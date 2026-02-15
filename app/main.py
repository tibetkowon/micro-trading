from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from app.config import settings
from app.database import engine
from app.models.base import Base

logger = logging.getLogger(__name__)


def _migrate_add_missing_columns(connection) -> None:
    """기존 SQLite DB에 누락된 컬럼을 자동 추가하는 마이그레이션."""
    import sqlalchemy as sa

    inspector = sa.inspect(connection)
    migrations: dict[str, list[tuple[str, str]]] = {
        "accounts": [
            ("initial_balance_krw", "FLOAT DEFAULT 100000000.0"),
            ("initial_balance_usd", "FLOAT DEFAULT 100000.0"),
            ("commission_rate", "FLOAT DEFAULT 0.0005"),
        ],
    }
    for table_name, columns in migrations.items():
        if not inspector.has_table(table_name):
            continue
        existing = {col["name"] for col in inspector.get_columns(table_name)}
        for col_name, col_def in columns:
            if col_name not in existing:
                connection.execute(
                    sa.text(f"ALTER TABLE {table_name} ADD COLUMN {col_name} {col_def}")
                )
                logger.info("Migrated: %s.%s added", table_name, col_name)
    # 기존 accounts 행의 initial_balance를 paper_balance 값으로 보정
    if inspector.has_table("accounts"):
        connection.execute(
            sa.text(
                "UPDATE accounts SET initial_balance_krw = paper_balance_krw "
                "WHERE initial_balance_krw = 100000000.0 AND paper_balance_krw != 100000000.0"
            )
        )
        connection.execute(
            sa.text(
                "UPDATE accounts SET initial_balance_usd = paper_balance_usd "
                "WHERE initial_balance_usd = 100000.0 AND paper_balance_usd != 100000.0"
            )
        )


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
        # 기존 DB에 누락된 컬럼 자동 추가 (SQLite ALTER TABLE)
        await conn.run_sync(_migrate_add_missing_columns)
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
