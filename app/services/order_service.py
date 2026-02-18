"""주문 관리 서비스: 잔고검증 → 브로커 실행 → 수수료 계산 → 잔고/포지션 업데이트."""

from __future__ import annotations

import logging
from datetime import datetime, timezone

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.broker.factory import get_broker
from app.config import settings as app_settings
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
        is_paper = req.trading_mode.value == "PAPER"

        # 매수 시 예상 금액으로 잔고 사전 검증
        if req.side.value == "BUY" and is_paper:
            estimated_price = req.price or 0.0
            if estimated_price <= 0:
                # 시장가 주문이면 현재가 조회
                price_info = await broker.get_current_price(req.symbol, req.market.value)
                estimated_price = price_info.price
            estimated_total = estimated_price * req.quantity
            commission_rate = (
                app_settings.real_commission_rate
                if req.trading_mode.value == "REAL"
                else account.commission_rate
            )
            commission = estimated_total * commission_rate
            required = estimated_total + commission
            available = account.paper_balance_krw
            if available < required:
                raise ValueError(
                    f"잔고 부족: 필요 {required:,.0f}, 보유 {available:,.0f}"
                )

        # 주문 레코드 생성
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

        # 브로커를 통해 주문 실행
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

            filled_price = order.filled_price or 0.0
            total_amount = filled_price * order.filled_quantity
            commission_rate = (
                app_settings.real_commission_rate
                if req.trading_mode.value == "REAL"
                else account.commission_rate
            )
            commission = round(total_amount * commission_rate, 2)

            # 거래 기록 생성 (수수료 포함)
            trade = Trade(
                account_id=account.id,
                order_id=order.id,
                symbol=req.symbol,
                market=req.market.value,
                side=req.side.value,
                quantity=order.filled_quantity,
                price=filled_price,
                total_amount=total_amount,
                commission=commission,
                trading_mode=req.trading_mode.value,
            )

            # 포지션 업데이트 + 잔고 변경
            await self._update_position(
                account, req.market.value, req.side.value,
                req.symbol, order.filled_quantity, filled_price,
                is_paper, trade, commission,
            )
            self.session.add(trade)
        else:
            order.status = "REJECTED"
            logger.warning("주문 거부: %s", result.message)

        await self.session.commit()
        return order

    async def _update_position(
        self,
        account: Account,
        market: str,
        side: str,
        symbol: str,
        quantity: int,
        price: float,
        is_paper: bool,
        trade: Trade,
        commission: float,
    ):
        """포지션 업데이트 및 가상 지갑 잔고 변경."""
        result = await self.session.execute(
            select(Position).where(
                Position.account_id == account.id,
                Position.symbol == symbol,
                Position.market == market,
                Position.is_paper == is_paper,
            )
        )
        position = result.scalar_one_or_none()
        total_amount = price * quantity

        if side == "BUY":
            if position is None:
                position = Position(
                    account_id=account.id,
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
            trade.cost_basis = total_amount

            # 매수: 잔고 차감 (체결금액 + 수수료)
            if is_paper:
                account.paper_balance_krw = round(account.paper_balance_krw - total_amount - commission, 2)

        else:  # SELL
            if position is None or position.quantity < quantity:
                raise ValueError(
                    f"{symbol} 보유 수량 부족: 보유 {position.quantity if position else 0}, 필요 {quantity}"
                )
            realized_pnl = (price - position.avg_price) * quantity
            trade.realized_pnl = round(realized_pnl, 2)
            trade.cost_basis = position.avg_price * quantity
            position.quantity -= quantity
            if position.quantity == 0:
                await self.session.delete(position)

            # 매도: 잔고 증가 (체결금액 - 수수료)
            if is_paper:
                account.paper_balance_krw = round(account.paper_balance_krw + total_amount - commission, 2)

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

    async def get_trades_by_order_ids(self, order_ids: list[int]) -> dict[int, float]:
        """주문 ID 목록에 대한 수수료 매핑 반환. {order_id: commission}"""
        if not order_ids:
            return {}
        stmt = select(Trade).where(Trade.order_id.in_(order_ids))
        result = await self.session.execute(stmt)
        return {t.order_id: t.commission for t in result.scalars().all()}

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
