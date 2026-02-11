"""Abstract strategy base class and Signal type."""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass
class Signal:
    symbol: str
    market: str = "KR"
    side: str = ""  # BUY / SELL / "" (no action)
    quantity: int = 0
    order_type: str = "MARKET"
    price: float | None = None
    reason: str = ""

    @property
    def is_active(self) -> bool:
        return self.side in ("BUY", "SELL") and self.quantity > 0


class AbstractStrategy(ABC):
    """Base class for all trading strategies."""

    name: str = "base"

    def __init__(self, params: dict[str, Any] | None = None):
        self.params = params or {}

    @abstractmethod
    async def evaluate(
        self,
        symbol: str,
        market: str,
        daily_prices: list[dict],
        current_price: float,
        position_qty: int,
    ) -> Signal:
        """Evaluate strategy and return a signal."""
        ...

    def get_param(self, key: str, default: Any = None) -> Any:
        return self.params.get(key, default)
