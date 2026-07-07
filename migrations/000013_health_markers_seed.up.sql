-- Seed do catálogo global de marcadores laboratoriais (scope = 'system').
-- Idempotente: ON CONFLICT DO NOTHING em ambas as tabelas.
-- Pode ser executado diretamente no console do banco sem risco de duplicação.

INSERT INTO health_markers (
    id, scope, canonical_name, normalized_key,
    category, comparability_class, canonical_unit,
    active, created_at, updated_at
) VALUES
    (gen_random_uuid(), 'system', 'Glicose',               'glicose',               'bioquimica',  'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Hemoglobina glicada',   'hemoglobina glicada',   'bioquimica',  'standardized',    '%',            true, now(), now()),
    (gen_random_uuid(), 'system', 'Creatinina',            'creatinina',            'renal',       'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Ureia',                 'ureia',                 'renal',       'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Ácido úrico',           'acido urico',           'bioquimica',  'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Colesterol total',      'colesterol total',      'lipidico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Colesterol HDL',        'colesterol hdl',        'lipidico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Colesterol LDL',        'colesterol ldl',        'lipidico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Triglicerídeos',        'triglicerideos',        'lipidico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'AST (TGO)',             'ast tgo',               'hepatico',    'standardized',    'U/L',          true, now(), now()),
    (gen_random_uuid(), 'system', 'ALT (TGP)',             'alt tgp',               'hepatico',    'standardized',    'U/L',          true, now(), now()),
    (gen_random_uuid(), 'system', 'Gama GT',               'gama gt',               'hepatico',    'standardized',    'U/L',          true, now(), now()),
    (gen_random_uuid(), 'system', 'Fosfatase alcalina',    'fosfatase alcalina',    'hepatico',    'standardized',    'U/L',          true, now(), now()),
    (gen_random_uuid(), 'system', 'Bilirrubina total',     'bilirrubina total',     'hepatico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Bilirrubina direta',    'bilirrubina direta',    'hepatico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Bilirrubina indireta',  'bilirrubina indireta',  'hepatico',    'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Albumina',              'albumina',              'bioquimica',  'standardized',    'g/dL',         true, now(), now()),
    (gen_random_uuid(), 'system', 'Proteína C reativa',    'proteina c reativa',    'inflamacao',  'standardized',    'mg/L',         true, now(), now()),
    (gen_random_uuid(), 'system', 'Hemoglobina',           'hemoglobina',           'hematologia', 'standardized',    'g/dL',         true, now(), now()),
    (gen_random_uuid(), 'system', 'Hematócrito',           'hematocrito',           'hematologia', 'standardized',    '%',            true, now(), now()),
    (gen_random_uuid(), 'system', 'Leucócitos',            'leucocitos',            'hematologia', 'standardized',    '/mm³',         true, now(), now()),
    (gen_random_uuid(), 'system', 'Plaquetas',             'plaquetas',             'hematologia', 'standardized',    '/mm³',         true, now(), now()),
    (gen_random_uuid(), 'system', 'Hemácias',              'hemacias',              'hematologia', 'standardized',    'milhões/mm³',  true, now(), now()),
    (gen_random_uuid(), 'system', 'VCM',                   'vcm',                   'hematologia', 'standardized',    'fL',           true, now(), now()),
    (gen_random_uuid(), 'system', 'Sódio',                 'sodio',                 'eletrolitos', 'standardized',    'mEq/L',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Potássio',              'potassio',              'eletrolitos', 'standardized',    'mEq/L',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Cálcio',                'calcio',                'eletrolitos', 'standardized',    'mg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Ferro sérico',          'ferro serico',          'bioquimica',  'standardized',    'µg/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'TSH',                   'tsh',                   'hormonios',   'method_dependent','µUI/mL',       true, now(), now()),
    (gen_random_uuid(), 'system', 'T4 livre',              't4 livre',              'hormonios',   'method_dependent','ng/dL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Insulina',              'insulina',              'hormonios',   'method_dependent','µUI/mL',       true, now(), now()),
    (gen_random_uuid(), 'system', 'Ferritina',             'ferritina',             'bioquimica',  'method_dependent','ng/mL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Vitamina D',            'vitamina d',            'vitaminas',   'method_dependent','ng/mL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Vitamina B12',          'vitamina b12',          'vitaminas',   'method_dependent','pg/mL',        true, now(), now()),
    (gen_random_uuid(), 'system', 'Homocisteína',          'homocisteina',          'bioquimica',  'method_dependent','µmol/L',       true, now(), now())
ON CONFLICT DO NOTHING;

-- ── Aliases ─────────────────────────────────────────────────────────────────
-- Cada INSERT seleciona o marker_id pelo normalized_key para ser idempotente.

INSERT INTO health_marker_aliases (id, marker_id, scope, alias, normalized_alias, source, created_at, updated_at)
SELECT gen_random_uuid(), m.id, 'system', a.alias, a.norm, 'seed', now(), now()
FROM health_markers m
JOIN (VALUES
    -- Glicose
    ('glicose',             'Glicemia',                        'glicemia'),
    ('glicose',             'Glicemia de jejum',               'glicemia de jejum'),
    ('glicose',             'GLU',                             'glu'),
    -- Hemoglobina glicada
    ('hemoglobina glicada', 'HbA1c',                           'hba1c'),
    ('hemoglobina glicada', 'A1C',                             'a1c'),
    ('hemoglobina glicada', 'Hemoglobina glicosilada',         'hemoglobina glicosilada'),
    -- Creatinina
    ('creatinina',          'CREA',                            'crea'),
    -- Ureia
    ('ureia',               'URE',                             'ure'),
    -- Ácido úrico
    ('acido urico',         'Urato',                           'urato'),
    -- Colesterol total
    ('colesterol total',    'Colesterol',                      'colesterol'),
    -- Colesterol HDL
    ('colesterol hdl',      'HDL',                             'hdl'),
    ('colesterol hdl',      'HDL-c',                           'hdl c'),
    -- Colesterol LDL
    ('colesterol ldl',      'LDL',                             'ldl'),
    ('colesterol ldl',      'LDL-c',                           'ldl c'),
    -- Triglicerídeos
    ('triglicerideos',      'Triglicérides',                   'triglicerides'),
    ('triglicerideos',      'TG',                              'tg'),
    -- AST (TGO)
    ('ast tgo',             'TGO',                             'tgo'),
    ('ast tgo',             'AST',                             'ast'),
    ('ast tgo',             'Aspartato aminotransferase',      'aspartato aminotransferase'),
    ('ast tgo',             'Transaminase oxalacética',        'transaminase oxalacetica'),
    -- ALT (TGP)
    ('alt tgp',             'TGP',                             'tgp'),
    ('alt tgp',             'ALT',                             'alt'),
    ('alt tgp',             'Alanina aminotransferase',        'alanina aminotransferase'),
    ('alt tgp',             'Transaminase pirúvica',           'transaminase piruvica'),
    -- Gama GT
    ('gama gt',             'GGT',                             'ggt'),
    ('gama gt',             'Gama glutamil transferase',       'gama glutamil transferase'),
    -- Fosfatase alcalina
    ('fosfatase alcalina',  'FA',                              'fa'),
    ('fosfatase alcalina',  'ALP',                             'alp'),
    -- Bilirrubinas
    ('bilirrubina total',   'BT',                              'bt'),
    ('bilirrubina direta',  'BD',                              'bd'),
    ('bilirrubina indireta','BI',                              'bi'),
    -- Proteína C reativa
    ('proteina c reativa',  'PCR',                             'pcr'),
    ('proteina c reativa',  'CRP',                             'crp'),
    -- Hemoglobina
    ('hemoglobina',         'Hb',                              'hb'),
    -- Hematócrito
    ('hematocrito',         'Ht',                              'ht'),
    ('hematocrito',         'HCT',                             'hct'),
    -- Leucócitos
    ('leucocitos',          'Glóbulos brancos',                'globulos brancos'),
    ('leucocitos',          'WBC',                             'wbc'),
    -- Plaquetas
    ('plaquetas',           'PLT',                             'plt'),
    -- Hemácias
    ('hemacias',            'Eritrócitos',                     'eritrocitos'),
    ('hemacias',            'Glóbulos vermelhos',              'globulos vermelhos'),
    ('hemacias',            'RBC',                             'rbc'),
    -- VCM
    ('vcm',                 'Volume corpuscular médio',        'volume corpuscular medio'),
    ('vcm',                 'MCV',                             'mcv'),
    -- Eletrólitos
    ('sodio',               'Na',                              'na'),
    ('potassio',            'K',                               'k'),
    ('calcio',              'Ca',                              'ca'),
    -- Ferro sérico
    ('ferro serico',        'Ferro',                           'ferro'),
    ('ferro serico',        'Fe',                              'fe'),
    -- TSH
    ('tsh',                 'Hormônio tireoestimulante',       'hormonio tireoestimulante'),
    ('tsh',                 'Tirotrofina',                     'tirotrofina'),
    -- T4 livre
    ('t4 livre',            'T4L',                             't4l'),
    ('t4 livre',            'Tiroxina livre',                  'tiroxina livre'),
    ('t4 livre',            'Free T4',                         'free t4'),
    -- Insulina
    ('insulina',            'Insulina de jejum',               'insulina de jejum'),
    -- Ferritina
    ('ferritina',           'FER',                             'fer'),
    -- Vitamina D
    ('vitamina d',          '25-OH vitamina D',                '25 oh vitamina d'),
    ('vitamina d',          '25 hidroxivitamina D',            '25 hidroxivitamina d'),
    -- Vitamina B12
    ('vitamina b12',        'B12',                             'b12'),
    ('vitamina b12',        'Cobalamina',                      'cobalamina')
) AS a(marker_key, alias, norm)
  ON m.normalized_key = a.marker_key AND m.scope = 'system' AND m.deleted_at IS NULL
ON CONFLICT DO NOTHING;
