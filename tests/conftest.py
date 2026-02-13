"""테스트 공통 설정: 인메모리 SQLite 세션 및 기본 계정 픽스처."""

from __future__ import annotations

import pytest
from sqlalchemy import event
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from app.models.base import Base
from app.models.account import Account


@pytest.fixture
async def session():
    """각 테스트마다 독립적인 인메모리 DB 세션 제공."""
    engine = create_async_engine("sqlite+aiosqlite://", echo=False)

    @event.listens_for(engine.sync_engine, "connect")
    def _set_pragmas(dbapi_conn, _):
        cursor = dbapi_conn.cursor()
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()

    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)
    async with factory() as sess:
        yield sess

    await engine.dispose()


@pytest.fixture
async def account(session: AsyncSession) -> Account:
    """초기 잔고가 설정된 기본 계정 생성."""
    acc = Account(
        name="test",
        broker_type="PAPER",
        paper_balance_krw=10_000_000.0,
        paper_balance_usd=10_000.0,
        initial_balance_krw=10_000_000.0,
        initial_balance_usd=10_000.0,
        commission_rate=0.0005,
    )
    session.add(acc)
    await session.commit()
    return acc
