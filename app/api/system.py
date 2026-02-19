from __future__ import annotations

from fastapi import APIRouter

from app.config import settings

router = APIRouter(tags=["system"])


@router.get(
    "/health",
    summary="헬스 체크",
    description="서버 구동 상태 및 현재 거래 모드(PAPER/REAL)를 반환합니다. "
                "배포 자동화 CI/CD의 헬스 체크 엔드포인트로 사용됩니다.",
)
async def health():
    return {
        "status": "ok",
        "trading_mode": settings.get_trading_mode().value,
        "version": "0.1.0",
    }
