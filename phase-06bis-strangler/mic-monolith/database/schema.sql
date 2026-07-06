-- =============================================================
-- MIC - Modernizziamo in Claude
-- Schema "didattico" anti-pattern: 2 tabelle generiche con tipi mischiati.
-- TUTTI i dati di business (clienti, articoli, ordini, fatture, ...)
-- vivono insieme in business_data + business_relations.
-- Brutto di proposito: e' la palestra perfetta per estrarre Bounded Context
-- con Strangler Fig.
-- =============================================================

CREATE DATABASE IF NOT EXISTS mic
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE mic;

DROP TABLE IF EXISTS business_relations;
DROP TABLE IF EXISTS business_data;

-- -------------------------------------------------------------
-- business_data
-- Una sola tabella che ospita 16+ tipi di entita' di business.
-- I campi sono volutamente generici (amount_1..4, text_1..5, date_1..3).
-- Il significato dipende da record_type. Es:
--   cliente:    code=P.IVA, name=ragione sociale, text_1=email, text_2=PEC, text_3=SDI, text_4=indirizzo, text_5=citta
--   articolo:   code=SKU,   name=nome,            description=descrizione, amount_1=prezzo_listino_base, amount_2=qta_min
--   ordine:     code=numero_ordine, amount_1=totale_imponibile, amount_2=totale_iva, amount_3=totale, status='bozza|confermato|spedito|annullato'
--   fattura:    code=numero, amount_1=imponibile, amount_2=iva, amount_3=totale, status='bozza|inviato|pagato|scaduto'
-- -------------------------------------------------------------
CREATE TABLE business_data (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  record_type VARCHAR(50) NOT NULL,
    -- 'cliente', 'fornitore', 'contatto', 'articolo', 'categoria',
    -- 'aliquota_iva', 'listino', 'voce_listino', 'sconto',
    -- 'magazzino', 'movimento', 'agente', 'ordine', 'riga_ordine',
    -- 'fattura', 'nota_credito', 'pagamento', 'utente', 'audit_log'

  code VARCHAR(100),                  -- SKU / P.IVA / numero_fattura / codice listino...
  name VARCHAR(500),                  -- ragione sociale / nome articolo / titolo...
  description TEXT,

  parent_id BIGINT,                   -- riferimento polimorfico (es. contatto -> cliente)
  parent_type VARCHAR(50),

  amount_1 DECIMAL(15,4),             -- prezzo, totale, percentuale, qta...
  amount_2 DECIMAL(15,4),
  amount_3 DECIMAL(15,4),
  amount_4 DECIMAL(15,4),

  date_1 DATE,                        -- creazione / scadenza / validita_da
  date_2 DATE,                        -- validita_a
  date_3 DATE,

  text_1 VARCHAR(500),                -- email / indirizzo / nota...
  text_2 VARCHAR(500),
  text_3 VARCHAR(500),
  text_4 VARCHAR(500),
  text_5 VARCHAR(500),

  status VARCHAR(50),                 -- 'attivo','bozza','confermato','inviato','pagato','annullato','scaduto',...

  payload_json JSON,                  -- discarica per i campi che non entrano sopra

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_type (record_type),
  INDEX idx_parent (parent_type, parent_id),
  INDEX idx_status (status),
  INDEX idx_code (code),
  INDEX idx_type_status (record_type, status),
  INDEX idx_type_date1 (record_type, date_1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -------------------------------------------------------------
-- business_relations
-- Tabella unica per TUTTE le relazioni N-N e 1-N "soft".
-- Esempi di relation_type:
--   'cliente_di_ordine'        source=ordine,  target=cliente
--   'agente_di_ordine'         source=ordine,  target=agente
--   'listino_di_ordine'        source=ordine,  target=listino
--   'articolo_in_riga_ordine'  source=riga,    target=articolo, amount=qta
--   'iva_di_riga'              source=riga,    target=aliquota_iva
--   'voce_di_listino'          source=voce,    target=listino
--   'articolo_di_voce'         source=voce,    target=articolo,  amount=prezzo
--   'fattura_di_ordine'        source=fattura, target=ordine
--   'pagamento_di_fattura'     source=pagamento,target=fattura,  amount=importo
--   'movimento_di_articolo'    source=movim.,  target=articolo,  amount=qta
--   'movimento_in_magazzino'   source=movim.,  target=magazzino
--   'sconto_su_ordine'         source=ordine,  target=sconto
--   'categoria_di_articolo'    source=articolo,target=categoria
-- -------------------------------------------------------------
CREATE TABLE business_relations (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  source_id BIGINT NOT NULL,
  target_id BIGINT NOT NULL,
  relation_type VARCHAR(50) NOT NULL,
  amount DECIMAL(15,4),                -- qta, prezzo unitario, sconto %, importo
  metadata JSON,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  INDEX idx_source (source_id, relation_type),
  INDEX idx_target (target_id, relation_type),
  INDEX idx_relation (relation_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
