from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel


class WatchlistItemCreate(BaseModel):
    symbol: str
    market: str = "KR"
    name: str
    memo: str | None = None
    sort_order: int = 0


class WatchlistItemUpdate(BaseModel):
    name: str | None = None
    memo: str | None = None
    sort_order: int | None = None


class WatchlistItemResponse(BaseModel):
    id: int
    symbol: str
    market: str
    name: str
    memo: str | None = None
    sort_order: int = 0
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}
