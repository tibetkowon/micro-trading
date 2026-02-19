from __future__ import annotations

import asyncio

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.market import PriceCacheResponse, PriceResponse
from app.services.market_service import MarketService
from app.services.stock_master_service import StockMasterService

router = APIRouter(prefix="/market", tags=["market"])


@router.get(
    "/price/{symbol}",
    response_model=PriceResponse,
    summary="종목 현재가 조회",
    description="실시간 현재가, 전일 대비 변동액/변동률, 거래량, MA5/MA20 이동평균을 반환합니다. "
                "3계층 캐시(메모리 15초 → DB → API) 순서로 조회하여 응답 속도를 최적화합니다.",
)
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


@router.get(
    "/daily-prices/{symbol}",
    summary="일별 시세 조회",
    description="최근 N일간의 일별 OHLCV(시가·고가·저가·종가·거래량) 데이터를 반환합니다. "
                "차트 렌더링 및 이동평균 계산에 사용합니다.",
)
async def get_daily_prices(
    symbol: str,
    market: str = "KR",
    days: int = 60,
    session: AsyncSession = Depends(get_session),
):
    svc = MarketService(session)
    return await svc.get_daily_prices(symbol, market, days)


@router.get(
    "/candles/{symbol}",
    summary="분봉 데이터 조회",
    description="1분봉(interval=1) 또는 5분봉(interval=5) 캔들 데이터를 반환합니다. "
                "5분봉은 서버에서 1분봉을 집계하여 생성합니다. KIS API 미연동 시 빈 목록을 반환합니다.",
)
async def get_intraday_candles(
    symbol: str,
    market: str = "KR",
    interval: int = Query(default=1, ge=1, le=60),
    session: AsyncSession = Depends(get_session),
):
    svc = MarketService(session)
    return await svc.get_intraday_candles(symbol, market, interval)


@router.get(
    "/cache",
    response_model=list[PriceCacheResponse],
    summary="시세 캐시 전체 조회",
    description="DB에 저장된 전체 종목 시세 캐시를 반환합니다. "
                "30초 간격 스케줄러가 관심종목·보유종목 시세를 갱신합니다.",
)
async def get_price_cache(session: AsyncSession = Depends(get_session)):
    from app.services.price_cache_service import PriceCacheService
    svc = PriceCacheService(session)
    items = await svc.get_all()
    return [PriceCacheResponse.model_validate(item) for item in items]
