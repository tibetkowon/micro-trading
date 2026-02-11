from __future__ import annotations

from fastapi import APIRouter

from app.schemas.market import PriceResponse
from app.services.market_service import MarketService

router = APIRouter(prefix="/market", tags=["market"])


@router.get("/price/{symbol}", response_model=PriceResponse)
async def get_price(symbol: str, market: str = "KR"):
    svc = MarketService()
    info = await svc.get_price(symbol, market)
    return PriceResponse(
        symbol=info.symbol,
        price=info.price,
        change=info.change,
        change_pct=info.change_pct,
        volume=info.volume,
        market=info.market,
    )
