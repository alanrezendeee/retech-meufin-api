-- Motivo do cancelamento (slug do catálogo global curado em código —
-- internal/domain/finance/cancel_reason.go).
--
-- Além de rastro para relatórios, o motivo carrega a intenção sobre a série
-- recorrente: "encerramento" para a extensão do grupo; "sem_cobranca_no_mes"
-- e afins cancelam só aquela ocorrência e a recorrência segue.
-- Cancelamentos anteriores a esta coluna ficam NULL e preservam o
-- comportamento antigo (tratados como encerramento).
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS cancel_reason VARCHAR(30);

-- Indicador: agregações por motivo de cancelamento (dashboards/insights).
CREATE INDEX IF NOT EXISTS idx_financial_entries_cancel_reason
    ON financial_entries (workspace_id, cancel_reason)
    WHERE cancel_reason IS NOT NULL;
