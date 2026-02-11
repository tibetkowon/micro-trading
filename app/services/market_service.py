"""Market data service with TTL cache."""

from __future__ import annotations

import logging
import time

from app.broker.base import PriceInfo
from app.broker.factory import get_broker
from app.config import TradingMode

logger = logging.getLogger(__name__)

# Simple TTL cache: {(symbol, market): (PriceInfo, timestamp)}
_price_cache: dict[tuple[str, str], tuple[PriceInfo, float]] = {}
CACHE_TTL = 30  # seconds


class MarketService:

    async def get_price(self, symbol: str, market: str = "KR") -> PriceInfo:
        key = (symbol, market)
        now = time.monotonic()
        cached = _price_cache.get(key)
        if cached and (now - cached[1]) < CACHE_TTL:
            return cached[0]

        # Always use KIS broker for real prices, even in paper mode
        try:
            from app.broker.kis.broker import KISBroker
            from app.config import settings
            if settings.kis_app_key:
                broker = await get_broker(TradingMode.REAL)
            else:
                broker = await get_broker()
        except Exception:
            broker = await get_broker()

        price_info = await broker.get_current_price(symbol, market)
        _price_cache[key] = (price_info, now)
        return price_info

    async def get_daily_prices(self, symbol: str, market: str = "KR", days: int = 60) -> list[dict]:
        try:
            from app.config import settings
            if settings.kis_app_key:
                broker = await get_broker(TradingMode.REAL)
            else:
                broker = await get_broker()
        except Exception:
            broker = await get_broker()
        return await broker.get_daily_prices(symbol, market, days)

    def clear_cache(self):
        _price_cache.clear()
