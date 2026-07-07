# Health Share Links — Links Públicos Temporários de Saúde

## Visão Geral

Permite ao usuário gerar um link público com prazo de validade para compartilhar resultados de exames com médicos, via WhatsApp, e-mail ou cópia direta. O destinatário acessa os dados sem precisar criar conta ou fazer login.

**Motivação:** O fluxo atual obriga o médico a ter acesso ao sistema para visualizar histórico do paciente. Links temporários eliminam essa fricção mantendo controle e segurança — o link expira, pode ser revogado a qualquer momento e não expõe outros dados do workspace.

---

## Escopo do MVP

### O que pode ser compartilhado

| Escopo (`scope`) | Descrição |
|---|---|
| `exam_result` | Um resultado de exame específico (todos os itens/marcadores) |
| `member_results` | Todos os resultados de exames de um membro (últimos 12 meses) |
| `exam_request` | Uma solicitação de exame específica |

### O que fica fora do MVP
- Compartilhamento de documentos (PDFs, imagens)
- Proteção por senha
- Notificação quando o link é acessado
- Links permanentes (sem expiração)

---

## Arquitetura

### Backend (retech-meufin-api)

#### Tabela: `health_share_links`

```sql
CREATE TABLE health_share_links (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID         NOT NULL REFERENCES workspaces(id),
    token        VARCHAR(64)  NOT NULL UNIQUE,          -- hex de 32 bytes aleatórios (256 bits)
    scope        VARCHAR(32)  NOT NULL,                  -- 'exam_result' | 'member_results' | 'exam_request'
    resource_id  UUID,                                   -- ID do ExamResult ou ExamRequest (NULL = member_results)
    family_member_id UUID REFERENCES health_family_members(id),
    title        VARCHAR(255),                           -- título opcional para identificação
    expires_at   TIMESTAMPTZ  NOT NULL,
    view_count   INT          NOT NULL DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    created_by   UUID         NOT NULL,                  -- user_id
    revoked_at   TIMESTAMPTZ,                            -- revogação manual
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_health_share_links_token
    ON health_share_links(token)
    WHERE deleted_at IS NULL AND revoked_at IS NULL;

CREATE INDEX idx_health_share_links_workspace
    ON health_share_links(workspace_id, expires_at)
    WHERE deleted_at IS NULL;
```

#### Geração do token

```go
// 32 bytes aleatórios via crypto/rand → hex string de 64 chars
func generateToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

256 bits de entropia — impossível de adivinhar por força bruta.

#### Endpoints autenticados (workspace)

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/api/v1/health/share-links` | Cria link |
| `GET` | `/api/v1/health/share-links` | Lista links ativos do workspace |
| `DELETE` | `/api/v1/health/share-links/:id` | Revoga link |

**POST body:**
```json
{
  "scope": "exam_result",
  "resource_id": "uuid-do-exam-result",
  "family_member_id": "uuid-do-membro",
  "title": "Resultados para Dr. João Silva",
  "expires_in_hours": 48
}
```

**Opções de `expires_in_hours`:** `1`, `6`, `12`, `24`, `48`, `72`, `168` (7 dias)

**POST response:**
```json
{
  "id": "uuid",
  "token": "a3f7...",
  "url": "https://meufin.com.br/compartilhado/a3f7...",
  "scope": "exam_result",
  "title": "Resultados para Dr. João Silva",
  "expires_at": "2026-07-09T14:00:00Z",
  "expires_in_hours": 48,
  "created_at": "2026-07-07T14:00:00Z"
}
```

#### Endpoint público (sem autenticação)

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/api/v1/public/health/:token` | Retorna dados do escopo |

**Comportamentos:**
- Token inexistente → `404 Not Found`
- Token expirado ou revogado → `410 Gone` com `{ "error": "link_expired" }`
- Token válido → `200 OK` com dados + incrementa `view_count` + atualiza `last_viewed_at`

**Response (scope = exam_result):**
```json
{
  "link": {
    "title": "Resultados para Dr. João Silva",
    "scope": "exam_result",
    "expires_at": "2026-07-09T14:00:00Z",
    "member_name": "Ana"
  },
  "exam_result": {
    "exam_date": "2026-07-01",
    "lab_name": "Fleury",
    "status": "completed",
    "items": [...]
  }
}
```

> **Privacidade:** o campo `member_name` expõe apenas o primeiro nome, nunca sobrenome, CPF, data de nascimento ou outros dados sensíveis do workspace.

**Rate limiting no endpoint público:** 30 req/min por IP.

#### Limpeza de links expirados

Job agendado (ou middleware lazy) remove links com `expires_at < now()` periodicamente. Alternativamente, a query de busca por token já filtra `expires_at > now()` e `revoked_at IS NULL` — links expirados simplesmente retornam 410 sem precisar deletar.

---

### Frontend Admin (retech-meufin-admin)

#### Fluxo de criação

1. Na tela **Resultados de Exames**, cada linha tem botão `Compartilhar` (ícone `ShareRounded`)
2. Abre dialog `ShareLinkDialog`:
   - Campo **Título** (ex.: "Para Dr. João Silva") — opcional
   - Selector **Validade**: 1h / 6h / 12h / 24h / 48h / 7 dias
   - Checkbox (opcional v2): incluir todos os resultados do membro
3. Ao confirmar → POST `/api/v1/health/share-links`
4. Dialog muda para tela de sucesso:
   - Input read-only com a URL
   - Botão **Copiar link**
   - Botão **WhatsApp** → abre `https://wa.me/?text=Encaminho+meus+resultados:+{url}`
   - Botão **E-mail** → abre `mailto:?subject=Resultados+de+Exames&body={url}`
   - Aviso: "Este link expira em 48 horas"

#### Tela de gerenciamento

Seção ou aba **Links Compartilhados** (dentro da tela de resultados ou como submenu separado):

| Coluna | Detalhe |
|---|---|
| Título | Nome do link |
| Escopo | "Resultado", "Todos do membro", "Solicitação" |
| Expiração | Timestamp formatado + badge "Expirado" se vencido |
| Visualizações | Contador de acessos |
| Ações | Copiar URL / Revogar |

Botão **Revogar** → `DELETE /api/v1/health/share-links/:id` → link some da lista.

#### Página pública (rota sem auth)

Rota separada no frontend: `/compartilhado/:token`

- **Header simples:** logo MeuFin + "Dados compartilhados pelo paciente"
- **Badge de validade:** "Expira em 23h 42min" ou "Link expirado" (estado de erro)
- **Conteúdo:** readonly dos dados do escopo — igual à tela interna mas sem ações
- **Footer:** "Para ver seu histórico completo, acesse meufin.com.br"
- **Sem navbar, sem login prompt**

Estados da página:
- `loading` — skeleton
- `valid` — conteúdo exibido
- `expired` — card de erro "Este link expirou ou foi revogado. Solicite um novo link ao paciente."
- `not_found` — card genérico (não revelar se token nunca existiu vs. expirado)

---

## Segurança

| Risco | Mitigação |
|---|---|
| Adivinhação de token | 256 bits de entropia (crypto/rand) |
| Token válido mas compartilhado indevidamente | Expiração curta (padrão 24h) + revogação manual |
| Exposição de dados sensíveis | Apenas primeiro nome do membro; sem workspace, CPF, email |
| Abuso do endpoint público | Rate limiting 30 req/min por IP |
| Enumeração de tokens expirados vs. inválidos | Ambos retornam 410 (mesmo código) |
| Logs de acesso | `view_count` + `last_viewed_at` auditáveis pelo dono |

---

## Implementação — Ordem de Entrega

### Sprint 1 — Backend (API)
1. Migration: tabela `health_share_links`
2. Domain: `ShareLink` struct, `ShareLinkRepository` interface
3. Application: `ShareLinkService` (Create, List, Revoke, GetByToken)
4. Infrastructure: `sharelink_repo.go`
5. Handler: `ShareLinkHandler` (endpoints autenticados + público)
6. Router: registrar rotas (autenticadas em `/api/v1/health/`, pública em `/api/v1/public/`)

### Sprint 2 — Frontend Admin
1. Tipos em `src/features/health/api.ts` (`ShareLink`, `CreateShareLinkInput`)
2. `ShareLinkDialog` — fluxo criar → sucesso com copiar/WhatsApp/e-mail
3. Botão "Compartilhar" na tabela de `ExamResultsPage`
4. Seção "Links Compartilhados" em `ExamResultsPage`

### Sprint 3 — Página Pública
1. Rota `/compartilhado/:token` (fora do layout de dashboard)
2. Componente `SharedHealthPage` — estados: loading / valid / expired / not_found
3. Renderização dos escopos: `exam_result`, `member_results`, `exam_request`

---

## Dependências e Pré-requisitos

- Tabela `health_family_members`, `health_exam_results`, `health_exam_requests` — já existem
- Middleware de rate limiting no router — a confirmar se já existe no projeto
- Domínio público configurado (ex.: `meufin.com.br`) para montar a URL do link
- Variável de ambiente: `PUBLIC_BASE_URL` (ex.: `https://meufin.com.br`)

---

## Métricas de Sucesso

- Links criados por mês
- Taxa de acesso (links criados vs. acessados ao menos 1x)
- Duração média selecionada
- Canais de compartilhamento (WhatsApp vs. e-mail vs. cópia direta) — via evento de analytics no click

---

## Decisões

1. **Nome do membro:** expor o nome completo do membro na página pública (não apenas iniciais).
2. **Notificação de acesso:** sim — notificar o dono do workspace quando o link for acessado pela primeira vez (canal a definir: push ou e-mail).
3. **Limite de links ativos por workspace:** máximo de **50 links ativos** por tenant. Será necessária uma área de gerenciamento para: listar links ativos, ver contagem de acessos, revogar links manualmente e filtrar por status (ativo/expirado/revogado). Implementar durante Sprint 2 (frontend admin).
4. **URL pública:** domínio do próprio app, via variável de ambiente `PUBLIC_BASE_URL` obrigatória (padrão do projeto). A API monta a URL completa no response de criação do link.

## Open Questions

1. **WhatsApp Business API:** integração futura para envio programático, não só link deep?
