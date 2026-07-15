-- Módulo: Saúde Familiar — dados corporais dos membros.
-- Adiciona altura (cm) e peso (kg) em health_family_members.
-- 100% idempotente: pode rodar múltiplas vezes sem erro.

ALTER TABLE health_family_members ADD COLUMN IF NOT EXISTS height_cm NUMERIC(5, 1);
ALTER TABLE health_family_members ADD COLUMN IF NOT EXISTS weight_kg NUMERIC(5, 2);
