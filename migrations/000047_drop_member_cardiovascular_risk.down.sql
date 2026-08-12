ALTER TABLE health_family_members
    ADD COLUMN IF NOT EXISTS cardiovascular_risk VARCHAR(20) NULL;
