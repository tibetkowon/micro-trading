from __future__ import annotations

from enum import Enum

from pydantic_settings import BaseSettings


class TradingMode(str, Enum):
    REAL = "REAL"
    PAPER = "PAPER"


class Settings(BaseSettings):
    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}

    # Trading
    trading_mode: TradingMode = TradingMode.PAPER

    # KIS OpenAPI
    kis_app_key: str = ""
    kis_app_secret: str = ""
    kis_account_number: str = ""
    kis_account_product_code: str = "01"
    kis_base_url: str = "https://openapivts.koreainvestment.com:9443"

    # Paper trading
    paper_balance_krw: float = 100_000_000
    paper_balance_usd: float = 100_000.0

    # Database
    database_url: str = "sqlite+aiosqlite:///./trading.db"

    # Server
    host: str = "0.0.0.0"
    port: int = 8000


settings = Settings()
