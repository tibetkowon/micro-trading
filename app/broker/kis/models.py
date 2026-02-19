"""KIS-specific data classes."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta


@dataclass
class KISToken:
    access_token: str = ""
    token_type: str = "Bearer"
    expires_at: datetime = field(default_factory=lambda: datetime.min)

    @property
    def is_expired(self) -> bool:
        return datetime.now() >= self.expires_at

    @property
    def is_expiring_soon(self) -> bool:
        """만료 1시간 이내인 경우 True — 선제 갱신 판단에 사용."""
        return datetime.now() >= self.expires_at - timedelta(hours=1)
