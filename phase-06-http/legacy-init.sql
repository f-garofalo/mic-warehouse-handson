-- Legacy database schema (legacy_db).
-- Mimics what the PHP monolith would expose for the dual-write source.
-- Simpler shape than warehouse_db on purpose: legacy stores price as a decimal,
-- with no currency split. The dual-write decorator translates between shapes
-- via the repository implementations. See ADR-013.

CREATE DATABASE IF NOT EXISTS legacy_db
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE legacy_db;

CREATE TABLE IF NOT EXISTS articles (
  id VARCHAR(36) PRIMARY KEY,
  sku VARCHAR(32) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  price DECIMAL(10, 2) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_sku (sku)
);

CREATE TABLE IF NOT EXISTS inventory (
  id VARCHAR(36) PRIMARY KEY,
  article_id VARCHAR(36) NOT NULL,
  quantity INT NOT NULL DEFAULT 0,
  location VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);
