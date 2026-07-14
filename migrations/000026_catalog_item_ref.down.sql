DROP INDEX IF EXISTS idx_vsoi_catalog;
ALTER TABLE vehicle_service_order_items DROP COLUMN IF EXISTS catalog_item_id;

DELETE FROM maintenance_catalog_items WHERE sort_order >= 30 AND category IN ('arrefecimento','pneus','ar_condicionado');
DELETE FROM maintenance_catalog_items WHERE sort_order >= 40 AND category IN ('carroceria','eletrico','servico');
DELETE FROM maintenance_catalog_items WHERE sort_order >= 50 AND category IN ('freios','transmissao');
DELETE FROM maintenance_catalog_items WHERE sort_order >= 60 AND category IN ('motor','suspensao');
