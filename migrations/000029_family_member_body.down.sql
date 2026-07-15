-- Reverte dados corporais dos membros da família.

ALTER TABLE health_family_members DROP COLUMN IF EXISTS weight_kg;
ALTER TABLE health_family_members DROP COLUMN IF EXISTS height_cm;
