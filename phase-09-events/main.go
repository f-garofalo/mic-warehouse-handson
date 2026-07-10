package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"warehouse.local/core/events"
	"warehouse.local/core/handlers"
	"warehouse.local/core/repositories"
	"warehouse.local/core/usecases"
)

func main() {
	// The Phase 04 dual-write is RETIRED in this phase. Not because it was
	// broken: because this phase shows event-driven integration, and with the
	// dual-write on there would be no gap to observe. The BC now writes only
	// its own store and loses its last dependency on the legacy MIC schema.
	// The visibility gap this opens on the not-yet-migrated MIC screens is
	// today's problem; events close it (events/hermes_publisher.go on this
	// side, mic-integration/consumer/consumer.php on the MIC side).
	warehouse, err := wireWarehouseDB()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer warehouse.Close()
	repo := repositories.NewMySQLArticleRepository(warehouse)

	// The dispatcher your use cases have fed since Phase 05 is no longer an
	// in-memory sink: it validates every event against its Data Product
	// schema (schemas/) and wraps it in a CloudEvents envelope. In this
	// exercise Hermes is local and in-memory; in production the same
	// publisher would push to the platform via the Hermes SDK.
	registry, err := events.LoadSchemaRegistry("./schemas")
	if err != nil {
		log.Fatalf("schema registry: %v", err)
	}
	publisher := events.NewHermesPublisher(registry)

	createUC := usecases.NewCreateArticleUseCase(repo, publisher)
	getUC := usecases.NewGetArticleUseCase(repo)
	listUC := usecases.NewListArticlesUseCase(repo)
	adjustUC := usecases.NewAdjustInventoryUseCase(repo, publisher)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status":         "ok",
			"write_mode":     "warehouse-only",
			"dual_write":     "retired in Phase 09",
			"hermes_records": len(publisher.PublishedCloudEvents()),
		})
	})

	// Teaching endpoint of the local Hermes mock: in a real platform you
	// would inspect the Data Product through Hermes tooling, not through a
	// BC debug route.
	e.GET("/debug/hermes/records", func(c echo.Context) error {
		typeFilter := strings.TrimSpace(c.QueryParam("type"))
		records := publisher.PublishedCloudEvents()
		if typeFilter != "" {
			filtered := make([]events.CloudEvent, 0, len(records))
			for _, record := range records {
				if record.Type == typeFilter {
					filtered = append(filtered, record)
				}
			}
			records = filtered
		}
		return c.JSON(200, records)
	})

	router := handlers.NewRouter(createUC, getUC, listUC, adjustUC)
	router.Register(e)

	log.Printf("Starting server on :8081 (write mode: warehouse-only, dual-write retired)")
	e.Logger.Fatal(e.Start(":8081"))
}

// wireWarehouseDB opens the warehouse MySQL connection. Note what is gone:
// no legacy DSN, no READ_MODE dial. The BC no longer knows the MIC database
// exists.
func wireWarehouseDB() (*sql.DB, error) {
	dsn := buildDSN(
		envOr("WAREHOUSE_DB_USER", "warehouse_user"),
		envOr("WAREHOUSE_DB_PASSWORD", "warehouse_pass"),
		envOr("WAREHOUSE_DB_HOST", "127.0.0.1"),
		envOr("WAREHOUSE_DB_PORT", "3306"),
		envOr("WAREHOUSE_DB_NAME", "warehouse_db"),
	)
	warehouse, err := openDB(dsn)
	if err != nil {
		return nil, fmt.Errorf("open warehouse db: %w", err)
	}
	return warehouse, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func buildDSN(user, pass, host, port, name string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		user, pass, host, port, name)
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
