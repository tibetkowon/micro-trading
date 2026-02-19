"""주문 관련 파셜 및 주문 제출 라우트."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, Form, Request
from fastapi.responses import HTMLResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.common import Market, OrderSide, OrderType, TradingMode
from app.schemas.order import OrderCreate
from app.services.order_service import OrderService
from app.services.portfolio_service import PortfolioService
from app.services.stock_master_service import StockMasterService
from app.web.routes._base import templates

logger = logging.getLogger(__name__)

router = APIRouter(tags=["web"])


@router.get("/partials/order-balance", response_class=HTMLResponse)
async def partial_order_balance(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """주문 폼에 표시할 주문 가능 잔고 — 거래 모드별 분기."""
    from sqlalchemy import select
    from app.models.account import Account

    mode = settings.get_trading_mode()
    if mode == TradingMode.REAL:
        from app.services.connection_service import ConnectionService
        real = await ConnectionService().get_real_balance()
        balance = real.get("cash_krw", 0.0) if not real.get("error") else 0.0
    else:
        result = await session.execute(select(Account).limit(1))
        account = result.scalar_one_or_none()
        balance = account.paper_balance_krw if account else 0.0

    return templates.TemplateResponse("partials/order_balance.html", {
        "request": request,
        "balance": balance,
    })


@router.get("/partials/portfolio-compact", response_class=HTMLResponse)
async def partial_portfolio_compact(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """포트폴리오 요약 파셜 (우측 사이드바)."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    return templates.TemplateResponse("partials/portfolio_compact.html", {
        "request": request,
        "summary": summary,
    })


@router.get("/partials/positions-compact", response_class=HTMLResponse)
async def partial_positions_compact(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """보유 포지션 목록 파셜 (우측 사이드바)."""
    svc = PortfolioService(session)
    positions = await svc.get_positions(
        is_paper=settings.get_trading_mode() == TradingMode.PAPER
    )
    return templates.TemplateResponse("partials/positions_compact.html", {
        "request": request,
        "positions": positions,
    })


@router.get("/partials/orders", response_class=HTMLResponse)
async def partial_orders(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """주문 내역 테이블 파셜 (commission_map 포함)."""
    svc = OrderService(session)
    orders = await svc.get_orders(limit=10)
    commission_map = await svc.get_trades_by_order_ids([o.id for o in orders])
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk(
        list({(o.symbol, o.market) for o in orders})
    )
    return templates.TemplateResponse("partials/order_table.html", {
        "request": request,
        "orders": orders,
        "commission_map": commission_map,
        "name_map": name_map,
    })


@router.get("/partials/orders-compact", response_class=HTMLResponse)
async def partial_orders_compact(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """최근 주문 콤팩트 파셜 (우측 사이드바)."""
    svc = OrderService(session)
    orders = await svc.get_orders(limit=10)
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk(
        list({(o.symbol, o.market) for o in orders})
    )
    return templates.TemplateResponse("partials/orders_compact.html", {
        "request": request,
        "orders": orders,
        "name_map": name_map,
    })


@router.get("/partials/positions", response_class=HTMLResponse)
async def partial_positions(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """보유 포지션 테이블 파셜 (레거시 페이지용)."""
    svc = PortfolioService(session)
    positions = await svc.get_positions(
        is_paper=settings.get_trading_mode() == TradingMode.PAPER
    )
    return templates.TemplateResponse("partials/position_table.html", {
        "request": request,
        "positions": positions,
    })


@router.get("/partials/summary", response_class=HTMLResponse)
async def partial_summary(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """포트폴리오 요약 카드 파셜 (레거시 페이지용)."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    return templates.TemplateResponse("partials/summary_cards.html", {
        "request": request,
        "summary": summary,
    })


@router.post("/orders/submit", response_class=HTMLResponse)
async def submit_order(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    side: str = Form(...),
    order_type: str = Form("MARKET"),
    quantity: int = Form(...),
    price: float | None = Form(None),
    session: AsyncSession = Depends(get_session),
):
    """주문 폼 제출 — 결과 HTML을 반환하고 refreshSidebar 이벤트를 트리거한다."""
    req = OrderCreate(
        symbol=symbol.upper(),
        market=Market(market),
        side=OrderSide(side),
        order_type=OrderType(order_type),
        quantity=quantity,
        price=price if order_type == "LIMIT" else None,
        trading_mode=TradingMode(settings.get_trading_mode().value),
    )
    svc = OrderService(session)

    try:
        await svc.create_order(req)
        side_label = "매수" if side == "BUY" else "매도"
        result_html = (
            f'<div class="order-result success">'
            f'{symbol} {side_label} {quantity}주 주문이 접수되었습니다.'
            f'</div>'
        )
    except Exception as e:
        result_html = (
            f'<div class="order-result error">'
            f'주문 실패: {e}'
            f'</div>'
        )

    resp = HTMLResponse(result_html)
    resp.headers["HX-Trigger"] = "refreshSidebar"
    return resp
