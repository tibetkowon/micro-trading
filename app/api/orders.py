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


@router.post(
    "",
    response_model=OrderResponse,
    summary="주문 생성",
    description="새 매수/매도 주문을 생성하고 즉시 체결을 시도합니다. "
                "모의투자 모드에서는 가상 잔고에서 차감되며, 실매매 모드에서는 KIS API로 실제 주문이 전송됩니다.",
)
async def create_order(
    req: OrderCreate,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.create_order(req)
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk([(order.symbol, order.market)])
    return _with_name(order, name_map)


@router.get(
    "",
    response_model=list[OrderResponse],
    summary="주문 목록 조회",
    description="거래 모드, 주문 상태, 최대 건수 필터로 주문 내역을 조회합니다. "
                "기본값은 최신 50건이며 created_at 내림차순으로 반환됩니다.",
)
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


@router.delete(
    "/{order_id}",
    response_model=OrderResponse,
    summary="주문 취소",
    description="PENDING 또는 SUBMITTED 상태의 주문을 취소합니다. "
                "이미 체결(FILLED)되었거나 취소(CANCELLED)된 주문은 취소할 수 없습니다.",
)
async def cancel_order(
    order_id: int,
    session: AsyncSession = Depends(get_session),
):
    svc = OrderService(session)
    order = await svc.cancel_order(order_id)
    stock_svc = StockMasterService(session)
    name_map = await stock_svc.get_names_bulk([(order.symbol, order.market)])
    return _with_name(order, name_map)
