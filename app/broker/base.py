from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Any


@dataclass
class OrderResult:
    success: bool
    broker_order_id: str | None = None
    filled_price: float | None = None
    filled_quantity: int | None = None
    message: str = ""


@dataclass
class BalanceInfo:
    cash_krw: float = 0.0
    cash_usd: float = 0.0
    total_value_krw: float = 0.0
    total_value_usd: float = 0.0


@dataclass
class PriceInfo:
    symbol: str
    price: float
    change: float = 0.0
    change_pct: float = 0.0
    volume: int = 0
    market: str = "KR"


class AbstractBroker(ABC):
    """Broker abstraction layer for both real and paper trading."""

    @abstractmethod
    async def connect(self) -> None:
        """Initialize connection / authenticate."""

    @abstractmethod
    async def disconnect(self) -> None:
        """Clean up resources."""

    @abstractmethod
    async def place_order(
        self,
        symbol: str,
        market: str,
        side: str,
        order_type: str,
        quantity: int,
        price: float | None = None,
        **kwargs: Any,
    ) -> OrderResult:
        """Submit a buy/sell order."""

    @abstractmethod
    async def cancel_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        """Cancel a pending order."""

    @abstractmethod
    async def get_order_status(self, broker_order_id: str, **kwargs: Any) -> dict:
        """Get current status of an order."""

    @abstractmethod
    async def get_balance(self) -> BalanceInfo:
        """Get account balance."""

    @abstractmethod
    async def get_current_price(self, symbol: str, market: str) -> PriceInfo:
        """Get current price for a symbol."""

    @abstractmethod
    async def get_daily_prices(self, symbol: str, market: str, days: int = 60) -> list[dict]:
        """Get daily OHLCV data for the last N days."""
