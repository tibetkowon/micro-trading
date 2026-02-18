"""KIS API 연결 상태 확인 및 관리 서비스."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import datetime

from app.config import settings
from app.schemas.common import TradingMode

logger = logging.getLogger(__name__)


@dataclass
class ConnectionStatus:
    """KIS API 연결 상태 정보."""
    has_app_key: bool = False
    has_app_secret: bool = False
    has_account_number: bool = False
    is_mock_server: bool = True
    base_url: str = ""
    connected: bool = False
    last_test_time: str | None = None
    error_message: str | None = None


class ConnectionService:
    """KIS API 연결 테스트 및 상태 관리."""

    def get_connection_status(self) -> ConnectionStatus:
        """API 키 설정 여부 및 서버 정보 반환."""
        return ConnectionStatus(
            has_app_key=bool(settings.kis_app_key),
            has_app_secret=bool(settings.kis_app_secret),
            has_account_number=bool(settings.kis_account_number),
            is_mock_server="vts" in settings.kis_base_url.lower(),
            base_url=settings.kis_base_url,
        )

    def is_kis_configured(self) -> bool:
        """KIS API 키가 모두 설정되었는지 확인."""
        return bool(
            settings.kis_app_key
            and settings.kis_app_secret
            and settings.kis_account_number
        )

    async def test_connection(self) -> ConnectionStatus:
        """KIS API 연결 테스트 — 토큰 발급 및 잔고 조회 시도."""
        status = self.get_connection_status()
        status.last_test_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        if not self.is_kis_configured():
            status.connected = False
            status.error_message = "API 키가 설정되지 않았습니다."
            return status

        try:
            from app.broker.factory import get_broker
            broker = await get_broker(TradingMode.REAL)
            balance = await broker.get_balance()
            status.connected = True
            status.error_message = None
            logger.info(
                "KIS 연결 테스트 성공 — 잔고: %s원",
                f"{balance.cash_krw:,.0f}",
            )
        except Exception as e:
            status.connected = False
            status.error_message = str(e)
            logger.error("KIS 연결 테스트 실패: %s", e)

        return status

    async def get_real_balance(self) -> dict:
        """KIS API로 실계좌 잔고 조회."""
        if not self.is_kis_configured():
            return {"error": "API 키가 설정되지 않았습니다."}

        try:
            from app.broker.factory import get_broker
            broker = await get_broker(TradingMode.REAL)
            balance = await broker.get_balance()
            return {
                "cash_krw": balance.cash_krw,
                "cash_usd": balance.cash_usd,
                "total_value_krw": balance.total_value_krw,
                "total_value_usd": balance.total_value_usd,
            }
        except Exception as e:
            return {"error": str(e)}

    async def get_paper_balance(self) -> dict:
        """모의투자 잔고 조회."""
        try:
            from app.broker.factory import get_broker
            broker = await get_broker(TradingMode.PAPER)
            balance = await broker.get_balance()
            return {
                "cash_krw": balance.cash_krw,
                "cash_usd": balance.cash_usd,
                "total_value_krw": balance.total_value_krw,
                "total_value_usd": balance.total_value_usd,
            }
        except Exception as e:
            return {"error": str(e)}

    @staticmethod
    def mask_key(key: str) -> str:
        """API 키를 마스킹하여 반환 (앞 4자리 + ****)."""
        if not key:
            return ""
        if len(key) <= 4:
            return "*" * len(key)
        return key[:4] + "****"  # 길이 고정으로 화면 오버플로우 방지
