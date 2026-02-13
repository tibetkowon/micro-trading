from __future__ import annotations

from datetime import datetime, timezone

from sqlalchemy import DateTime, String, UniqueConstraint, func
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base


class PriceCache(Base):
    """시세 캐시 모델 (재시작 시 마지막 시세 복원용)"""

    __tablename__ = "price_cache"
    __table_args__ = (UniqueConstraint("symbol", "market", name="uq_price_cache"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))
    price: Mapped[float] = mapped_column(default=0.0)
    change: Mapped[float] = mapped_column(default=0.0)
    change_pct: Mapped[float] = mapped_column(default=0.0)
    volume: Mapped[int] = mapped_column(default=0)
    high: Mapped[float] = mapped_column(default=0.0)
    low: Mapped[float] = mapped_column(default=0.0)
    open: Mapped[float] = mapped_column(default=0.0)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
        onupdate=lambda: datetime.now(timezone.utc),
        server_default=func.now(),
    )

    def __repr__(self) -> str:
        return f"<PriceCache symbol={self.symbol!r} market={self.market} price={self.price}>"
