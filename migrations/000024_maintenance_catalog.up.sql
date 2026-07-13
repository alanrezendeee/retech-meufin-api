-- Catálogo global de itens de manutenção — usado para sugestão automática de intervalos
CREATE TABLE IF NOT EXISTS maintenance_catalog_items (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category                  TEXT NOT NULL,
    item_type                 TEXT NOT NULL CHECK (item_type IN ('product','service')),
    name                      TEXT NOT NULL,
    description               TEXT,
    default_interval_km       INT,
    default_interval_months   INT,
    default_warranty_months   INT,
    active                    BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order                INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_mci_category ON maintenance_catalog_items(category);
CREATE INDEX idx_mci_active   ON maintenance_catalog_items(active);

-- ─── SEED: Motor ───────────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('motor','product','Óleo do Motor 0W20','Óleo sintético recomendado para motores flex modernos e turbo',7500,6,NULL,10),
('motor','product','Óleo do Motor 5W30','Óleo sintético / semissintético para uso geral',5000,6,NULL,11),
('motor','product','Óleo do Motor 5W40','Óleo sintético para motores a gasolina e diesel',5000,6,NULL,12),
('motor','product','Óleo do Motor 10W40','Óleo semissintético para motores mais antigos',5000,6,NULL,13),
('motor','product','Filtro de Óleo','Troca obrigatória a cada troca de óleo',5000,6,NULL,20),
('motor','product','Filtro de Combustível','Filtro da linha de combustível (gasolina/etanol)',30000,24,NULL,30),
('motor','product','Filtro de Ar do Motor','Filtra o ar que entra no motor',20000,12,NULL,31),
('motor','product','Vela de Ignição (comum)','Velas de ignição de cobre ou níquel',30000,24,NULL,40),
('motor','product','Vela de Ignição (iridium/platina)','Velas de alta durabilidade',80000,48,NULL,41),
('motor','product','Correia Dentada (kit distribuição)','Kit completo: correia, tensores, bomba d''água',80000,48,12,50),
('motor','product','Correia Acessórios / Poly-V','Correia que aciona alternador, compressor AC e direção',60000,48,NULL,51),
('motor','product','Tensor da Correia','Tensor da correia de acessórios',60000,48,NULL,52),
('motor','product','Bomba d''Água','Bomba do sistema de arrefecimento (geralmente trocada com a correia)',80000,48,12,53),
('motor','product','Bomba de Combustível','Bomba elétrica da linha de combustível',100000,60,NULL,54),
('motor','product','Vela de Aquecimento (diesel)','Vela de pré-aquecimento para motores diesel',60000,48,NULL,55),
('motor','service','Limpeza de Injetores','Limpeza ultrassônica ou on-car dos injetores de combustível',40000,24,NULL,60),
('motor','service','Descarbonização','Limpeza interna do motor e câmara de combustão',60000,36,NULL,61);

-- ─── SEED: Freios ──────────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('freios','product','Pastilha de Freio Dianteiro','Jogo de pastilhas dianteiras',40000,NULL,12,10),
('freios','product','Pastilha de Freio Traseiro','Jogo de pastilhas traseiras',60000,NULL,12,11),
('freios','product','Disco de Freio Dianteiro','Par de discos dianteiros',80000,NULL,12,12),
('freios','product','Disco de Freio Traseiro','Par de discos traseiros',80000,NULL,12,13),
('freios','product','Fluido de Freio DOT4','Fluido de freio padrão DOT4',40000,24,NULL,20),
('freios','product','Fluido de Freio DOT4 LV','Fluido de freio baixa viscosidade (veículos modernos)',40000,24,NULL,21),
('freios','product','Fluido de Freio DOT5.1','Fluido de freio de alto desempenho',40000,24,NULL,22),
('freios','product','Tambor de Freio Traseiro','Par de tambores (veículos com freio a tambor)',80000,NULL,12,30),
('freios','product','Sapata de Freio Traseiro','Jogo de sapatas (veículos com tambor)',60000,NULL,12,31),
('freios','service','Troca de Fluido de Freio','Sangria e substituição completa do fluido',40000,24,NULL,40),
('freios','service','Retífica de Disco de Freio','Torno dos discos para eliminar empenamento',NULL,NULL,NULL,41),
('freios','service','Retífica de Tambor de Freio','Torno dos tambores para eliminar empenamento',NULL,NULL,NULL,42);

-- ─── SEED: Suspensão ───────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('suspensao','product','Amortecedor Dianteiro','Par de amortecedores dianteiros',60000,NULL,12,10),
('suspensao','product','Amortecedor Traseiro','Par de amortecedores traseiros',60000,NULL,12,11),
('suspensao','product','Bucha da Bandeja Dianteira','Buchas de borracha da bandeja de suspensão',60000,NULL,12,20),
('suspensao','product','Bucha do Estabilizador','Buchas de borracha do estabilizador/barra)',40000,24,12,21),
('suspensao','product','Terminal de Direção','Par de terminais da barra de direção',60000,NULL,12,22),
('suspensao','product','Pivô de Direção','Par de pivôs (ball joint)',80000,NULL,12,23),
('suspensao','product','Rolamento de Roda Dianteiro','Rolamento do cubo de roda dianteiro',80000,NULL,12,30),
('suspensao','product','Rolamento de Roda Traseiro','Rolamento do cubo de roda traseiro',80000,NULL,12,31),
('suspensao','product','Mola Helicoidal Dianteira','Par de molas dianteiras',100000,NULL,12,40),
('suspensao','product','Mola Helicoidal Traseira','Par de molas traseiras',100000,NULL,12,41),
('suspensao','service','Alinhamento e Balanceamento','Alinhamento de direção e balanceamento das rodas',10000,12,NULL,50),
('suspensao','service','Geometria 3D','Alinhamento com medição computadorizada de 3 eixos',20000,12,NULL,51);

-- ─── SEED: Transmissão ─────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('transmissao','product','Óleo do Câmbio Manual','Lubrificante para câmbio mecânico',60000,36,NULL,10),
('transmissao','product','Óleo do Câmbio Automático','Fluido ATF para câmbio automático',60000,36,NULL,11),
('transmissao','product','Filtro do Câmbio Automático','Filtro interno do câmbio automático',60000,36,NULL,12),
('transmissao','product','Óleo do Diferencial','Lubrificante do diferencial',60000,36,NULL,13),
('transmissao','product','Coifa / Junta Homocinética','Coifa de borracha da homocinética',80000,NULL,12,20),
('transmissao','product','Kit de Embreagem','Kit completo: disco, platô e rolamento de pressão',80000,NULL,12,30),
('transmissao','service','Troca de Óleo do Câmbio','Drenagem e substituição do óleo do câmbio',60000,36,NULL,40);

-- ─── SEED: Arrefecimento ───────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('arrefecimento','product','Fluido de Arrefecimento','Aditivo para radiador (refrigerante)',40000,24,NULL,10),
('arrefecimento','product','Termostato','Válvula termostática do sistema de arrefecimento',60000,36,12,11),
('arrefecimento','product','Tampa do Radiador','Tampa de pressão do radiador',40000,NULL,NULL,12),
('arrefecimento','product','Mangueira do Radiador','Mangueiras superior e inferior do radiador',60000,NULL,NULL,13),
('arrefecimento','service','Troca de Fluido de Arrefecimento','Limpeza do sistema e substituição do fluido',40000,24,NULL,20);

-- ─── SEED: Elétrico ────────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('eletrico','product','Bateria','Bateria do veículo',NULL,48,12,10),
('eletrico','product','Alternador','Gerador de energia do veículo',100000,NULL,12,11),
('eletrico','product','Motor de Partida','Arranque do motor',100000,NULL,12,12),
('eletrico','product','Sensor de Oxigênio (Lambda)','Sonda lambda para controle da mistura ar/combustível',80000,48,12,20),
('eletrico','product','Sensor de Temperatura do Motor','Sensor de temperatura (para o computador de bordo)',80000,NULL,NULL,21),
('eletrico','product','Sensor de Rotação (CKP/CMP)','Sensor de rotação do virabrequim ou comando',80000,NULL,NULL,22),
('eletrico','service','Diagnóstico por Scanner','Leitura de códigos de falha com scanner OBD2',NULL,NULL,NULL,30),
('eletrico','service','Serviço de Scanner','Scanner completo com análise de sensores ao vivo',NULL,NULL,NULL,31);

-- ─── SEED: Pneus ───────────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('pneus','product','Pneu','Pneu (por unidade)',40000,48,NULL,10),
('pneus','product','Estepe (step)','Pneu estepe sobressalente',NULL,60,NULL,11),
('pneus','service','Rodízio de Pneus','Rotação cruzada dos pneus para desgaste uniforme',10000,6,NULL,20),
('pneus','service','Alinhamento de Direção','Alinhamento das rodas dianteiras e traseiras',10000,12,NULL,21),
('pneus','service','Balanceamento de Rodas','Balanceamento estático/dinâmico',10000,12,NULL,22),
('pneus','service','Calibragem de Pneus','Verificação e ajuste da pressão dos pneus',NULL,1,NULL,23);

-- ─── SEED: Ar Condicionado ─────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('ar_condicionado','product','Filtro de Cabine / Pólen','Filtro que purifica o ar que entra na cabine',15000,12,NULL,10),
('ar_condicionado','product','Gás Refrigerante R134a','Recarga do gás refrigerante do ar condicionado',NULL,24,NULL,11),
('ar_condicionado','product','Gás Refrigerante R1234yf','Gás refrigerante de nova geração',NULL,24,NULL,12),
('ar_condicionado','service','Higienização do Ar Condicionado','Limpeza e desodorização do sistema de ventilação',NULL,12,NULL,20),
('ar_condicionado','service','Recarga de Gás do AC','Verificação de vazamentos e recarga',NULL,24,NULL,21);

-- ─── SEED: Carroceria ──────────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('carroceria','product','Jogo de Palheta do Parabrisa','Palhetas dianteiras do limpador',NULL,12,NULL,10),
('carroceria','product','Palheta Traseira','Palheta do limpador traseiro',NULL,12,NULL,11),
('carroceria','product','Fluido do Limpador de Para-brisa','Fluido para o reservatório do limpador',NULL,6,NULL,12),
('carroceria','product','Lâmpada de Farol (halógena)','Lâmpada H4, H7 ou similar',NULL,NULL,NULL,20),
('carroceria','product','Lâmpada de Farol (LED/Xenon)','Lâmpada de alta eficiência',NULL,NULL,NULL,21),
('carroceria','product','Lâmpada de Freio','Lâmpada da luz de freio',NULL,NULL,NULL,22),
('carroceria','service','Polimento e Cristalização','Polimento da pintura e aplicação de cera/coating',NULL,12,NULL,30),
('carroceria','service','Funilaria e Pintura','Serviço de lataria e pintura automotiva',NULL,NULL,NULL,31);

-- ─── SEED: Serviços gerais ─────────────────────────────────────────────────────
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('servico','service','Revisão Geral','Revisão completa do veículo conforme manual do fabricante',NULL,12,NULL,10),
('servico','service','Higienização do Motor','Limpeza externa do motor com produto específico',NULL,24,NULL,11),
('servico','service','Lavagem e Limpeza','Lavagem externa e interna do veículo',NULL,NULL,NULL,12),
('servico','service','Mão de Obra','Hora de trabalho do mecânico (item genérico)',NULL,NULL,NULL,13),
('servico','service','Diagnóstico RuidCar','Diagnóstico especializado em ruídos e vibrações',NULL,NULL,NULL,14),
('servico','product','Anel de Vedação','Anel de vedação (retentor) — uso genérico',NULL,NULL,NULL,20),
('servico','product','Junta','Junta de vedação — uso genérico',NULL,NULL,NULL,21);
