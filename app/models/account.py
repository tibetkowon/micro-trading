from __future__ import annotations

from sqlalchemy import String
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class Account(TimestampMixin, Base):
    __tablename__ = "accounts"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(100), index=True)
    broker_type: Mapped[str] = mapped_column(String(20), default="KIS")
    account_number: Mapped[str] = mapped_column(String(50), default="")
    paper_balance_krw: Mapped[float] = mapped_column(default=100_000_000.0)
    paper_balance_usd: Mapped[float] = mapped_column(default=100_000.0)
    initial_balance_krw: Mapped[float] = mapped_column(default=100_000_000.0)
    initial_balance_usd: Mapped[float] = mapped_column(default=100_000.0)
    commission_rate: Mapped[float] = mapped_column(default=0.0005)

    def __repr__(self) -> str:
        return f"<Account id={self.id} name={self.name!r} broker={self.broker_type}>"
