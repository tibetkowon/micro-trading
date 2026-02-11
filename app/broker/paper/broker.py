"""Paper broker - simulates trading using real market data."""

from __future__ import annotations

import logging
from typing import Any

from app.broker.base import AbstractBroker, BalanceInfo, OrderResult, PriceInfo
from app.broker.paper.engine import PaperExecutionEngine
from app.config import settings

logger = logging.getLogger(__name__)


class PaperBroker(AbstractBroker):
    """Paper trading broker backed by SQLite for balances and real KIS for prices."""

    def __init__(self):
        self._engine = PaperExecutionEngine()
        self._kis_broker = None  # Lazy init for price data

    async def _get_kis_broker(self):
        """Get KIS broker for price data (lazy init)."""
        if self._kis_broker is None:
            if settings.kis_app_key:
                from app.broker.kis.broker import KISBroker
                self._kis_broker = KISBroker()
                await self._kis_broker.connect()
            else:
                logger.warning("KIS credentials not set, paper broker will use dummy prices")
        return self._kis_broker

    async def connect(self) -> None:
        logger.info("PaperBroker connected")

    async def disconnect(self) -> None:
        if self._kis_broker:
            await self._kis_broker.disconnect()
        logger.info("PaperBroker disconnected")

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
        # Get real price
        current_price = await self.get_current_price(symbol, market)

        if order_type == "MARKET":
            result = self._engine.execute_market_order(symbol, side, quantity, current_price)
        else:
            if price is None:
                return OrderResult(success=False, message="Limit order requires price")
            result = self._engine.execute_limit_order(symbol, side, quantity, price, current_price)
            if result is None:
                return OrderResult(
                    success=False,
                    message=f"Limit price {price} not met (current: {current_price.price})",
                )

        logger.info(
            "Paper %s %s %d x %s @ %.4f",
            side, symbol, quantity, market, result["filled_price"],
        )

        return OrderResult(
            success=True,
            broker_order_id=result["broker_order_id"],
            filled_price=result["filled_price"],
            filled_quantity=result["filled_quantity"],
        )

    async def cancel_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        return OrderResult(success=True, message="Paper order cancelled")

    async def get_order_status(self, broker_order_id: str, **kwargs: Any) -> dict:
        return {"broker_order_id": broker_order_id, "status": "FILLED"}

    async def get_balance(self) -> BalanceInfo:
        from app.database import async_session
        from app.models.account import Account
        from sqlalchemy import select

        async with async_session() as session:
            result = await session.execute(select(Account).limit(1))
            account = result.scalar_one_or_none()
            if not account:
                return BalanceInfo()
            return BalanceInfo(
                cash_krw=account.paper_balance_krw,
                cash_usd=account.paper_balance_usd,
            )

    async def get_current_price(self, symbol: str, market: str) -> PriceInfo:
        kis = await self._get_kis_broker()
        if kis:
            try:
                return await kis.get_current_price(symbol, market)
            except Exception as e:
                logger.warning("KIS price fetch failed for %s: %s", symbol, e)
        # Fallback dummy price
        return PriceInfo(symbol=symbol, price=0.0, market=market)

    async def get_daily_prices(self, symbol: str, market: str, days: int = 60) -> list[dict]:
        kis = await self._get_kis_broker()
        if kis:
            try:
                return await kis.get_daily_prices(symbol, market, days)
            except Exception as e:
                logger.warning("KIS daily prices failed for %s: %s", symbol, e)
        return []
