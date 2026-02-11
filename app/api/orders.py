from __future__ import annotations

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import get_session
from app.schemas.order import OrderCreate, OrderResponse
from app.services.order_service import OrderService

router = APIRouter(prefix="/orders", tags=["orders"])


@router.post("", response_model=OrderResponse)
async def create_order(
    req: OrderCreate,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.create_order(req)
    return OrderResponse.model_validate(order)


@router.get("", response_model=list[OrderResponse])
async def list_orders(
    trading_mode: str | None = None,
    status: str | None = None,
    limit: int = 50,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    orders = await svc.get_orders(trading_mode=trading_mode, status=status, limit=limit)
    return [OrderResponse.model_validate(o) for o in orders]


@router.delete("/{order_id}", response_model=OrderResponse)
async def cancel_order(
    order_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.cancel_order(order_id)
    return OrderResponse.model_validate(order)
