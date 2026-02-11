from __future__ import annotations

from sqlalchemy import String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class StockMemo(TimestampMixin, Base):
    __tablename__ = "stock_memos"
    __table_args__ = (UniqueConstraint("symbol", "market"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))  # KR / US
    name: Mapped[str] = mapped_column(String(100))
    memo: Mapped[str | None] = mapped_column(String(200), default=None)
