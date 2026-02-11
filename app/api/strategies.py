from __future__ import annotations

import json

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.strategy import StrategyCreate, StrategyResponse, StrategyUpdate
from app.services.strategy_service import StrategyService

router = APIRouter(prefix="/strategies", tags=["strategies"])


def _to_response(config) -> StrategyResponse:
    return StrategyResponse(
        id=config.id,
        name=config.name,
        strategy_type=config.strategy_type,
        symbols=json.loads(config.symbols) if isinstance(config.symbols, str) else config.symbols,
        market=config.market,
        params=json.loads(config.params) if isinstance(config.params, str) else config.params,
        trading_mode=config.trading_mode,
        is_active=config.is_active,
        schedule_cron=config.schedule_cron,
    )


@router.post("", response_model=StrategyResponse)
async def create_strategy(
    req: StrategyCreate,
    session: AsyncSession = Depends(get_session),
):
    svc = StrategyService(session)
    config = await svc.create(req)
    return _to_response(config)


@router.get("", response_model=list[StrategyResponse])
async def list_strategies(
    active_only: bool = False,
    session: AsyncSession = Depends(get_session),
):
    svc = StrategyService(session)
    configs = await svc.list_all(active_only)
    return [_to_response(c) for c in configs]


@router.get("/{strategy_id}", response_model=StrategyResponse)
async def get_strategy(
    strategy_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = StrategyService(session)
    config = await svc.get(strategy_id)
    if not config:
        raise HTTPException(status_code=404, detail="Strategy not found")
    return _to_response(config)


@router.patch("/{strategy_id}", response_model=StrategyResponse)
async def update_strategy(
    strategy_id: int,
    req: StrategyUpdate,
    session: AsyncSession = Depends(get_session),
):
    svc = StrategyService(session)
    try:
        config = await svc.update(strategy_id, req)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    return _to_response(config)


@router.delete("/{strategy_id}")
async def delete_strategy(
    strategy_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = StrategyService(session)
    try:
        await svc.delete(strategy_id)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
    return {"status": "deleted"}
