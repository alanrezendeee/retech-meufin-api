# MeuFin — Roadmap & Entrega da Apresentação (2026-07-15)

## PRs para merge (nesta ordem)
1. API: https://github.com/alanrezendeee/retech-meufin-api/pull/57
2. Admin: https://github.com/alanrezendeee/retech-meufin-admin/pull/72

## O que foi entregue nesta rodada (branch `feat/presentation-mega`)

### Correção crítica de produção
- **Crash "Dirty database version 27"**: a migration 27 dropava `vehicle_service_order_items` antes de `vehicle_maintenance_schedules`, que tinha FK para ela. Corrigido o arquivo (ordem dos DROPs) e o banco de produção foi normalizado manualmente (version=28, dirty=false). API de produção voltou a responder (`/health` ok).

### Conta demo para a apresentação
- **Login**: `demo@meufin.app` / senha `MeuFin@Demo2026` (role master)
- **Workspace/tenant**: `be5efc4d-5463-4cf4-88b1-a78b4d6d4e90`
- **Família Oliveira**: Carlos (titular), Fernanda (cônjuge), Julia, Pedro e Sofia (filhos), José (avô)
- Dados históricos de **Jan/2025 a Dez/2026** (realizado até 14/07/2026, previsto adiante):
  - 1.200+ lançamentos (salários, aluguel recebido, dividendos, freelas, 13º, férias; despesas fixas com reajuste anual, sazonalidade de luz, IPVA/IPTU/seguros anuais)
  - 6 parcelamentos grandes (geladeira 10x, TV 12x, notebook 12x, sofá 8x, iPhone 12x, ar-condicionado 6x)
  - ~48 faturas de cartão importadas (2 cartões, pai+filhos com data de compra)
  - 56 cupons fiscais com 990 itens — **picanha comprada 48 vezes** com inflação de ~28% a.a. (case do gráfico de inflação por item)
  - Saúde: 2 laboratórios, 15 pedidos + 15 resultados de exame, 90 itens de resultado com evolução (glicada do avô caindo com tratamento, colesterol do Carlos melhorando)
  - Frota: 3 veículos (CR-V, Argo, CG 160), 6 manutenções com itens, 6 agendamentos, 42 meses de histórico FIPE

### Novos módulos (API + Admin)
1. **Aniversariantes + dados corporais** — birth_date/altura/peso nos membros, idade calculada, quadro de aniversariantes no painel principal
2. **Patrimônio** — imóveis (com documentos/escritura PDF) + impostos de bens (IPTU/IPVA/licenciamento/seguros) com dashboard de evolução anual e inflação dos impostos
3. **Garantias** — controle de garantia legal + fabricante + estendida de qualquer bem, com documentos e alertas de vencimento
4. **Educação / Material escolar** — matrículas por membro/ano letivo, listas de material com preço referência × pago, dashboard de custo
5. **Segurança do Lar** — itens de segurança física/química/biológica (gás, caixa d'água, extintor, dedetização...) com validade, periodicidade e histórico de manutenção
6. **Dashboard Fiscal** — inflação pessoal por item comprado, histórico de preço por produto, top compras

## Sugestões para as próximas noites (roadmap)

### Curto prazo
- **Alertas unificados**: central de notificações (e-mail/push/WhatsApp) agregando vencimentos de impostos, garantias, segurança do lar, agendamentos de frota e contas do dia
- **Metas e orçamento por categoria** (budget) com acompanhamento mensal e projeção de estouro
- **Importação automática**: OCR de cupom via foto no celular (hoje o fluxo é PDF), open finance para extratos bancários
- **PDF com senha** nos documentos fiscais (pendência conhecida da extração)
- **Migration no retech-auth-api** para `user_applications.tenant_id` (drift corrigido só em prod)

### Médio prazo
- **Inflação pessoal vs IPCA**: comparar o índice pessoal da família com o IPCA oficial (API do IBGE/SIDRA é pública) — argumento forte de venda
- **Preços de material escolar**: Procon-SP publica pesquisa anual em planilha; importar como referência e alertar abuso (não há API estável; avaliar scraping/planilha)
- **Assistente IA**: "quanto gastei com churrasco em 2025?" — RAG sobre os lançamentos (retech-meufin-rag)
- **Renegociação/rotativo** (fase 3 do modelo de liquidação)
- **Compartilhamento por membro**: login individual por membro da família com permissões (adolescente vê só a mesada)

### Longo prazo
- **Marketplace de seguros/garantia estendida** (monetização por lead)
- **Score de saúde financeira familiar** (índice proprietário — vendável para bancos)
- **App mobile** (React Native reaproveitando o design system)

## Pendências operacionais pós-merge
1. Merge das PRs (API e Admin) → deploy Railway. Migrations 29–33 JÁ aplicadas em prod (schema_migrations em 33) — deploy não re-roda nada
2. Boot da API sincroniza o manifesto de permissions novo (patrimony.*, warranties.*, education.*, homesafety.*, finance.fiscal-dashboard) com o auth
3. Validar login demo e navegação em todas as áreas novas
