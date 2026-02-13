from __future__ import annotations

from sqlalchemy import String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class WatchlistItem(TimestampMixin, Base):
    """관심종목 모델"""

    __tablename__ = "watchlist_items"
    __table_args__ = (UniqueConstraint("symbol", "market"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))  # KR / US
    name: Mapped[str] = mapped_column(String(100))
    memo: Mapped[str | None] = mapped_column(String(500), default=None)
    sort_order: Mapped[int] = mapped_column(default=0)

    def __repr__(self) -> str:
        return f"<WatchlistItem id={self.id} symbol={self.symbol!r} market={self.market}>"
