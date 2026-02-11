"""Web dashboard routes (Jinja2 + HTMX)."""

from __future__ import annotations

import json

from fastapi import APIRouter, Depends, Request, Form
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.services.order_service import OrderService
from app.services.portfolio_service import PortfolioService
from app.services.strategy_service import StrategyService
from app.services.stock_memo_service import StockMemoService
from app.schemas.order import OrderCreate
from app.schemas.common import Market, OrderSide, OrderType, TradingMode
from app.web.stock_list import KR_STOCKS, US_STOCKS

import pathlib

templates = Jinja2Templates(directory=str(pathlib.Path(__file__).parent / "templates"))

web_router = APIRouter(tags=["web"])


@web_router.get("/", response_class=HTMLResponse)
async def dashboard(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.trading_mode.value)
    positions = await svc.get_positions(is_paper=settings.trading_mode == TradingMode.PAPER)
    snapshots = await svc.get_snapshots(settings.trading_mode.value)
    snapshots_data = [
        {"date": str(s.date), "value": s.total_value, "pnl": s.realized_pnl + s.unrealized_pnl}
        for s in reversed(snapshots)
    ]

    order_svc = OrderService(session)
    recent_orders = await order_svc.get_orders(limit=10)

    return templates.TemplateResponse("index.html", {
        "request": request,
        "summary": summary,
        "positions": positions[:5],
        "recent_orders": recent_orders,
        "snapshots_json": json.dumps(snapshots_data),
        "trading_mode": settings.trading_mode.value,
    })


@web_router.get("/orders", response_class=HTMLResponse)
async def orders_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = OrderService(session)
    orders = await svc.get_orders(limit=100)
    memo_svc = StockMemoService(session)
    memos = await memo_svc.list_all()
    return templates.TemplateResponse("orders.html", {
        "request": request,
        "orders": orders,
        "memos": memos,
        "trading_mode": settings.trading_mode.value,
    })


@web_router.post("/orders/submit", response_class=HTMLResponse)
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
    req = OrderCreate(
        symbol=symbol.upper(),
        market=Market(market),
        side=OrderSide(side),
        order_type=OrderType(order_type),
        quantity=quantity,
        price=price if order_type == "LIMIT" else None,
        trading_mode=TradingMode(settings.trading_mode.value),
    )
    svc = OrderService(session)
    await svc.create_order(req)

    orders = await svc.get_orders(limit=100)
    return templates.TemplateResponse("partials/order_table.html", {
        "request": request,
        "orders": orders,
    })


@web_router.get("/positions", response_class=HTMLResponse)
async def positions_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    is_paper = settings.trading_mode == TradingMode.PAPER
    positions = await svc.get_positions(is_paper=is_paper)
    return templates.TemplateResponse("positions.html", {
        "request": request,
        "positions": positions,
        "trading_mode": settings.trading_mode.value,
    })


@web_router.get("/portfolio", response_class=HTMLResponse)
async def portfolio_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.trading_mode.value)
    snapshots = await svc.get_snapshots(settings.trading_mode.value)
    snapshots_data = [
        {"date": str(s.date), "value": s.total_value, "pnl": s.realized_pnl + s.unrealized_pnl}
        for s in reversed(snapshots)
    ]
    return templates.TemplateResponse("portfolio.html", {
        "request": request,
        "summary": summary,
        "snapshots_json": json.dumps(snapshots_data),
        "trading_mode": settings.trading_mode.value,
    })


@web_router.get("/stocks", response_class=HTMLResponse)
async def stocks_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = StockMemoService(session)
    memos = await svc.list_all()
    return templates.TemplateResponse("stocks.html", {
        "request": request,
        "memos": memos,
        "kr_stocks": KR_STOCKS,
        "us_stocks": US_STOCKS,
        "trading_mode": settings.trading_mode.value,
    })


@web_router.post("/stocks/memo", response_class=HTMLResponse)
async def add_memo(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    name: str = Form(...),
    memo: str = Form(""),
    session: AsyncSession = Depends(get_session),
):
    svc = StockMemoService(session)
    try:
        await svc.add(symbol, market, name, memo or None)
    except Exception:
        pass  # ignore duplicate
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@web_router.delete("/stocks/memo/{memo_id}", response_class=HTMLResponse)
async def delete_memo(
    request: Request,
    memo_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = StockMemoService(session)
    await svc.remove(memo_id)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@web_router.get("/strategies", response_class=HTMLResponse)
async def strategies_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = StrategyService(session)
    strategies = await svc.list_all()
    strats = []
    for s in strategies:
        strats.append({
            "id": s.id,
            "name": s.name,
            "strategy_type": s.strategy_type,
            "symbols": json.loads(s.symbols) if isinstance(s.symbols, str) else s.symbols,
            "market": s.market,
            "params": json.loads(s.params) if isinstance(s.params, str) else s.params,
            "trading_mode": s.trading_mode,
            "is_active": s.is_active,
            "schedule_cron": s.schedule_cron,
        })
    memo_svc = StockMemoService(session)
    memos = await memo_svc.list_all()
    return templates.TemplateResponse("strategies.html", {
        "request": request,
        "strategies": strats,
        "memos": memos,
        "trading_mode": settings.trading_mode.value,
    })


# HTMX partials

@web_router.get("/partials/positions", response_class=HTMLResponse)
async def partial_positions(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    is_paper = settings.trading_mode == TradingMode.PAPER
    positions = await svc.get_positions(is_paper=is_paper)
    return templates.TemplateResponse("partials/position_table.html", {
        "request": request,
        "positions": positions,
    })


@web_router.get("/partials/orders", response_class=HTMLResponse)
async def partial_orders(request: Request, session: AsyncSession = Depends(get_session)):
    svc = OrderService(session)
    orders = await svc.get_orders(limit=50)
    return templates.TemplateResponse("partials/order_table.html", {
        "request": request,
        "orders": orders,
    })


@web_router.get("/partials/summary", response_class=HTMLResponse)
async def partial_summary(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.trading_mode.value)
    return templates.TemplateResponse("partials/summary_cards.html", {
        "request": request,
        "summary": summary,
    })


@web_router.get("/partials/memos", response_class=HTMLResponse)
async def partial_memos(request: Request, session: AsyncSession = Depends(get_session)):
    svc = StockMemoService(session)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@web_router.get("/partials/memo-options", response_class=HTMLResponse)
async def partial_memo_options(request: Request, session: AsyncSession = Depends(get_session)):
    svc = StockMemoService(session)
    memos = await svc.list_all()
    options = '<option value="">-- 메모 종목 선택 --</option>'
    for m in memos:
        options += f'<option value="{m.symbol}" data-market="{m.market}">{m.name} ({m.symbol} / {m.market})</option>'
    return HTMLResponse(options)
