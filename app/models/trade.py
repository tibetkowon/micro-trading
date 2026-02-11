from __future__ import annotations

from sqlalchemy import ForeignKey, String
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class Trade(TimestampMixin, Base):
    __tablename__ = "trades"

    id: Mapped[int] = mapped_column(primary_key=True)
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"))
    order_id: Mapped[int] = mapped_column(ForeignKey("orders.id"))
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))
    side: Mapped[str] = mapped_column(String(4))
    quantity: Mapped[int] = mapped_column()
    price: Mapped[float] = mapped_column()
    realized_pnl: Mapped[float] = mapped_column(default=0.0)
    cost_basis: Mapped[float] = mapped_column(default=0.0)
    trading_mode: Mapped[str] = mapped_column(String(5))
