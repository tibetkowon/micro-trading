from __future__ import annotations

from pydantic import BaseModel, Field

from app.schemas.common import Market, StrategyType, TradingMode


class StrategyCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100)
    strategy_type: StrategyType
    symbols: list[str] = Field(default_factory=list)
    market: Market = Market.KR
    params: dict = Field(default_factory=dict)
    trading_mode: TradingMode = TradingMode.PAPER
    schedule_cron: str = ""


class StrategyUpdate(BaseModel):
    is_active: bool | None = None
    params: dict | None = None
    symbols: list[str] | None = None
    schedule_cron: str | None = None


class StrategyResponse(BaseModel):
    id: int
    name: str
    strategy_type: str
    symbols: list[str] = []
    market: str
    params: dict = {}
    trading_mode: str
    is_active: bool
    schedule_cron: str

    model_config = {"from_attributes": True}
