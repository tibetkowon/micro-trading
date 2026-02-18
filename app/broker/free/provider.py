"""Free market data provider using pykrx (KR)."""

from __future__ import annotations

import asyncio
import logging
from datetime import datetime, timedelta

from app.broker.base import PriceInfo

logger = logging.getLogger(__name__)


class FreeMarketProvider:
    """Provides market data without KIS credentials."""

    async def get_current_price(self, symbol: str, market: str) -> PriceInfo:
        return await self._get_kr_price(symbol)

    async def get_daily_prices(
        self, symbol: str, market: str, days: int = 60
    ) -> list[dict]:
        return await self._get_kr_daily(symbol, days)

    # ── KR (pykrx) ──────────────────────────────────────────────

    async def _get_kr_price(self, symbol: str) -> PriceInfo:
        def _fetch():
            from pykrx import stock as pykrx_stock

            today = datetime.now().strftime("%Y%m%d")
            week_ago = (datetime.now() - timedelta(days=7)).strftime("%Y%m%d")

            df = pykrx_stock.get_market_ohlcv_by_date(week_ago, today, symbol)
            if df.empty:
                return PriceInfo(symbol=symbol, price=0.0, market="KR")

            latest = df.iloc[-1]
            price = float(latest["종가"])
            volume = int(latest["거래량"])

            if len(df) >= 2:
                prev_close = float(df.iloc[-2]["종가"])
                change = price - prev_close
                change_pct = (change / prev_close * 100) if prev_close else 0.0
            else:
                change = 0.0
                change_pct = 0.0

            return PriceInfo(
                symbol=symbol,
                price=price,
                change=change,
                change_pct=change_pct,
                volume=volume,
                market="KR",
            )

        return await asyncio.to_thread(_fetch)

    async def _get_kr_daily(self, symbol: str, days: int) -> list[dict]:
        def _fetch():
            from pykrx import stock as pykrx_stock

            today = datetime.now().strftime("%Y%m%d")
            start = (datetime.now() - timedelta(days=int(days * 1.6))).strftime(
                "%Y%m%d"
            )

            df = pykrx_stock.get_market_ohlcv_by_date(start, today, symbol)
            if df.empty:
                return []

            df = df.tail(days)
            result = []
            for date_idx, row in df.iterrows():
                result.append(
                    {
                        "date": date_idx.strftime("%Y-%m-%d"),
                        "open": float(row["시가"]),
                        "high": float(row["고가"]),
                        "low": float(row["저가"]),
                        "close": float(row["종가"]),
                        "volume": int(row["거래량"]),
                    }
                )
            return result

        return await asyncio.to_thread(_fetch)
