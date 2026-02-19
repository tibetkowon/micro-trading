"""KIS broker implementation for real trading."""

from __future__ import annotations

import logging
from datetime import datetime
from typing import Any

from app.broker.base import AbstractBroker, BalanceInfo, OrderResult, PriceInfo
from app.broker.kis.client import KISClient
from app.broker.kis import endpoints as ep
from app.config import settings

logger = logging.getLogger(__name__)


def _aggregate_candles(candles: list[dict], interval: int) -> list[dict]:
    """1분봉 목록을 N분봉으로 집계한다 (오름차순 입력 기준)."""
    if interval <= 1 or not candles:
        return candles
    aggregated = []
    bucket: list[dict] = []
    for candle in candles:
        bucket.append(candle)
        if len(bucket) == interval:
            aggregated.append({
                "datetime": bucket[0]["datetime"],
                "open": bucket[0]["open"],
                "high": max(c["high"] for c in bucket),
                "low": min(c["low"] for c in bucket),
                "close": bucket[-1]["close"],
                "volume": sum(c["volume"] for c in bucket),
            })
            bucket = []
    return aggregated


class KISBroker(AbstractBroker):

    def __init__(self):
        self._client = KISClient()

    @property
    def is_mock(self) -> bool:
        return self._client.is_mock

    def _account_params(self) -> dict[str, str]:
        return {
            "CANO": settings.kis_account_number,
            "ACNT_PRDT_CD": settings.kis_account_product_code,
        }

    def _kr_order_tr(self, side: str) -> str:
        if self.is_mock:
            return ep.KR_BUY_TR_MOCK if side == "BUY" else ep.KR_SELL_TR_MOCK
        return ep.KR_BUY_TR if side == "BUY" else ep.KR_SELL_TR

    async def connect(self) -> None:
        await self._client.open()
        logger.info("KISBroker connected (mock=%s)", self.is_mock)

    async def disconnect(self) -> None:
        await self._client.close()

    # ---- Orders ----

    async def place_order(
        self,
        symbol: str,
        market: str,
        side: str,
        order_type: str,
        quantity: int,
        price: float | None = None,
        **kwargs: Any,
    ) -> OrderResult:
        return await self._place_kr_order(symbol, side, order_type, quantity, price)

    async def _place_kr_order(
        self, symbol: str, side: str, order_type: str, quantity: int, price: float | None
    ) -> OrderResult:
        tr_id = self._kr_order_tr(side)
        ord_dvsn = "01" if order_type == "MARKET" else "00"
        body = {
            **self._account_params(),
            "PDNO": symbol,
            "ORD_DVSN": ord_dvsn,
            "ORD_QTY": str(quantity),
            "ORD_UNPR": str(int(price)) if price else "0",
        }
        try:
            data = await self._client.post(ep.KR_ORDER_PATH, tr_id, body)
            output = data.get("output", {})
            return OrderResult(
                success=data.get("rt_cd") == "0",
                broker_order_id=output.get("ODNO", ""),
                message=data.get("msg1", ""),
            )
        except Exception as e:
            logger.error("KR order failed: %s", e)
            return OrderResult(success=False, message=str(e))

    async def cancel_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        return await self._cancel_kr_order(broker_order_id, **kwargs)

    async def _cancel_kr_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        tr_id = ep.KR_CANCEL_TR_MOCK if self.is_mock else ep.KR_CANCEL_TR
        body = {
            **self._account_params(),
            "KRX_FWDG_ORD_ORGNO": "",
            "ORGN_ODNO": broker_order_id,
            "ORD_DVSN": "00",
            "RVSE_CNCL_DVSN_CD": "02",
            "ORD_QTY": str(kwargs.get("quantity", 0)),
            "ORD_UNPR": "0",
            "QTY_ALL_ORD_YN": "Y",
        }
        try:
            data = await self._client.post(ep.KR_ORDER_CANCEL_PATH, tr_id, body)
            return OrderResult(
                success=data.get("rt_cd") == "0",
                message=data.get("msg1", ""),
            )
        except Exception as e:
            return OrderResult(success=False, message=str(e))

    async def get_order_status(self, broker_order_id: str, **kwargs: Any) -> dict:
        # KIS doesn't have a single-order status endpoint; use daily orders
        return {"broker_order_id": broker_order_id, "status": "UNKNOWN"}

    # ---- Balance ----

    async def get_balance(self) -> BalanceInfo:
        try:
            tr_id = ep.KR_BALANCE_TR_MOCK if self.is_mock else ep.KR_BALANCE_TR
            params = {
                **self._account_params(),
                "AFHR_FLPR_YN": "N",
                "OFL_YN": "",
                "INQR_DVSN": "02",
                "UNPR_DVSN": "01",
                "FUND_STTL_ICLD_YN": "N",
                "FNCG_AMT_AUTO_RDPT_YN": "N",
                "PRCS_DVSN": "01",
                "CTX_AREA_FK100": "",
                "CTX_AREA_NK100": "",
            }
            data = await self._client.get(ep.KR_BALANCE_PATH, tr_id, params)
            output2 = data.get("output2", [{}])
            summary = output2[0] if output2 else {}
            return BalanceInfo(
                cash_krw=float(summary.get("dnca_tot_amt", 0)),
                total_value_krw=float(summary.get("tot_evlu_amt", 0)),
            )
        except Exception as e:
            logger.error("Balance query failed: %s", e)
            return BalanceInfo()

    # ---- Prices ----

    async def get_current_price(self, symbol: str, market: str) -> PriceInfo:
        return await self._get_kr_price(symbol)

    async def _get_kr_price(self, symbol: str) -> PriceInfo:
        params = {"FID_COND_MRKT_DIV_CODE": "J", "FID_INPUT_ISCD": symbol}
        data = await self._client.get(ep.KR_PRICE_PATH, ep.KR_PRICE_TR, params)
        output = data.get("output", {})
        return PriceInfo(
            symbol=symbol,
            price=float(output.get("stck_prpr", 0)),
            change=float(output.get("prdy_vrss", 0)),
            change_pct=float(output.get("prdy_ctrt", 0)),
            volume=int(output.get("acml_vol", 0)),
            market="KR",
        )

    async def get_daily_prices(self, symbol: str, market: str, days: int = 60) -> list[dict]:
        return await self._get_kr_daily(symbol)

    async def _get_kr_daily(self, symbol: str) -> list[dict]:
        params = {
            "FID_COND_MRKT_DIV_CODE": "J",
            "FID_INPUT_ISCD": symbol,
            "FID_PERIOD_DIV_CODE": "D",
            "FID_ORG_ADJ_PRC": "0",
        }
        data = await self._client.get(ep.KR_DAILY_PRICE_PATH, ep.KR_DAILY_PRICE_TR, params)
        output = data.get("output", [])
        return [
            {
                "date": item.get("stck_bsop_date", ""),
                "open": float(item.get("stck_oprc", 0)),
                "high": float(item.get("stck_hgpr", 0)),
                "low": float(item.get("stck_lwpr", 0)),
                "close": float(item.get("stck_clpr", 0)),
                "volume": int(item.get("acml_vol", 0)),
            }
            for item in output
            if item.get("stck_bsop_date")
        ]

    # ---- Intraday candles ----

    async def get_intraday_candles(
        self, symbol: str, market: str, interval: int = 1
    ) -> list[dict]:
        """KIS API로 분봉 조회 후 interval(분)에 맞게 집계."""
        raw = await self._get_kr_minute_candles(symbol)
        if interval == 1 or not raw:
            return raw
        return _aggregate_candles(raw, interval)

    async def _get_kr_minute_candles(self, symbol: str) -> list[dict]:
        """KIS 분봉 차트 API 호출 (1분봉 기준, 당일 데이터)."""
        now_str = datetime.now().strftime("%H%M%S")
        params = {
            "FID_ETC_CLS_CODE": "",
            "FID_COND_MRKT_DIV_CODE": "J",
            "FID_INPUT_ISCD": symbol,
            "FID_INPUT_HOUR_1": now_str,
            "FID_PW_DATA_INCU_YN": "Y",
        }
        try:
            data = await self._client.get(ep.KR_MINUTE_CHART_PATH, ep.KR_MINUTE_CHART_TR, params)
            output = data.get("output2", [])
            result = []
            for item in output:
                date = item.get("stck_bsop_date", "")
                time = item.get("stck_cntg_hour", "")
                if not date or not time:
                    continue
                result.append({
                    "datetime": f"{date[:4]}-{date[4:6]}-{date[6:]} {time[:2]}:{time[2:4]}:{time[4:]}",
                    "open": float(item.get("stck_oprc", 0)),
                    "high": float(item.get("stck_hgpr", 0)),
                    "low": float(item.get("stck_lwpr", 0)),
                    "close": float(item.get("stck_prpr", 0)),
                    "volume": int(item.get("cntg_vol", 0)),
                })
            return list(reversed(result))  # 오름차순 정렬
        except Exception as e:
            logger.warning("분봉 조회 실패 %s: %s", symbol, e)
            return []
