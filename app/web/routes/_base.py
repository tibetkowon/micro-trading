"""공유 Jinja2 템플릿 인스턴스, 필터, 헬퍼 함수."""

from __future__ import annotations

import pathlib
from datetime import datetime, timezone, timedelta

from fastapi.templating import Jinja2Templates
from sqlalchemy.ext.asyncio import AsyncSession

from app.services.watchlist_service import WatchlistService
from app.web.stock_list import KR_STOCKS

# 템플릿 경로: app/web/templates/
templates = Jinja2Templates(
    directory=str(pathlib.Path(__file__).parent.parent / "templates")
)


# ── Jinja2 커스텀 필터 ──────────────────────────────────────

def _to_kst(dt: datetime | None) -> str:
    """UTC datetime을 KST(UTC+9) 문자열로 변환."""
    if dt is None:
        return ""
    kst = timezone(timedelta(hours=9))
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(kst).strftime("%m-%d %H:%M")


def _source_label(source: str | None) -> str:
    """주문 출처를 한국어 라벨로 변환."""
    if not source or source == "manual":
        return "직접주문"
    return "자동전략"


templates.env.filters["to_kst"] = _to_kst
templates.env.filters["source_label"] = _source_label


# ── 종목명 조회 헬퍼 ────────────────────────────────────────

_STOCK_NAMES: dict[str, str] = {s["symbol"]: s["name"] for s in KR_STOCKS}


async def get_stock_name(symbol: str, market: str, session: AsyncSession) -> str:
    """종목 코드에 대한 종목명을 조회한다 (KR_STOCKS → DB → 관심종목 순)."""
    if symbol in _STOCK_NAMES:
        return _STOCK_NAMES[symbol]

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

    memo_svc = WatchlistService(session)
    for m in await memo_svc.list_all():
        if m.symbol == symbol:
            return m.name
    return symbol
