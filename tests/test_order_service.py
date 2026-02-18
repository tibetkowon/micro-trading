"""OrderService 테스트: 잔고검증, 수수료, 포지션 업데이트, 통화 구분."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from sqlalchemy import select

from app.broker.base import OrderResult, PriceInfo
from app.models.account import Account
from app.models.position import Position
from app.models.trade import Trade
from app.schemas.common import Market, OrderSide, OrderType, TradingMode
from app.schemas.order import OrderCreate
from app.services.order_service import OrderService


def _mock_broker(fill_price: float = 50000.0):
    """성공하는 mock 브로커 생성."""
    broker = AsyncMock()
    broker.place_order.return_value = OrderResult(
        success=True,
        broker_order_id="PAPER-TEST001",
        filled_price=fill_price,
        filled_quantity=None,  # OrderService가 req.quantity 사용
    )
    broker.get_current_price.return_value = PriceInfo(
        symbol="005930", price=fill_price, market="KR",
    )
    return broker


@pytest.mark.asyncio
async def test_buy_deducts_balance_and_commission(session, account):
    """매수 시 잔고에서 체결금액 + 수수료 차감 확인."""
    broker = _mock_broker(fill_price=50000.0)

    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.BUY,
            order_type=OrderType.MARKET, quantity=10,
            trading_mode=TradingMode.PAPER,
        )
        order = await svc.create_order(req)

    assert order.status == "FILLED"

    # 잔고 확인: 10,000,000 - (50,000 * 10) - (500,000 * 0.0005)
    await session.refresh(account)
    expected_cost = 50000.0 * 10  # 500,000
    commission = expected_cost * 0.0005  # 250
    assert account.paper_balance_krw == pytest.approx(10_000_000 - expected_cost - commission)

    # 거래 기록에 수수료 기록 확인
    trade = (await session.execute(select(Trade))).scalar_one()
    assert trade.commission == pytest.approx(commission)
    assert trade.total_amount == pytest.approx(expected_cost)


@pytest.mark.asyncio
async def test_sell_increases_balance(session, account):
    """매도 시 잔고에 체결금액 - 수수료 입금 확인."""
    # 먼저 포지션 생성
    pos = Position(
        account_id=account.id, symbol="005930", market="KR",
        quantity=10, avg_price=45000.0, is_paper=True,
    )
    session.add(pos)
    await session.commit()

    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.SELL,
            order_type=OrderType.MARKET, quantity=10,
            trading_mode=TradingMode.PAPER,
        )
        order = await svc.create_order(req)

    assert order.status == "FILLED"

    await session.refresh(account)
    sell_amount = 50000.0 * 10  # 500,000
    commission = sell_amount * 0.0005  # 250
    assert account.paper_balance_krw == pytest.approx(10_000_000 + sell_amount - commission)

    # realized_pnl 확인
    trade = (await session.execute(select(Trade))).scalar_one()
    expected_pnl = (50000.0 - 45000.0) * 10  # 50,000
    assert trade.realized_pnl == pytest.approx(expected_pnl)


@pytest.mark.asyncio
async def test_buy_insufficient_balance_raises(session, account):
    """잔고 부족 시 ValueError 발생 확인."""
    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.BUY,
            order_type=OrderType.MARKET, quantity=300,  # 50,000 * 300 = 15,000,000 > 10,000,000
            trading_mode=TradingMode.PAPER,
        )
        with pytest.raises(ValueError, match="잔고 부족"):
            await svc.create_order(req)


@pytest.mark.asyncio
async def test_sell_insufficient_position_raises(session, account):
    """보유 수량 부족 시 ValueError 발생 확인."""
    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.SELL,
            order_type=OrderType.MARKET, quantity=5,
            trading_mode=TradingMode.PAPER,
        )
        with pytest.raises(ValueError, match="보유 수량 부족"):
            await svc.create_order(req)


@pytest.mark.asyncio
async def test_buy_creates_position(session, account):
    """매수 후 포지션 생성 확인."""
    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.BUY,
            order_type=OrderType.MARKET, quantity=10,
            trading_mode=TradingMode.PAPER,
        )
        await svc.create_order(req)

    pos = (await session.execute(
        select(Position).where(Position.symbol == "005930")
    )).scalar_one()
    assert pos.quantity == 10
    assert pos.avg_price == pytest.approx(50000.0, rel=0.01)


@pytest.mark.asyncio
async def test_sell_all_deletes_position(session, account):
    """전량 매도 시 포지션 삭제 확인."""
    pos = Position(
        account_id=account.id, symbol="005930", market="KR",
        quantity=10, avg_price=45000.0, is_paper=True,
    )
    session.add(pos)
    await session.commit()

    broker = _mock_broker(fill_price=50000.0)
    with patch("app.services.order_service.get_broker", return_value=broker):
        svc = OrderService(session)
        req = OrderCreate(
            symbol="005930", market=Market.KR, side=OrderSide.SELL,
            order_type=OrderType.MARKET, quantity=10,
            trading_mode=TradingMode.PAPER,
        )
        await svc.create_order(req)

    result = await session.execute(
        select(Position).where(Position.symbol == "005930")
    )
    assert result.scalar_one_or_none() is None
