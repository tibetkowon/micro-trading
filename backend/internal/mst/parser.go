package mst

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

// Record lengths per market type (bytes per line, excluding \n).
const (
	kospiRecordLen  = 288
	kosdaqRecordLen = 282

	codeOffset  = 0
	codeLen     = 6
	isinOffset  = 9
	isinLen     = 12
	nameOffset  = 21
	nameLen     = 40 // EUC-KR bytes
	groupOffset = 61
	groupLen    = 2
)

// StockMaster holds parsed fields for one stock from the KIS MST binary file.
type StockMaster struct {
	StockCode           string
	StockName           string
	ISIN                string
	MarketType          string // "KOSPI" | "KOSDAQ"
	GroupCode           string // "ST"=주식, "EF"=ETF, "BC"=뮤추얼펀드, "FS"=외국주식, "MF"=매매정지, "RT"=리츠
	IsETF               bool
	IsDomesticEquityETF bool // true=비과세(국내주식형 ETF)
	UpdatedAt           time.Time
}

// ParseMST parses all records from a raw MST file (byte slice) for the given market.
// market must be "KOSPI" or "KOSDAQ".
func ParseMST(data []byte, market string) ([]*StockMaster, error) {
	lines := splitLines(data)
	result := make([]*StockMaster, 0, len(lines))
	for i, line := range lines {
		if len(line) < 63 {
			continue
		}
		m, err := parseRecord(line, market)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if m.StockCode == "" {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// splitLines splits data on \n, stripping \r if present.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := data[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(data) {
		line := data[start:]
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}

// parseRecord parses a single fixed-width binary record.
func parseRecord(rec []byte, market string) (*StockMaster, error) {
	if len(rec) < groupOffset+groupLen {
		return nil, fmt.Errorf("record too short: %d bytes", len(rec))
	}

	code := strings.TrimSpace(string(rec[codeOffset : codeOffset+codeLen]))
	isin := strings.TrimSpace(string(rec[isinOffset : isinOffset+isinLen]))

	nameBytes := rec[nameOffset : nameOffset+nameLen]
	nameUTF8, err := eucKRToUTF8(nameBytes)
	if err != nil {
		// Fall back to lossy conversion
		nameUTF8 = strings.TrimSpace(string(nameBytes))
	}
	name := strings.TrimSpace(nameUTF8)

	group := strings.TrimSpace(string(rec[groupOffset : groupOffset+groupLen]))

	isETF := group == "EF"
	isDomestic := isETF && classifyDomesticEquityETF(name)

	return &StockMaster{
		StockCode:           code,
		StockName:           name,
		ISIN:                isin,
		MarketType:          market,
		GroupCode:           group,
		IsETF:               isETF,
		IsDomesticEquityETF: isDomestic,
		UpdatedAt:           time.Now(),
	}, nil
}

// eucKRToUTF8 converts EUC-KR encoded bytes to a UTF-8 string.
func eucKRToUTF8(b []byte) (string, error) {
	dec := korean.EUCKR.NewDecoder()
	out, _, err := transform.Bytes(dec, b)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// classifyDomesticEquityETF returns true if the ETF name suggests a domestic equity index ETF
// (non-taxable under Korean tax law). Classification is keyword-based and may occasionally
// misclassify — the is_domestic_equity_etf column can be manually corrected in the DB.
func classifyDomesticEquityETF(name string) bool {
	// Foreign / non-domestic keywords: if found, it's NOT domestic equity
	foreign := []string{
		"미국", "나스닥", "S&P", "SP500", "달러", "채권", "금", "원유",
		"합성", "커버드콜", "배당", "리츠", "인버스", "레버리지",
		"원자재", "유럽", "일본", "중국", "글로벌", "해외",
	}
	for _, kw := range foreign {
		if strings.Contains(name, kw) {
			return false
		}
	}
	// Domestic equity keywords
	domestic := []string{"200", "코스닥", "코스피", "KRX", "KTOP", "KQ150", "배럴"}
	for _, kw := range domestic {
		if strings.Contains(name, kw) {
			return true
		}
	}
	// Default: treat as non-domestic (apply tax) to be conservative
	return false
}
