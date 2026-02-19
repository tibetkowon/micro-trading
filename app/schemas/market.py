from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, Field


class PriceResponse(BaseModel):
    symbol: str = Field(..., description="종목 코드")
    name: str | None = Field(None, description="종목명 (StockMaster 조회)")
    price: float = Field(..., description="현재가")
    change: float = Field(0.0, description="전일 대비 변동액")
    change_pct: float = Field(0.0, description="전일 대비 변동률 (%)")
    volume: int = Field(0, description="거래량")
    market: str = Field("KR", description="시장 구분 (KR/US)")
    ma5: float | None = Field(None, description="5일 이동평균 (MA5)")
    ma20: float | None = Field(None, description="20일 이동평균 (MA20)")


class PriceCacheResponse(BaseModel):
    symbol: str = Field(..., description="종목 코드")
    market: str = Field(..., description="시장 구분 (KR/US)")
    price: float = Field(..., description="캐시된 현재가")
    change: float = Field(0.0, description="전일 대비 변동액")
    change_pct: float = Field(0.0, description="전일 대비 변동률 (%)")
    volume: int = Field(0, description="거래량")
    high: float = Field(0.0, description="당일 고가")
    low: float = Field(0.0, description="당일 저가")
    open: float = Field(0.0, description="당일 시가")
    updated_at: datetime | None = Field(None, description="캐시 마지막 갱신 시각 (UTC)")

    model_config = {"from_attributes": True}
