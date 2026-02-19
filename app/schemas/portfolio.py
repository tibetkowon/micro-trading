from datetime import date as DateType

from pydantic import BaseModel, Field


class PortfolioSummary(BaseModel):
    total_value: float = Field(0.0, description="총 평가금액 = 보유 포지션 시가총액 + 현금 잔고")
    total_invested: float = Field(0.0, description="총 투자 원금 (보유 포지션 평균 매입가 × 수량 합계)")
    cash_krw: float = Field(0.0, description="KRW 현금 잔고")
    cash_usd: float = Field(0.0, description="USD 현금 잔고")
    initial_balance_krw: float = Field(0.0, description="초기 KRW 자산 (수익률 계산 기준)")
    initial_balance_usd: float = Field(0.0, description="초기 USD 자산")
    realized_pnl: float = Field(0.0, description="실현 손익 (매도 완료 포지션 누계)")
    unrealized_pnl: float = Field(0.0, description="미실현 손익 (보유 포지션 평가 기준)")
    total_pnl: float = Field(0.0, description="총 손익 = 실현 + 미실현")
    return_pct: float = Field(0.0, description="수익률 (%) = total_pnl / initial_balance × 100")
    orderable_krw: float = Field(0.0, description="수수료 차감 후 실질 KRW 주문가능 금액")
    orderable_usd: float = Field(0.0, description="수수료 차감 후 실질 USD 주문가능 금액")


class OrderableResponse(BaseModel):
    """AI 자동매매용 주문가능 금액 응답 (수수료 반영)."""
    trading_mode: str = Field(..., description="거래 모드 (PAPER/REAL)")
    cash_krw: float = Field(..., description="현재 KRW 현금 잔고")
    cash_usd: float = Field(..., description="현재 USD 현금 잔고")
    commission_rate: float = Field(..., description="적용 수수료율 (모의투자: 0.0005, 실계좌: 0.003)")
    orderable_krw: float = Field(..., description="수수료 차감 후 실질 KRW 주문가능 금액")
    orderable_usd: float = Field(..., description="수수료 차감 후 실질 USD 주문가능 금액")


class SymbolPnlItem(BaseModel):
    symbol: str = Field(..., description="종목 코드")
    market: str = Field(..., description="시장 구분 (KR/US)")
    name: str = Field(..., description="종목명")
    realized_pnl: float = Field(..., description="종목별 실현 손익 합계")
    total_commission: float = Field(..., description="종목별 수수료 합계")
    trade_count: int = Field(..., description="거래 횟수")


class DailyReturnItem(BaseModel):
    date: str = Field(..., description="날짜 (YYYY-MM-DD)")
    total_value: float = Field(..., description="해당일 총 평가금액")
    realized_pnl: float = Field(..., description="해당일 누적 실현 손익")
    unrealized_pnl: float = Field(..., description="해당일 미실현 손익")
    total_pnl: float = Field(..., description="해당일 총 손익 = 실현 + 미실현")
    return_pct: float = Field(..., description="해당일 수익률 (%)")


class PnlAnalysisResponse(BaseModel):
    """수익률 분석 응답 — 종목별 실현손익 및 일별 누적 수익률."""
    trading_mode: str = Field(..., description="거래 모드 (PAPER/REAL)")
    total_realized_pnl: float = Field(..., description="전체 종목 실현 손익 합계")
    symbol_pnl: list[SymbolPnlItem] = Field(..., description="종목별 실현 손익 목록 (실현손익 내림차순)")
    daily_returns: list[DailyReturnItem] = Field(..., description="일별 수익률 목록 (날짜 오름차순)")


class SnapshotResponse(BaseModel):
    date: DateType = Field(..., description="스냅샷 날짜")
    trading_mode: str = Field(..., description="거래 모드 (PAPER/REAL)")
    total_value: float = Field(..., description="총 평가금액")
    total_invested: float = Field(..., description="총 투자 원금")
    realized_pnl: float = Field(..., description="실현 손익")
    unrealized_pnl: float = Field(..., description="미실현 손익")

    model_config = {"from_attributes": True}
