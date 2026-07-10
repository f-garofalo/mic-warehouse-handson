USE mic;

-- Phase 09 read model populated by the MIC Hermes consumer.
-- It intentionally does not use record_type='articolo': the Warehouse BC remains
-- the source of truth and MIC owns only this projection for legacy order entry.
CREATE INDEX idx_business_data_type_code_projection ON business_data(record_type, code);
