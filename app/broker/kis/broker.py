"""KIS broker implementation for real trading."""

from __future__ import annotations

import logging
from typing import Any

from app.broker.base import AbstractBroker, BalanceInfo, OrderResult, PriceInfo
from app.broker.kis.client import KISClient
from app.broker.kis import endpoints as ep
from app.config import settings

logger = logging.getLogger(__name__)


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

    def _us_order_tr(self, side: str) -> str:
        if self.is_mock:
            return ep.US_BUY_TR_MOCK if side == "BUY" else ep.US_SELL_TR_MOCK
        return ep.US_BUY_TR if side == "BUY" else ep.US_SELL_TR

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
        if market == "KR":
            return await self._place_kr_order(symbol, side, order_type, quantity, price)
        return await self._place_us_order(symbol, side, order_type, quantity, price, **kwargs)

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

    async def _place_us_order(
        self,
        symbol: str,
        side: str,
        order_type: str,
        quantity: int,
        price: float | None,
        exchange: str = "NAS",
        **kwargs: Any,
    ) -> OrderResult:
        tr_id = self._us_order_tr(side)
        body = {
            **self._account_params(),
            "OVRS_EXCG_CD": exchange,
            "PDNO": symbol,
            "ORD_DVSN": "00" if order_type == "LIMIT" else "01",
            "ORD_QTY": str(quantity),
            "OVRS_ORD_UNPR": str(price) if price else "0",
        }
        try:
            data = await self._client.post(ep.US_ORDER_PATH, tr_id, body)
            output = data.get("output", {})
            return OrderResult(
                success=data.get("rt_cd") == "0",
                broker_order_id=output.get("ODNO", ""),
                message=data.get("msg1", ""),
            )
        except Exception as e:
            logger.error("US order failed: %s", e)
            return OrderResult(success=False, message=str(e))

    async def cancel_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        market = kwargs.get("market", "KR")
        if market == "KR":
            return await self._cancel_kr_order(broker_order_id, **kwargs)
        return await self._cancel_us_order(broker_order_id, **kwargs)

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

    async def _cancel_us_order(self, broker_order_id: str, **kwargs: Any) -> OrderResult:
        tr_id = ep.US_CANCEL_TR_MOCK if self.is_mock else ep.US_CANCEL_TR
        body = {
            **self._account_params(),
            "OVRS_EXCG_CD": kwargs.get("exchange", "NAS"),
            "ORGN_ODNO": broker_order_id,
            "RVSE_CNCL_DVSN_CD": "02",
            "ORD_QTY": str(kwargs.get("quantity", 0)),
            "OVRS_ORD_UNPR": "0",
        }
        try:
            data = await self._client.post(ep.US_ORDER_CANCEL_PATH, tr_id, body)
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
        if market == "KR":
            return await self._get_kr_price(symbol)
        return await self._get_us_price(symbol)

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

    async def _get_us_price(self, symbol: str) -> PriceInfo:
        params = {
            "AUTH": "",
            "EXCD": "NAS",
            "SYMB": symbol,
        }
        data = await self._client.get(ep.US_PRICE_PATH, ep.US_PRICE_TR, params)
        output = data.get("output", {})
        return PriceInfo(
            symbol=symbol,
            price=float(output.get("last", 0)),
            change=float(output.get("diff", 0)),
            change_pct=float(output.get("rate", 0)),
            volume=int(output.get("tvol", 0)),
            market="US",
        )

    async def get_daily_prices(self, symbol: str, market: str, days: int = 60) -> list[dict]:
        if market == "KR":
            return await self._get_kr_daily(symbol)
        return await self._get_us_daily(symbol)

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

    async def _get_us_daily(self, symbol: str) -> list[dict]:
        params = {
            "AUTH": "",
            "EXCD": "NAS",
            "SYMB": symbol,
            "GUBN": "0",
            "BYMD": "",
            "MODP": "1",
        }
        data = await self._client.get(ep.US_DAILY_PRICE_PATH, ep.US_DAILY_PRICE_TR, params)
        output = data.get("output2", [])
        return [
            {
                "date": item.get("xymd", ""),
                "open": float(item.get("open", 0)),
                "high": float(item.get("high", 0)),
                "low": float(item.get("low", 0)),
                "close": float(item.get("clos", 0)),
                "volume": int(item.get("tvol", 0)),
            }
            for item in output
            if item.get("xymd")
        ]
