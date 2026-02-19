from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, Field

from app.schemas.common import Market, OrderSide, OrderStatus, OrderType, TradingMode


class OrderCreate(BaseModel):
    symbol: str = Field(..., min_length=1, max_length=20)
    market: Market = Market.KR
    side: OrderSide
    order_type: OrderType = OrderType.MARKET
    quantity: int = Field(..., gt=0)
    price: float | None = None
    trading_mode: TradingMode = TradingMode.PAPER


class OrderResponse(BaseModel):
    id: int
    broker_order_id: str | None = None
    symbol: str
    name: str | None = None
    market: str
    side: str
    order_type: str
    quantity: int
    price: float | None
    filled_quantity: int
    filled_price: float | None
    trading_mode: str
    status: str
    reject_reason: str | None = None
    source: str
    strategy_name: str | None = None
    created_at: datetime
    filled_at: datetime | None = None

    model_config = {"from_attributes": True}
