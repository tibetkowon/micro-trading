from __future__ import annotations

import logging

from apscheduler.schedulers.asyncio import AsyncIOScheduler

logger = logging.getLogger(__name__)

_scheduler: AsyncIOScheduler | None = None


def get_scheduler() -> AsyncIOScheduler:
    global _scheduler
    if _scheduler is None:
        _scheduler = AsyncIOScheduler(timezone="Asia/Seoul")
    return _scheduler


def start_scheduler() -> AsyncIOScheduler:
    scheduler = get_scheduler()
    from app.scheduler.jobs import register_jobs
    register_jobs(scheduler)
    scheduler.start()
    logger.info("Scheduler started")
    return scheduler
