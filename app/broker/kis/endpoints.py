"""KIS OpenAPI endpoint paths and transaction IDs."""

# OAuth
TOKEN_PATH = "/oauth2/tokenP"
HASHKEY_PATH = "/uapi/hashkey"

# --- 국내주식 ---
KR_ORDER_PATH = "/uapi/domestic-stock/v1/trading/order-cash"
KR_ORDER_CANCEL_PATH = "/uapi/domestic-stock/v1/trading/order-rvsecncl"
KR_BALANCE_PATH = "/uapi/domestic-stock/v1/trading/inquire-balance"
KR_PRICE_PATH = "/uapi/domestic-stock/v1/quotations/inquire-price"
KR_DAILY_PRICE_PATH = "/uapi/domestic-stock/v1/quotations/inquire-daily-price"
KR_ORDERS_PATH = "/uapi/domestic-stock/v1/trading/inquire-daily-ccld"

# Transaction IDs - 실전
KR_BUY_TR = "TTTC0802U"
KR_SELL_TR = "TTTC0801U"
KR_CANCEL_TR = "TTTC0803U"
KR_BALANCE_TR = "TTTC8434R"
KR_PRICE_TR = "FHKST01010100"
KR_DAILY_PRICE_TR = "FHKST01010400"
KR_ORDERS_TR = "TTTC8001R"

# Transaction IDs - 모의투자
KR_BUY_TR_MOCK = "VTTC0802U"
KR_SELL_TR_MOCK = "VTTC0801U"
KR_CANCEL_TR_MOCK = "VTTC0803U"
KR_BALANCE_TR_MOCK = "VTTC8434R"
KR_ORDERS_TR_MOCK = "VTTC8001R"

# --- 해외주식 ---
US_ORDER_PATH = "/uapi/overseas-stock/v1/trading/order"
US_ORDER_CANCEL_PATH = "/uapi/overseas-stock/v1/trading/order-rvsecncl"
US_BALANCE_PATH = "/uapi/overseas-stock/v1/trading/inquire-balance"
US_PRICE_PATH = "/uapi/overseas-price/v1/quotations/price"
US_DAILY_PRICE_PATH = "/uapi/overseas-price/v1/quotations/dailyprice"

# Transaction IDs - 실전
US_BUY_TR = "JTTT1002U"
US_SELL_TR = "JTTT1006U"
US_CANCEL_TR = "JTTT1004U"
US_BALANCE_TR = "JTTT3012R"
US_PRICE_TR = "HHDFS00000300"
US_DAILY_PRICE_TR = "HHDFS76240000"

# Transaction IDs - 모의투자
US_BUY_TR_MOCK = "VTTT1002U"
US_SELL_TR_MOCK = "VTTT1001U"
US_CANCEL_TR_MOCK = "VTTT1004U"
US_BALANCE_TR_MOCK = "VTTS3012R"

# Exchange code mapping
US_EXCHANGE_MAP = {
    "NYSE": "NYS",
    "NASDAQ": "NAS",
    "AMEX": "AMS",
}
