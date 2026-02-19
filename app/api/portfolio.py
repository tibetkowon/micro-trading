from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import settings
from app.database import get_session
from app.schemas.portfolio import PortfolioSummary, SnapshotResponse, OrderableResponse, PnlAnalysisResponse
from app.services.portfolio_service import PortfolioService

router = APIRouter(prefix="/portfolio", tags=["portfolio"])


@router.get(
    "/summary",
    response_model=PortfolioSummary,
    summary="포트폴리오 요약 조회",
    description="총 평가금액, 현금 잔고, 실현/미실현 손익, 수익률, 주문가능 금액을 반환합니다. "
                "trading_mode 미지정 시 현재 런타임 모드(PAPER/REAL)가 자동 적용됩니다.",
)
async def portfolio_summary(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    return await svc.get_summary(mode)


@router.get(
    "/snapshots",
    response_model=list[SnapshotResponse],
    summary="일별 포트폴리오 스냅샷 조회",
    description="지정 기간의 일별 총 평가금액 및 손익 기록을 반환합니다. "
                "매일 16:00 KST(장 마감 후) 스케줄러가 자동으로 스냅샷을 생성합니다.",
)
async def portfolio_snapshots(
    trading_mode: str | None = None,
    limit: int = 90,
    session: AsyncSession = Depends(get_session),
):
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    snapshots = await svc.get_snapshots(mode, limit)
    return [SnapshotResponse.model_validate(s) for s in snapshots]


@router.get(
    "/orderable",
    response_model=OrderableResponse,
    summary="주문가능 금액 조회 (AI용)",
    description="수수료를 반영한 실질 주문가능 금액을 반환합니다. "
                "모의투자는 계정 설정 수수료율, 실계좌는 0.3% 안전 마진을 적용합니다. "
                "자동매매 AI가 안전한 최대 주문 금액을 계산하는 데 활용합니다.",
)
async def portfolio_orderable(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    info = await svc.get_orderable_info(mode)
    return OrderableResponse(**info)


@router.get(
    "/pnl-analysis",
    response_model=PnlAnalysisResponse,
    summary="수익률 분석 조회",
    description="종목별 실현 손익 합계와 일별 누적 수익률을 반환합니다. "
                "AI 자동매매의 전략 성과 분석 및 포트폴리오 리밸런싱 판단에 활용합니다.",
)
async def portfolio_pnl_analysis(
    trading_mode: str | None = None,
    session: AsyncSession = Depends(get_session),
):
    mode = trading_mode or settings.get_trading_mode().value
    svc = PortfolioService(session)
    result = await svc.get_pnl_analysis(mode)
    return PnlAnalysisResponse(**result)


@router.post(
    "/snapshot",
    summary="포트폴리오 스냅샷 수동 생성",
    description="현재 포트폴리오 상태를 수동으로 스냅샷으로 저장합니다. "
                "당일 스냅샷이 이미 존재하면 중복 생성하지 않습니다.",
)
async def take_snapshot(
    session: AsyncSession = Depends(get_session),
):
    svc = PortfolioService(session)
    await svc.take_daily_snapshot()
    return {"status": "ok"}
