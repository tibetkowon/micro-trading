from __future__ import annotations

from pydantic import BaseModel


class PriceResponse(BaseModel):
    symbol: str
    price: float
    change: float = 0.0
    change_pct: float = 0.0
    volume: int = 0
    market: str = "KR"
