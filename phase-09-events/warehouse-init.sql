-- Warehouse BC database schema (warehouse_db).
-- Owned by the Go service. Mirrors the Article aggregate persistence shape.

CREATE DATABASE IF NOT EXISTS warehouse_db
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE warehouse_db;

CREATE TABLE IF NOT EXISTS articles (
  id VARCHAR(36) PRIMARY KEY,
  sku VARCHAR(32) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  price_cents BIGINT NOT NULL,
  currency CHAR(3) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_sku (sku)
);

CREATE TABLE IF NOT EXISTS inventory_levels (
  id VARCHAR(36) PRIMARY KEY,
  article_id VARCHAR(36) NOT NULL,
  location_code VARCHAR(20) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  reserved INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
  UNIQUE KEY uniq_article_location (article_id, location_code),
  INDEX idx_article_id (article_id)
);
