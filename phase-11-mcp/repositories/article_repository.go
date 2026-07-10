package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrArticleNotFound is returned by FindByID and FindBySKU when no row matches.
var ErrArticleNotFound = errors.New("repositories: article not found")

// MySQLArticleRepository implements interfaces.ArticleRepository against MySQL.
// The schema (warehouse-init.sql) stores the aggregate's outer fields. Inventory
// levels are stored in a child table managed by the same repository.
type MySQLArticleRepository struct {
	db *sql.DB
}

func NewMySQLArticleRepository(db *sql.DB) *MySQLArticleRepository {
	return &MySQLArticleRepository{db: db}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*MySQLArticleRepository)(nil)

func (r *MySQLArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	now := time.Now().UTC()
	a.UpdatedAt = now

	// Upsert: insert if new, update if existing. We rely on the aggregate's
	// invariants (NewArticle / ChangePrice) to have validated the inputs.
	const upsert = `
		INSERT INTO articles (id, sku, name, description, price_cents, currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  sku = VALUES(sku),
		  name = VALUES(name),
		  description = VALUES(description),
		  price_cents = VALUES(price_cents),
		  currency = VALUES(currency),
		  updated_at = VALUES(updated_at)
	`
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	_, err := r.db.ExecContext(ctx, upsert,
		a.ID, a.SKU.Code, a.Name, a.Description,
		a.Price.AmountCents, a.Price.Currency,
		a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return err
	}
	return r.saveInventoryLevels(ctx, a)
}

func (r *MySQLArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price_cents, currency, created_at, updated_at FROM articles WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, id)
	a, err := scanArticle(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadInventoryLevels(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MySQLArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price_cents, currency, created_at, updated_at FROM articles WHERE sku = ?`
	row := r.db.QueryRowContext(ctx, q, skuCode)
	a, err := scanArticle(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadInventoryLevels(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *MySQLArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price_cents, currency, created_at, updated_at FROM articles ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Article
	for rows.Next() {
		a, err := scanArticleFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *MySQLArticleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// saveInventoryLevels upserts the aggregate's per-location stock rows.
// Until Phase 09 the adjust flow looked persisted but was not: the aggregate
// updated its in-memory levels and Save silently dropped them.
func (r *MySQLArticleRepository) saveInventoryLevels(ctx context.Context, a *entities.Article) error {
	const upsert = `
		INSERT INTO inventory_levels (id, article_id, location_code, quantity, reserved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  quantity = VALUES(quantity),
		  reserved = VALUES(reserved),
		  updated_at = VALUES(updated_at)
	`
	for i := range a.Inventories {
		lvl := &a.Inventories[i]
		if lvl.CreatedAt.IsZero() {
			lvl.CreatedAt = a.UpdatedAt
		}
		lvl.UpdatedAt = a.UpdatedAt
		if _, err := r.db.ExecContext(ctx, upsert,
			lvl.ID, a.ID, lvl.LocationCode, lvl.Quantity, lvl.Reserved,
			lvl.CreatedAt, lvl.UpdatedAt,
		); err != nil {
			return fmt.Errorf("save inventory level %s: %w", lvl.LocationCode, err)
		}
	}
	return nil
}

// loadInventoryLevels attaches the persisted stock rows to the aggregate.
func (r *MySQLArticleRepository) loadInventoryLevels(ctx context.Context, a *entities.Article) error {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, location_code, quantity, reserved, created_at, updated_at
		   FROM inventory_levels WHERE article_id = ? ORDER BY location_code`, a.ID)
	if err != nil {
		return fmt.Errorf("load inventory levels: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		lvl := entities.InventoryLevel{ArticleID: a.ID}
		if err := rows.Scan(&lvl.ID, &lvl.LocationCode, &lvl.Quantity, &lvl.Reserved, &lvl.CreatedAt, &lvl.UpdatedAt); err != nil {
			return fmt.Errorf("scan inventory level: %w", err)
		}
		a.Inventories = append(a.Inventories, lvl)
	}
	return rows.Err()
}

// scanArticle decodes one row into an *entities.Article. It bypasses the
// aggregate's NewArticle factory because the row is loaded from a trusted
// source (the BC's own table). Reconstruction must not fail on data that
// was previously persisted as valid.
func scanArticle(row *sql.Row) (*entities.Article, error) {
	var (
		id, sku, name, desc, currency string
		priceCents                    int64
		createdAt, updatedAt          time.Time
	)
	if err := row.Scan(&id, &sku, &name, &desc, &priceCents, &currency, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, fmt.Errorf("scan article: %w", err)
	}
	skuVO, err := entities.NewSKU(sku)
	if err != nil {
		return nil, fmt.Errorf("rehydrate SKU: %w", err)
	}
	priceVO, err := entities.NewMoney(priceCents, currency)
	if err != nil {
		return nil, fmt.Errorf("rehydrate Money: %w", err)
	}
	a := &entities.Article{
		ID:          id,
		SKU:         *skuVO,
		Name:        name,
		Description: desc,
		Price:       *priceVO,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	return a, nil
}

func scanArticleFromRows(rows *sql.Rows) (*entities.Article, error) {
	var (
		id, sku, name, desc, currency string
		priceCents                    int64
		createdAt, updatedAt          time.Time
	)
	if err := rows.Scan(&id, &sku, &name, &desc, &priceCents, &currency, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	skuVO, err := entities.NewSKU(sku)
	if err != nil {
		return nil, fmt.Errorf("rehydrate SKU: %w", err)
	}
	priceVO, err := entities.NewMoney(priceCents, currency)
	if err != nil {
		return nil, fmt.Errorf("rehydrate Money: %w", err)
	}
	return &entities.Article{
		ID:          id,
		SKU:         *skuVO,
		Name:        name,
		Description: desc,
		Price:       *priceVO,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
