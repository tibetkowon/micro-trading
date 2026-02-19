from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, Field

from app.schemas.common import Market, OrderSide, OrderStatus, OrderType, TradingMode


class OrderCreate(BaseModel):
    symbol: str = Field(..., min_length=1, max_length=20, description="종목 코드 (예: '005930', 'AAPL')")
    market: Market = Field(Market.KR, description="시장 구분 (KR: 국내, US: 해외)")
    side: OrderSide = Field(..., description="주문 방향 (BUY: 매수, SELL: 매도)")
    order_type: OrderType = Field(OrderType.MARKET, description="주문 유형 (MARKET: 시장가, LIMIT: 지정가)")
    quantity: int = Field(..., gt=0, description="주문 수량 (1 이상 정수)")
    price: float | None = Field(None, description="지정가 단가. 시장가 주문 시 null, LIMIT 주문 시 필수")
    trading_mode: TradingMode = Field(TradingMode.PAPER, description="거래 모드 (PAPER: 모의투자, REAL: 실매매)")


class OrderResponse(BaseModel):
    id: int = Field(..., description="주문 고유 ID")
    broker_order_id: str | None = Field(None, description="증권사 접수 주문번호 (KIS ODNO)")
    symbol: str = Field(..., description="종목 코드")
    name: str | None = Field(None, description="종목명 (StockMaster 조회)")
    market: str = Field(..., description="시장 구분 (KR/US)")
    side: str = Field(..., description="주문 방향 (BUY/SELL)")
    order_type: str = Field(..., description="주문 유형 (MARKET/LIMIT)")
    quantity: int = Field(..., description="주문 수량")
    price: float | None = Field(None, description="주문 단가 (지정가 주문 시)")
    filled_quantity: int = Field(..., description="체결 수량")
    filled_price: float | None = Field(None, description="체결 단가 (체결 전 null)")
    trading_mode: str = Field(..., description="거래 모드 (PAPER/REAL)")
    status: str = Field(..., description="주문 상태 (SUBMITTED/FILLED/REJECTED/CANCELLED)")
    reject_reason: str | None = Field(None, description="주문 거부 사유 (REJECTED 상태일 때만)")
    source: str = Field(..., description="주문 출처 (manual: 직접주문, strategy: 자동전략)")
    strategy_name: str | None = Field(None, description="자동전략 주문 시 전략명")
    created_at: datetime = Field(..., description="주문 생성 시각 (UTC)")
    filled_at: datetime | None = Field(None, description="체결 시각 (UTC, 체결 전 null)")

    model_config = {"from_attributes": True}
