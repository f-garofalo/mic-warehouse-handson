// Package main is the Phase 04 seed CLI: a small program that exercises the
// dual-write decorator against the two real MySQL containers, without needing
// HTTP handlers (HTTP arrives in Phase 06). When Phase 06 adds the HTTP layer,
// the handlers will call the same dual.Save() / dual.FindByID() this CLI uses.
//
// Subcommands (output is human-readable; this is a demo, not a script target):
//
//	seed write <id> <sku> <name> <price-cents> <currency>
//	    Build an Article and call dual.Save(). Prints OK/FAIL per side and a
//	    verify-by-read confirmation.
//
//	seed read <id>
//	    Call dual.FindByID() with the current DUAL_WRITE_READ_MODE.
//
//	seed compare <id>
//	    Read legacy_db and warehouse_db directly through their SQL adapters and
//	    report whether the two stores are aligned, without modifying either DB.
//
// Environment variables (same as the app service):
//
//	LEGACY_DB_HOST / PORT / USER / PASSWORD / NAME
//	WAREHOUSE_DB_HOST / PORT / USER / PASSWORD / NAME
//	DUAL_WRITE_READ_MODE = legacy | bc   (default: legacy)
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"warehouse.local/core/entities"
	"warehouse.local/core/repositories"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	stack, err := openStack(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ stack init failed: %v\n", err)
		os.Exit(1)
	}
	defer stack.close()

	switch cmd {
	case "write":
		os.Exit(runWrite(stack, args))
	case "read":
		os.Exit(runRead(stack, args))
	case "compare":
		os.Exit(runCompare(stack, args))
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "✗ unknown subcommand: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Phase 04 dual-write seed CLI

Usage:
  seed write <id> <sku> <name> <price-cents> <currency>
  seed read <id>
  seed compare <id>

Environment:
  DUAL_WRITE_READ_MODE = legacy | bc   (default: legacy)
  LEGACY_DB_*, WAREHOUSE_DB_*          (see docker-compose.yml)

Examples:
  seed write demo-1 ABC-001 "Widget" 2999 EUR
  seed read demo-1
  seed compare demo-1`)
}

// --- stack setup --------------------------------------------------------------

type stack struct {
	legacyDB    *sql.DB
	warehouseDB *sql.DB
	legacyRepo  *repositories.LegacyMySQLArticleRepository
	bcRepo      *repositories.MySQLArticleRepository
	dual        *repositories.DualWriteArticleRepository
	mode        repositories.ReadMode
}

func openStack(ctx context.Context) (*stack, error) {
	legacyDSN := buildDSN(
		envOr("LEGACY_DB_USER", "legacy_user"),
		envOr("LEGACY_DB_PASSWORD", "legacy_pass"),
		envOr("LEGACY_DB_HOST", "127.0.0.1"),
		envOr("LEGACY_DB_PORT", "3306"),
		envOr("LEGACY_DB_NAME", "legacy_db"),
	)
	warehouseDSN := buildDSN(
		envOr("WAREHOUSE_DB_USER", "warehouse_user"),
		envOr("WAREHOUSE_DB_PASSWORD", "warehouse_pass"),
		envOr("WAREHOUSE_DB_HOST", "127.0.0.1"),
		envOr("WAREHOUSE_DB_PORT", "3307"),
		envOr("WAREHOUSE_DB_NAME", "warehouse_db"),
	)
	legacyDB, err := openDB(ctx, legacyDSN)
	if err != nil {
		return nil, fmt.Errorf("open legacy: %w", err)
	}
	warehouseDB, err := openDB(ctx, warehouseDSN)
	if err != nil {
		legacyDB.Close()
		return nil, fmt.Errorf("open warehouse: %w", err)
	}
	mode, err := parseMode(envOr("DUAL_WRITE_READ_MODE", "legacy"))
	if err != nil {
		legacyDB.Close()
		warehouseDB.Close()
		return nil, err
	}
	legacyRepo := repositories.NewLegacyMySQLArticleRepository(legacyDB)
	bcRepo := repositories.NewMySQLArticleRepository(warehouseDB)
	dual := repositories.NewDualWriteArticleRepository(legacyRepo, bcRepo, mode)
	return &stack{
		legacyDB:    legacyDB,
		warehouseDB: warehouseDB,
		legacyRepo:  legacyRepo,
		bcRepo:      bcRepo,
		dual:        dual,
		mode:        mode,
	}, nil
}

func (s *stack) close() {
	if s.legacyDB != nil {
		s.legacyDB.Close()
	}
	if s.warehouseDB != nil {
		s.warehouseDB.Close()
	}
}

func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
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

func parseMode(s string) (repositories.ReadMode, error) {
	switch strings.ToLower(s) {
	case "legacy":
		return repositories.ReadFromLegacy, nil
	case "bc":
		return repositories.ReadFromBC, nil
	default:
		return 0, errors.New("DUAL_WRITE_READ_MODE must be legacy|bc")
	}
}

func modeName(m repositories.ReadMode) string {
	switch m {
	case repositories.ReadFromLegacy:
		return "legacy"
	case repositories.ReadFromBC:
		return "bc"
	}
	return "unknown"
}

// --- write --------------------------------------------------------------------

func runWrite(s *stack, args []string) int {
	if len(args) != 5 {
		fmt.Fprintln(os.Stderr, "✗ write requires: <id> <sku> <name> <price-cents> <currency>")
		return 2
	}
	id, skuCode, name := args[0], args[1], args[2]
	priceCents, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ price-cents must be int64: %v\n", err)
		return 2
	}
	currency := args[4]

	skuVO, err := entities.NewSKU(skuCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ NewSKU(%q): %v\n", skuCode, err)
		return 1
	}
	priceVO, err := entities.NewMoney(priceCents, currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ NewMoney(%d, %q): %v\n", priceCents, currency, err)
		return 1
	}
	article, err := entities.NewArticle(id, *skuVO, name, "seed-created", *priceVO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ NewArticle: %v\n", err)
		return 1
	}

	ctx := context.Background()
	fmt.Printf("→ dual-write Save id=%s sku=%s name=%q price=%d %s\n", id, skuCode, name, priceCents, currency)
	if err := s.dual.Save(ctx, article); err != nil {
		fmt.Printf("✗ dual.Save: %v\n", err)
		return 1
	}
	fmt.Println("✓ legacy write OK")
	fmt.Println("✓ BC write OK")

	// Verify-by-read: read each side independently and report.
	legacyArt, lerr := s.legacyRepo.FindByID(ctx, id)
	bcArt, berr := s.bcRepo.FindByID(ctx, id)
	switch {
	case lerr == nil && berr == nil:
		fmt.Printf("✓ read-back id=%s confirmed: legacy.price=%s, bc.price_cents=%d %s\n",
			id, fmtPriceFromArticle(legacyArt), bcArt.Price.AmountCents, bcArt.Price.Currency)
	case lerr != nil:
		fmt.Printf("⚠ read-back: legacy FindByID failed: %v\n", lerr)
	case berr != nil:
		fmt.Printf("⚠ read-back: BC FindByID failed: %v\n", berr)
	}
	return 0
}

func fmtPriceFromArticle(a *entities.Article) string {
	whole := a.Price.AmountCents / 100
	frac := a.Price.AmountCents % 100
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%d.%02d", whole, frac)
}

// --- read ---------------------------------------------------------------------

func runRead(s *stack, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "✗ read requires: <id>")
		return 2
	}
	id := args[0]
	ctx := context.Background()
	fmt.Printf("→ dual.FindByID id=%s mode=%s\n", id, modeName(s.mode))
	got, err := s.dual.FindByID(ctx, id)
	if err != nil {
		fmt.Printf("✗ dual.FindByID: %v\n", err)
		return 1
	}
	fmt.Printf("✓ returned: id=%s sku=%s name=%q price=%d %s\n",
		got.ID, got.SKU.Code, got.Name, got.Price.AmountCents, got.Price.Currency)
	return 0
}

// --- compare ------------------------------------------------------------------

func runCompare(s *stack, args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "✗ compare requires: <id>")
		return 2
	}
	id := args[0]
	ctx := context.Background()
	fmt.Printf("→ compare id=%s across legacy_db and warehouse_db\n", id)

	legacyArt, lerr := s.legacyRepo.FindByID(ctx, id)
	bcArt, berr := s.bcRepo.FindByID(ctx, id)
	if lerr != nil || berr != nil {
		fmt.Println("DIVERGENCE: one side cannot read the article")
		if lerr != nil {
			fmt.Printf("  legacy:    %v\n", lerr)
		} else {
			fmt.Printf("  legacy:    %s\n", articleSummary(legacyArt))
		}
		if berr != nil {
			fmt.Printf("  warehouse: %v\n", berr)
		} else {
			fmt.Printf("  warehouse: %s\n", articleSummary(bcArt))
		}
		return 1
	}

	if articlesAligned(legacyArt, bcArt) {
		fmt.Println("OK: stores aligned")
		fmt.Printf("  legacy:    %s\n", articleSummary(legacyArt))
		fmt.Printf("  warehouse: %s\n", articleSummary(bcArt))
		return 0
	}

	fmt.Println("DIVERGENCE: article fields differ")
	printFieldDiff("ID", legacyArt.ID, bcArt.ID)
	printFieldDiff("SKU", legacyArt.SKU.Code, bcArt.SKU.Code)
	printFieldDiff("Name", legacyArt.Name, bcArt.Name)
	printFieldDiff("Description", legacyArt.Description, bcArt.Description)
	printFieldDiff("Price.AmountCents", legacyArt.Price.AmountCents, bcArt.Price.AmountCents)
	printFieldDiff("Price.Currency", legacyArt.Price.Currency, bcArt.Price.Currency)
	return 1
}

func articlesAligned(a, b *entities.Article) bool {
	return a.ID == b.ID &&
		a.SKU.Code == b.SKU.Code &&
		a.Name == b.Name &&
		a.Description == b.Description &&
		a.Price.AmountCents == b.Price.AmountCents &&
		a.Price.Currency == b.Price.Currency
}

func articleSummary(a *entities.Article) string {
	return fmt.Sprintf("id=%s sku=%s name=%q price=%d %s",
		a.ID, a.SKU.Code, a.Name, a.Price.AmountCents, a.Price.Currency)
}

func printFieldDiff[T comparable](field string, legacy, warehouse T) {
	if legacy != warehouse {
		fmt.Printf("  %-18s legacy=%v warehouse=%v\n", field, legacy, warehouse)
	}
}
