USE warehouse_db;

INSERT INTO articles (id, sku, name, description, price_cents, currency, created_at, updated_at)
SELECT
  CAST(id AS CHAR) AS id,
  LEFT(code, 32) AS sku,
  COALESCE(name, code) AS name,
  COALESCE(description, '') AS description,
  CAST(ROUND(COALESCE(amount_1, 0) * 100) AS SIGNED) AS price_cents,
  'EUR' AS currency,
  COALESCE(created_at, CURRENT_TIMESTAMP) AS created_at,
  COALESCE(updated_at, CURRENT_TIMESTAMP) AS updated_at
FROM mic.business_data
WHERE record_type = 'articolo'
  AND code IS NOT NULL
  AND code <> ''
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  price_cents = VALUES(price_cents),
  currency = VALUES(currency),
  updated_at = VALUES(updated_at);

SELECT COUNT(*) AS warehouse_articles_after_backfill FROM articles;
