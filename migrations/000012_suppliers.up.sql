-- Fornecedores: cadastro híbrido (global = workspace_id NULL + tenant-próprio).
-- financial_entries ganha supplier_id opcional para vincular despesas ao payee.

CREATE TABLE suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NULL,                          -- NULL = global (gerido por nós)
    name VARCHAR(150) NOT NULL,
    category VARCHAR(30) NOT NULL DEFAULT 'outros',  -- servicos_publicos|telecom|streaming|varejo|farmacia|saude|seguros|financeiro|educacao|alimentacao|transporte|academia|outros
    default_billing_type VARCHAR(20) NULL,           -- boleto|pix|cartao_credito|debito_automatico|debito|transferencia
    pix_key VARCHAR(150) NULL,
    bank_name VARCHAR(100) NULL,
    notes TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_suppliers_workspace ON suppliers (workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_suppliers_global ON suppliers (category) WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX idx_suppliers_global_name ON suppliers (name) WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX idx_suppliers_tenant_name ON suppliers (workspace_id, name) WHERE workspace_id IS NOT NULL;

ALTER TABLE financial_entries
    ADD COLUMN supplier_id UUID NULL REFERENCES suppliers (id) ON DELETE SET NULL;

CREATE INDEX idx_financial_entries_supplier ON financial_entries (supplier_id) WHERE supplier_id IS NOT NULL;

-- ============================================================
-- SEED: fornecedores globais
-- ============================================================

INSERT INTO suppliers (name, category, default_billing_type) VALUES

-- ── ÁGUA / SANEAMENTO ────────────────────────────────────────
('Sabesp', 'servicos_publicos', 'boleto'),
('Cedae', 'servicos_publicos', 'boleto'),
('Casan', 'servicos_publicos', 'boleto'),
('Sanepar', 'servicos_publicos', 'boleto'),
('Embasa', 'servicos_publicos', 'boleto'),
('Copasa', 'servicos_publicos', 'boleto'),
('Corsan', 'servicos_publicos', 'boleto'),
('Compesa', 'servicos_publicos', 'boleto'),
('Caema', 'servicos_publicos', 'boleto'),
('Caern', 'servicos_publicos', 'boleto'),
('Cagepa', 'servicos_publicos', 'boleto'),
('Casal', 'servicos_publicos', 'boleto'),
('Deso', 'servicos_publicos', 'boleto'),
('Agespisa', 'servicos_publicos', 'boleto'),
('Cosanpa', 'servicos_publicos', 'boleto'),
('BRK Ambiental', 'servicos_publicos', 'boleto'),
('Águas do Brasil', 'servicos_publicos', 'boleto'),
('Equatorial Saneamento', 'servicos_publicos', 'boleto'),

-- ── ENERGIA ELÉTRICA ─────────────────────────────────────────
('Celesc', 'servicos_publicos', 'boleto'),
('CEMIG', 'servicos_publicos', 'boleto'),
('Copel', 'servicos_publicos', 'boleto'),
('Enel SP', 'servicos_publicos', 'boleto'),
('Enel CE', 'servicos_publicos', 'boleto'),
('Enel RJ', 'servicos_publicos', 'boleto'),
('CPFL Paulista', 'servicos_publicos', 'boleto'),
('CPFL Piratininga', 'servicos_publicos', 'boleto'),
('EDP SP', 'servicos_publicos', 'boleto'),
('EDP ES', 'servicos_publicos', 'boleto'),
('Light', 'servicos_publicos', 'boleto'),
('Elektro', 'servicos_publicos', 'boleto'),
('Equatorial MA', 'servicos_publicos', 'boleto'),
('Equatorial PA', 'servicos_publicos', 'boleto'),
('Equatorial GO', 'servicos_publicos', 'boleto'),
('Equatorial AL', 'servicos_publicos', 'boleto'),
('Equatorial PI', 'servicos_publicos', 'boleto'),
('CELPE', 'servicos_publicos', 'boleto'),
('COELBA', 'servicos_publicos', 'boleto'),
('COSERN', 'servicos_publicos', 'boleto'),
('Energisa', 'servicos_publicos', 'boleto'),
('CEB', 'servicos_publicos', 'boleto'),
('Neoenergia Brasília', 'servicos_publicos', 'boleto'),
('RGE Sul', 'servicos_publicos', 'boleto'),

-- ── GÁS ─────────────────────────────────────────────────────
('Comgás', 'servicos_publicos', 'boleto'),
('CEG', 'servicos_publicos', 'boleto'),
('Copagás', 'servicos_publicos', 'boleto'),
('Cegás', 'servicos_publicos', 'boleto'),
('Gasmig', 'servicos_publicos', 'boleto'),
('Sulgás', 'servicos_publicos', 'boleto'),
('Mitsugás', 'servicos_publicos', 'boleto'),
('Gasbrasília', 'servicos_publicos', 'boleto'),

-- ── TELECOMUNICAÇÕES ─────────────────────────────────────────
('Claro', 'telecom', 'boleto'),
('Vivo', 'telecom', 'boleto'),
('TIM', 'telecom', 'boleto'),
('Oi', 'telecom', 'boleto'),
('SKY', 'telecom', 'boleto'),
('Algar Telecom', 'telecom', 'boleto'),
('Brisanet', 'telecom', 'boleto'),
('Desktop Fibra', 'telecom', 'boleto'),
('Starlink', 'telecom', 'cartao_credito'),

-- ── STREAMING ────────────────────────────────────────────────
('Netflix', 'streaming', 'cartao_credito'),
('Spotify', 'streaming', 'cartao_credito'),
('Disney+', 'streaming', 'cartao_credito'),
('Max', 'streaming', 'cartao_credito'),
('Amazon Prime Video', 'streaming', 'cartao_credito'),
('Apple TV+', 'streaming', 'cartao_credito'),
('Globoplay', 'streaming', 'cartao_credito'),
('Paramount+', 'streaming', 'cartao_credito'),
('Crunchyroll', 'streaming', 'cartao_credito'),
('YouTube Premium', 'streaming', 'cartao_credito'),
('Deezer', 'streaming', 'cartao_credito'),
('Apple Music', 'streaming', 'cartao_credito'),
('Apple One', 'streaming', 'cartao_credito'),
('Xbox Game Pass', 'streaming', 'cartao_credito'),
('PlayStation Plus', 'streaming', 'cartao_credito'),
('Nintendo Switch Online', 'streaming', 'cartao_credito'),
('Mubi', 'streaming', 'cartao_credito'),
('Telecine', 'streaming', 'cartao_credito'),

-- ── SUPERMERCADOS ────────────────────────────────────────────
('Carrefour', 'varejo', NULL),
('Pão de Açúcar', 'varejo', NULL),
('Extra', 'varejo', NULL),
('Assaí Atacadista', 'varejo', NULL),
('Atacadão', 'varejo', NULL),
('Makro', 'varejo', NULL),
('Sam''s Club', 'varejo', NULL),
('BIG', 'varejo', NULL),
('Dia%', 'varejo', NULL),
('Sonda', 'varejo', NULL),
('Prezunic', 'varejo', NULL),
('Guanabara', 'varejo', NULL),
('Condor', 'varejo', NULL),
('Giassi', 'varejo', NULL),
('Zaffari', 'varejo', NULL),
('Angeloni', 'varejo', NULL),
('Coop', 'varejo', NULL),
('Atakarejo', 'varejo', NULL),
('Nordestão', 'varejo', NULL),
('Supernosso', 'varejo', NULL),
('Mart Minas', 'varejo', NULL),

-- ── FARMÁCIAS ────────────────────────────────────────────────
('Drogasil', 'farmacia', NULL),
('Droga Raia', 'farmacia', NULL),
('Ultrafarma', 'farmacia', NULL),
('Panvel', 'farmacia', NULL),
('Pague Menos', 'farmacia', NULL),
('São João Farmácias', 'farmacia', NULL),
('Nissei', 'farmacia', NULL),
('Pacheco', 'farmacia', NULL),
('Drogaria SP', 'farmacia', NULL),
('Farmácias Nordeste', 'farmacia', NULL),
('Bom Preço Farmácias', 'farmacia', NULL),

-- ── PLANOS DE SAÚDE ─────────────────────────────────────────
('Unimed', 'saude', 'boleto'),
('Amil', 'saude', 'boleto'),
('Hapvida', 'saude', 'boleto'),
('NotreDame Intermédica', 'saude', 'boleto'),
('SulAmérica Saúde', 'saude', 'boleto'),
('Bradesco Saúde', 'saude', 'boleto'),
('Porto Seguro Saúde', 'saude', 'boleto'),
('Golden Cross', 'saude', 'boleto'),
('Prevent Senior', 'saude', 'boleto'),
('São Cristóvão Saúde', 'saude', 'boleto'),
('Mediplan', 'saude', 'boleto'),
('Cassi', 'saude', 'boleto'),

-- ── SEGUROS ─────────────────────────────────────────────────
('Porto Seguro', 'seguros', 'boleto'),
('Bradesco Seguros', 'seguros', 'boleto'),
('SulAmérica Seguros', 'seguros', 'boleto'),
('Allianz', 'seguros', 'boleto'),
('Liberty Seguros', 'seguros', 'boleto'),
('HDI Seguros', 'seguros', 'boleto'),
('Mapfre', 'seguros', 'boleto'),
('Tokio Marine', 'seguros', 'boleto'),
('Azul Seguros', 'seguros', 'boleto'),
('Caixa Seguros', 'seguros', 'boleto'),
('Sompo Seguros', 'seguros', 'boleto'),
('Zurich', 'seguros', 'boleto'),

-- ── FINANCEIRO (faturas / financiamentos) ────────────────────
('Nubank', 'financeiro', 'boleto'),
('Itaú', 'financeiro', 'boleto'),
('Bradesco', 'financeiro', 'boleto'),
('Santander', 'financeiro', 'boleto'),
('Banco do Brasil', 'financeiro', 'boleto'),
('Caixa Econômica Federal', 'financeiro', 'boleto'),
('Inter', 'financeiro', 'boleto'),
('C6 Bank', 'financeiro', 'boleto'),
('BTG Pactual', 'financeiro', 'boleto'),
('XP Investimentos', 'financeiro', 'boleto'),
('PicPay', 'financeiro', 'pix'),
('Mercado Pago', 'financeiro', 'pix'),
('Neon', 'financeiro', 'boleto'),
('Next', 'financeiro', 'boleto'),
('Will Bank', 'financeiro', 'boleto'),

-- ── EDUCAÇÃO ────────────────────────────────────────────────
('Udemy', 'educacao', 'cartao_credito'),
('Coursera', 'educacao', 'cartao_credito'),
('Alura', 'educacao', 'cartao_credito'),
('Descomplica', 'educacao', 'cartao_credito'),
('Rocketseat', 'educacao', 'cartao_credito'),
('LinkedIn Learning', 'educacao', 'cartao_credito'),
('Duolingo', 'educacao', 'cartao_credito'),
('Hotmart', 'educacao', 'cartao_credito'),
('Estratégia Concursos', 'educacao', 'boleto'),
('Gran Cursos Online', 'educacao', 'boleto'),

-- ── ALIMENTAÇÃO / DELIVERY ───────────────────────────────────
('iFood', 'alimentacao', 'cartao_credito'),
('Rappi', 'alimentacao', 'cartao_credito'),
('Uber Eats', 'alimentacao', 'cartao_credito'),

-- ── TRANSPORTE ───────────────────────────────────────────────
('Uber', 'transporte', 'cartao_credito'),
('99', 'transporte', 'cartao_credito'),
('Cabify', 'transporte', 'cartao_credito'),
('Buser', 'transporte', 'pix'),
('FlixBus', 'transporte', 'boleto'),
('Comigo / Shell Box', 'transporte', NULL),
('Auto Posto (Ipiranga)', 'transporte', NULL),
('Auto Posto (BR / Petrobras)', 'transporte', NULL),

-- ── ACADEMIA / BEM-ESTAR ─────────────────────────────────────
('SmartFit', 'academia', 'boleto'),
('Bodytech', 'academia', 'boleto'),
('Bluefit', 'academia', 'boleto'),
('Bio Ritmo', 'academia', 'boleto'),
('Selfit', 'academia', 'boleto'),
('Wellhub (Gympass)', 'academia', 'cartao_credito'),

-- ── E-COMMERCE ───────────────────────────────────────────────
('Mercado Livre', 'varejo', NULL),
('Amazon Brasil', 'varejo', NULL),
('Shopee', 'varejo', NULL),
('Magazine Luiza', 'varejo', NULL),
('Americanas', 'varejo', NULL),
('Casas Bahia', 'varejo', NULL),
('Submarino', 'varejo', NULL),
('Centauro', 'varejo', NULL),
('Netshoes', 'varejo', NULL);
