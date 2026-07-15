-- Reverte avatares.
DROP INDEX IF EXISTS idx_user_profiles_workspace;
DROP TABLE IF EXISTS user_profiles;

ALTER TABLE health_family_members
    DROP COLUMN IF EXISTS avatar_object_key;
