"""종목 마스터 동기화 및 검색 서비스.

pykrx를 사용하여 KRX 전체 상장 종목을 DB에 동기화하고,
초성 검색을 포함한 종목 검색 기능을 제공한다.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.stock_master import StockMaster
from app.web.hangul_util import extract_chosung, is_chosung_only, match_chosung

logger = logging.getLogger(__name__)

# 동기화 주기 (7일)
_SYNC_INTERVAL_DAYS = 7


class StockMasterService:

    def __init__(self, session: AsyncSession):
        self.session = session

    async def sync_if_needed(self) -> None:
        """DB에 데이터가 없거나 마지막 동기화가 7일 이상 경과했으면 재수집한다."""
        result = await self.session.execute(
            select(StockMaster.updated_at)
            .order_by(StockMaster.updated_at.desc())
            .limit(1)
        )
        last_updated = result.scalar_one_or_none()

        if last_updated is not None:
            # timezone-aware 비교
            if last_updated.tzinfo is None:
                last_updated = last_updated.replace(tzinfo=timezone.utc)
            elapsed = (datetime.now(timezone.utc) - last_updated).days
            if elapsed < _SYNC_INTERVAL_DAYS:
                logger.info("종목 마스터 동기화 불필요 (마지막: %d일 전)", elapsed)
                return

        logger.info("종목 마스터 동기화 시작...")
        await self.sync_kr_stocks()
        logger.info("종목 마스터 동기화 완료")

    async def sync_kr_stocks(self) -> int:
        """pykrx로 KOSPI/KOSDAQ 전종목을 수집하여 DB에 upsert한다."""
        try:
            from pykrx import stock as pykrx_stock
        except ImportError:
            logger.warning("pykrx가 설치되지 않아 KR 종목 동기화를 건너뜁니다.")
            return 0

        count = 0
        today = datetime.now().strftime("%Y%m%d")

        for market_name in ("KOSPI", "KOSDAQ"):
            try:
                tickers = pykrx_stock.get_market_ticker_list(today, market=market_name)
            except Exception:
                logger.exception("pykrx %s 종목 목록 조회 실패", market_name)
                continue

            for ticker in tickers:
                try:
                    name = pykrx_stock.get_market_ticker_name(ticker)
                except Exception:
                    continue

                if not name:
                    continue

                await self._upsert(symbol=ticker, market="KR", name=name, sector=market_name)
                count += 1

        if count > 0:
            await self.session.commit()
            logger.info("KR 종목 %d건 동기화 완료", count)
        return count

    async def _upsert(
        self, *, symbol: str, market: str, name: str, sector: str | None
    ) -> None:
        """종목을 insert 또는 update한다."""
        result = await self.session.execute(
            select(StockMaster).where(
                StockMaster.symbol == symbol,
                StockMaster.market == market,
            )
        )
        existing = result.scalar_one_or_none()

        if existing:
            existing.name = name
            if sector is not None:
                existing.sector = sector
            existing.updated_at = datetime.now(timezone.utc)
        else:
            self.session.add(
                StockMaster(symbol=symbol, market=market, name=name, sector=sector)
            )

    async def search(
        self, query: str, limit: int = 15
    ) -> list[dict]:
        """종목을 검색한다. 초성 검색을 지원한다.

        Returns:
            match_type이 포함된 dict 리스트:
            [{"symbol": "005930", "market": "KR", "name": "삼성전자", "match_type": "prefix"}]
        """
        if not query:
            return []

        # 초성 전용 쿼리인 경우
        if is_chosung_only(query):
            return await self._search_chosung(query, limit)

        # 일반 검색: symbol 또는 name LIKE 매칭
        q_upper = query.upper()
        q_like = f"%{query}%"

        result = await self.session.execute(
            select(StockMaster).where(
                (StockMaster.symbol.ilike(q_like))
                | (StockMaster.name.ilike(q_like))
            ).limit(200)  # 정렬을 위해 넉넉히 가져옴
        )
        rows = list(result.scalars().all())

        # 정렬: exact > prefix > contains
        scored: list[tuple[int, StockMaster]] = []
        for row in rows:
            sym_upper = row.symbol.upper()
            name_upper = row.name.upper()

            if sym_upper == q_upper or name_upper == q_upper:
                scored.append((0, row))  # exact
            elif sym_upper.startswith(q_upper) or name_upper.startswith(q_upper):
                scored.append((1, row))  # prefix
            else:
                scored.append((2, row))  # contains

        scored.sort(key=lambda x: x[0])

        match_labels = {0: "exact", 1: "prefix", 2: "contains"}
        return [
            {
                "symbol": row.symbol,
                "market": row.market,
                "name": row.name,
                "match_type": match_labels[score],
            }
            for score, row in scored[:limit]
        ]

    async def _search_chosung(self, query: str, limit: int) -> list[dict]:
        """초성 검색 — KR 종목 전체를 가져와 Python에서 필터링한다.

        SQLite에는 한글 초성 함수가 없으므로 Python에서 처리한다.
        메모리 절약을 위해 symbol, name만 로드한다.
        """
        result = await self.session.execute(
            select(
                StockMaster.symbol,
                StockMaster.market,
                StockMaster.name,
            ).where(StockMaster.market == "KR")
        )
        rows = result.all()

        matches: list[dict] = []
        for symbol, market, name in rows:
            if match_chosung(query, name):
                matches.append({
                    "symbol": symbol,
                    "market": market,
                    "name": name,
                    "match_type": "chosung",
                })
                if len(matches) >= limit:
                    break

        return matches

    async def get_name(self, symbol: str, market: str) -> str | None:
        """단일 종목명 조회. 없으면 None 반환."""
        result = await self.session.execute(
            select(StockMaster.name).where(
                StockMaster.symbol == symbol,
                StockMaster.market == market,
            )
        )
        return result.scalar_one_or_none()

    async def get_names_bulk(self, symbols: list[tuple[str, str]]) -> dict[tuple[str, str], str]:
        """여러 종목의 이름을 한 번에 조회. {(symbol, market): name} 반환."""
        if not symbols:
            return {}
        from sqlalchemy import or_, and_, tuple_
        conditions = [
            and_(StockMaster.symbol == sym, StockMaster.market == mkt)
            for sym, mkt in symbols
        ]
        result = await self.session.execute(
            select(StockMaster.symbol, StockMaster.market, StockMaster.name)
            .where(or_(*conditions))
        )
        return {(row.symbol, row.market): row.name for row in result.all()}

    async def get_count(self) -> int:
        """저장된 종목 수를 반환한다."""
        result = await self.session.execute(
            select(func.count(StockMaster.id))
        )
        return result.scalar_one()
