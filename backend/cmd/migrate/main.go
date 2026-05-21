package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/micro-trading-for-agent/backend/internal/config"
	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/logger"
	"github.com/micro-trading-for-agent/backend/internal/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	logger.Setup()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()

	var opts []option.ClientOption
	if cfg.FirebaseCredentialsJSON != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsJSON))
	}
	fs, err := firestore.NewClient(ctx, cfg.FirebaseProjectID, opts...)
	if err != nil {
		slog.Error("firestore init failed", "error", err)
		os.Exit(1)
	}
	defer fs.Close()

	db, err := database.New(cfg.SQLitePath)
	if err != nil {
		slog.Error("sqlite init failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	migrateSettings(ctx, fs, db)
	migrateOrders(ctx, fs, db)
	migratePositions(ctx, fs, db)
	migrateScanLogs(ctx, fs, db)
	migrateTradeReports(ctx, fs, db)
	migrateDailyReports(ctx, fs, db)
	migrateSimulationResults(ctx, fs, db)
	migrateTokens(ctx, fs, db)
	slog.Info("migration complete")
}

func migrateSettings(ctx context.Context, fs *firestore.Client, db *database.DB) {
	doc, err := fs.Collection("settings").Doc("config").Get(ctx)
	if err != nil {
		slog.Warn("settings not found", "error", err)
		return
	}
	count := 0
	for k, v := range doc.Data() {
		var strVal string
		switch v.(type) {
		case []interface{}, map[string]interface{}:
			b, merr := json.Marshal(v)
			if merr != nil {
				slog.Error("marshal setting failed", "key", k, "error", merr)
				continue
			}
			strVal = string(b)
		default:
			strVal = fmt.Sprintf("%v", v)
		}
		if err := db.SetSetting(ctx, k, strVal); err != nil {
			slog.Error("set setting failed", "key", k, "error", err)
			continue
		}
		count++
	}
	slog.Info("settings migrated", "count", count)
}

func migrateOrders(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("orders").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("orders iter error", "error", err)
			break
		}
		var o models.Order
		if err := doc.DataTo(&o); err == nil {
			if _, err := db.CreateOrder(ctx, &o); err != nil {
				slog.Error("order migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("orders migrated", "count", count)
}

func migratePositions(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("monitored_positions").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("positions iter error", "error", err)
			break
		}
		var p models.MonitoredPosition
		if err := doc.DataTo(&p); err == nil {
			if err := db.CreatePosition(ctx, &p); err != nil {
				slog.Error("position migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("positions migrated", "count", count)
}

func migrateScanLogs(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("scan_logs").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("scan logs iter error", "error", err)
			break
		}
		var l models.ScanLog
		if err := doc.DataTo(&l); err == nil {
			if _, err := db.CreateScanLog(ctx, &l); err != nil {
				slog.Error("scan log migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("scan logs migrated", "count", count)
}

func migrateTradeReports(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("trade_reports").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("trade reports iter error", "error", err)
			break
		}
		var r models.TradeReport
		if err := doc.DataTo(&r); err == nil {
			if _, err := db.CreateTradeReport(ctx, &r); err != nil {
				slog.Error("trade report migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("trade reports migrated", "count", count)
}

func migrateDailyReports(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("daily_reports").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("daily reports iter error", "error", err)
			break
		}
		var r models.DailyReport
		if err := doc.DataTo(&r); err == nil {
			if err := db.InsertOrUpdateDailyReport(ctx, r); err != nil {
				slog.Error("daily report migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("daily reports migrated", "count", count)
}

func migrateSimulationResults(ctx context.Context, fs *firestore.Client, db *database.DB) {
	count := 0
	iter := fs.Collection("simulation_results").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("simulation results iter error", "error", err)
			break
		}
		var r models.SimulationResult
		if err := doc.DataTo(&r); err == nil {
			if err := db.UpsertSimulationResult(ctx, r); err != nil {
				slog.Error("simulation result migrate failed", "id", doc.Ref.ID, "error", err)
			}
		}
		count++
	}
	slog.Info("simulation results migrated", "count", count)
}

func migrateTokens(ctx context.Context, fs *firestore.Client, db *database.DB) {
	doc, err := fs.Collection("tokens").Doc("current").Get(ctx)
	if err != nil {
		slog.Warn("token not found", "error", err)
		return
	}
	var t models.Token
	if err := doc.DataTo(&t); err != nil {
		slog.Error("token decode failed", "error", err)
		return
	}
	if err := db.SaveToken(ctx, &t); err != nil {
		slog.Error("token migrate failed", "error", err)
		return
	}
	slog.Info("tokens migrated", "count", 1)
}
