from __future__ import annotations

import logging

from pydantic_settings import BaseSettings

from app.schemas.common import TradingMode

logger = logging.getLogger(__name__)


# 런타임에서 변경 가능한 거래 모드 (Settings는 frozen이므로 별도 관리)
_runtime_trading_mode: TradingMode | None = None


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
    paper_commission_rate: float = 0.0005  # 0.05%

    # Database
    database_url: str = "sqlite+aiosqlite:///./trading.db"

    # Server
    host: str = "0.0.0.0"
    port: int = 8000

    def get_trading_mode(self) -> TradingMode:
        """현재 활성 거래 모드 반환 (런타임 변경값 우선)."""
        if _runtime_trading_mode is not None:
            return _runtime_trading_mode
        return self.trading_mode

    def switch_trading_mode(self, mode: TradingMode) -> None:
        """런타임에서 거래 모드 전환."""
        global _runtime_trading_mode
        _runtime_trading_mode = mode
        logger.info("거래 모드 전환: %s", mode.value)


settings = Settings()
