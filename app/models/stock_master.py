from __future__ import annotations

from sqlalchemy import String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class StockMaster(TimestampMixin, Base):
    """KRX/US 전체 상장 종목 마스터 테이블"""

    __tablename__ = "stock_master"
    __table_args__ = (UniqueConstraint("symbol", "market"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))  # KR / US
    name: Mapped[str] = mapped_column(String(100))
    sector: Mapped[str | None] = mapped_column(String(100), default=None)

    def __repr__(self) -> str:
        return f"<StockMaster symbol={self.symbol!r} market={self.market} name={self.name!r}>"
