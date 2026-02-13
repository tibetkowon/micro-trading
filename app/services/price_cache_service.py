"""시세 캐시 DB 영속화 서비스."""

from __future__ import annotations

import logging
from datetime import datetime, timezone

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.broker.base import PriceInfo
from app.models.price_cache import PriceCache

logger = logging.getLogger(__name__)


class PriceCacheService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def upsert(self, info: PriceInfo, ohlcv: dict | None = None) -> PriceCache:
        """시세 정보를 DB에 저장 (있으면 갱신, 없으면 생성)."""
        result = await self.session.execute(
            select(PriceCache).where(
                PriceCache.symbol == info.symbol,
                PriceCache.market == info.market,
            )
        )
        cache = result.scalar_one_or_none()

        if cache is None:
            cache = PriceCache(symbol=info.symbol, market=info.market)
            self.session.add(cache)

        cache.price = info.price
        cache.change = info.change
        cache.change_pct = info.change_pct
        cache.volume = info.volume
        if ohlcv:
            cache.high = ohlcv.get("high", 0.0)
            cache.low = ohlcv.get("low", 0.0)
            cache.open = ohlcv.get("open", 0.0)
        cache.updated_at = datetime.now(timezone.utc)

        await self.session.commit()
        return cache

    async def get(self, symbol: str, market: str) -> PriceCache | None:
        """단일 종목 시세 캐시 조회."""
        result = await self.session.execute(
            select(PriceCache).where(
                PriceCache.symbol == symbol,
                PriceCache.market == market,
            )
        )
        return result.scalar_one_or_none()

    async def get_all(self) -> list[PriceCache]:
        """전체 시세 캐시 조회."""
        result = await self.session.execute(
            select(PriceCache).order_by(PriceCache.updated_at.desc())
        )
        return list(result.scalars().all())
