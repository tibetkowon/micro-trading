from __future__ import annotations

import logging

from apscheduler.schedulers.asyncio import AsyncIOScheduler

logger = logging.getLogger(__name__)


async def run_strategy_tick():
    """Execute all active strategies."""
    from app.strategies.runner import StrategyRunner
    runner = StrategyRunner()
    await runner.run_all()


async def take_portfolio_snapshot():
    """Take a daily portfolio snapshot for all accounts."""
    from app.database import async_session
    from app.services.portfolio_service import PortfolioService
    async with async_session() as session:
        svc = PortfolioService(session)
        await svc.take_daily_snapshot()


def register_jobs(scheduler: AsyncIOScheduler):
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

    logger.info("Registered scheduled jobs")
