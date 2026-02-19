from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.position import PositionResponse
from app.services.portfolio_service import PortfolioService

router = APIRouter(prefix="/positions", tags=["positions"])


@router.get(
    "",
    response_model=list[PositionResponse],
    summary="보유 포지션 조회",
    description="현재 보유 중인 모든 포지션을 현재가 기준 평가금액 및 미실현 손익률과 함께 반환합니다. "
                "is_paper=true이면 모의투자, false이면 실매매 포지션을 조회합니다.",
)
async def list_positions(
    is_paper: bool = True,
    session: AsyncSession = Depends(get_session),
):
    svc = PortfolioService(session)
    positions = await svc.get_positions(is_paper=is_paper)
    return [PositionResponse(**p) for p in positions]
