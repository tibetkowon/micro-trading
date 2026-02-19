from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.order import OrderCreate, OrderResponse
from app.services.order_service import OrderService
from app.services.stock_master_service import StockMasterService

router = APIRouter(prefix="/orders", tags=["orders"])


def _with_name(order, name_map: dict) -> OrderResponse:
    """주문 객체를 OrderResponse로 변환하며 종목명을 추가한다."""
    resp = OrderResponse.model_validate(order)
    resp.name = name_map.get((order.symbol, order.market))
    return resp


@router.post("", response_model=OrderResponse)
async def create_order(
    req: OrderCreate,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.create_order(req)
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk([(order.symbol, order.market)])
    return _with_name(order, name_map)


@router.get("", response_model=list[OrderResponse])
async def list_orders(
    trading_mode: str | None = None,
    status: str | None = None,
    limit: int = 50,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    orders = await svc.get_orders(trading_mode=trading_mode, status=status, limit=limit)
    stock_svc = StockMasterService(session)
    symbols = list({(o.symbol, o.market) for o in orders})
    name_map = await stock_svc.get_names_bulk(symbols)
    return [_with_name(o, name_map) for o in orders]


@router.delete("/{order_id}", response_model=OrderResponse)
async def cancel_order(
    order_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.cancel_order(order_id)
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk([(order.symbol, order.market)])
    return _with_name(order, name_map)
