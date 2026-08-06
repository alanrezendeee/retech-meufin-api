-- Renegociação de dívida (novação): as cobranças em aberto de um parcelamento
-- são encerradas e uma nova série de parcelas nasce no lugar.
--
-- Modelado como EVENTO, e não como ponteiro de um lançamento para outro,
-- porque a relação é N→M: dezenas de parcelas previstas mais os residuais de
-- pagamentos parciais viram um acordo novo com outra quantidade de parcelas.
-- O evento permite navegar dos dois lados e encadear renegociações sucessivas.
CREATE TABLE IF NOT EXISTS finance_renegotiations (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id         UUID         NOT NULL,
    date                 DATE         NOT NULL,
    description          VARCHAR(255) NOT NULL,
    -- Saldo apurado nas origens: parcelas previstas + residuais em aberto.
    -- Parcelas já pagas NÃO entram (são fato consumado).
    settled_amount_cents BIGINT       NOT NULL,
    -- Total do novo acordo (parcelas × valor).
    new_amount_cents     BIGINT       NOT NULL,
    -- new_amount - settled_amount. Positivo = encargo/juros; negativo = desconto.
    adjustment_cents     BIGINT       NOT NULL,
    origin_count         INTEGER      NOT NULL,
    new_count            INTEGER      NOT NULL,
    notes                TEXT,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_workspace
    ON finance_renegotiations (workspace_id);
CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_date
    ON finance_renegotiations (workspace_id, date DESC);

-- Vínculo dos dois lados: origens encerradas e parcelas novas apontam para o
-- mesmo evento. Sem isso, os relatórios contariam a dívida repactuada como
-- endividamento adicional.
ALTER TABLE financial_entries
    ADD COLUMN IF NOT EXISTS renegotiation_id UUID
    REFERENCES finance_renegotiations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_financial_entries_renegotiation
    ON financial_entries (workspace_id, renegotiation_id)
    WHERE renegotiation_id IS NOT NULL;
