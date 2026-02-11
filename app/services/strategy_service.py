"""Strategy CRUD service."""

from __future__ import annotations

import json
import logging

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.account import Account
from app.models.strategy import StrategyConfig
from app.schemas.strategy import StrategyCreate, StrategyUpdate

logger = logging.getLogger(__name__)


class StrategyService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def create(self, req: StrategyCreate) -> StrategyConfig:
        account = (await self.session.execute(select(Account).limit(1))).scalar_one()
        config = StrategyConfig(
            account_id=account.id,
            name=req.name,
            strategy_type=req.strategy_type.value,
            symbols=json.dumps(req.symbols),
            market=req.market.value,
            params=json.dumps(req.params),
            trading_mode=req.trading_mode.value,
            schedule_cron=req.schedule_cron,
        )
        self.session.add(config)
        await self.session.commit()
        return config

    async def update(self, strategy_id: int, req: StrategyUpdate) -> StrategyConfig:
        result = await self.session.execute(
            select(StrategyConfig).where(StrategyConfig.id == strategy_id)
        )
        config = result.scalar_one_or_none()
        if not config:
            raise ValueError(f"Strategy {strategy_id} not found")

        if req.is_active is not None:
            config.is_active = req.is_active
        if req.params is not None:
            config.params = json.dumps(req.params)
        if req.symbols is not None:
            config.symbols = json.dumps(req.symbols)
        if req.schedule_cron is not None:
            config.schedule_cron = req.schedule_cron

        await self.session.commit()
        return config

    async def delete(self, strategy_id: int):
        result = await self.session.execute(
            select(StrategyConfig).where(StrategyConfig.id == strategy_id)
        )
        config = result.scalar_one_or_none()
        if not config:
            raise ValueError(f"Strategy {strategy_id} not found")
        await self.session.delete(config)
        await self.session.commit()

    async def get(self, strategy_id: int) -> StrategyConfig | None:
        result = await self.session.execute(
            select(StrategyConfig).where(StrategyConfig.id == strategy_id)
        )
        return result.scalar_one_or_none()

    async def list_all(self, active_only: bool = False) -> list[StrategyConfig]:
        stmt = select(StrategyConfig).order_by(StrategyConfig.id)
        if active_only:
            stmt = stmt.where(StrategyConfig.is_active == True)
        result = await self.session.execute(stmt)
        return list(result.scalars().all())
