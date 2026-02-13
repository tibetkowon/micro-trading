from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.market import PriceCacheResponse, PriceResponse
from app.services.market_service import MarketService

router = APIRouter(prefix="/market", tags=["market"])


@router.get("/price/{symbol}", response_model=PriceResponse)
async def get_price(
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    svc = MarketService(session)
    info = await svc.get_price(symbol, market)
    return PriceResponse(
        symbol=info.symbol,
        price=info.price,
        change=info.change,
        change_pct=info.change_pct,
        volume=info.volume,
        market=info.market,
    )


@router.get("/daily-prices/{symbol}")
async def get_daily_prices(
    symbol: str,
    market: str = "KR",
    days: int = 60,
    session: AsyncSession = Depends(get_session),
):
    svc = MarketService(session)
    return await svc.get_daily_prices(symbol, market, days)


@router.get("/cache", response_model=list[PriceCacheResponse])
async def get_price_cache(session: AsyncSession = Depends(get_session)):
    """DB에 저장된 시세 캐시 전체 조회."""
    from app.services.price_cache_service import PriceCacheService
    svc = PriceCacheService(session)
    items = await svc.get_all()
    return [PriceCacheResponse.model_validate(item) for item in items]
