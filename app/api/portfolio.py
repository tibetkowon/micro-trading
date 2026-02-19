from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.portfolio import PortfolioSummary, SnapshotResponse, OrderableResponse, PnlAnalysisResponse
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


@router.get("/orderable", response_model=OrderableResponse)
async def portfolio_orderable(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    """AI 자동매매용 실질 주문가능 금액 조회 (수수료 0.3% 반영)."""
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    info = await svc.get_orderable_info(mode)
    return OrderableResponse(**info)


@router.get("/pnl-analysis", response_model=PnlAnalysisResponse)
async def portfolio_pnl_analysis(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    """수익률 분석 — 종목별 실현손익 및 일별 누적 수익률 조회."""
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    result = await svc.get_pnl_analysis(mode)
    return PnlAnalysisResponse(**result)


@router.post("/snapshot")
async def take_snapshot(
    session: AsyncSession = Depends(get_session),
):
    svc = PortfolioService(session)
    await svc.take_daily_snapshot()
    return {"status": "ok"}
