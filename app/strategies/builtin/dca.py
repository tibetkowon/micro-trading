"""Dollar-Cost Averaging (DCA) strategy - periodic fixed-amount buying."""

from __future__ import annotations

from app.strategies.base import AbstractStrategy, Signal


class DCAStrategy(AbstractStrategy):
    """Buy a fixed amount at regular intervals regardless of price.

    Params:
        amount_per_buy: Amount in currency to spend per interval (default: 100000 KRW)
    """

    name = "DCA"

    async def evaluate(
        self,
        symbol: str,
        market: str,
        daily_prices: list[dict],
        current_price: float,
        position_qty: int,
    ) -> Signal:
        if current_price <= 0:
            return Signal(symbol=symbol, market=market, reason="No price data")

        amount = self.get_param("amount_per_buy", 100_000)
        quantity = int(amount / current_price)

        if quantity <= 0:
            return Signal(
                symbol=symbol,
                market=market,
                reason=f"Price {current_price} too high for budget {amount}",
            )

        return Signal(
            symbol=symbol,
            market=market,
            side="BUY",
            quantity=quantity,
            reason=f"DCA buy {quantity} shares at {current_price}",
        )
