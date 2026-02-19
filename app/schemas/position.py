from __future__ import annotations

from pydantic import BaseModel, Field


class PositionResponse(BaseModel):
    id: int = Field(..., description="포지션 고유 ID")
    symbol: str = Field(..., description="종목 코드")
    name: str | None = Field(None, description="종목명 (StockMaster 조회)")
    market: str = Field(..., description="시장 구분 (KR/US)")
    quantity: int = Field(..., description="보유 수량")
    avg_price: float = Field(..., description="평균 매입 단가")
    is_paper: bool = Field(..., description="모의투자 포지션 여부 (true: 모의, false: 실매매)")
    current_price: float = Field(0.0, description="현재가 (실시간 조회)")
    unrealized_pnl: float = Field(0.0, description="미실현 손익 = (현재가 - 평균매입가) × 수량")
    unrealized_pnl_pct: float = Field(0.0, description="미실현 손익률 (%) = (현재가 / 평균매입가 - 1) × 100")

    model_config = {"from_attributes": True}
