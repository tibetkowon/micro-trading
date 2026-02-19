from __future__ import annotations

import asyncio

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.market import PriceCacheResponse, PriceResponse
from app.services.market_service import MarketService
from app.services.stock_master_service import StockMasterService

router = APIRouter(prefix="/market", tags=["market"])


@router.get("/price/{symbol}", response_model=PriceResponse)
async def get_price(
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    svc = MarketService(session)
    stock_svc = StockMasterService(session)
    info = await svc.get_price(symbol, market)
    name, indicators = await asyncio.gather(
        stock_svc.get_name(symbol, market),
        svc.get_latest_indicators(symbol, market),
    )
    return PriceResponse(
        symbol=info.symbol,
        name=name,
        price=info.price,
        change=info.change,
        change_pct=info.change_pct,
        volume=info.volume,
        market=info.market,
        ma5=indicators["ma5"],
        ma20=indicators["ma20"],
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
