-- Trilha de eventos do ciclo de vida dos lançamentos (fase 4 do modelo
-- contábil): registro imutável de cada transição — liquidação, reabertura,
-- cancelamento, mudança de vencimento — com data, valores e autor.
--
-- Motivação: o momento da liquidação nunca foi um fato próprio; o único
-- vestígio era updated_at, que é "último toque" e foi contaminado por
-- edições em massa (ex.: rename de parcelamento). Evento é append-only:
-- "quando isso foi pago de verdade?" passa a ter resposta por construção.
CREATE TABLE IF NOT EXISTS finance_entry_events (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID        NOT NULL,
    entry_id          UUID        NOT NULL REFERENCES financial_entries (id) ON DELETE CASCADE,
    event             VARCHAR(30) NOT NULL, -- confirmed|settled|reopened|cancelled|due_date_changed
    from_status       VARCHAR(15),
    to_status         VARCHAR(15),
    paid_at           TIMESTAMPTZ,
    paid_amount_cents BIGINT,
    cancel_reason     VARCHAR(30),
    old_due_date      DATE,
    new_due_date      DATE,
    actor_user_id     UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_finance_entry_events_entry
    ON finance_entry_events (workspace_id, entry_id, created_at);
