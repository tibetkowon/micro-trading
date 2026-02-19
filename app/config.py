from __future__ import annotations

import json
import logging
import pathlib

from pydantic_settings import BaseSettings

from app.schemas.common import TradingMode

logger = logging.getLogger(__name__)

# 런타임 설정 영속화 파일 경로 (재기동 후에도 유지)
_RUNTIME_SETTINGS_PATH = pathlib.Path("runtime_settings.json")

# 런타임에서 변경 가능한 거래 모드 (Settings는 frozen이므로 별도 관리)
_runtime_trading_mode: TradingMode | None = None


def _load_runtime_settings() -> None:
    """앱 시작 시 파일에 저장된 런타임 설정을 메모리에 로드."""
    global _runtime_trading_mode
    if not _RUNTIME_SETTINGS_PATH.exists():
        return
    try:
        data = json.loads(_RUNTIME_SETTINGS_PATH.read_text(encoding="utf-8"))
        mode_val = data.get("trading_mode")
        if mode_val:
            _runtime_trading_mode = TradingMode(mode_val)
            logger.info("런타임 설정 복원: trading_mode=%s", mode_val)
    except Exception:
        logger.warning("런타임 설정 파일 로드 실패, 기본값 사용")


def _save_runtime_settings(mode: TradingMode) -> None:
    """현재 런타임 설정을 파일에 저장."""
    try:
        _RUNTIME_SETTINGS_PATH.write_text(
            json.dumps({"trading_mode": mode.value}),
            encoding="utf-8",
        )
    except Exception:
        logger.warning("런타임 설정 파일 저장 실패")


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
    real_commission_rate: float = 0.0015  # KIS 실거래 수수료 기본값 0.15%

    # Database
    database_url: str = "sqlite+aiosqlite:///./trading.db"

    # Server
    host: str = "0.0.0.0"
    port: int = 8000

    def get_trading_mode(self) -> TradingMode:
        """현재 활성 거래 모드 반환 (영속화 파일 → .env 순 우선순위)."""
        global _runtime_trading_mode
        # 첫 접근 시 파일에서 복원 (모듈 로드 시점에는 로그 설정이 안 됐을 수 있으므로 지연 로드)
        if _runtime_trading_mode is None:
            _load_runtime_settings()
        if _runtime_trading_mode is not None:
            return _runtime_trading_mode
        return self.trading_mode

    def switch_trading_mode(self, mode: TradingMode) -> None:
        """런타임에서 거래 모드 전환 후 파일에 영속화."""
        global _runtime_trading_mode
        _runtime_trading_mode = mode
        _save_runtime_settings(mode)
        logger.info("거래 모드 전환: %s", mode.value)


settings = Settings()
