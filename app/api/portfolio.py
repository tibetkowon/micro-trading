from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.portfolio import PortfolioSummary, SnapshotResponse
from app.services.portfolio_service import PortfolioService

router = APIRouter(prefix="/portfolio", tags=["portfolio"])


@router.get("/summary", response_model=PortfolioSummary)
async def portfolio_summary(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    # trading_mode 미지정 시 현재 런타임 모드 사용
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    return await svc.get_summary(mode)


@router.get("/snapshots", response_model=list[SnapshotResponse])
async def portfolio_snapshots(
    trading_mode: str | None = None,
    limit: int = 90,
    session: AsyncSession = Depends(get_session),
):
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    snapshots = await svc.get_snapshots(mode, limit)
    return [SnapshotResponse.model_validate(s) for s in snapshots]


@router.post("/snapshot")
async def take_snapshot(
    session: AsyncSession = Depends(get_session),
):
    svc = PortfolioService(session)
    await svc.take_daily_snapshot()
    return {"status": "ok"}
