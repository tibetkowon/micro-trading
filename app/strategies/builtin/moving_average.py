"""Moving Average Crossover strategy."""

from __future__ import annotations

from app.strategies.base import AbstractStrategy, Signal


class MovingAverageStrategy(AbstractStrategy):
    """Buy when short MA crosses above long MA, sell when it crosses below.

    Params:
        short_period: Short MA period (default: 5)
        long_period: Long MA period (default: 20)
        quantity: Number of shares per trade (default: 10)
    """

    name = "MOVING_AVERAGE"

    async def evaluate(
        self,
        symbol: str,
        market: str,
        daily_prices: list[dict],
        current_price: float,
        position_qty: int,
    ) -> Signal:
        short_period = self.get_param("short_period", 5)
        long_period = self.get_param("long_period", 20)
        quantity = self.get_param("quantity", 10)

        if len(daily_prices) < long_period + 1:
            return Signal(
                symbol=symbol, market=market,
                reason=f"Not enough data ({len(daily_prices)}/{long_period + 1})",
            )

        # Calculate MAs (prices are newest-first, reverse for chronological)
        closes = [p["close"] for p in reversed(daily_prices)]

        short_ma_now = sum(closes[-short_period:]) / short_period
        short_ma_prev = sum(closes[-short_period - 1:-1]) / short_period
        long_ma_now = sum(closes[-long_period:]) / long_period
        long_ma_prev = sum(closes[-long_period - 1:-1]) / long_period

        # Golden cross: short crosses above long
        if short_ma_prev <= long_ma_prev and short_ma_now > long_ma_now:
            return Signal(
                symbol=symbol, market=market,
                side="BUY", quantity=quantity,
                reason=f"Golden cross: SMA{short_period}={short_ma_now:.2f} > SMA{long_period}={long_ma_now:.2f}",
            )

        # Death cross: short crosses below long → sell if holding
        if short_ma_prev >= long_ma_prev and short_ma_now < long_ma_now and position_qty > 0:
            sell_qty = min(quantity, position_qty)
            return Signal(
                symbol=symbol, market=market,
                side="SELL", quantity=sell_qty,
                reason=f"Death cross: SMA{short_period}={short_ma_now:.2f} < SMA{long_period}={long_ma_now:.2f}",
            )

        return Signal(
            symbol=symbol, market=market,
            reason=f"No cross: SMA{short_period}={short_ma_now:.2f}, SMA{long_period}={long_ma_now:.2f}",
        )
