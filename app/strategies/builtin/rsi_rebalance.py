"""RSI-based rebalancing strategy."""

from __future__ import annotations

from app.strategies.base import AbstractStrategy, Signal


def _compute_rsi(closes: list[float], period: int = 14) -> float:
    """Compute RSI from a chronological list of close prices."""
    if len(closes) < period + 1:
        return 50.0  # neutral default

    deltas = [closes[i] - closes[i - 1] for i in range(1, len(closes))]
    recent = deltas[-period:]

    gains = [d for d in recent if d > 0]
    losses = [-d for d in recent if d < 0]

    avg_gain = sum(gains) / period if gains else 0.0
    avg_loss = sum(losses) / period if losses else 0.0001

    rs = avg_gain / avg_loss
    return 100 - (100 / (1 + rs))


class RSIRebalanceStrategy(AbstractStrategy):
    """Buy when RSI is oversold, sell when overbought.

    Params:
        rsi_period: RSI calculation period (default: 14)
        oversold: RSI threshold to buy (default: 30)
        overbought: RSI threshold to sell (default: 70)
        quantity: Shares per trade (default: 10)
    """

    name = "RSI_REBALANCE"

    async def evaluate(
        self,
        symbol: str,
        market: str,
        daily_prices: list[dict],
        current_price: float,
        position_qty: int,
    ) -> Signal:
        rsi_period = self.get_param("rsi_period", 14)
        oversold = self.get_param("oversold", 30)
        overbought = self.get_param("overbought", 70)
        quantity = self.get_param("quantity", 10)

        if len(daily_prices) < rsi_period + 2:
            return Signal(
                symbol=symbol, market=market,
                reason=f"Not enough data ({len(daily_prices)}/{rsi_period + 2})",
            )

        closes = [p["close"] for p in reversed(daily_prices)]
        rsi = _compute_rsi(closes, rsi_period)

        if rsi < oversold:
            return Signal(
                symbol=symbol, market=market,
                side="BUY", quantity=quantity,
                reason=f"RSI={rsi:.1f} < {oversold} (oversold)",
            )

        if rsi > overbought and position_qty > 0:
            sell_qty = min(quantity, position_qty)
            return Signal(
                symbol=symbol, market=market,
                side="SELL", quantity=sell_qty,
                reason=f"RSI={rsi:.1f} > {overbought} (overbought)",
            )

        return Signal(
            symbol=symbol, market=market,
            reason=f"RSI={rsi:.1f} (neutral range {oversold}-{overbought})",
        )
