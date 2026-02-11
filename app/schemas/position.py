from __future__ import annotations

from pydantic import BaseModel


class PositionResponse(BaseModel):
    id: int
    symbol: str
    market: str
    quantity: int
    avg_price: float
    is_paper: bool
    current_price: float = 0.0
    unrealized_pnl: float = 0.0
    unrealized_pnl_pct: float = 0.0

    model_config = {"from_attributes": True}
