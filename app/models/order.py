from __future__ import annotations

from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, String
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class Order(TimestampMixin, Base):
    __tablename__ = "orders"

    id: Mapped[int] = mapped_column(primary_key=True)
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"))
    broker_order_id: Mapped[str | None] = mapped_column(String(100), default=None)
    symbol: Mapped[str] = mapped_column(String(20), index=True)
    market: Mapped[str] = mapped_column(String(5))  # KR / US
    side: Mapped[str] = mapped_column(String(4))  # BUY / SELL
    order_type: Mapped[str] = mapped_column(String(10))  # MARKET / LIMIT
    quantity: Mapped[int] = mapped_column()
    price: Mapped[float | None] = mapped_column(default=None)
    filled_quantity: Mapped[int] = mapped_column(default=0)
    filled_price: Mapped[float | None] = mapped_column(default=None)
    trading_mode: Mapped[str] = mapped_column(String(5))  # REAL / PAPER
    status: Mapped[str] = mapped_column(String(20), default="PENDING")
    source: Mapped[str] = mapped_column(String(20), default="manual")
    strategy_name: Mapped[str | None] = mapped_column(String(100), default=None)
    filled_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), default=None)
