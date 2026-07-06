package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// LegacyMICArticleRepository is the Anti-Corruption Layer between the Warehouse
// BC and the MIC monolith's god-table `business_data` (database `mic`).
//
// Article rows are the subset WHERE record_type='articolo' AND code IS NOT
// NULL AND code <> ''. Column meaning for that record_type (see
// mic-monolith/database/schema.sql and ArticleController.php):
//
//	id        BIGINT AUTO_INCREMENT -> Article.ID (decimal string)
//	code                            -> SKU
//	name                            -> Name (fallback: code)
//	description                     -> Description
//	amount_1  DECIMAL(15,4)         -> Price (EUR, converted to integer cents
//	                                   via string parsing; never float64)
//	amount_2 / text_1 / text_2 / status / payload_json
//	                                -> qta_minima / categoria / iva_default /
//	                                   status / {unita,peso_kg}: NOT owned by
//	                                   the BC; preserved on update, defaulted
//	                                   on insert (the write-side seam).
//
// Inventories are out of scope for the migration: this adapter persists and
// rehydrates the article row only, always with empty Inventories.
type LegacyMICArticleRepository struct {
	db *sql.DB
}

func NewLegacyMICArticleRepository(db *sql.DB) *LegacyMICArticleRepository {
	return &LegacyMICArticleRepository{db: db}
}

var _ interfaces.ArticleRepository = (*LegacyMICArticleRepository)(nil)

// Save persists the aggregate into business_data.
//
// New aggregate (ID == ""): INSERT. The legacy store is the system of record
// during the migration, so it MINTS the id: after the insert we read
// LAST_INSERT_ID and assign it (as a decimal string) to article.ID.
//
// Existing aggregate (ID != ""): UPDATE of the BC-owned columns only (code,
// name, description, amount_1). Columns the BC does not own (categoria,
// iva_default, qta_minima, status, payload_json) are left exactly as the
// monolith wrote them: clobbering them would corrupt data the monolith still
// reads.
func (r *LegacyMICArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	now := time.Now().UTC()
	a.UpdatedAt = now
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	priceDecimal := centsToDecimal(a.Price.AmountCents)

	if a.ID == "" {
		// WRITE-SIDE SEAM: the BC's model does not carry qta_minima, categoria,
		// iva_default, status or unita, but the monolith expects every articolo
		// row to have them. Until those concepts migrate (or die), the ACL
		// writes the monolith's own defaults, matching what
		// ArticleController::fromPayload would have produced for a minimal
		// create: amount_2=1, text_1='WAREHOUSE', text_2='IVA22',
		// status='attivo', payload_json={"unita":"pz"}.
		const insert = `
			INSERT INTO business_data
			  (record_type, code, name, description, amount_1,
			   amount_2, text_1, text_2, status, payload_json,
			   created_at, updated_at)
			VALUES ('articolo', ?, ?, ?, ?,
			        1, 'WAREHOUSE', 'IVA22', 'attivo', '{"unita": "pz"}',
			        ?, ?)
		`
		res, err := r.db.ExecContext(ctx, insert,
			a.SKU.Code, a.Name, a.Description, priceDecimal,
			a.CreatedAt, a.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("legacy insert article: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("legacy insert article: read minted id: %w", err)
		}
		a.ID = strconv.FormatInt(id, 10)
		return nil
	}

	id, err := parseLegacyID(a.ID)
	if err != nil {
		return err
	}
	// Update only the columns the BC owns; everything else stays untouched.
	const update = `
		UPDATE business_data
		SET code = ?, name = ?, description = ?, amount_1 = ?, updated_at = ?
		WHERE id = ? AND record_type = 'articolo'
	`
	_, err = r.db.ExecContext(ctx, update,
		a.SKU.Code, a.Name, a.Description, priceDecimal, a.UpdatedAt, id,
	)
	if err != nil {
		return fmt.Errorf("legacy update article %s: %w", a.ID, err)
	}
	return nil
}

const legacyArticleColumns = `id, code, COALESCE(name, ''), COALESCE(description, ''), COALESCE(amount_1, 0), created_at, updated_at`

const legacyArticleFilter = `record_type = 'articolo' AND code IS NOT NULL AND code <> ''`

func (r *LegacyMICArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	numericID, err := parseLegacyID(id)
	if err != nil {
		// A non-numeric id cannot exist in business_data (BIGINT pk).
		return nil, ErrArticleNotFound
	}
	q := `SELECT ` + legacyArticleColumns + ` FROM business_data WHERE ` + legacyArticleFilter + ` AND id = ?`
	row := r.db.QueryRowContext(ctx, q, numericID)
	return scanLegacyArticle(row)
}

func (r *LegacyMICArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	q := `SELECT ` + legacyArticleColumns + ` FROM business_data WHERE ` + legacyArticleFilter + ` AND code = ?`
	row := r.db.QueryRowContext(ctx, q, skuCode)
	return scanLegacyArticle(row)
}

func (r *LegacyMICArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	q := `SELECT ` + legacyArticleColumns + ` FROM business_data WHERE ` + legacyArticleFilter + ` ORDER BY id`
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

func (r *LegacyMICArticleRepository) Delete(ctx context.Context, id string) error {
	numericID, err := parseLegacyID(id)
	if err != nil {
		return ErrArticleNotFound
	}
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM business_data WHERE record_type = 'articolo' AND id = ?`, numericID)
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

// parseLegacyID validates that the aggregate id is a decimal string matching
// business_data's BIGINT primary key.
func parseLegacyID(id string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("legacy article id %q is not a business_data id: %w", id, err)
	}
	return n, nil
}

func scanLegacyArticle(row *sql.Row) (*entities.Article, error) {
	var (
		id, code, name, desc string
		priceDecimal         string
		createdAt, updatedAt time.Time
	)
	if err := row.Scan(&id, &code, &name, &desc, &priceDecimal, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, fmt.Errorf("scan legacy article: %w", err)
	}
	return rehydrateLegacyArticle(id, code, name, desc, priceDecimal, createdAt, updatedAt)
}

func scanLegacyArticleFromRows(rows *sql.Rows) (*entities.Article, error) {
	var (
		id, code, name, desc string
		priceDecimal         string
		createdAt, updatedAt time.Time
	)
	if err := rows.Scan(&id, &code, &name, &desc, &priceDecimal, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return rehydrateLegacyArticle(id, code, name, desc, priceDecimal, createdAt, updatedAt)
}

// rehydrateLegacyArticle translates one business_data row into the domain
// aggregate. The SKU goes through the domain factory (garbage codes in the
// god-table must fail loudly, not leak into the BC); the price is converted
// from the legacy DECIMAL string to integer cents without float64. The
// currency is always EUR: the monolith has no currency column.
func rehydrateLegacyArticle(id, code, name, desc, priceDecimal string, createdAt, updatedAt time.Time) (*entities.Article, error) {
	skuVO, err := entities.NewSKU(code)
	if err != nil {
		return nil, fmt.Errorf("rehydrate SKU from legacy code %q: %w", code, err)
	}
	if strings.TrimSpace(name) == "" {
		name = code
	}
	priceCents, err := decimalToCents(priceDecimal)
	if err != nil {
		return nil, fmt.Errorf("rehydrate price: %w", err)
	}
	priceVO, err := entities.NewMoney(priceCents, "EUR")
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

// centsToDecimal renders integer cents as a legacy DECIMAL string ("29.99").
// Pure string/integer arithmetic: no float64 anywhere near money.
func centsToDecimal(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	if frac < 0 {
		frac = -frac
	}
	sign := ""
	if cents < 0 && whole == 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%02d", sign, whole, frac)
}

// decimalToCents parses a legacy DECIMAL string into integer cents, rounding
// half-up beyond two fractional digits (business_data stores DECIMAL(15,4)).
// Pure string/integer arithmetic: no float64 anywhere near money.
func decimalToCents(s string) (int64, error) {
	neg := false
	rest := s
	if len(rest) > 0 && (rest[0] == '-' || rest[0] == '+') {
		neg = rest[0] == '-'
		rest = rest[1:]
	}

	dot := -1
	for i, c := range rest {
		if c == '.' {
			dot = i
			break
		}
	}

	var wholeStr, fracStr string
	if dot < 0 {
		wholeStr = rest
		fracStr = "00"
	} else {
		wholeStr = rest[:dot]
		fracStr = rest[dot+1:]
		if len(fracStr) == 0 {
			fracStr = "00"
		} else if len(fracStr) == 1 {
			fracStr += "0"
		} else if len(fracStr) > 2 {
			lead := fracStr[:2]
			tail := fracStr[2:]
			n, err := strconv.ParseInt(lead, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("decimal %q: bad fractional %q", s, fracStr)
			}
			if tail[0] >= '5' {
				n++
				if n >= 100 {
					n -= 100
					w, werr := strconv.ParseInt(wholeStr, 10, 64)
					if werr != nil {
						return 0, fmt.Errorf("decimal %q: bad whole %q", s, wholeStr)
					}
					wholeStr = strconv.FormatInt(w+1, 10)
				}
			}
			fracStr = fmt.Sprintf("%02d", n)
		}
	}

	if wholeStr == "" {
		wholeStr = "0"
	}
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimal %q: bad whole %q", s, wholeStr)
	}
	frac, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimal %q: bad fractional %q", s, fracStr)
	}
	if whole > math.MaxInt64/100 {
		return 0, fmt.Errorf("decimal %q: overflow", s)
	}
	cents := whole*100 + frac
	if neg {
		cents = -cents
	}
	return cents, nil
}
