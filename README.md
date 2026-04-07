# ReTechFin API

API REST em Go para o SaaS de gestão financeira familiar (ledger, categorias, contas, orçamentos). Arquitetura em camadas com **DDD**, **SOLID** e **Clean Architecture**.

## Por que GORM (e não sqlc)?

- **GORM** encaixa bem na fase atual: agregados com CRUD, evolução de schema via migrations SQL versionadas, mapeamento explícito modelo persistência ↔ entidade de domínio nos repositórios.
- **sqlc** é excelente quando a maior parte do acesso é SQL fixo e você quer tipagem gerada em tempo de compilação; para um domínio financeiro que tende a ganhar relatórios e regras complexas, podemos introduzir leituras otimizadas com sqlc **só nas queries quentes**, mantendo comandos em GORM — decisão documentada para revisão futura.

## Estrutura de pastas

| Caminho | Papel |
|--------|--------|
| `cmd/api` | Ponto de entrada, composição (DI manual), subida do servidor |
| `configs` | Carregamento estrito de variáveis de ambiente |
| `internal/domain` | Entidades, invariantes, erros e **ports** (interfaces de repositório) |
| `internal/application` | Casos de uso / serviços de aplicação, orquestração, DTOs de entrada |
| `internal/infrastructure/persistence` | GORM, Postgres, migrations (`golang-migrate`), adaptadores dos repositórios |
| `internal/interfaces/http` | Gin: rotas, middlewares, handlers (DTOs JSON), mapeamento HTTP ↔ aplicação |
| `migrations` | SQL up/down (obrigatório; sem `AutoMigrate` em produção) |
| `pkg/logger` | Logger estruturado (`log/slog`) |

**Regra:** `domain` não importa `infrastructure` nem `gin`.

## Variáveis de ambiente

Todas são **obrigatórias**. Se faltar qualquer uma, o processo encerra com erro.

Veja `.env.example`.

## Rodar com Docker Compose

As variáveis necessárias já estão definidas no `docker-compose.yml` para desenvolvimento local.

```bash
docker compose up --build
```

- API: `http://localhost:8080`
- Postgres: `localhost:5432`

Health: `GET http://localhost:8080/health`

## Rodar manualmente (sem Docker da API)

1. Suba um Postgres 16+ e crie o banco.
2. Copie e ajuste `.env` (incluindo `MIGRATIONS_PATH` apontando para `./migrations`).
3. Instale dependências e execute:

```bash
go mod download
set -a && source .env && set +a   # ou export manual de cada variável
go run ./cmd/api
```

Migrations rodam automaticamente na subida do processo.

## Railway / Docker em produção

- Faça deploy **da imagem Docker** (mesmo `Dockerfile`).
- Configure o banco **externo** no painel e defina **todas** as variáveis (equivalentes a `DB_*`, `APP_*`, `LOG_LEVEL`, `MIGRATIONS_PATH`).
- `MIGRATIONS_PATH` na imagem padrão: `/app/migrations` (já copiado no build).
- `DB_SSLMODE` geralmente `require` em provedores gerenciados.

## API (`/api/v1`)

Todas as rotas versionadas exigem o header:

```http
X-Workspace-ID: <uuid>
```

Representa o isolamento lógico do workspace (família/organização). Autenticação JWT pode popular esse header via API gateway no futuro.

### Endpoints principais

| Método | Caminho | Descrição |
|--------|---------|-----------|
| `GET` | `/health` | Saúde do processo |
| `POST` | `/api/v1/accounts` | Criar conta financeira |
| `GET` | `/api/v1/accounts` | Listar (paginação `?limit=&offset=`) |
| `GET` | `/api/v1/accounts/:id` | Obter |
| `PUT` | `/api/v1/accounts/:id` | Atualizar |
| `DELETE` | `/api/v1/accounts/:id` | Remover |
| `POST` | `/api/v1/categories` | Criar categoria (`kind`: `income` \| `expense`) |
| `GET` | `/api/v1/categories` | Listar |
| … | … | CRUD idem |
| `POST` | `/api/v1/transactions` | **Criar lançamento** (`flow`: `in` \| `out`, `amount_cents` > 0) |
| `GET` | `/api/v1/transactions` | **Listar lançamentos** |
| `POST` | `/api/v1/budgets` | **Criar orçamento** (só para categoria `expense`) |
| `POST` | `/api/v1/budgets/validate` | **Validar estouro** no mês (`year`, `month` no JSON) |
| … | CRUD `/api/v1/budgets` | Demais operações |

### Exemplos

**Criar conta**

```bash
curl -s -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -H "X-Workspace-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{"name":"Conta principal","currency":"BRL"}'
```

**Criar categoria de despesa**

```bash
curl -s -X POST http://localhost:8080/api/v1/categories \
  -H "Content-Type: application/json" \
  -H "X-Workspace-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{"name":"Moradia","kind":"expense"}'
```

**Criar lançamento (saída)**

```bash
curl -s -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -H "X-Workspace-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "account_id":"<UUID-da-conta>",
    "category_id":"<UUID-da-categoria-expense>",
    "amount_cents":15000,
    "flow":"out",
    "description":"Aluguel",
    "occurred_at":"2026-04-01T10:00:00Z"
  }'
```

**Criar orçamento mensal**

```bash
curl -s -X POST http://localhost:8080/api/v1/budgets \
  -H "Content-Type: application/json" \
  -H "X-Workspace-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "category_id":"<UUID-da-categoria-expense>",
    "year":2026,
    "month":4,
    "limit_cents":300000
  }'
```

**Validar orçamento**

```bash
curl -s -X POST http://localhost:8080/api/v1/budgets/validate \
  -H "Content-Type: application/json" \
  -H "X-Workspace-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{"year":2026,"month":4}'
```

Resposta inclui `lines[]` com `limit_cents`, `spent_cents`, `over_budget`, `remaining_cents` por categoria orçada.

## Erros

Envelope padrão:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "...",
    "request_id": "..."
  }
}
```

## Testes

```bash
go test ./...
```

## Licença

Proprietário — The Retech / ReTechFin.
