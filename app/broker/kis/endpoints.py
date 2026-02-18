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
