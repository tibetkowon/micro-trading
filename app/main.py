from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from app.config import settings
from app.database import engine
from app.models.base import Base

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    logger.info("Starting micro-trading (mode=%s)", settings.trading_mode.value)

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    logger.info("Database tables ready")

    # Ensure default account exists
    from app.database import async_session
    from app.models.account import Account
    from sqlalchemy import select

    async with async_session() as session:
        result = await session.execute(select(Account).limit(1))
        if result.scalar_one_or_none() is None:
            session.add(Account(
                name="default",
                broker_type="KIS",
                account_number=settings.kis_account_number,
                paper_balance_krw=settings.paper_balance_krw,
                paper_balance_usd=settings.paper_balance_usd,
                initial_balance_krw=settings.paper_balance_krw,
                initial_balance_usd=settings.paper_balance_usd,
                commission_rate=settings.paper_commission_rate,
            ))
            await session.commit()
            logger.info("Created default account")

    # Start scheduler
    from app.scheduler.scheduler import start_scheduler
    scheduler = start_scheduler()

    yield

    # Shutdown
    if scheduler.running:
        scheduler.shutdown(wait=False)
    await engine.dispose()
    logger.info("Shutdown complete")


def create_app() -> FastAPI:
    app = FastAPI(
        title="Micro Trading",
        version="0.1.0",
        lifespan=lifespan,
    )

    # API routes
    from app.api.router import api_router
    app.include_router(api_router, prefix="/api")

    # Web routes
    from app.web.routes import web_router
    app.include_router(web_router)

    # Static files
    import pathlib
    static_dir = pathlib.Path(__file__).parent / "web" / "static"
    app.mount("/static", StaticFiles(directory=str(static_dir)), name="static")

    return app


app = create_app()
