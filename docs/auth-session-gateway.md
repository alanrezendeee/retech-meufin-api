# Gateway de sessão (token handler / BFF)

## Problema

Até aqui o admin fazia login direto no `retech-auth-api` e guardava `access_token`
(24h) **e** `refresh_token` (7 dias) em `localStorage`. Qualquer XSS no admin lia os
dois e tinha uma semana de acesso — o auth não tem revogação de refresh nem logout.
Reduzir `JWT_EXPIRATION_HOURS` sozinho não resolvia: o refresh continuava no browser.

## Solução

A API vira a borda de autenticação do browser (mesmo padrão do CashFlowfy):

```
browser ──(email/senha)──▶ POST /api/v1/auth/login ──▶ retech-auth-api /v1/authenticate
        ◀── Set-Cookie: meufin_session=<256 bits aleatórios>; HttpOnly; Secure; SameSite=Lax
                                   │
                                   └─ auth_sessions: sha256(token) → access/refresh cifrados (AES-256-GCM)

browser ──(cookie)──▶ /api/v1/**  ──▶ RequireAuth: cookie → sessão → access token
                                        ├─ expira em < 30s? POST /v1/refresh no auth, grava tokens novos
                                        └─ valida JWT via JWKS (igual ao Bearer) → tenant_id → workspace
```

- **Nenhum JWT chega ao JavaScript.** O cookie é um id opaco; vale só com o banco da API.
- **Refresh é server-side.** O access token do auth pode ficar curto (15 min) sem deslogar ninguém.
- **Logout real.** `POST /auth/logout` revoga a sessão no banco; o cookie morre na hora.
- **`Authorization: Bearer`** só durante o rollout (`AUTH_BEARER_ENABLED=true`). Depois, `false`:
  a API passa a aceitar **apenas** o cookie e `Authorization` sai do CORS — idêntico ao CashFlowfy.
  Integrações futuras devem usar credencial própria (API key / client credentials), não o JWT de usuário.
- **Proxy IAM.** As telas de usuários/roles/permissions falam com o auth via
  `/api/v1/iam/v1/*` — a API injeta o Bearer da sessão. Allowlist: `users`, `roles`, `permissions`.

## CSRF

Cookie `SameSite=Lax` já impede o browser de enviar o cookie em POST cross-site. Como
defesa em profundidade, `RequireAuth` recusa (403) requisições por cookie que mudam
estado quando `Sec-Fetch-Site: cross-site` ou `Origin` fora de `CORS_ALLOWED_ORIGINS`
e diferente do host da API. `GET/HEAD/OPTIONS` e Bearer não passam por isso.

## Variáveis

| Var | Padrão | Nota |
|-----|--------|------|
| `SESSION_ENCRYPTION_KEY` | — | base64 de 32 bytes (`openssl rand -base64 32`). Vazia = gateway desligado. **Obrigatória em produção.** |
| `SESSION_COOKIE_NAME` | `meufin_session` | Produção: `__Host-meufin_session` (exige Secure e sem Domain; bloqueia injeção por subdomínio). |
| `SESSION_COOKIE_SECURE` | `true` | `false` só em dev http. Produção falha no boot se `false`. |
| `SESSION_COOKIE_DOMAIN` | vazio | Vazio = host da API. Só preencher se admin e API estiverem em subdomínios diferentes sem proxy. |
| `SESSION_TTL_HOURS` | `12` | Validade absoluta da sessão (igual ao CashFlowfy). Precisa caber no `JWT_REFRESH_EXPIRATION_HOURS` do auth. |
| `AUTH_BEARER_ENABLED` | `true` | `false` após o rollout do admin: só cookie. |
| `APP_APPLICATION_CODE` | `meufin` | `application_code` do `/v1/authenticate`. |
| `AUTH_API_BASE_URL` | — | Já existia (esqueci a senha). Agora também alimenta login/refresh/me e o proxy IAM. |

## Deploy same-origin (recomendado)

O admin (nginx) faz `proxy_pass` de `/api/` para a API. Browser só enxerga uma origem:
sem CORS, sem `Domain` no cookie, `SameSite=Lax` funciona por construção. No admin,
`API_UPSTREAM` é a URL da API (Railway: preferir a URL interna `http://<serviço>.railway.internal:<porta>`).

Alternativa cross-origin (admin e API em domínios distintos): `CORS_ALLOWED_ORIGINS`
com a origem do admin (já emite `Allow-Credentials: true`), `SESSION_COOKIE_DOMAIN`
apontando para o domínio pai comum (`.meufin.app`) e `VITE_API_BASE_URL` no admin.
Se não houver domínio pai comum, `SameSite=Lax` bloqueia o cookie — use same-origin.

## Rollout

1. API: definir `SESSION_ENCRYPTION_KEY` (+ `SESSION_COOKIE_SECURE=true`) e deployar. Nada
   quebra: Bearer continua funcionando, o admin antigo segue logando direto no auth.
2. Admin: deployar a versão com cookie + `API_UPSTREAM`. Usuários logados caem para a
   tela de login uma vez (a sessão antiga em `localStorage` é descartada).
3. API: `AUTH_BEARER_ENABLED=false`. Fecha a porta Bearer; só cookie.
4. Auth central: `JWT_EXPIRATION_MINUTES=15`. Pré-requisito: **todo** consumidor renova
   server-side. Hoje só CashFlowfy e MeuFin usam o auth; os dois renovam no servidor.

## Headers e IP real

- Toda resposta sai com `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`
  e `Cache-Control: no-store` (igual ao CashFlowfy).
- Rate limit por IP usa o **último hop** de `X-Forwarded-For` (anexado pelo proxy à frente,
  não controlado pelo cliente). O nginx do admin repassa o XFF do edge sem anexar o seu.

## Operação

- Sessões expiradas/revogadas são apagadas após 7 dias (job a cada 6h) — ficam para auditoria (`ip`, `user_agent`, `last_seen_at`).
- Rotação de `SESSION_ENCRYPTION_KEY` invalida todas as sessões (tokens não decifram → 401 → cookie limpo → relogin). Aceitável; fazer fora de horário de pico.
- `Service.RevokeAllForUser` existe para troca de senha/bloqueio; ainda não exposto em rota.
