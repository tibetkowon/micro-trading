"""Order management service: validate → broker → DB → position update."""

from __future__ import annotations

import logging
from datetime import datetime, timezone

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.broker.factory import get_broker
from app.models.order import Order
from app.models.position import Position
from app.models.trade import Trade
from app.models.account import Account
from app.schemas.order import OrderCreate

logger = logging.getLogger(__name__)


class OrderService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def _get_default_account(self) -> Account:
        result = await self.session.execute(select(Account).limit(1))
        account = result.scalar_one_or_none()
        if not account:
            raise ValueError("No account found")
        return account

    async def create_order(
        self,
        req: OrderCreate,
        source: str = "manual",
        strategy_name: str | None = None,
    ) -> Order:
        account = await self._get_default_account()
        broker = await get_broker(req.trading_mode.value)

        # Create order record
        order = Order(
            account_id=account.id,
            symbol=req.symbol,
            market=req.market.value,
            side=req.side.value,
            order_type=req.order_type.value,
            quantity=req.quantity,
            price=req.price,
            trading_mode=req.trading_mode.value,
            status="SUBMITTED",
            source=source,
            strategy_name=strategy_name,
        )
        self.session.add(order)
        await self.session.flush()

        # Place via broker
        result = await broker.place_order(
            symbol=req.symbol,
            market=req.market.value,
            side=req.side.value,
            order_type=req.order_type.value,
            quantity=req.quantity,
            price=req.price,
        )

        if result.success:
            order.broker_order_id = result.broker_order_id
            order.status = "FILLED"
            order.filled_quantity = result.filled_quantity or req.quantity
            order.filled_price = result.filled_price or req.price
            order.filled_at = datetime.now(timezone.utc)

            # Record trade
            filled_price = order.filled_price or 0.0
            trade = Trade(
                account_id=account.id,
                order_id=order.id,
                symbol=req.symbol,
                market=req.market.value,
                side=req.side.value,
                quantity=order.filled_quantity,
                price=filled_price,
                trading_mode=req.trading_mode.value,
            )

            # Update position
            is_paper = req.trading_mode == "PAPER"
            await self._update_position(
                account.id, req.symbol, req.market.value, req.side.value,
                order.filled_quantity, filled_price, is_paper, trade,
            )
            self.session.add(trade)
        else:
            order.status = "REJECTED"
            logger.warning("Order rejected: %s", result.message)

        await self.session.commit()
        return order

    async def _update_position(
        self,
        account_id: int,
        symbol: str,
        market: str,
        side: str,
        quantity: int,
        price: float,
        is_paper: bool,
        trade: Trade,
    ):
        result = await self.session.execute(
            select(Position).where(
                Position.account_id == account_id,
                Position.symbol == symbol,
                Position.market == market,
                Position.is_paper == is_paper,
            )
        )
        position = result.scalar_one_or_none()

        if side == "BUY":
            if position is None:
                position = Position(
                    account_id=account_id,
                    symbol=symbol,
                    market=market,
                    quantity=quantity,
                    avg_price=price,
                    is_paper=is_paper,
                )
                self.session.add(position)
            else:
                total_cost = position.avg_price * position.quantity + price * quantity
                position.quantity += quantity
                position.avg_price = total_cost / position.quantity if position.quantity > 0 else 0
            trade.cost_basis = price * quantity
        else:  # SELL
            if position is None or position.quantity < quantity:
                raise ValueError(f"Insufficient position for {symbol}: have {position.quantity if position else 0}, need {quantity}")
            realized_pnl = (price - position.avg_price) * quantity
            trade.realized_pnl = realized_pnl
            trade.cost_basis = position.avg_price * quantity
            position.quantity -= quantity
            if position.quantity == 0:
                await self.session.delete(position)

    async def cancel_order(self, order_id: int) -> Order:
        result = await self.session.execute(select(Order).where(Order.id == order_id))
        order = result.scalar_one_or_none()
        if not order:
            raise ValueError(f"Order {order_id} not found")
        if order.status not in ("PENDING", "SUBMITTED"):
            raise ValueError(f"Cannot cancel order in status {order.status}")

        broker = await get_broker(order.trading_mode)
        if order.broker_order_id:
            await broker.cancel_order(order.broker_order_id, market=order.market)

        order.status = "CANCELLED"
        await self.session.commit()
        return order

    async def get_orders(
        self,
        trading_mode: str | None = None,
        status: str | None = None,
        limit: int = 50,
    ) -> list[Order]:
        stmt = select(Order).order_by(Order.created_at.desc()).limit(limit)
        if trading_mode:
            stmt = stmt.where(Order.trading_mode == trading_mode)
        if status:
            stmt = stmt.where(Order.status == status)
        result = await self.session.execute(stmt)
        return list(result.scalars().all())
