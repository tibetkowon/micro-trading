"""Portfolio tracking, P&L calculation, daily snapshots."""

from __future__ import annotations

import logging
from datetime import date

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.account import Account
from app.models.position import Position
from app.models.trade import Trade
from app.models.portfolio_snapshot import PortfolioSnapshot
from app.schemas.portfolio import PortfolioSummary
from app.services.market_service import MarketService

logger = logging.getLogger(__name__)


class PortfolioService:

    def __init__(self, session: AsyncSession):
        self.session = session
        self.market_svc = MarketService(session)

    async def get_positions(self, is_paper: bool = True) -> list[dict]:
        result = await self.session.execute(
            select(Position).where(Position.is_paper == is_paper, Position.quantity > 0)
        )
        positions = result.scalars().all()

        # 종목명 일괄 조회
        from app.services.stock_master_service import StockMasterService
        stock_svc = StockMasterService(self.session)
        symbols = [(p.symbol, p.market) for p in positions]
        name_map = await stock_svc.get_names_bulk(symbols)

        enriched = []
        for pos in positions:
            try:
                price_info = await self.market_svc.get_price(pos.symbol, pos.market)
                current_price = price_info.price
            except Exception:
                current_price = pos.avg_price

            unrealized_pnl = (current_price - pos.avg_price) * pos.quantity
            pnl_pct = ((current_price / pos.avg_price) - 1) * 100 if pos.avg_price > 0 else 0

            enriched.append({
                "id": pos.id,
                "symbol": pos.symbol,
                "name": name_map.get((pos.symbol, pos.market)),
                "market": pos.market,
                "quantity": pos.quantity,
                "avg_price": pos.avg_price,
                "is_paper": pos.is_paper,
                "current_price": current_price,
                "unrealized_pnl": round(unrealized_pnl, 2),
                "unrealized_pnl_pct": round(pnl_pct, 2),
            })
        return enriched

    async def get_summary(self, trading_mode: str = "PAPER") -> PortfolioSummary:
        is_paper = trading_mode == "PAPER"
        account = (await self.session.execute(select(Account).limit(1))).scalar_one_or_none()
        if not account:
            return PortfolioSummary()

        positions = await self.get_positions(is_paper)

        total_invested = sum(p["avg_price"] * p["quantity"] for p in positions)
        total_market_value = sum(p["current_price"] * p["quantity"] for p in positions)
        unrealized_pnl = total_market_value - total_invested

        # Realized P&L from trades
        result = await self.session.execute(
            select(func.coalesce(func.sum(Trade.realized_pnl), 0.0))
            .where(Trade.trading_mode == trading_mode)
        )
        realized_pnl = float(result.scalar())

        cash_krw = account.paper_balance_krw if is_paper else 0.0
        cash_usd = 0.0
        total_value = total_market_value + cash_krw

        total_pnl = realized_pnl + unrealized_pnl
        initial = account.paper_balance_krw if is_paper else 0.0
        return_pct = (total_pnl / initial * 100) if initial > 0 else 0.0

        # 실질 주문가능 금액: 수수료 차감 후 실제 매수에 쓸 수 있는 금액
        commission_rate = account.commission_rate if is_paper else 0.0015
        orderable_krw = round(cash_krw / (1 + commission_rate), 2) if cash_krw > 0 else 0.0
        orderable_usd = round(cash_usd / (1 + commission_rate), 2) if cash_usd > 0 else 0.0

        return PortfolioSummary(
            total_value=round(total_value, 2),
            total_invested=round(total_invested, 2),
            cash_krw=round(cash_krw, 2),
            cash_usd=round(cash_usd, 2),
            initial_balance_krw=account.initial_balance_krw,
            initial_balance_usd=account.initial_balance_usd,
            realized_pnl=round(realized_pnl, 2),
            unrealized_pnl=round(unrealized_pnl, 2),
            total_pnl=round(total_pnl, 2),
            return_pct=round(return_pct, 2),
            orderable_krw=orderable_krw,
            orderable_usd=orderable_usd,
        )

    async def take_daily_snapshot(self):
        today = date.today()
        accounts = (await self.session.execute(select(Account))).scalars().all()

        for account in accounts:
            for mode in ("PAPER", "REAL"):
                existing = await self.session.execute(
                    select(PortfolioSnapshot).where(
                        PortfolioSnapshot.account_id == account.id,
                        PortfolioSnapshot.date == today,
                        PortfolioSnapshot.trading_mode == mode,
                    )
                )
                if existing.scalar_one_or_none():
                    continue

                summary = await self.get_summary(mode)
                snapshot = PortfolioSnapshot(
                    account_id=account.id,
                    date=today,
                    trading_mode=mode,
                    total_value=summary.total_value,
                    total_invested=summary.total_invested,
                    realized_pnl=summary.realized_pnl,
                    unrealized_pnl=summary.unrealized_pnl,
                )
                self.session.add(snapshot)

        await self.session.commit()
        logger.info("Daily snapshot taken for %s", today)

    async def get_snapshots(
        self,
        trading_mode: str = "PAPER",
        limit: int = 90,
    ) -> list[PortfolioSnapshot]:
        result = await self.session.execute(
            select(PortfolioSnapshot)
            .where(PortfolioSnapshot.trading_mode == trading_mode)
            .order_by(PortfolioSnapshot.date.desc())
            .limit(limit)
        )
        return list(result.scalars().all())
