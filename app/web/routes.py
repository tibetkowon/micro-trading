"""Web dashboard routes (Jinja2 + HTMX)."""

from __future__ import annotations

import json
import logging

from fastapi import APIRouter, Depends, Request, Form
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.services.market_service import MarketService
from app.services.order_service import OrderService
from app.services.portfolio_service import PortfolioService
from app.services.strategy_service import StrategyService
from app.services.watchlist_service import WatchlistService
from app.services.stock_master_service import StockMasterService
from app.schemas.order import OrderCreate
from app.schemas.common import Market, OrderSide, OrderType, TradingMode
from app.web.stock_list import KR_STOCKS

import pathlib

logger = logging.getLogger(__name__)

templates = Jinja2Templates(directory=str(pathlib.Path(__file__).parent / "templates"))

web_router = APIRouter(tags=["web"])


# ──────────────────────────────────────────────
# Helper: build stock name lookup
# ──────────────────────────────────────────────

_STOCK_NAMES: dict[str, str] = {}
for _s in KR_STOCKS:
    _STOCK_NAMES[_s["symbol"]] = _s["name"]


async def _get_stock_name(symbol: str, market: str, session: AsyncSession) -> str:
    """Resolve a human-readable name for a symbol."""
    if symbol in _STOCK_NAMES:
        return _STOCK_NAMES[symbol]
    # DB 마스터에서 검색
    from app.models.stock_master import StockMaster
    from sqlalchemy import select
    result = await session.execute(
        select(StockMaster.name).where(
            StockMaster.symbol == symbol, StockMaster.market == market
        )
    )
    name = result.scalar_one_or_none()
    if name:
        return name
    # 관심종목에서 검색
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    for m in memos:
        if m.symbol == symbol:
            return m.name
    return symbol


# ──────────────────────────────────────────────
# Main trading view
# ──────────────────────────────────────────────

@web_router.get("/", response_class=HTMLResponse)
async def trading_view(request: Request, session: AsyncSession = Depends(get_session)):
    """3-panel trading view (main page)."""
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

    # Watchlist items (memos)
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()

    return templates.TemplateResponse("trading.html", {
        "request": request,
        "summary": summary,
        "positions": positions,
        "recent_orders": recent_orders,
        "orders": recent_orders,
        "snapshots_json": json.dumps(snapshots_data),
        "trading_mode": settings.get_trading_mode().value,
        "selected_symbol": None,
        "memos": memos,
    })


@web_router.get("/stock/{symbol}", response_class=HTMLResponse)
async def stock_page(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """Full page load with a specific stock selected."""
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    positions = await svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)

    order_svc = OrderService(session)
    recent_orders = await order_svc.get_orders(limit=10)

    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()

    stock_name = await _get_stock_name(symbol, market, session)

    # Get price for the selected stock
    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    # Find position for this stock
    position = None
    for p in positions:
        if p.symbol == symbol and p.market == market:
            position = p
            break

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


# ──────────────────────────────────────────────
# Watchlist partials (left panel)
# ──────────────────────────────────────────────

@web_router.get("/partials/watchlist/items", response_class=HTMLResponse)
async def partial_watchlist_items(
    request: Request,
    tab: str = "watchlist",
    session: AsyncSession = Depends(get_session),
):
    """Return watchlist items with prices for the given tab."""
    market_svc = MarketService(session)
    items = []

    if tab == "watchlist":
        memo_svc = WatchlistService(session)
        memos = await memo_svc.list_all()
        for m in memos:
            try:
                p = await market_svc.get_price(m.symbol, m.market)
                items.append({
                    "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                    "price": p.price, "change": p.change, "change_pct": p.change_pct,
                })
            except Exception:
                items.append({
                    "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                    "price": 0, "change": 0, "change_pct": 0,
                })
    elif tab == "kr":
        for s in KR_STOCKS:
            try:
                p = await market_svc.get_price(s["symbol"], "KR")
                items.append({
                    "symbol": s["symbol"], "market": "KR", "name": s["name"],
                    "price": p.price, "change": p.change, "change_pct": p.change_pct,
                })
            except Exception:
                items.append({
                    "symbol": s["symbol"], "market": "KR", "name": s["name"],
                    "price": 0, "change": 0, "change_pct": 0,
                })
    return templates.TemplateResponse("partials/watchlist_items.html", {
        "request": request,
        "items": items,
        "tab": tab,
        "selected_symbol": None,
    })


@web_router.get("/partials/watchlist/search", response_class=HTMLResponse)
async def partial_watchlist_search(
    request: Request,
    q: str = "",
    session: AsyncSession = Depends(get_session),
):
    """종목 검색 — DB 기반, 초성 검색 지원."""
    q = q.strip()
    if not q:
        return HTMLResponse("")

    svc = StockMasterService(session)
    results = await svc.search(q, limit=15)

    return templates.TemplateResponse("partials/watchlist_search_results.html", {
        "request": request,
        "results": results,
        "q": q,
    })


# ──────────────────────────────────────────────
# Dashboard partials
# ──────────────────────────────────────────────

@web_router.get("/partials/dashboard/watchlist-prices", response_class=HTMLResponse)
async def partial_dashboard_watchlist_prices(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """대시보드 관심종목 실시간 시세 카드 그리드."""
    market_svc = MarketService(session)
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    items = []
    for m in memos:
        try:
            p = await market_svc.get_price(m.symbol, m.market)
            items.append({
                "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": p.price, "change": p.change, "change_pct": p.change_pct,
            })
        except Exception:
            items.append({
                "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": 0, "change": 0, "change_pct": 0,
            })
    return templates.TemplateResponse("partials/dashboard_watchlist_prices.html", {
        "request": request,
        "items": items,
    })


# ──────────────────────────────────────────────
# Stock detail partials (center panel)
# ──────────────────────────────────────────────

@web_router.get("/partials/stock-detail/{symbol}", response_class=HTMLResponse)
async def partial_stock_detail(
    request: Request,
    symbol: str,
    market: str = "KR",
    name: str = "",
    session: AsyncSession = Depends(get_session),
):
    """Load stock detail into center panel via HTMX."""
    stock_name = name or await _get_stock_name(symbol, market, session)

    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    # Find position
    portfolio_svc = PortfolioService(session)
    positions = await portfolio_svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)
    position = None
    for p in positions:
        if p.symbol == symbol and p.market == market:
            position = p
            break

    return templates.TemplateResponse("partials/stock_detail.html", {
        "request": request,
        "symbol": symbol,
        "market": market,
        "stock_name": stock_name,
        "price_info": price_info,
        "position": position,
    })


@web_router.get("/partials/stock-price/{symbol}", response_class=HTMLResponse)
async def partial_stock_price(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """Real-time price hero (10s polling)."""
    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    return templates.TemplateResponse("partials/stock_price_hero.html", {
        "request": request,
        "price_info": price_info,
        "market": market,
    })


@web_router.get("/partials/stock-position/{symbol}", response_class=HTMLResponse)
async def partial_stock_position(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """Position card for a specific stock."""
    svc = PortfolioService(session)
    positions = await svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)
    position = None
    for p in positions:
        if p.symbol == symbol and p.market == market:
            position = p
            break

    return templates.TemplateResponse("partials/stock_position_card.html", {
        "request": request,
        "position": position,
    })


# ──────────────────────────────────────────────
# Order helper partials
# ──────────────────────────────────────────────

@web_router.get("/partials/order-balance", response_class=HTMLResponse)
async def partial_order_balance(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """주문 폼에 표시할 주문 가능 잔고 — 거래 모드별 분기."""
    from app.models.account import Account
    from sqlalchemy import select
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


# ──────────────────────────────────────────────
# Portfolio sidebar partials (right panel)
# ──────────────────────────────────────────────

@web_router.get("/partials/portfolio-compact", response_class=HTMLResponse)
async def partial_portfolio_compact(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    return templates.TemplateResponse("partials/portfolio_compact.html", {
        "request": request,
        "summary": summary,
    })


@web_router.get("/partials/positions-compact", response_class=HTMLResponse)
async def partial_positions_compact(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    positions = await svc.get_positions(is_paper=settings.get_trading_mode() == TradingMode.PAPER)
    return templates.TemplateResponse("partials/positions_compact.html", {
        "request": request,
        "positions": positions,
    })


@web_router.get("/partials/orders-compact", response_class=HTMLResponse)
async def partial_orders_compact(request: Request, session: AsyncSession = Depends(get_session)):
    svc = OrderService(session)
    orders = await svc.get_orders(limit=10)
    return templates.TemplateResponse("partials/orders_compact.html", {
        "request": request,
        "orders": orders,
    })


# ──────────────────────────────────────────────
# Order submission (Phase 6 - inline result)
# ──────────────────────────────────────────────

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
        trading_mode=TradingMode(settings.get_trading_mode().value),
    )
    svc = OrderService(session)

    try:
        order = await svc.create_order(req)
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

    # Check if request is from the new trading view (HTMX to #order-result)
    hx_target = request.headers.get("hx-target", "")
    if hx_target == "order-result":
        from fastapi.responses import HTMLResponse as HR
        resp = HR(result_html)
        # Trigger sidebar refresh
        resp.headers["HX-Trigger"] = "refreshSidebar"
        return resp

    # Legacy: orders page expects full order table
    orders = await svc.get_orders(limit=100)
    commission_map = await svc.get_trades_by_order_ids([o.id for o in orders])
    return templates.TemplateResponse("partials/order_table.html", {
        "request": request,
        "orders": orders,
        "commission_map": commission_map,
    })


# ──────────────────────────────────────────────
# Legacy pages (kept accessible)
# ──────────────────────────────────────────────

@web_router.get("/orders", response_class=HTMLResponse)
async def orders_page(request: Request, session: AsyncSession = Depends(get_session)):
    svc = OrderService(session)
    orders = await svc.get_orders(limit=100)
    commission_map = await svc.get_trades_by_order_ids([o.id for o in orders])
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    return templates.TemplateResponse("orders.html", {
        "request": request,
        "orders": orders,
        "commission_map": commission_map,
        "memos": memos,
        "trading_mode": settings.get_trading_mode().value,
    })



@web_router.get("/portfolio", response_class=HTMLResponse)
async def portfolio_page(request: Request, session: AsyncSession = Depends(get_session)):
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



@web_router.post("/stocks/memo", response_class=HTMLResponse)
async def add_memo(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    name: str = Form(...),
    memo: str = Form(""),
    session: AsyncSession = Depends(get_session),
):
    svc = WatchlistService(session)
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
    svc = WatchlistService(session)
    await svc.remove(memo_id)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@web_router.post("/watchlist/item", response_class=HTMLResponse)
async def add_watchlist_item(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    name: str = Form(...),
    memo: str = Form(""),
    session: AsyncSession = Depends(get_session),
):
    """관심종목 패널에서 종목 추가 후 목록 갱신."""
    svc = WatchlistService(session)
    try:
        await svc.add(symbol, market, name, memo or None)
    except Exception:
        pass  # 중복 무시
    market_svc = MarketService(session)
    memos = await svc.list_all()
    items = []
    for m in memos:
        try:
            p = await market_svc.get_price(m.symbol, m.market)
            items.append({
                "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": p.price, "change": p.change, "change_pct": p.change_pct,
            })
        except Exception:
            items.append({
                "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": 0, "change": 0, "change_pct": 0,
            })
    return templates.TemplateResponse("partials/watchlist_items.html", {
        "request": request,
        "items": items,
        "tab": "watchlist",
        "selected_symbol": None,
    })


@web_router.delete("/watchlist/item/{item_id}", response_class=HTMLResponse)
async def delete_watchlist_item(
    request: Request,
    item_id: int,
    session: AsyncSession = Depends(get_session),
):
    """관심종목 패널에서 종목 제거 후 목록 갱신."""
    svc = WatchlistService(session)
    await svc.remove(item_id)
    market_svc = MarketService(session)
    memos = await svc.list_all()
    items = []
    for m in memos:
        try:
            p = await market_svc.get_price(m.symbol, m.market)
            items.append({
                "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": p.price, "change": p.change, "change_pct": p.change_pct,
            })
        except Exception:
            items.append({
                "id": m.id, "symbol": m.symbol, "market": m.market, "name": m.name,
                "price": 0, "change": 0, "change_pct": 0,
            })
    return templates.TemplateResponse("partials/watchlist_items.html", {
        "request": request,
        "items": items,
        "tab": "watchlist",
        "selected_symbol": None,
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
    memo_svc = WatchlistService(session)
    memos = await memo_svc.list_all()
    return templates.TemplateResponse("strategies.html", {
        "request": request,
        "strategies": strats,
        "memos": memos,
        "trading_mode": settings.get_trading_mode().value,
    })


# ──────────────────────────────────────────────
# Existing HTMX partials (for legacy pages)
# ──────────────────────────────────────────────

@web_router.get("/partials/positions", response_class=HTMLResponse)
async def partial_positions(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    is_paper = settings.get_trading_mode() == TradingMode.PAPER
    positions = await svc.get_positions(is_paper=is_paper)
    return templates.TemplateResponse("partials/position_table.html", {
        "request": request,
        "positions": positions,
    })


@web_router.get("/partials/orders", response_class=HTMLResponse)
async def partial_orders(request: Request, session: AsyncSession = Depends(get_session)):
    svc = OrderService(session)
    orders = await svc.get_orders(limit=50)
    commission_map = await svc.get_trades_by_order_ids([o.id for o in orders])
    return templates.TemplateResponse("partials/order_table.html", {
        "request": request,
        "orders": orders,
        "commission_map": commission_map,
    })


@web_router.get("/partials/summary", response_class=HTMLResponse)
async def partial_summary(request: Request, session: AsyncSession = Depends(get_session)):
    svc = PortfolioService(session)
    summary = await svc.get_summary(settings.get_trading_mode().value)
    return templates.TemplateResponse("partials/summary_cards.html", {
        "request": request,
        "summary": summary,
    })


@web_router.get("/partials/memos", response_class=HTMLResponse)
async def partial_memos(request: Request, session: AsyncSession = Depends(get_session)):
    svc = WatchlistService(session)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@web_router.get("/partials/memo-options", response_class=HTMLResponse)
async def partial_memo_options(request: Request, session: AsyncSession = Depends(get_session)):
    svc = WatchlistService(session)
    memos = await svc.list_all()
    options = '<option value="">-- 메모 종목 선택 --</option>'
    for m in memos:
        options += f'<option value="{m.symbol}" data-market="{m.market}">{m.name} ({m.symbol} / {m.market})</option>'
    return HTMLResponse(options)


# ──────────────────────────────────────────────
# Settings (Phase 9 & 10)
# ──────────────────────────────────────────────

@web_router.get("/settings", response_class=HTMLResponse)
async def settings_page(request: Request):
    """설정 페이지."""
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


@web_router.post("/settings/test-connection", response_class=HTMLResponse)
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


@web_router.get("/partials/account-info", response_class=HTMLResponse)
async def partial_account_info(request: Request):
    """계좌 잔고 정보 partial."""
    from app.services.connection_service import ConnectionService
    conn_svc = ConnectionService()
    paper_balance = await conn_svc.get_paper_balance()
    real_balance = await conn_svc.get_real_balance()

    return templates.TemplateResponse("partials/account_info.html", {
        "request": request,
        "paper_balance": paper_balance,
        "real_balance": real_balance,
        "commission_rate": settings.paper_commission_rate,
        "real_commission_rate": settings.real_commission_rate,
    })


@web_router.post("/settings/trading-mode", response_class=HTMLResponse)
async def switch_trading_mode(request: Request, mode: str = Form(...)):
    """런타임 거래 모드 전환 (Phase 10)."""
    from app.services.connection_service import ConnectionService

    try:
        target_mode = TradingMode(mode)
    except ValueError:
        return HTMLResponse(f"잘못된 거래 모드 값입니다: {mode}", status_code=400)

    # REAL 전환 시 KIS API 키 확인
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
