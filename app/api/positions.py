from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.position import PositionResponse
from app.services.portfolio_service import PortfolioService

router = APIRouter(prefix="/positions", tags=["positions"])


@router.get("", response_model=list[PositionResponse])
async def list_positions(
    is_paper: bool = True,
    session: AsyncSession = Depends(get_session),
):
    svc = PortfolioService(session)
    positions = await svc.get_positions(is_paper=is_paper)
    return [PositionResponse(**p) for p in positions]
