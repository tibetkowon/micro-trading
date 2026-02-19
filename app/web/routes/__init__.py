"""Web routes 패키지 — 서브 라우터를 web_router로 통합한다."""

from fastapi import APIRouter

from app.web.routes import trading, watchlist, stock, orders, settings as settings_routes

web_router = APIRouter(tags=["web"])

web_router.include_router(trading.router)
web_router.include_router(watchlist.router)
web_router.include_router(stock.router)
web_router.include_router(orders.router)
web_router.include_router(settings_routes.router)

__all__ = ["web_router"]
