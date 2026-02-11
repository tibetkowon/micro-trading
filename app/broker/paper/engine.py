"""Virtual execution engine for paper trading."""

from __future__ import annotations

import logging
import uuid
from datetime import datetime, timezone

from app.broker.base import PriceInfo

logger = logging.getLogger(__name__)


class PaperExecutionEngine:
    """Simulates order execution against real market prices."""

    def execute_market_order(
        self,
        symbol: str,
        side: str,
        quantity: int,
        current_price: PriceInfo,
    ) -> dict:
        """Execute a market order at current price with simulated slippage."""
        # Simulate tiny slippage (0.01%) for realism
        slippage = 0.0001
        if side == "BUY":
            fill_price = current_price.price * (1 + slippage)
        else:
            fill_price = current_price.price * (1 - slippage)

        return {
            "broker_order_id": f"PAPER-{uuid.uuid4().hex[:12].upper()}",
            "filled_price": round(fill_price, 4),
            "filled_quantity": quantity,
            "filled_at": datetime.now(timezone.utc),
        }

    def execute_limit_order(
        self,
        symbol: str,
        side: str,
        quantity: int,
        limit_price: float,
        current_price: PriceInfo,
    ) -> dict | None:
        """Try to execute a limit order. Returns None if price not met."""
        if side == "BUY" and current_price.price <= limit_price:
            return {
                "broker_order_id": f"PAPER-{uuid.uuid4().hex[:12].upper()}",
                "filled_price": current_price.price,
                "filled_quantity": quantity,
                "filled_at": datetime.now(timezone.utc),
            }
        elif side == "SELL" and current_price.price >= limit_price:
            return {
                "broker_order_id": f"PAPER-{uuid.uuid4().hex[:12].upper()}",
                "filled_price": current_price.price,
                "filled_quantity": quantity,
                "filled_at": datetime.now(timezone.utc),
            }
        return None
