package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/handlers"
	"warehouse.local/core/repositories"
	"warehouse.local/core/usecases"
)

func main() {
	readModeRaw, readMode, err := readModeFromEnv()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	legacy, warehouse, err := wireRepositories()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer legacy.Close()
	defer warehouse.Close()

	// Migration wiring (ADR-013): the Phase 04 dual-write decorator, now in
	// production. Writes go legacy-first (the MIC god-table mints the id),
	// then to the warehouse store; reads follow the READ_MODE dial.
	legacyACL := repositories.NewLegacyMICArticleRepository(legacy)
	warehouseRepo := repositories.NewMySQLArticleRepository(warehouse)
	repo := repositories.NewDualWriteArticleRepository(legacyACL, warehouseRepo, readMode)
	disp := dispatcher.NewInMemoryDispatcher()

	createUC := usecases.NewCreateArticleUseCase(repo, disp)
	getUC := usecases.NewGetArticleUseCase(repo)
	listUC := usecases.NewListArticlesUseCase(repo)
	adjustUC := usecases.NewAdjustInventoryUseCase(repo, disp)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]any{
			"status":    "ok",
			"read_mode": readModeRaw,
		})
	})

	router := handlers.NewRouter(createUC, getUC, listUC, adjustUC)
	router.Register(e)

	log.Printf("Starting server on :8081")
	e.Logger.Fatal(e.Start(":8081"))
}

// readModeFromEnv parses the READ_MODE dial: "legacy" (default) or
// "warehouse". Any other value is a config error and aborts startup.
func readModeFromEnv() (string, repositories.ReadMode, error) {
	raw := envOr("READ_MODE", "legacy")
	switch raw {
	case "legacy":
		return raw, repositories.ReadFromLegacy, nil
	case "warehouse":
		return raw, repositories.ReadFromBC, nil
	default:
		return "", 0, fmt.Errorf("invalid READ_MODE %q: must be \"legacy\" or \"warehouse\"", raw)
	}
}

// wireRepositories opens both MySQL connections: the MIC monolith database
// (legacy side of the dual-write) and the warehouse BC database.
func wireRepositories() (legacy *sql.DB, warehouse *sql.DB, err error) {
	legacyDSN := buildDSN(
		envOr("LEGACY_DB_USER", "root"),
		envOr("LEGACY_DB_PASSWORD", "root"),
		envOr("LEGACY_DB_HOST", "127.0.0.1"),
		envOr("LEGACY_DB_PORT", "3306"),
		envOr("LEGACY_DB_NAME", "mic"),
	)
	legacy, err = openDB(legacyDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("open legacy (mic) db: %w", err)
	}

	warehouseDSN := buildDSN(
		envOr("WAREHOUSE_DB_USER", "warehouse_user"),
		envOr("WAREHOUSE_DB_PASSWORD", "warehouse_pass"),
		envOr("WAREHOUSE_DB_HOST", "127.0.0.1"),
		envOr("WAREHOUSE_DB_PORT", "3306"),
		envOr("WAREHOUSE_DB_NAME", "warehouse_db"),
	)
	warehouse, err = openDB(warehouseDSN)
	if err != nil {
		legacy.Close()
		return nil, nil, fmt.Errorf("open warehouse db: %w", err)
	}
	return legacy, warehouse, nil
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
