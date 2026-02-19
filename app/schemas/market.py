from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel


class PriceResponse(BaseModel):
    symbol: str
    name: str | None = None
    price: float
    change: float = 0.0
    change_pct: float = 0.0
    volume: int = 0
    market: str = "KR"


class PriceCacheResponse(BaseModel):
    symbol: str
    market: str
    price: float
    change: float = 0.0
    change_pct: float = 0.0
    volume: int = 0
    high: float = 0.0
    low: float = 0.0
    open: float = 0.0
    updated_at: datetime | None = None

    model_config = {"from_attributes": True}
