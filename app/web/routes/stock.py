"""종목 상세 패널 파셜 라우트 (센터 패널)."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, Request
from fastapi.responses import HTMLResponse
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.common import TradingMode
from app.services.market_service import MarketService
from app.services.portfolio_service import PortfolioService
from app.web.routes._base import get_stock_name, templates

logger = logging.getLogger(__name__)

router = APIRouter(tags=["web"])


@router.get("/partials/stock-detail/{symbol}", response_class=HTMLResponse)
async def partial_stock_detail(
    request: Request,
    symbol: str,
    market: str = "KR",
    name: str = "",
    session: AsyncSession = Depends(get_session),
):
    """HTMX로 센터 패널에 종목 상세를 로드한다."""
    stock_name = name or await get_stock_name(symbol, market, session)

    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    portfolio_svc = PortfolioService(session)
    positions = await portfolio_svc.get_positions(
        is_paper=settings.get_trading_mode() == TradingMode.PAPER
    )
    position = next(
        (p for p in positions if p.symbol == symbol and p.market == market), None
    )

    return templates.TemplateResponse("partials/stock_detail.html", {
        "request": request,
        "symbol": symbol,
        "market": market,
        "stock_name": stock_name,
        "price_info": price_info,
        "position": position,
    })


@router.get("/partials/stock-price/{symbol}", response_class=HTMLResponse)
async def partial_stock_price(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """현재가 히어로 파셜 (10초 폴링)."""
    market_svc = MarketService(session)
    try:
        price_info = await market_svc.get_price(symbol, market)
    except Exception:
        price_info = None

    return templates.TemplateResponse("partials/stock_price_hero.html", {
        "request": request,
        "price_info": price_info,
        "market": market,
    })


@router.get("/partials/stock-position/{symbol}", response_class=HTMLResponse)
async def partial_stock_position(
    request: Request,
    symbol: str,
    market: str = "KR",
    session: AsyncSession = Depends(get_session),
):
    """특정 종목의 보유 포지션 카드."""
    svc = PortfolioService(session)
    positions = await svc.get_positions(
        is_paper=settings.get_trading_mode() == TradingMode.PAPER
    )
    position = next(
        (p for p in positions if p.symbol == symbol and p.market == market), None
    )

    return templates.TemplateResponse("partials/stock_position_card.html", {
        "request": request,
        "position": position,
    })
