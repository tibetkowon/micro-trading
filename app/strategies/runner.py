"""Strategy runner - loads active strategies, evaluates, and places orders."""

from __future__ import annotations

import json
import logging

from sqlalchemy import select

from app.database import async_session
from app.models.position import Position
from app.models.strategy import StrategyConfig
from app.schemas.common import Market, OrderSide, OrderType, TradingMode
from app.schemas.order import OrderCreate
from app.services.market_service import MarketService
from app.services.order_service import OrderService
from app.strategies.base import AbstractStrategy

logger = logging.getLogger(__name__)

# Strategy type registry
STRATEGY_REGISTRY: dict[str, type[AbstractStrategy]] = {}


def _load_registry():
    if STRATEGY_REGISTRY:
        return
    from app.strategies.builtin.dca import DCAStrategy
    from app.strategies.builtin.moving_average import MovingAverageStrategy
    from app.strategies.builtin.rsi_rebalance import RSIRebalanceStrategy

    STRATEGY_REGISTRY["DCA"] = DCAStrategy
    STRATEGY_REGISTRY["MOVING_AVERAGE"] = MovingAverageStrategy
    STRATEGY_REGISTRY["RSI_REBALANCE"] = RSIRebalanceStrategy


class StrategyRunner:

    def __init__(self):
        self.market_svc = MarketService()
        _load_registry()

    async def run_all(self):
        """Run all active strategies."""
        async with async_session() as session:
            result = await session.execute(
                select(StrategyConfig).where(StrategyConfig.is_active == True)
            )
            configs = result.scalars().all()

        for config in configs:
            try:
                await self._run_strategy(config)
            except Exception as e:
                logger.error("Strategy %s failed: %s", config.name, e, exc_info=True)

    async def _run_strategy(self, config: StrategyConfig):
        strategy_cls = STRATEGY_REGISTRY.get(config.strategy_type)
        if not strategy_cls:
            logger.warning("Unknown strategy type: %s", config.strategy_type)
            return

        params = json.loads(config.params) if isinstance(config.params, str) else config.params
        strategy = strategy_cls(params)
        symbols = json.loads(config.symbols) if isinstance(config.symbols, str) else config.symbols

        for symbol in symbols:
            try:
                await self._evaluate_and_act(strategy, config, symbol)
            except Exception as e:
                logger.error("Strategy %s symbol %s error: %s", config.name, symbol, e)

    async def _evaluate_and_act(
        self,
        strategy: AbstractStrategy,
        config: StrategyConfig,
        symbol: str,
    ):
        market = config.market

        # Get market data
        price_info = await self.market_svc.get_price(symbol, market)
        daily_prices = await self.market_svc.get_daily_prices(symbol, market)

        # Get current position
        is_paper = config.trading_mode == "PAPER"
        async with async_session() as session:
            result = await session.execute(
                select(Position).where(
                    Position.symbol == symbol,
                    Position.market == market,
                    Position.is_paper == is_paper,
                )
            )
            position = result.scalar_one_or_none()
            position_qty = position.quantity if position else 0

        # Evaluate
        signal = await strategy.evaluate(
            symbol=symbol,
            market=market,
            daily_prices=daily_prices,
            current_price=price_info.price,
            position_qty=position_qty,
        )

        logger.info("Strategy %s → %s %s: %s", config.name, symbol, signal.side or "HOLD", signal.reason)

        if not signal.is_active:
            return

        # Place order
        order_req = OrderCreate(
            symbol=symbol,
            market=Market(market),
            side=OrderSide(signal.side),
            order_type=OrderType(signal.order_type),
            quantity=signal.quantity,
            price=signal.price,
            trading_mode=TradingMode(config.trading_mode),
        )

        async with async_session() as session:
            order_svc = OrderService(session)
            order = await order_svc.create_order(
                order_req, source="strategy", strategy_name=config.name,
            )
            logger.info(
                "Strategy %s placed order #%d: %s %s %d @ %s",
                config.name, order.id, signal.side, symbol,
                signal.quantity, order.filled_price,
            )
