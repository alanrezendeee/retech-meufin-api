# Plano — Módulo Saúde Familiar (MeuFin)

> Status: **planejamento**. Nada implementado. Documento para revisão antes de codar.
> Decisões travadas: catálogo **híbrido (system + tenant)**, **soft-delete** nas tabelas de saúde, **sem armazenar senha de laboratório** no MVP.

## 1. Contexto e princípios

- Módulo dentro do `retech-meufin-api` (monólito, bounded context próprio), **não** microserviço.
- Espelha o padrão atual: `internal/domain/health`, `internal/application/health`, `internal/infrastructure/persistence`, `internal/interfaces/http/handlers`.
- Coluna de tenant no banco = **`workspace_id`** (convenção existente), alimentada pelo `tenant_id` do token via `middleware.WorkspaceID`. Onde a spec original diz `tenant_id`, lê-se `workspace_id`.
- Rotas sob `/api/v1/health`, protegidas pelo `RequireAuth` (JWT/JWKS) já existente. Isolamento por `workspace_id` em **toda** query.
- Soft-delete: todas as tabelas health têm `deleted_at TIMESTAMPTZ NULL`. DELETE = soft. Queries filtram `deleted_at IS NULL`.

## 2. Coração do módulo: catálogo canônico + dedup

### 2.1 Por que
Guardar `marker_name` como texto livre quebra histórico/evolução e torna dedup inviável. Resultados referenciam catálogo por **FK**.

### 2.2 Dois níveis
- **Marcador/analito** (`health_markers`) — unidade de histórico (glicose, ferritina, TGO, HbA1c). **Must-have.**
- **Exame/painel** (`health_exams` + `health_exam_markers`) — agrupa marcadores (Hemograma → Hemoglobina, Hematócrito...). **Nice-to-have** (Fase pós-MVP).

### 2.3 Escopo híbrido
- `scope = 'system'`: seed global, `workspace_id NULL`, compartilhado, editável só por seed/admin.
- `scope = 'tenant'`: criado pelo tenant, `workspace_id` setado.
- Dedup e resolução consideram os **dois** escopos (system + o próprio tenant).

### 2.4 Estratégia de dedup (em camadas)
1. **Normalização** (função Go no domínio + índice Postgres `unaccent`+`lower`): minúsculo, sem acento/pontuação, espaços colapsados.
2. **Unicidade por índice parcial:**
   - system: `normalized_key` único global (`WHERE scope='system' AND deleted_at IS NULL`).
   - tenant: único por `(workspace_id, normalized_key)` (`WHERE scope='tenant' AND deleted_at IS NULL`).
   - aliases: `normalized_alias` único no escopo.
   - `loinc_code` único no escopo quando não nulo.
3. **Criar novo marcador:** normaliza → busca em `markers.normalized_key` + `aliases.normalized_alias` (system + tenant). Match exato → **409 + sugestão** do existente.
4. **Near-duplicate:** `pg_trgm similarity` > 0.85 → **warning** ("parecido com X, confirmar?"), não bloqueia.
5. **Unidade na ingestão:** valida unidade do resultado contra a esperada do marcador; divergência → flag (evita misturar mg/dL com mmol/L no histórico).

### 2.5 Resolução na extração (válvula de segurança)
LLM/OCR gera nomes crus → etapa de resolução mapeia `raw_marker_name → marker_id` via normalized/alias. Não-resolvidos vão para **fila de revisão humana** (confirma existente ou cria novo). LLM nunca cria marcador sozinho. `raw_marker_name` sempre preservado.

## 3. Modelagem (migration `000002_health`)

> Convenções: `id UUID PK DEFAULT gen_random_uuid()`, `workspace_id UUID`, `created_at/updated_at TIMESTAMPTZ DEFAULT now()`, `deleted_at TIMESTAMPTZ NULL`. Índice `(workspace_id)` em toda tabela com escopo de tenant. FKs com `ON DELETE RESTRICT` (soft-delete cuida da remoção lógica).

### Catálogo
```sql
-- extensões
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

health_markers
  id, scope VARCHAR(10) NOT NULL,           -- 'system' | 'tenant'
  workspace_id UUID NULL,                    -- NULL quando system
  canonical_name VARCHAR(255) NOT NULL,
  normalized_key VARCHAR(255) NOT NULL,      -- gerado no app (unaccent+lower+trim+collapse)
  loinc_code VARCHAR(20) NULL,
  category VARCHAR(50) NOT NULL,             -- hematologia|bioquimica|hormonios|vitaminas|hepatico|lipidico|outros
  default_unit VARCHAR(30) NULL,
  default_ref_min NUMERIC NULL,
  default_ref_max NUMERIC NULL,
  default_ref_text VARCHAR(255) NULL,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at, updated_at, deleted_at
-- índices parciais de dedup:
CREATE UNIQUE INDEX uq_markers_system_key ON health_markers (normalized_key)
  WHERE scope='system' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_markers_tenant_key ON health_markers (workspace_id, normalized_key)
  WHERE scope='tenant' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_markers_system_loinc ON health_markers (loinc_code)
  WHERE scope='system' AND loinc_code IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_markers_trgm ON health_markers USING gin (normalized_key gin_trgm_ops);

health_marker_aliases
  id, marker_id UUID NOT NULL REFERENCES health_markers(id),
  scope VARCHAR(10) NOT NULL, workspace_id UUID NULL,
  alias VARCHAR(255) NOT NULL, normalized_alias VARCHAR(255) NOT NULL,
  source VARCHAR(50) NULL, created_at, updated_at, deleted_at
CREATE UNIQUE INDEX uq_alias_system ON health_marker_aliases (normalized_alias)
  WHERE scope='system' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_alias_tenant ON health_marker_aliases (workspace_id, normalized_alias)
  WHERE scope='tenant' AND deleted_at IS NULL;
```

### Domínio (todas com workspace_id + soft-delete)
```
health_family_members
  full_name, relationship (self|spouse|child|parent|other), birth_date,
  gender NULL, document NULL, notes NULL, active BOOL

health_labs                     -- SEM login_username/password (decisão)
  name, website_url NULL, exam_results_url NULL,
  contact_phone NULL, address NULL, notes NULL, active BOOL

health_exam_requests
  family_member_id FK, requested_by NULL, request_date,
  lab_id FK NULL, status (draft|requested|collected|partially_resulted|resulted|canceled), notes NULL

health_exam_request_items
  exam_request_id FK, marker_id FK NULL, exam_name, exam_code NULL,
  body_area NULL, notes NULL, status (pending|collected|resulted|canceled)

health_exam_results
  family_member_id FK, lab_id FK NULL, exam_request_id FK NULL,
  exam_date, collection_date NULL, release_date NULL,
  source_type (manual|pdf|image|ocr|llm),
  status (draft|processing|extracted|reviewed|failed), summary NULL, notes NULL

health_exam_result_items
  exam_result_id FK, marker_id FK NULL,          -- FK dirige histórico
  raw_marker_name VARCHAR(255) NULL,             -- o que o lab/LLM escreveu
  result_value VARCHAR(255), result_numeric NUMERIC NULL, unit VARCHAR(30) NULL,
  reference_min NUMERIC NULL, reference_max NUMERIC NULL, reference_text VARCHAR(255) NULL,
  interpretation (low|normal|high|critical|inconclusive) NULL,
  interpretation_computed (idem) NULL,           -- calculada por value×ref
  method NULL, material NULL, raw_text NULL
CREATE INDEX idx_result_items_history ON health_exam_result_items (workspace_id, marker_id)
  WHERE deleted_at IS NULL;

health_documents
  family_member_id FK NULL, lab_id FK NULL, exam_request_id FK NULL, exam_result_id FK NULL,
  document_type (exam_request|exam_result|image_report|medical_report|prescription|other),
  file_name, original_file_name, mime_type, size_bytes,
  storage_provider (minio) DEFAULT 'minio', bucket, object_key, checksum NULL,
  uploaded_by_user_id UUID NOT NULL,
  extraction_status (pending|processing|extracted|failed|not_required),
  extracted_text NULL, extracted_json JSONB NULL, metadata JSONB NULL

health_extraction_jobs
  document_id FK, provider, model NULL,
  status (pending|processing|completed|failed), input_type (pdf|image),
  prompt_version NULL, raw_response JSONB NULL, error_message NULL,
  started_at NULL, finished_at NULL

health_audit_logs
  workspace_id, user_id, action, entity_type, entity_id, metadata JSONB, created_at
  -- auditável: upload, extração, resultado revisado, download, lab criado/editado
```

### Relacionamentos
Iguais à spec (family_members/labs 1:N requests/results/documents; requests 1:N items/results/documents; results 1:N items/documents; documents 1:N extraction_jobs), **acrescido** de `markers 1:N result_items` e `markers 1:N request_items` e `markers 1:N aliases`.

### Seed (migration idempotente, `INSERT ... ON CONFLICT (normalized_key) DO NOTHING`)
~40 marcadores `scope='system'` com `canonical_name`, `category`, `default_unit`, aliases e `loinc_code` quando conhecido:
glicose, hemoglobina glicada (HbA1c), creatinina, ureia, ácido úrico, ferritina, TGO/AST, TGP/ALT, GGT, fosfatase alcalina, bilirrubina total/direta/indireta, colesterol total/HDL/LDL/VLDL, triglicerídeos, vitamina D (25-OH), vitamina B12, homocisteína, hemoglobina, hematócrito, leucócitos, plaquetas, TSH, T4 livre, PCR, sódio, potássio, etc. Aliases: TGO↔AST, TGP↔ALT, GGT↔Gama GT, HbA1c↔Hemoglobina glicada.

## 4. API (`/api/v1/health`)
Conforme spec (family-members, labs, exam-requests + items + documents, exam-results + items + documents, documents upload/download-url/extract/extraction-status, dashboard + evolução). **Acrescentar** endpoints de catálogo:
```
GET    /markers?query=&category=          # busca (system + tenant)
POST   /markers                            # cria tenant (dedup 409 + sugestão)
GET    /markers/:id
PUT    /markers/:id                        # só tenant-scope
DELETE /markers/:id                        # só tenant-scope (soft)
POST   /markers/resolve                    # [{raw_name, unit}] → [{marker_id?, candidates[], action}]
```

## 5. Upload / MinIO
ENV obrigatórias: `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_HEALTH`, `MINIO_USE_SSL`, `HEALTH_MAX_UPLOAD_MB`. Aceitar PDF/JPG/PNG. Path `tenants/{workspace_id}/health/{family_member_id}/{yyyy}/{mm}/{uuid}-{filename}`. Download só via **presigned URL**. `object_key` nunca exposto sem autorização.

## 6. Extração OCR/LLM
Porta/adaptador `HealthDocumentExtractor.Extract(ctx, file, type) -> (text, structuredJSON)`. ENV: `HEALTH_EXTRACTION_PROVIDER` (anthropic|openai|google|local|disabled), `HEALTH_EXTRACTION_MODEL`, `HEALTH_EXTRACTION_API_KEY`, `HEALTH_EXTRACTION_BASE_URL?`. App só sobe se `provider != disabled` exigir as ENVs. `disabled` → upload funciona, extração retorna erro controlado.
- **Async**: upload responde rápido; `health_extraction_jobs` processa em background (goroutine worker no MVP); status por polling.
- **Provider default recomendado: Anthropic Claude** (vision + structured output pro JSON alvo da spec). Ao implementar, consultar referência da API Claude.
- Saída = JSON estruturado da spec (patient_name, exam_date, exams[], summary, warnings). **Não** é diagnóstico — só extração.
- Pós-extração → **resolução** raw→marker_id (seção 2.5) antes de virar histórico.

## 7. Segurança / LGPD
Dado de saúde = categoria especial (LGPD art. 11). Isolamento por `workspace_id` em toda query; permissões CASL; não logar conteúdo extraído nem PII; `object_key` protegido; auditoria (`health_audit_logs`). Sem credencial de portal de lab (decisão). Prever retenção/minimização e consentimento por membro em fase futura.

## 8. Impacto cross-repo
- **retech-auth-api**: registrar abilities `health:read|create|update|delete|upload|extract` e associar à role do usuário (abilities vêm do `/v1/me`).
- **retech-meufin-admin**: menu "Saúde Familiar" + subjects CASL + telas (Dashboard, Membros, Laboratórios, Solicitações, Resultados, Documentos) seguindo o padrão visual; cliente usa `meufinClient` já existente.
- **retech-meufin-api**: tudo do §3–§7.

## 9. Faseamento
| Fase | Entrega | Depende |
|---|---|---|
| 0 — Catálogo | markers + aliases + normalização + dedup + resolve + seed | — |
| 1 — Core CRUD | family_members, labs, requests/items, results/items (manual), isolamento, audit | 0 |
| 2 — Documentos | MinIO upload + presigned + health_documents | 1 |
| 3 — Extração | jobs async + Anthropic + tela de resolução | 0,2 |
| 4 — Dashboard | evolução por marcador + cards + gráficos | 1 |
Cada fase mergeável isolada. Frontend acompanha por fase.

## 10. Boas práticas (mantidas da spec)
DDD/Clean Arch (domain sem infra), repos por interface, use cases separados, DTO ≠ entidade, validações de domínio, erros padronizados (reusar `errrespond`), logs estruturados, migrations versionadas. Frontend: componentes reutilizáveis, forms validados, loading/empty/error states, confirmação antes de excluir, responsivo.

## 11. Fora de escopo agora
RAG, n8n, Mastra, diagnóstico/recomendação clínica, integração automática com portal de lab, notificações complexas, microserviço. Modelagem já deixa preparada.

## 12. Gráfico de evolução: absoluto + normalizado (cross-lab)

Referência de exame **varia por laboratório** (método, equipamento, reagente, população). Mas o problema real são **dois**:
1. A faixa de referência muda entre labs.
2. O valor medido em si é comparável entre labs? → depende do analito.

Por isso o catálogo tem `comparability_class`:
- `standardized` — valor comparável entre labs (hematócrito, glicose, creatinina, hemoglobina...). Um gráfico absoluto único combina labs sem problema.
- `method_dependent` — valor cru varia com o método (ferritina, vit D, TSH...). Comparar absoluto entre labs engana.
- `qualitative` — sem valor numérico contínuo.

**Gráfico único com 2 modos:**
- **Absoluto (default)** — valor na unidade canônica, X=data, banda de referência sombreada, ponto colorido por laboratório (legenda). Tooltip com valor, unidade, lab, método e a referência daquele resultado. Ótimo para `standardized`.
- **Normalizado (toggle)** — cada ponto pela posição na própria referência:
  ```
  x' = 2·(x − ref_min)/(ref_max − ref_min) − 1
   0 = meio da faixa; ±1 = limites; >+1 acima; <−1 abaixo
  ```
  Resolve "referências diferentes num gráfico só" de forma honesta. É o default para `method_dependent`.

Regras: unidade **canônica obrigatória** na ingestão (converter antes de plotar); referência aberta ("< 200") usa distância ao limite; sem referência → default do catálogo ou sai do modo normalizado. **Não usar z-score** (exige mesma população; engana em longitudinal cross-lab).

Endpoint: `GET /health/dashboard/markers/:markerId/evolution?family_member_id=&from=&to=&mode=absolute|normalized`.

## 13. Padrão de UI: popover de regra de negócio no menu (admin)

Regras de negócio complexas e críticas devem ficar **documentadas na própria UI**, no padrão BioPass: ao lado do item de menu, um ícone **ⓘ**; ao passar o mouse, um **popover** com texto limpo, direto, com ícones/imagens evidenciando a regra.

- Componente reutilizável `MenuInfoPopover` (admin) + **registry** de conteúdo por chave de menu (`business-rules.ts`): título, corpo (markdown/JSX), ícones, imagens.
- Primeiras regras a registrar: **dedup de marcadores** (por que não pode duplicar, como resolvemos) e **referência cross-lab** (absoluto vs normalizado).
- Renderiza no hover do ícone ⓘ do item de menu; acessível (focus/teclado), responsivo.

## 14. Estado da implementação
- **Fase 0 (catálogo) — implementada** neste repo: migration `000002_health_catalog`, domínio `internal/domain/health`, serviço `internal/application/health` (dedup + resolve + seed), repo, handlers, rotas `/api/v1/health/markers` (+`/resolve`), seed `cmd/seed`. Fonte da verdade do schema = a migration.
- Próximas fases (1–4) conforme §9.

## 15. Decisões ainda em aberto (não bloqueiam Fase 0)
- Painel de exames (`health_exams`) no MVP ou só marcadores? (sugestão: só marcadores agora)
- Conversão automática de unidades vs só flag de divergência.
- Referência default do catálogo por sexo/idade (fase futura).
- Provider de extração definitivo e prompt versioning.
