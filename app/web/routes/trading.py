"""메인 트레이딩 뷰 및 포트폴리오·전략 페이지 라우트."""

from __future__ import annotations

import json
import logging

from fastapi import APIRouter, Depends, Request
from fastapi.responses import HTMLResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.common import TradingMode
from app.services.market_service import MarketService
from app.services.order_service import OrderService
from app.services.portfolio_service import PortfolioService
from app.services.strategy_service import StrategyService
from app.services.watchlist_service import WatchlistService
from app.web.routes._base import get_stock_name, templates

logger = logging.getLogger(__name__)

router = APIRouter(tags=["web"])


@router.get("/", response_class=HTMLResponse)
async def trading_view(request: Request, session: AsyncSession = Depends(get_session)):
    """3-패널 메인 트레이딩 뷰."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    positions = await svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)
    snapshots = await svc.get_snapshots(settings.get_trading_mode().value)
    snapshots_data = [
        {"date": str(s.date), "value": s.total_value, "pnl": s.realized_pnl + s.unrealized_pnl}
        for s in reversed(snapshots)
    ]

    order_svc = OrderService(session)
    recent_orders = await order_svc.get_orders(limit=10)
    commission_map = await order_svc.get_trades_by_order_ids([o.id for o in recent_orders])

    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()

    return templates.TemplateResponse("trading.html", {
        "request": request,
        "summary": summary,
        "positions": positions,
        "recent_orders": recent_orders,
        "orders": recent_orders,
        "commission_map": commission_map,
        "snapshots_json": json.dumps(snapshots_data),
        "trading_mode": settings.get_trading_mode().value,
        "selected_symbol": None,
        "memos": memos,
    })


@router.get("/stock/{symbol}", response_class=HTMLResponse)
async def stock_page(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """특정 종목이 선택된 상태로 전체 페이지 로드."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    positions = await svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)

    order_svc = OrderService(session)
    recent_orders = await order_svc.get_orders(limit=10)

    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()

    stock_name = await get_stock_name(symbol, market, session)

    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    position = next((p for p in positions if p.symbol == symbol and p.market == market), None)

    return templates.TemplateResponse("trading.html", {
        "request": request,
        "summary": summary,
        "positions": positions,
        "recent_orders": recent_orders,
        "orders": recent_orders,
        "snapshots_json": "[]",
        "trading_mode": settings.get_trading_mode().value,
        "selected_symbol": symbol,
        "symbol": symbol,
        "market": market,
        "stock_name": stock_name,
        "price_info": price_info,
        "position": position,
        "memos": memos,
    })


@router.get("/portfolio", response_class=HTMLResponse)
async def portfolio_page(request: Request, session: AsyncSession = Depends(get_session)):
    """포트폴리오 차트 페이지."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    snapshots = await svc.get_snapshots(settings.get_trading_mode().value)
    snapshots_data = [
        {"date": str(s.date), "value": s.total_value, "pnl": s.realized_pnl + s.unrealized_pnl}
        for s in reversed(snapshots)
    ]
    return templates.TemplateResponse("portfolio.html", {
        "request": request,
        "summary": summary,
        "snapshots_json": json.dumps(snapshots_data),
        "trading_mode": settings.get_trading_mode().value,
    })


@router.get("/strategies", response_class=HTMLResponse)
async def strategies_page(request: Request, session: AsyncSession = Depends(get_session)):
    """자동매매 전략 관리 페이지."""
    svc = StrategyService(session)
    strategies = await svc.list_all()
    strats = [
        {
            "id": s.id,
            "name": s.name,
            "strategy_type": s.strategy_type,
            "symbols": json.loads(s.symbols) if isinstance(s.symbols, str) else s.symbols,
            "market": s.market,
            "params": json.loads(s.params) if isinstance(s.params, str) else s.params,
            "trading_mode": s.trading_mode,
            "is_active": s.is_active,
            "schedule_cron": s.schedule_cron,
        }
        for s in strategies
    ]
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    return templates.TemplateResponse("strategies.html", {
        "request": request,
        "strategies": strats,
        "memos": memos,
        "trading_mode": settings.get_trading_mode().value,
    })
