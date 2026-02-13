from __future__ import annotations

from sqlalchemy import ForeignKey, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class Position(TimestampMixin, Base):
    __tablename__ = "positions"
    __table_args__ = (
        UniqueConstraint("account_id", "symbol", "market", "is_paper", name="uq_position"),
    )

    id: Mapped[int] = mapped_column(primary_key=True)
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"), index=True)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))
    quantity: Mapped[int] = mapped_column(default=0)
    avg_price: Mapped[float] = mapped_column(default=0.0)
    is_paper: Mapped[bool] = mapped_column(default=True)

    def __repr__(self) -> str:
        return f"<Position id={self.id} {self.symbol} qty={self.quantity} avg={self.avg_price}>"
