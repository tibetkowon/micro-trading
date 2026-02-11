from __future__ import annotations

from sqlalchemy import ForeignKey, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from app.models.base import Base, TimestampMixin


class StrategyConfig(TimestampMixin, Base):
    __tablename__ = "strategy_configs"

    id: Mapped[int] = mapped_column(primary_key=True)
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id"))
    name: Mapped[str] = mapped_column(String(100), unique=True)
    strategy_type: Mapped[str] = mapped_column(String(50))
    symbols: Mapped[str] = mapped_column(Text, default="[]")  # JSON list
    market: Mapped[str] = mapped_column(String(5))
    params: Mapped[str] = mapped_column(Text, default="{}")  # JSON dict
    trading_mode: Mapped[str] = mapped_column(String(5), default="PAPER")
    is_active: Mapped[bool] = mapped_column(default=False)
    schedule_cron: Mapped[str] = mapped_column(String(100), default="")
