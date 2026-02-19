"""KIS OpenAPI HTTP client with OAuth token management."""

from __future__ import annotations

import asyncio
import hashlib
import json
import logging
from datetime import datetime, timedelta
from typing import Any

import httpx

from app.config import settings
from app.broker.kis.endpoints import HASHKEY_PATH, TOKEN_PATH
from app.broker.kis.models import KISToken

logger = logging.getLogger(__name__)


class KISClient:
    """Low-level HTTP client for KIS OpenAPI."""

    def __init__(self):
        self._client: httpx.AsyncClient | None = None
        self._token = KISToken()
        self._is_mock = "vts" in settings.kis_base_url.lower()

    @property
    def is_mock(self) -> bool:
        return self._is_mock

    async def open(self):
        self._client = httpx.AsyncClient(
            base_url=settings.kis_base_url,
            timeout=30.0,
            verify=not self._is_mock,  # VTS 서버 인증서가 실서버 도메인으로 발급됨
        )

    async def close(self):
        if self._client:
            await self._client.aclose()
            self._client = None

    async def _ensure_token(self):
        if not self._token.is_expired:
            return
        if not self._client:
            await self.open()
        resp = await self._client.post(
            TOKEN_PATH,
            json={
                "grant_type": "client_credentials",
                "appkey": settings.kis_app_key,
                "appsecret": settings.kis_app_secret,
            },
        )
        resp.raise_for_status()
        data = resp.json()
        self._token = KISToken(
            access_token=data["access_token"],
            token_type=data.get("token_type", "Bearer"),
            expires_at=datetime.now() + timedelta(hours=23),
        )
        logger.info("KIS token refreshed (mock=%s)", self._is_mock)

    async def force_refresh_token(self) -> None:
        """토큰 강제 갱신 — 선제 갱신 스케줄러 및 401 재시도에 사용."""
        self._token = KISToken()  # 만료 상태로 초기화
        await self._ensure_token()

    async def _get_hashkey(self, body: dict) -> str:
        if not self._client:
            await self.open()
        resp = await self._client.post(
            HASHKEY_PATH,
            json=body,
            headers={
                "content-type": "application/json; charset=utf-8",
                "appKey": settings.kis_app_key,
                "appSecret": settings.kis_app_secret,
            },
        )
        if not resp.is_success:
            logger.error("해시키 발급 실패 [%s]: %s", resp.status_code, resp.text)
        resp.raise_for_status()
        return resp.json()["HASH"]

    def _base_headers(self, tr_id: str) -> dict[str, str]:
        return {
            "content-type": "application/json; charset=utf-8",
            "authorization": f"{self._token.token_type} {self._token.access_token}",
            "appkey": settings.kis_app_key,
            "appsecret": settings.kis_app_secret,
            "tr_id": tr_id,
            "custtype": "P",  # 개인투자자 구분 (KIS API 필수 헤더)
        }

    async def get(self, path: str, tr_id: str, params: dict[str, str] | None = None) -> dict[str, Any]:
        await self._ensure_token()
        if not self._client:
            await self.open()
        headers = self._base_headers(tr_id)
        resp = await self._client.get(path, headers=headers, params=params)
        if resp.status_code == 401:
            # 세션 만료로 인한 401 — 토큰 강제 갱신 후 1회 재시도
            logger.warning("KIS 401 응답 — 토큰 강제 갱신 후 재시도: %s", path)
            await self.force_refresh_token()
            headers = self._base_headers(tr_id)
            resp = await self._client.get(path, headers=headers, params=params)
        if not resp.is_success:
            logger.error("KIS GET 오류 [%s] %s: %s", resp.status_code, path, resp.text)
        resp.raise_for_status()
        return resp.json()

    async def post(
        self,
        path: str,
        tr_id: str,
        body: dict[str, Any],
        use_hashkey: bool = True,
        _retry: int = 2,
    ) -> dict[str, Any]:
        await self._ensure_token()
        if not self._client:
            await self.open()
        headers = self._base_headers(tr_id)
        if use_hashkey:
            headers["hashkey"] = await self._get_hashkey(body)
        resp = await self._client.post(path, headers=headers, json=body)

        if resp.status_code == 401:
            # 세션 만료로 인한 401 — 토큰 강제 갱신 후 1회 재시도
            logger.warning("KIS 401 응답 — 토큰 강제 갱신 후 재시도: %s (tr_id=%s)", path, tr_id)
            await self.force_refresh_token()
            headers = self._base_headers(tr_id)
            if use_hashkey:
                headers["hashkey"] = await self._get_hashkey(body)
            resp = await self._client.post(path, headers=headers, json=body)

        # 5xx 서버 오류는 일시적 장애일 수 있으므로 최대 _retry회 재시도
        if resp.status_code >= 500 and _retry > 0:
            logger.warning(
                "KIS POST 5xx 오류 [%s] — %d회 재시도 예정: %s (tr_id=%s)",
                resp.status_code, _retry, path, tr_id,
            )
            await asyncio.sleep(1)
            return await self.post(path, tr_id, body, use_hashkey=use_hashkey, _retry=_retry - 1)

        if not resp.is_success:
            logger.error(
                "KIS POST 오류 [%s] %s (tr_id=%s): %s",
                resp.status_code, path, tr_id, resp.text,
            )
        resp.raise_for_status()
        return resp.json()
