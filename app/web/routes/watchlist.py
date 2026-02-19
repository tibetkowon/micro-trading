"""관심종목 패널 파셜 및 뮤테이션 라우트."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, Form, Request
from fastapi.responses import HTMLResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.services.market_service import MarketService
from app.services.watchlist_service import WatchlistService
from app.services.stock_master_service import StockMasterService
from app.web.routes._base import templates
from app.web.stock_list import KR_STOCKS

logger = logging.getLogger(__name__)

router = APIRouter(tags=["web"])


async def _build_watchlist_items(
    svc: WatchlistService,
    market_svc: MarketService,
) -> list[dict]:
    """관심종목 목록에 시세를 붙여 반환한다."""
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
    return items


@router.get("/partials/watchlist/items", response_class=HTMLResponse)
async def partial_watchlist_items(
    request: Request,
    tab: str = "watchlist",
    session: AsyncSession = Depends(get_session),
):
    """탭별 관심종목 목록과 시세를 반환한다."""
    market_svc = MarketService(session)
    items = []

    if tab == "watchlist":
        memo_svc = WatchlistService(session)
        items = await _build_watchlist_items(memo_svc, market_svc)
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


@router.get("/partials/watchlist/search", response_class=HTMLResponse)
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


@router.get("/partials/dashboard/watchlist-prices", response_class=HTMLResponse)
async def partial_dashboard_watchlist_prices(
    request: Request,
    session: AsyncSession = Depends(get_session),
):
    """대시보드 관심종목 실시간 시세 카드 그리드."""
    market_svc = MarketService(session)
    memo_svc = WatchlistService(session)
    items = await _build_watchlist_items(memo_svc, market_svc)
    return templates.TemplateResponse("partials/dashboard_watchlist_prices.html", {
        "request": request,
        "items": items,
    })


@router.post("/watchlist/item", response_class=HTMLResponse)
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
    items = await _build_watchlist_items(svc, market_svc)
    return templates.TemplateResponse("partials/watchlist_items.html", {
        "request": request,
        "items": items,
        "tab": "watchlist",
        "selected_symbol": None,
    })


@router.delete("/watchlist/item/{item_id}", response_class=HTMLResponse)
async def delete_watchlist_item(
    request: Request,
    item_id: int,
    session: AsyncSession = Depends(get_session),
):
    """관심종목 패널에서 종목 제거 후 목록 갱신."""
    svc = WatchlistService(session)
    await svc.remove(item_id)
    market_svc = MarketService(session)
    items = await _build_watchlist_items(svc, market_svc)
    return templates.TemplateResponse("partials/watchlist_items.html", {
        "request": request,
        "items": items,
        "tab": "watchlist",
        "selected_symbol": None,
    })


# ── 레거시 memo 엔드포인트 (기존 memo 테이블 전용) ──────────

@router.post("/stocks/memo", response_class=HTMLResponse)
async def add_memo(
    request: Request,
    symbol: str = Form(...),
    market: str = Form("KR"),
    name: str = Form(...),
    memo: str = Form(""),
    session: AsyncSession = Depends(get_session),
):
    """레거시 메모 추가."""
    svc = WatchlistService(session)
    try:
        await svc.add(symbol, market, name, memo or None)
    except Exception:
        pass
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@router.delete("/stocks/memo/{memo_id}", response_class=HTMLResponse)
async def delete_memo(
    request: Request,
    memo_id: int,
    session: AsyncSession = Depends(get_session),
):
    """레거시 메모 삭제."""
    svc = WatchlistService(session)
    await svc.remove(memo_id)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@router.get("/partials/memos", response_class=HTMLResponse)
async def partial_memos(request: Request, session: AsyncSession = Depends(get_session)):
    """레거시 메모 테이블 파셜."""
    svc = WatchlistService(session)
    memos = await svc.list_all()
    return templates.TemplateResponse("partials/memo_table.html", {
        "request": request,
        "memos": memos,
    })


@router.get("/partials/memo-options", response_class=HTMLResponse)
async def partial_memo_options(request: Request, session: AsyncSession = Depends(get_session)):
    """레거시 메모 종목 선택 옵션."""
    svc = WatchlistService(session)
    memos = await svc.list_all()
    options = '<option value="">-- 메모 종목 선택 --</option>'
    for m in memos:
        options += (
            f'<option value="{m.symbol}" data-market="{m.market}">'
            f'{m.name} ({m.symbol} / {m.market})</option>'
        )
    return HTMLResponse(options)
