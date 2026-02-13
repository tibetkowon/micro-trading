"""관심종목(watchlist) 서비스."""

from __future__ import annotations

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.watchlist import WatchlistItem


class WatchlistService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def add(self, symbol: str, market: str, name: str, memo: str | None = None) -> WatchlistItem:
        item = WatchlistItem(
            symbol=symbol.upper().strip(),
            market=market.upper().strip(),
            name=name.strip(),
            memo=memo.strip() if memo else None,
        )
        self.session.add(item)
        await self.session.commit()
        return item

    async def remove(self, item_id: int) -> None:
        result = await self.session.execute(
            select(WatchlistItem).where(WatchlistItem.id == item_id)
        )
        item = result.scalar_one_or_none()
        if item:
            await self.session.delete(item)
            await self.session.commit()

    async def list_all(self) -> list[WatchlistItem]:
        result = await self.session.execute(
            select(WatchlistItem).order_by(WatchlistItem.sort_order, WatchlistItem.id.desc())
        )
        return list(result.scalars().all())
