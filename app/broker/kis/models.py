"""KIS-specific data classes."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class KISToken:
    access_token: str = ""
    token_type: str = "Bearer"
    expires_at: datetime = field(default_factory=lambda: datetime.min)

    @property
    def is_expired(self) -> bool:
        return datetime.now() >= self.expires_at
