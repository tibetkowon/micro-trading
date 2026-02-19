"""설정 페이지 및 계좌 정보 파셜 라우트."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, Form, Request
from fastapi.responses import HTMLResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.common import TradingMode
from app.web.routes._base import templates

logger = logging.getLogger(__name__)

router = APIRouter(tags=["web"])


@router.get("/settings", response_class=HTMLResponse)
async def settings_page(request: Request):
    """설정 페이지 — KIS API 키 및 거래 모드."""
    from app.services.connection_service import ConnectionService
    conn_svc = ConnectionService()
    status = conn_svc.get_connection_status()
    mode = settings.get_trading_mode()

    return templates.TemplateResponse("settings.html", {
        "request": request,
        "trading_mode": mode.value,
        "connection_status": status,
        "kis_configured": conn_svc.is_kis_configured(),
        "masked_app_key": conn_svc.mask_key(settings.kis_app_key),
        "masked_app_secret": conn_svc.mask_key(settings.kis_app_secret),
        "masked_account": conn_svc.mask_key(settings.kis_account_number),
    })


@router.post("/settings/test-connection", response_class=HTMLResponse)
async def test_connection(request: Request):
    """KIS API 연결 테스트 (HTMX)."""
    from app.services.connection_service import ConnectionService
    conn_svc = ConnectionService()
    status = await conn_svc.test_connection()

    if status.connected:
        html = (
            '<div class="order-result success">'
            f'연결 성공 — {status.last_test_time}'
            '</div>'
        )
    else:
        html = (
            '<div class="order-result error">'
            f'연결 실패: {status.error_message}'
            '</div>'
        )
    return HTMLResponse(html)


@router.get("/partials/account-info", response_class=HTMLResponse)
async def partial_account_info(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """계좌 잔고 및 주문가능금액 파셜."""
    from app.services.connection_service import ConnectionService
    conn_svc = ConnectionService()
    paper_balance = await conn_svc.get_paper_balance()
    real_balance = await conn_svc.get_real_balance()

    paper_cash = paper_balance.get("cash_krw", 0.0) if not paper_balance.get("error") else 0.0
    paper_orderable = round(paper_cash / (1 + settings.paper_commission_rate), 0) if paper_cash > 0 else 0.0

    real_cash = real_balance.get("cash_krw", 0.0) if not real_balance.get("error") else 0.0
    real_orderable = round(real_cash / (1 + 0.003), 0) if real_cash > 0 else 0.0  # 실계좌 0.3% 안전 마진

    return templates.TemplateResponse("partials/account_info.html", {
        "request": request,
        "paper_balance": paper_balance,
        "real_balance": real_balance,
        "paper_orderable": paper_orderable,
        "real_orderable": real_orderable,
        "commission_rate": settings.paper_commission_rate,
        "real_commission_rate": settings.real_commission_rate,
    })


@router.post("/settings/trading-mode", response_class=HTMLResponse)
async def switch_trading_mode(request: Request, mode: str = Form(...)):
    """런타임 거래 모드 전환 — 전환 후 페이지 새로고침."""
    from app.services.connection_service import ConnectionService

    try:
        target_mode = TradingMode(mode)
    except ValueError:
        return HTMLResponse(f"잘못된 거래 모드 값입니다: {mode}", status_code=400)

    if target_mode == TradingMode.REAL:
        conn_svc = ConnectionService()
        if not conn_svc.is_kis_configured():
            return HTMLResponse(
                "KIS API 키가 설정되지 않아 실매매 모드로 전환할 수 없습니다.",
                status_code=400,
            )

    settings.switch_trading_mode(target_mode)
    resp = HTMLResponse("OK")
    resp.headers["HX-Refresh"] = "true"
    return resp
