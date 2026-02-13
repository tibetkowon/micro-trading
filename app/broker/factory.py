"""Broker factory - returns the right broker based on trading mode."""

from __future__ import annotations

from app.broker.base import AbstractBroker
from app.schemas.common import TradingMode


_broker_cache: dict[TradingMode, AbstractBroker] = {}


async def get_broker(mode: TradingMode | str | None = None) -> AbstractBroker:
    """Get or create a broker instance for the given trading mode."""
    if mode is None:
        from app.config import settings
        mode = settings.trading_mode
    if isinstance(mode, str):
        mode = TradingMode(mode)

    if mode not in _broker_cache:
        if mode == TradingMode.REAL:
            from app.broker.kis.broker import KISBroker
            broker = KISBroker()
        else:
            from app.broker.paper.broker import PaperBroker
            broker = PaperBroker()
        await broker.connect()
        _broker_cache[mode] = broker

    return _broker_cache[mode]


async def close_all_brokers():
    for broker in _broker_cache.values():
        await broker.disconnect()
    _broker_cache.clear()
