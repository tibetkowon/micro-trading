package mst

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/micro-trading-for-agent/backend/internal/logger"
)

const (
	kospiURL  = "https://new.real.download.dws.co.kr/common/master/kospi_code.mst.zip"
	kosdaqURL = "https://new.real.download.dws.co.kr/common/master/kosdaq_code.mst.zip"

	maxRetries      = 3
	retryInterval   = 5 * time.Minute
	downloadTimeout = 60 * time.Second
)

// DownloadResult holds parsed StockMaster records for one market.
type DownloadResult struct {
	Market  string
	Records []*StockMaster
}

// DownloadAndParse downloads and parses both KOSPI and KOSDAQ MST files.
// It retries up to maxRetries times on failure, waiting retryInterval between attempts.
// Returns parsed records for both markets.
func DownloadAndParse(ctx context.Context) ([]DownloadResult, error) {
	markets := []struct {
		name string
		url  string
	}{
		{"KOSPI", kospiURL},
		{"KOSDAQ", kosdaqURL},
	}

	var results []DownloadResult
	for _, m := range markets {
		var records []*StockMaster
		var lastErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			records, lastErr = downloadAndParseOne(ctx, m.url, m.name)
			if lastErr == nil {
				break
			}
			logger.Warn("mst: download failed, will retry",
				map[string]any{
					"market":  m.name,
					"attempt": attempt,
					"error":   lastErr.Error(),
				})
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryInterval):
				}
			}
		}
		if lastErr != nil {
			return nil, fmt.Errorf("mst: %s download failed after %d attempts: %w", m.name, maxRetries, lastErr)
		}
		logger.Info("mst: downloaded", map[string]any{"market": m.name, "count": len(records)})
		results = append(results, DownloadResult{Market: m.name, Records: records})
	}
	return results, nil
}

// downloadAndParseOne downloads a single ZIP from url, extracts the .mst file, and parses it.
func downloadAndParseOne(ctx context.Context, url, market string) ([]*StockMaster, error) {
	httpCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Unzip and find the .mst file
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read zip entry %s: %w", f.Name, err)
		}
		return ParseMST(data, market)
	}
	return nil, fmt.Errorf("no files found in zip")
}
