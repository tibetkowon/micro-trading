"""시장 데이터 서비스 (메모리 TTL 캐시 + DB 영속 캐시)."""

from __future__ import annotations

import logging
import time

from sqlalchemy.ext.asyncio import AsyncSession

from app.broker.base import PriceInfo
from app.broker.factory import get_broker
from app.schemas.common import TradingMode

logger = logging.getLogger(__name__)

# 메모리 TTL 캐시: {(symbol, market): (PriceInfo, timestamp)}
_price_cache: dict[tuple[str, str], tuple[PriceInfo, float]] = {}
CACHE_TTL = 15  # seconds


def _add_moving_averages(prices: list[dict]) -> list[dict]:
    """일별 가격 목록에 MA5/MA20을 계산해 추가.

    날짜 오름차순(과거→현재)으로 정렬 후 계산, 동일 순서로 반환.
    데이터가 부족한 초반 행은 None으로 표시.
    """
    if not prices:
        return prices

    # 날짜 오름차순 정렬 (broker별 정렬 순서 차이 정규화)
    sorted_prices = sorted(prices, key=lambda x: x["date"])
    closes = [p["close"] for p in sorted_prices]

    for i, p in enumerate(sorted_prices):
        p["ma5"] = round(sum(closes[i - 4:i + 1]) / 5, 2) if i >= 4 else None
        p["ma20"] = round(sum(closes[i - 19:i + 1]) / 20, 2) if i >= 19 else None

    return sorted_prices


class MarketService:

    def __init__(self, session: AsyncSession | None = None):
        self.session = session

    async def get_price(self, symbol: str, market: str = "KR") -> PriceInfo:
        """시세 조회 (메모리 캐시 → API → DB 저장)."""
        key = (symbol, market)
        now = time.monotonic()

        # 1) 메모리 캐시 확인
        cached = _price_cache.get(key)
        if cached and (now - cached[1]) < CACHE_TTL:
            return cached[0]

        # 2) API에서 실시간 시세 조회
        try:
            broker = await self._get_broker()
            price_info = await broker.get_current_price(symbol, market)
            _price_cache[key] = (price_info, now)

            # 3) DB 캐시에 저장 (session이 있을 때만)
            if self.session and price_info.price > 0:
                await self._save_to_db(price_info)

            return price_info

        except Exception as e:
            logger.warning("API 시세 조회 실패 %s/%s: %s", symbol, market, e)

            # 4) API 실패 시 DB 캐시에서 복원
            if self.session:
                db_price = await self._load_from_db(symbol, market)
                if db_price:
                    return db_price

            # 5) 메모리 캐시에 만료된 데이터라도 있으면 반환
            if cached:
                return cached[0]

            return PriceInfo(symbol=symbol, price=0.0, market=market)

    async def get_daily_prices(self, symbol: str, market: str = "KR", days: int = 60) -> list[dict]:
        broker = await self._get_broker()
        prices = await broker.get_daily_prices(symbol, market, days)
        return _add_moving_averages(prices)

    async def refresh_watchlist_prices(self) -> int:
        """관심종목 + 보유종목 시세 일괄 갱신 (스케줄러용). 갱신 건수 반환."""
        if not self.session:
            return 0

        from app.models.watchlist import WatchlistItem
        from app.models.position import Position
        from sqlalchemy import select

        # 관심종목 조회
        result = await self.session.execute(select(WatchlistItem))
        watchlist = list(result.scalars().all())

        # 보유종목 조회 (수량 > 0)
        result = await self.session.execute(
            select(Position).where(Position.quantity > 0)
        )
        positions = list(result.scalars().all())

        # 중복 제거
        symbols: set[tuple[str, str]] = set()
        for item in watchlist:
            symbols.add((item.symbol, item.market))
        for pos in positions:
            symbols.add((pos.symbol, pos.market))

        count = 0
        for symbol, market in symbols:
            try:
                await self.get_price(symbol, market)
                count += 1
            except Exception as e:
                logger.warning("시세 갱신 실패 %s/%s: %s", symbol, market, e)

        logger.info("시세 갱신 완료: %d/%d건", count, len(symbols))
        return count

    def clear_cache(self):
        _price_cache.clear()

    async def _get_broker(self):
        """가격 조회용 브로커 반환."""
        try:
            from app.config import settings
            if settings.kis_app_key:
                return await get_broker(TradingMode.REAL)
        except Exception:
            pass
        return await get_broker()

    async def _save_to_db(self, info: PriceInfo) -> None:
        """시세를 DB 캐시에 저장."""
        try:
            from app.services.price_cache_service import PriceCacheService
            svc = PriceCacheService(self.session)
            await svc.upsert(info)
        except Exception as e:
            logger.debug("DB 캐시 저장 실패: %s", e)

    async def _load_from_db(self, symbol: str, market: str) -> PriceInfo | None:
        """DB 캐시에서 시세 복원."""
        try:
            from app.services.price_cache_service import PriceCacheService
            svc = PriceCacheService(self.session)
            cache = await svc.get(symbol, market)
            if cache and cache.price > 0:
                return PriceInfo(
                    symbol=cache.symbol,
                    price=cache.price,
                    change=cache.change,
                    change_pct=cache.change_pct,
                    volume=cache.volume,
                    market=cache.market,
                )
        except Exception as e:
            logger.debug("DB 캐시 조회 실패: %s", e)
        return None
