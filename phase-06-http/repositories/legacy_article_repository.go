package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// LegacyMySQLArticleRepository implements interfaces.ArticleRepository against
// the simplified legacy MySQL schema (legacy-init.sql):
//
//	legacy_db.articles(id, sku, name, description, price DECIMAL(10,2),
//	                   created_at, updated_at)   -- NO currency column
//
// It is an Anti-Corruption Layer: it converts between the legacy DECIMAL price
// (no currency) and the domain Money value object (integer cents + currency),
// so DualWriteArticleRepository and entities.Article never learn legacy details.
type LegacyMySQLArticleRepository struct {
	db *sql.DB
}

func NewLegacyMySQLArticleRepository(db *sql.DB) *LegacyMySQLArticleRepository {
	return &LegacyMySQLArticleRepository{db: db}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*LegacyMySQLArticleRepository)(nil)

// legacyCurrency is the currency the ACL assumes for every legacy row. The
// legacy schema has no currency column, so we invent EUR on the way out (read)
// and drop it on the way in (write).
const legacyCurrency = "EUR"

// Save upserts the article into legacy_db.articles. The Money amount is written
// as a DECIMAL string (2999 -> "29.99") and the currency is dropped (legacy has
// no such column). Article.Inventories is out of scope for this phase.
func (r *LegacyMySQLArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	now := time.Now().UTC()
	a.UpdatedAt = now
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	const upsert = `
		INSERT INTO articles (id, sku, name, description, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  sku = VALUES(sku),
		  name = VALUES(name),
		  description = VALUES(description),
		  price = VALUES(price),
		  updated_at = VALUES(updated_at)
	`
	_, err := r.db.ExecContext(ctx, upsert,
		a.ID, a.SKU.Code, a.Name, a.Description,
		centsToDecimal(a.Price.AmountCents),
		a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *LegacyMySQLArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles WHERE id = ?`
	return scanLegacyArticle(r.db.QueryRowContext(ctx, q, id))
}

func (r *LegacyMySQLArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles WHERE sku = ?`
	return scanLegacyArticle(r.db.QueryRowContext(ctx, q, skuCode))
}

func (r *LegacyMySQLArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Article
	for rows.Next() {
		a, err := scanLegacyArticleFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *LegacyMySQLArticleRepository) Delete(ctx context.Context, id string) error {
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

// scanLegacyArticle rehydrates one legacy row into an *entities.Article. Distinct
// from the BC adapter's scanArticle: the legacy price is a DECIMAL string and the
// currency is defaulted to EUR.
func scanLegacyArticle(row *sql.Row) (*entities.Article, error) {
	var (
		id, sku, name, desc, price string
		createdAt, updatedAt       time.Time
	)
	if err := row.Scan(&id, &sku, &name, &desc, &price, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, fmt.Errorf("scan legacy article: %w", err)
	}
	return rehydrateLegacyArticle(id, sku, name, desc, price, createdAt, updatedAt)
}

func scanLegacyArticleFromRows(rows *sql.Rows) (*entities.Article, error) {
	var (
		id, sku, name, desc, price string
		createdAt, updatedAt       time.Time
	)
	if err := rows.Scan(&id, &sku, &name, &desc, &price, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return rehydrateLegacyArticle(id, sku, name, desc, price, createdAt, updatedAt)
}

// rehydrateLegacyArticle rebuilds the aggregate from legacy columns, converting
// the DECIMAL price to cents and rebuilding SKU/Money through the domain factories.
func rehydrateLegacyArticle(id, sku, name, desc, price string, createdAt, updatedAt time.Time) (*entities.Article, error) {
	cents, err := decimalToCents(price)
	if err != nil {
		return nil, fmt.Errorf("rehydrate legacy price %q: %w", price, err)
	}
	skuVO, err := entities.NewSKU(sku)
	if err != nil {
		return nil, fmt.Errorf("rehydrate SKU: %w", err)
	}
	priceVO, err := entities.NewMoney(cents, legacyCurrency)
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

// centsToDecimal converts integer cents into the legacy DECIMAL string, e.g.
// 2999 -> "29.99". Integer/string math only — never float64, which would
// silently corrupt exact monetary values. Money.AmountCents is always >= 0.
func centsToDecimal(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// decimalToCents parses a legacy DECIMAL string back into integer cents:
// "1" -> 100, "1.5" -> 150, "29.99" -> 2999. No float64: it splits on the dot
// and does integer arithmetic. A fractional part longer than two digits is
// rounded half-up.
func decimalToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("decimalToCents: empty string")
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	wholeVal, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimalToCents: invalid integer part in %q: %w", s, err)
	}
	if !hasFrac {
		return wholeVal * 100, nil
	}

	// Normalize the fractional part to exactly two digits.
	switch {
	case len(frac) == 0:
		frac = "00"
	case len(frac) == 1:
		frac += "0"
	case len(frac) == 2:
		// already two digits
	default: // more than two digits: round half-up on the third digit
		roundUp := frac[2] >= '5'
		fv, perr := strconv.ParseInt(frac[:2], 10, 64)
		if perr != nil {
			return 0, fmt.Errorf("decimalToCents: invalid fractional part in %q: %w", s, perr)
		}
		if roundUp {
			fv++ // fv may reach 100, which carries correctly into the whole part below
		}
		return wholeVal*100 + fv, nil
	}
	fv, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimalToCents: invalid fractional part in %q: %w", s, err)
	}
	return wholeVal*100 + fv, nil
}
