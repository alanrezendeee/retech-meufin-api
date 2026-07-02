# Módulo Financeiro — Roadmap pré-dash + Dashboard v1.1

> Definido em 2026-07-01. Ordem sugerida: cadastros base → contas do dia → liquidação → docs de membro → dashboard. A dash fica por último porque a liquidação muda a semântica de "realizado" (valor pago vs previsto).

## Ordem de implementação

1. Cadastros base: contas + formas de pagamento
2. Tela "Contas do dia" (vencendo hoje / em atraso)
3. Liquidação com forma de pagamento + comprovante
4. Documentos de membros da família
5. Dashboard v1.1

---

## 1. Contas e formas de pagamento

**Cadastro de contas** — nova tabela `finance_accounts` (CRUD simples, padrão `income_sources`):
- `name`, `bank_name?`, `kind` (corrente | poupança | carteira/dinheiro | digital), `notes?`
- Endpoints: `GET/POST/PUT/DELETE /finance/accounts`

**Forma de pagamento** — não precisa de tabela própria; é um enum + referências:
- `pix`, `debito`, `transferencia`, `boleto`, `dinheiro`, `cartao_credito`
- pix/débito/transferência/boleto → apontam para `account_id`
- `cartao_credito` → aponta para `card_id` (tabela `credit_cards` já existe)

**Modelagem da liquidação**: colunas no próprio `financial_entries` (relação 1:1, simples):
`paid_at`, `paid_amount_cents`, `payment_method`, `payment_account_id?`, `payment_card_id?`.
Tabela separada `financial_settlements` só se um dia tiver pagamento parcial (v2).

⚠️ Decisão em aberto: pagar uma despesa **com cartão de crédito** na prática deveria virar item da fatura futura do cartão (senão duplica quando a fatura chegar). v1: só registra a forma e alerta o usuário; automatizar na v2.

## 2. Tela "Contas do dia"

- Backend: reusar `GET /finance/entries` com filtros novos no `FinancialEntryFilter`:
  - `due_before` / `due_on` (vencendo hoje) e `overdue=true` (due_date < hoje AND status=prevista)
- Front: tela "A pagar" com 3 seções — **Em atraso** / **Hoje** / **Próximos 7 dias** — cada linha com ação "Liquidar"
- Mesma tela serve receitas ("A receber hoje") trocando `kind` — mesmo componente

## 3. Liquidação (settle)

- `POST /finance/entries/:id/settle` — evolução do `confirm` atual (que só troca status):
  ```json
  { "paid_at": "...", "paid_amount_cents": 12345, "payment_method": "pix", "account_id": "...", "card_id": null, "notes": "..." }
  ```
- `paid_amount_cents` default = `amount_cents`; pode diferir (juros/multa/desconto) — **guardar os dois**
- Status vira `realizada`. `confirm` continua existindo (liquidação rápida sem detalhes)
- **Comprovante**: reusar infra `finance_documents` + MinIO (`internal/infrastructure/storage/minio.go`):
  - `POST /finance/entries/:id/receipt` (multipart) — pdf/jpg/png/heic; múltiplos permitidos
  - `GET .../receipts` + download-url assinada (padrão já existe nos documents)
  - Migration: `entry_id` nullable em `finance_documents` (ou kind `receipt`)
- Migration: colunas de settlement em `financial_entries` + índice se precisar
- ⚠️ Decisão em aberto: liquidar **fatura pai** liquida os filhos em cascata? (sugestão: sim)

## 4. Documentos de membros da família

- Nova tabela `family_member_documents` (padrão `finance_documents`, storage MinIO):
  - `family_member_id` (FK `health_family_members`), `doc_type`, `label?`, `doc_number?`, `valid_until?`, `notes?`, arquivo
- `doc_type` catálogo: `cpf`, `rg`, `cnh`, `passaporte`, `carteira_trabalho`, `certidao_nascimento`, `titulo_eleitor`, `cartao_sus`, `plano_saude`, `outro` (com label livre)
- `valid_until` importa: CNH e passaporte vencem → alerta futuro
- Endpoints: `POST/GET/DELETE /health/family-members/:id/documents` (+ download-url)
- Upload de quantos documentos quiser, qualquer formato comum (pdf/jpg/png/heic/doc)
- Front: aba "Documentos" dentro do cadastro/edição do membro

## 5. Dashboard v1.1

Princípio: 4 perguntas — Como estou este mês? / O que ainda vem? / Pra onde foi o dinheiro? / Quanto do futuro já está comprometido?

**Layout** (uma tela, seletor mês/ano + filtro membro):
1. 3 cards do mês: Receitas (real/prev), Despesas (real/prev), Saldo (real + prev)
2. 2 cards pendência: A receber, A pagar
3. Gráfico anual: barras receita×despesa por mês (split realizado/previsto) + linha de saldo
4. Despesas por categoria (barras horizontais, maior→menor)
5. Faixa "Parcelas futuras": total comprometido nos próximos meses

**Regras de agregação** (críticas — fatura pai/filho duplica valor):
- Totais/cards/série anual: só `parent_id IS NULL`
- Categorias: só folhas — exclui pai que tem filhos (senão tudo vira "cartao"); pode divergir do total se amount da fatura foi sobrescrito (aceito)
- Sempre `status != 'cancelada'` e `deleted_at IS NULL`
- realizado = `realizada` (**usar `paid_amount_cents` quando existir**, senão `amount_cents` — integra com a liquidação acima); previsto = `prevista`+`realizada`; a pagar/receber = `prevista` do mês
- Mês: `due_date >= dia1 AND < dia1_mês_seguinte`
- Parcelas futuras: `prevista` débito com `installment_number NOT NULL` e vencimento após o mês (parcelas manuais já materializam; de fatura importada não — limitação v1)
- Filtro membro: fatura/filhos sem `family_member_id` somem do filtro — limitação v1

**Backend** (2 endpoints, ~4 queries, índices existentes bastam — zero migration):
- `GET /finance/dashboard?year=&month=&family_member_id=` → cards + pendências + categorias + parcelas futuras (`SUM(...) FILTER (WHERE ...)` num scan só)
- `GET /finance/dashboard/monthly?year=` → 12 meses com realizado×previsto separados
- Valores em cents (int64). Arquivos espelham dashboard da Saúde: `domain/finance/dashboard.go`, `persistence/finance_dashboard_repo.go`, `application/finance/dashboard_service.go`, `handlers/finance_dashboard_handler.go`, rotas + wire no `main.go`

**Front**: página em `src/features/finance/pages/` (recharts + react-query, padrão HealthDashboardPage), rota `financeiro` guarded (subject `finance.dashboard` — registrar no catálogo do auth), item no menu.

---

## Sugestões extras (avaliar)

- **Liquidar receitas** no mesmo fluxo (recebimento + comprovante/recibo) — quase de graça, mesmo componente
- **Alerta de documento vencendo** (CNH/passaporte via `valid_until`) — card na dash ou badge no menu (v2)
- **Saldo/extrato por conta** (v2): liquidação registrando `account_id` já deixa os dados prontos
- **Cascata na fatura**: liquidar pai → filhos `realizada` juntos
- **Juros/multa visíveis**: diferença `paid_amount − amount` pode aparecer como linha "custo de atraso" na dash (v2)
- **Pagamento parcial** (v2): exigiria tabela `financial_settlements`
- **Conciliação OFX/extrato bancário** (v3)
- **Notificação de vencimento** push/email (v2 — precisa de scheduler)

## Decisões em aberto (resolver antes de codar)

1. Liquidar fatura pai liquida filhos em cascata? (sugestão: sim)
2. Dash usa valor pago ou previsto no "realizado"? (sugestão: pago quando existir)
3. Pagar despesa com cartão de crédito vira item de fatura futura? (sugestão: v1 só registra, v2 automatiza)
4. Permissões: um subject por tela (`finance.accounts`, `finance.payables`, `finance.dashboard`) ou reusar `finance.entries`?
