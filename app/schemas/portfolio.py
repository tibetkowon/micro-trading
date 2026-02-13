from __future__ import annotations

from datetime import date

from pydantic import BaseModel


class PortfolioSummary(BaseModel):
    total_value: float = 0.0
    total_invested: float = 0.0
    cash_krw: float = 0.0
    cash_usd: float = 0.0
    initial_balance_krw: float = 0.0
    initial_balance_usd: float = 0.0
    realized_pnl: float = 0.0
    unrealized_pnl: float = 0.0
    total_pnl: float = 0.0
    return_pct: float = 0.0


class SnapshotResponse(BaseModel):
    date: date
    trading_mode: str
    total_value: float
    total_invested: float
    realized_pnl: float
    unrealized_pnl: float

    model_config = {"from_attributes": True}
