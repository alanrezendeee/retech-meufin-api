-- Reverte Consultas & Agenda + Planos de Saúde.

DROP TABLE IF EXISTS health_appointments;
DROP TABLE IF EXISTS health_plan_members;
DROP TABLE IF EXISTS health_plans;

ALTER TABLE health_labs DROP COLUMN IF EXISTS kind;
