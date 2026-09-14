-- Quitação de contrato e troca de bem financiado.
--
-- O evento de renegociação já resolve o núcleo do problema — apurar o saldo em
-- aberto de um parcelamento, encerrar as cobranças e manter o vínculo duplo —
-- mas só sabia fazer uma coisa: substituir a série por outra série. Quitar um
-- financiamento (com desconto, pago pelo próprio usuário ou por um terceiro,
-- como a concessionária numa troca) é estruturalmente a mesma operação, com
-- outro desfecho. Em vez de uma segunda máquina paralela, a tabela de eventos
-- ganha um `kind`:
--
--   renegociacao  série antiga → série nova (comportamento original)
--   quitacao      série antiga → um lançamento realizado de quitação
--   troca_bem     série antiga → quitação + venda do bem (troca) + série nova
--                 no bem novo, tudo num único evento
--
-- Os lançamentos ganham vínculo com o bem (asset_type/asset_id): é o que
-- permite abrir um veículo e ver o contrato que o financia — hoje a dívida
-- não sabe qual bem paga.

ALTER TABLE finance_renegotiations
    ADD COLUMN IF NOT EXISTS kind                   VARCHAR(20) NOT NULL DEFAULT 'renegociacao',
    -- Quitação: quanto foi efetivamente pago para encerrar o saldo apurado.
    ADD COLUMN IF NOT EXISTS payoff_cents           BIGINT,
    -- Quem pagou a quitação: 'proprio' (caixa do usuário) ou 'terceiro'
    -- (concessionária/comprador abateu do valor do bem — sem caixa).
    ADD COLUMN IF NOT EXISTS payer                  VARCHAR(20),
    ADD COLUMN IF NOT EXISTS payoff_entry_id        UUID REFERENCES financial_entries (id) ON DELETE SET NULL,
    -- Bem envolvido: o financiado/quitado (asset_id) e, na troca, o novo.
    ADD COLUMN IF NOT EXISTS asset_type             VARCHAR(20),
    ADD COLUMN IF NOT EXISTS asset_id               UUID,
    ADD COLUMN IF NOT EXISTS new_asset_id           UUID,
    -- Troca: valor de avaliação do bem usado e entrada em dinheiro.
    ADD COLUMN IF NOT EXISTS trade_in_cents         BIGINT,
    ADD COLUMN IF NOT EXISTS trade_in_entry_id      UUID REFERENCES financial_entries (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cash_downpayment_cents BIGINT,
    ADD COLUMN IF NOT EXISTS downpayment_entry_id   UUID REFERENCES financial_entries (id) ON DELETE SET NULL;

ALTER TABLE finance_renegotiations DROP CONSTRAINT IF EXISTS chk_finance_renegotiations_kind;
ALTER TABLE finance_renegotiations
    ADD CONSTRAINT chk_finance_renegotiations_kind
    CHECK (kind IN ('renegociacao', 'quitacao', 'troca_bem'));

ALTER TABLE finance_renegotiations DROP CONSTRAINT IF EXISTS chk_finance_renegotiations_payer;
ALTER TABLE finance_renegotiations
    ADD CONSTRAINT chk_finance_renegotiations_payer
    CHECK (payer IS NULL OR payer IN ('proprio', 'terceiro'));

CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_asset
    ON finance_renegotiations (workspace_id, asset_id) WHERE asset_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_new_asset
    ON finance_renegotiations (workspace_id, new_asset_id) WHERE new_asset_id IS NOT NULL;

-- Vínculo lançamento ↔ bem. Sem FK física: o bem pode viver em tabelas
-- diferentes (vehicles, properties) e o módulo financeiro não deve depender
-- da existência de cada uma.
ALTER TABLE financial_entries
    ADD COLUMN IF NOT EXISTS asset_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS asset_id   UUID;

ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS chk_financial_entries_asset_type;
ALTER TABLE financial_entries
    ADD CONSTRAINT chk_financial_entries_asset_type
    CHECK (asset_type IS NULL OR asset_type IN ('vehicle', 'property'));

CREATE INDEX IF NOT EXISTS idx_financial_entries_asset
    ON financial_entries (workspace_id, asset_type, asset_id) WHERE asset_id IS NOT NULL;

-- Receita de venda de bem (troca, venda direta) entra no catálogo curado.
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS chk_financial_entries_type;
ALTER TABLE financial_entries
    ADD CONSTRAINT chk_financial_entries_type CHECK (
        type IS NULL
        OR kind = 'debit'
        OR (kind = 'credit' AND type IN (
            'salario','pj_contrato','pro_labore','dividendos','rendimento',
            'aluguel','freela','ferias_13','beneficio','reembolso','venda_bem','outro'
        ))
    );
