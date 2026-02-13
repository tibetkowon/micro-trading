from __future__ import annotations

from enum import Enum


class Market(str, Enum):
    KR = "KR"
    US = "US"


class OrderSide(str, Enum):
    BUY = "BUY"
    SELL = "SELL"


class OrderType(str, Enum):
    MARKET = "MARKET"
    LIMIT = "LIMIT"


class OrderStatus(str, Enum):
    PENDING = "PENDING"
    SUBMITTED = "SUBMITTED"
    FILLED = "FILLED"
    PARTIALLY_FILLED = "PARTIALLY_FILLED"
    CANCELLED = "CANCELLED"
    REJECTED = "REJECTED"


class TradingMode(str, Enum):
    REAL = "REAL"
    PAPER = "PAPER"


class OrderSource(str, Enum):
    MANUAL = "MANUAL"
    STRATEGY = "STRATEGY"


class StrategyType(str, Enum):
    DCA = "DCA"
    MOVING_AVERAGE = "MOVING_AVERAGE"
    RSI_REBALANCE = "RSI_REBALANCE"
