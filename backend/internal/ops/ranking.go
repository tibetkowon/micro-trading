package ops

import (
	"context"

	"github.com/micro-trading-for-agent/backend/internal/kis"
)

// GetVolumeRank returns the volume ranking (거래량 순위).
// market: "J"=KRX(default).
// inputIscd: "0000"=전체(default), "0001"=KOSPI, "1001"=KOSDAQ, "2001"=KOSPI200.
// sort (FID_BLNG_CLS_CODE): "0"=평균거래량, "1"=거래량증가율, "2"=거래회전율, "3"=거래대금순, "4"=평균거래대금.
// priceMin/priceMax: 가격 범위 (빈값="" 이면 전체).
// excludeCls: fid_trgt_exls_cls_code 10자리 문자열 (빈값이면 "1111111111" 사용).
func GetVolumeRank(ctx context.Context, client *kis.Client, market, inputIscd, sort, priceMin, priceMax, excludeCls string) ([]kis.VolumeRankItem, error) {
	if market == "" {
		market = "J"
	}
	if sort == "" {
		sort = "0"
	}
	return client.GetVolumeRank(ctx, market, inputIscd, sort, priceMin, priceMax, excludeCls)
}

// GetStrengthRank returns the execution strength ranking (체결강도 상위).
// market: "0000"=전체(default), "0001"=거래소, "1001"=코스닥, "2001"=코스피200.
// priceMin/priceMax: 가격 범위 (빈값="" 이면 전체).
// excludeCls: fid_trgt_exls_cls_code 10자리 문자열 (빈값이면 "1111111111" 사용).
func GetStrengthRank(ctx context.Context, client *kis.Client, market, priceMin, priceMax, excludeCls string) ([]kis.StrengthRankItem, error) {
	if market == "" {
		market = "0000"
	}
	return client.GetStrengthRank(ctx, market, priceMin, priceMax, excludeCls)
}

// GetFluctuationRank returns the price change rate ranking (등락률 순위 FHPST01700000).
// market: "0000"=전체(default), "0001"=거래소(KOSPI), "1001"=코스닥.
// priceMin/priceMax: 가격 범위 (빈값="" 이면 전체).
// excludeCls: fid_trgt_exls_cls_code 10자리 문자열 (빈값이면 "1111111111" 사용).
func GetFluctuationRank(ctx context.Context, client *kis.Client, market, priceMin, priceMax, excludeCls string) ([]kis.FluctuationRankItem, error) {
	if market == "" {
		market = "0000"
	}
	return client.GetFluctuationRank(ctx, market, priceMin, priceMax, excludeCls)
}

// GetVIStatus returns today's VI 발동현황 (변동성완화장치) activated stocks (FHPST01390000).
// date: "YYYYMMDD". 미해제(vi_cncl_hour=="") 건을 포함하여 반환하므로 호출자에서 필터링 필요.
func GetVIStatus(ctx context.Context, client *kis.Client, date string) ([]kis.VIStatusItem, error) {
	return client.GetVIStatus(ctx, date)
}
