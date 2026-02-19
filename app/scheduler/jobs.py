from __future__ import annotations

import logging

from apscheduler.schedulers.asyncio import AsyncIOScheduler

logger = logging.getLogger(__name__)


async def run_strategy_tick():
    """Execute all active strategies."""
    from app.strategies.runner import StrategyRunner
    runner = StrategyRunner()
    await runner.run_all()


async def refresh_prices():
    """관심종목 및 보유종목 시세 갱신 (30초 간격)."""
    from app.database import async_session
    from app.services.market_service import MarketService
    async with async_session() as session:
        svc = MarketService(session)
        await svc.refresh_watchlist_prices()


async def take_portfolio_snapshot():
    """Take a daily portfolio snapshot for all accounts."""
    from app.database import async_session
    from app.services.portfolio_service import PortfolioService
    async with async_session() as session:
        svc = PortfolioService(session)
        await svc.take_daily_snapshot()


async def refresh_kis_token():
    """KIS 토큰 선제 갱신 — 만료 1시간 이내일 때 미리 갱신하여 세션 끊김 방지."""
    from app.config import settings
    if not settings.kis_app_key or not settings.kis_app_secret:
        return
    from app.broker.factory import _broker_cache
    from app.schemas.common import TradingMode
    broker = _broker_cache.get(TradingMode.REAL)
    if broker is None:
        return  # 실매매 브로커가 아직 초기화되지 않음
    client = getattr(broker, "_client", None)
    if client is None:
        return
    try:
        if client._token.is_expired or client._token.is_expiring_soon:
            await client.force_refresh_token()
            logger.info("KIS 토큰 선제 갱신 완료")
    except Exception as e:
        logger.error("KIS 토큰 선제 갱신 실패: %s", e)


def register_jobs(scheduler: AsyncIOScheduler):
    # 시세 갱신: 30초 간격 (장중 KST 09:00-15:30 + US 프리마켓 포함)
    scheduler.add_job(
        refresh_prices,
        "interval",
        seconds=30,
        id="refresh_prices",
        replace_existing=True,
    )

    # Strategy tick every minute during market hours (KST 09:00-15:30)
    scheduler.add_job(
        run_strategy_tick,
        "cron",
        hour="9-15",
        minute="*/1",
        id="strategy_tick",
        replace_existing=True,
    )

    # Daily portfolio snapshot at 16:00 KST (after market close)
    scheduler.add_job(
        take_portfolio_snapshot,
        "cron",
        hour=16,
        minute=0,
        id="portfolio_snapshot",
        replace_existing=True,
    )

    # KIS 토큰 선제 갱신: 30분 간격 (만료 1시간 이내일 때만 실제 갱신)
    scheduler.add_job(
        refresh_kis_token,
        "interval",
        minutes=30,
        id="refresh_kis_token",
        replace_existing=True,
    )

    logger.info("Registered scheduled jobs")
