from __future__ import annotations

from fastapi import APIRouter

from app.api.system import router as system_router
from app.api.orders import router as orders_router
from app.api.positions import router as positions_router
from app.api.portfolio import router as portfolio_router
from app.api.strategies import router as strategies_router
from app.api.market import router as market_router

api_router = APIRouter()
api_router.include_router(system_router)
api_router.include_router(orders_router)
api_router.include_router(positions_router)
api_router.include_router(portfolio_router)
api_router.include_router(strategies_router)
api_router.include_router(market_router)
