-- Vincula cada item de OS a um item do catálogo (nullable = item não catalogado)
ALTER TABLE vehicle_service_order_items
    ADD COLUMN catalog_item_id UUID REFERENCES maintenance_catalog_items(id) ON DELETE SET NULL;

CREATE INDEX idx_vsoi_catalog ON vehicle_service_order_items(catalog_item_id) WHERE catalog_item_id IS NOT NULL;

-- ─── Expansão do catálogo ──────────────────────────────────────────────────────

-- Motor (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('motor','product','Injetor de Combustível','Injetor eletromagnético (troca)',80000,NULL,12,62),
('motor','product','Corpo de Borboleta','Válvula borboleta (TBI/throttle body)',NULL,NULL,12,63),
('motor','product','Bobina de Ignição','Bobina individual ou de bloco',80000,NULL,12,64),
('motor','product','Cabo de Vela','Cabos de ignição (jogo)',30000,24,NULL,65),
('motor','product','Sensor MAP / MAF','Sensor de pressão ou fluxo de ar',NULL,NULL,NULL,66),
('motor','product','Sensor de Detonação (Knock)','Sensor anti-detonante',NULL,NULL,NULL,67),
('motor','product','Válvula PCV / Respiro','Válvula de ventilação do cárter',40000,24,NULL,68),
('motor','product','Junta do Cabeçote','Junta de vedação do cabeçote',NULL,NULL,NULL,69),
('motor','product','Junta do Coletor de Admissão','Junta de vedação do coletor',NULL,NULL,NULL,70),
('motor','product','Retentor Dianteiro do Motor','Retentor do virabrequim (frente)',NULL,NULL,NULL,71),
('motor','product','Retentor Traseiro do Motor','Retentor do virabrequim (traseiro)',NULL,NULL,NULL,72),
('motor','product','Tampa de Válvulas','Tampa de válvulas com junta',NULL,NULL,NULL,73),
('motor','service','Limpeza do Corpo de Borboleta','Limpeza interna da válvula borboleta',20000,12,NULL,74),
('motor','service','Descarbonização dos Pistões','Limpeza por produto químico sem desmontagem',60000,36,NULL,75);

-- Freios (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('freios','product','Pinça de Freio Dianteira','Pinça do freio a disco dianteiro',100000,NULL,12,50),
('freios','product','Pinça de Freio Traseira','Pinça do freio a disco traseiro',100000,NULL,12,51),
('freios','product','Cilindro de Roda','Cilindro escravo do freio a tambor',80000,NULL,12,52),
('freios','product','Cilindro Mestre de Freio','Bomba do freio principal',100000,NULL,12,53),
('freios','product','Servo Freio (Hidrovácuo)','Servofreio a vácuo',100000,NULL,12,54),
('freios','product','Mangueira de Freio','Mangueira flexível do circuito de freio',60000,NULL,12,55),
('freios','product','Cabo de Freio de Mão','Cabo de tração do freio de estacionamento',80000,NULL,12,56),
('freios','product','Sensor ABS (roda)','Sensor de velocidade da roda para ABS',NULL,NULL,12,57);

-- Suspensão (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('suspensao','product','Coxim do Motor','Suporte antivibração do motor',80000,NULL,12,60),
('suspensao','product','Coxim da Caixa de Câmbio','Suporte antivibração do câmbio',80000,NULL,12,61),
('suspensao','product','Batente de Amortecedor','Borracha de batente (bump stop)',60000,NULL,12,62),
('suspensao','product','Link / Pivô do Estabilizador','Bieleta da barra estabilizadora',40000,NULL,12,63),
('suspensao','product','Bandeja Dianteira Completa','Bandeja de suspensão com bucha',80000,NULL,12,64),
('suspensao','product','Cubo de Roda','Cubo com rolamento integrado',100000,NULL,12,65),
('suspensao','service','Troca de Fluido de Direção Hidráulica','Substituição do fluido da direção',40000,24,NULL,70);

-- Transmissão (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('transmissao','product','Semi-eixo Dianteiro','Semi-eixo / homocinética completo',100000,NULL,12,50),
('transmissao','product','Óleo CVT','Fluido para câmbio continuamente variável',40000,24,NULL,51),
('transmissao','product','Coifa da Trizeta','Coifa interna do semi-eixo (trizeta)',60000,NULL,NULL,52),
('transmissao','product','Cilindro Escravo de Embreagem','Cilindro hidráulico da embreagem',80000,NULL,12,53);

-- Arrefecimento (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('arrefecimento','product','Radiador','Radiador de arrefecimento do motor',150000,NULL,12,30),
('arrefecimento','product','Ventoinha do Radiador','Motor elétrico da ventoinha',80000,NULL,12,31),
('arrefecimento','product','Reservatório de Expansão','Vasilhame do fluido de arrefecimento',NULL,NULL,NULL,32),
('arrefecimento','service','Limpeza do Sistema de Arrefecimento','Lavagem química do circuito de arrefecimento',40000,24,NULL,40);

-- Elétrico (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('eletrico','product','Sensor de Detonação (Knock)','Sensor para detecção de batida de pino',NULL,NULL,NULL,40),
('eletrico','product','Motor do Limpador de Para-brisa','Motor elétrico do sistema de limpeza',NULL,NULL,12,41),
('eletrico','product','Motor do Vidro Elétrico','Motor do elevador de vidro elétrico',NULL,NULL,12,42),
('eletrico','product','Fusível (jogo)','Fusíveis do painel e do motor',NULL,NULL,NULL,43),
('eletrico','product','Cabo de Bateria','Cabos positivo/negativo da bateria',NULL,NULL,NULL,44),
('eletrico','product','Sensor de Estacionamento','Sensor de ré / Park Assist',NULL,NULL,12,45),
('eletrico','product','Módulo de Ignição','Central de ignição eletrônica',NULL,NULL,12,46),
('eletrico','service','Polimento de Faróis','Restauração da transparência dos faróis amarelados',NULL,12,NULL,50);

-- Pneus (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('pneus','product','Válvula de Pneu','Válvula de ar da roda',NULL,NULL,NULL,30),
('pneus','service','Reparo de Pneu (remendo)','Remendo interno para furos simples',NULL,NULL,NULL,31),
('pneus','service','Desmontagem e Montagem de Pneu','Desmontagem/montagem sem troca de pneu',NULL,NULL,NULL,32);

-- Ar-condicionado (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('ar_condicionado','product','Compressor de A/C','Compressor do sistema de ar-condicionado',100000,NULL,12,30),
('ar_condicionado','product','Condensador de A/C','Condensador (radiador do A/C)',100000,NULL,12,31),
('ar_condicionado','product','Filtro Secador de A/C','Filtro-desidratador do circuito de A/C',NULL,24,NULL,32),
('ar_condicionado','product','Válvula de Expansão de A/C','Válvula termostática de expansão',NULL,NULL,12,33);

-- Carroceria (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('carroceria','product','Para-brisa','Para-brisa dianteiro (troca)',NULL,NULL,NULL,40),
('carroceria','product','Vidro Lateral','Vidro de porta ou janela',NULL,NULL,NULL,41),
('carroceria','product','Vidro Traseiro','Luneta traseira',NULL,NULL,NULL,42),
('carroceria','product','Insulfilm','Película de controle solar',NULL,NULL,NULL,43),
('carroceria','service','Reparo de Para-brisa (trinca)','Resina de restauração de trinca pequena',NULL,NULL,NULL,44),
('carroceria','service','Lataria e Funilaria','Reparo de amassados sem pintura (PDR ou convencional)',NULL,NULL,NULL,45),
('carroceria','service','Higienização Interna','Limpeza profunda do interior do veículo',NULL,6,NULL,46);

-- Serviço (complementar)
INSERT INTO maintenance_catalog_items (category, item_type, name, description, default_interval_km, default_interval_months, default_warranty_months, sort_order) VALUES
('servico','service','Revisão de 10.000 km','Revisão programada conforme manual (10k km)',10000,12,NULL,50),
('servico','service','Revisão de 20.000 km','Revisão programada conforme manual (20k km)',20000,24,NULL,51),
('servico','service','Revisão de 30.000 km','Revisão programada conforme manual (30k km)',30000,36,NULL,52),
('servico','service','Revisão de 40.000 km','Revisão programada conforme manual (40k km)',40000,48,NULL,53),
('servico','service','Revisão de 50.000 km','Revisão programada conforme manual (50k km)',50000,60,NULL,54),
('servico','service','Revisão de 60.000 km','Revisão programada conforme manual (60k km)',60000,72,NULL,55),
('servico','service','Vistoria / Inspeção Veicular','Vistoria obrigatória ou cautelar',NULL,12,NULL,56),
('servico','service','Reboque (guincho)','Serviço de reboque/socorro em estrada',NULL,NULL,NULL,57);
