"""Stock memo (favorites) service."""

from __future__ import annotations

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.stock_memo import StockMemo


class StockMemoService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def add(self, symbol: str, market: str, name: str, memo: str | None = None) -> StockMemo:
        stock_memo = StockMemo(
            symbol=symbol.upper().strip(),
            market=market.upper().strip(),
            name=name.strip(),
            memo=memo.strip() if memo else None,
        )
        self.session.add(stock_memo)
        await self.session.commit()
        return stock_memo

    async def remove(self, memo_id: int) -> None:
        result = await self.session.execute(
            select(StockMemo).where(StockMemo.id == memo_id)
        )
        item = result.scalar_one_or_none()
        if item:
            await self.session.delete(item)
            await self.session.commit()

    async def list_all(self) -> list[StockMemo]:
        result = await self.session.execute(
            select(StockMemo).order_by(StockMemo.id.desc())
        )
        return list(result.scalars().all())
