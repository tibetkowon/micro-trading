from __future__ import annotations

from fastapi import APIRouter

from app.config import settings

router = APIRouter(tags=["system"])


@router.get("/health")
async def health():
    return {
        "status": "ok",
        "trading_mode": settings.get_trading_mode().value,
        "version": "0.1.0",
    }
