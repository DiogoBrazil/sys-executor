# Relatório — novo visual da interface ("capa dos autos refinada")

- **Pedido do usuário (10/11/2026):** as telas estão bagunçadas. Ele quer um visual moderno, agradável e fácil de usar, com animações suaves.
- **Plano aprovado:** `~/.claude/plans/delightful-nibbling-sunrise.md`, em 3 fases.
- **Decisões:**
  - manter a identidade (pasta, aba bordô no PADS e azul-petróleo na SR), sem tema escuro;
  - no processamento, etapas à esquerda e abas (Análise, Suas respostas, Autos);
  - ajustes funcionais P1, P2, P3 e itens 1 e 2 na Fase 1;
  - galeria local com capturas sem GPU.
- **Restrições:**
  - sem dependência npm nova (continua com 40 npm);
  - `data-teste` preservados;
  - sem mudança em regras jurídicas, golden ou V4.

## Fase 1: fundação, galeria e ajustes pendentes
- **Base visual (`frontend/src/estilo.css`):**
  - tokens de cor, forma (raios e sombras em dois níveis) e movimento (`--dur-*`, `--ease`);
  - `--acento` definido pelo `App.vue` conforme o tipo da tarefa;
  - estilos globais de campos, botões (`.primario` na cor do procedimento, `.icone`), `.cartao`, `.chip`, `.selo` e callouts com ícone;
  - transições `folha` e `esmaecer`;
  - `prefers-reduced-motion` respeitado.
  - Saíram os estilos de campo duplicados em `CartaoPergunta`, `Montagem`, `CampoBusca`, `ConfigGemini`, `EditorEnquadramento`, `PainelSEI` e `VisualizadorFontes`.
- **Componentes `components/ui/`:**
  - `Icone.vue`: 20 ícones SVG embutidos;
  - `Recolhivel.vue`: altura animada e conteúdo mantido no DOM;
  - `Abas.vue`: `tablist` com setas do teclado, sublinhado deslizante e painéis em `v-show`.
  - `AvisoErro.vue` passou a ser callout com ícone.
- **Galeria de telas (só desenvolvimento; não entra no `.exe`):**
  - arquivos: `frontend/galeria.html`, `src/galeria/` (API falsa, cenários sintéticos, entrada) e `vite.galeria.config.ts`;
  - para abrir: `npm run galeria`, com a tela escolhida por `#cenario`.
  - O `index.html`, o `vite.config.ts` e o `embed.go` não mudaram.
- **Ajustes funcionais:**
  - **P1:** a conclusão da SR leva o sindicado no título ("Conclusão sobre o fato F-1: …"), e "Suas respostas" distingue as duas.
  - **P2:**
    - orientação na seção Ofendidos;
    - aviso quando o nome parece instituição (Administração, Estado, Fazenda, Polícia…);
    - na conferência, a opção "Menor de 13 anos" fica escondida para esses nomes.
  - **P3:**
    - depoente com o mesmo nome de pessoa já cadastrada recebe "Esta pessoa já está cadastrada como …" com o botão **Usar a mesma**, que vincula e remove a pessoa duplicada;
    - na conferência, avisos (que não impedem concluir) para sindicado com outro papel no depoimento e para pessoa cadastrada mais de uma vez.
  - **Item 1:** o acusado do PADS vem com o posto do cadastro também quando a IA não lê nenhum acusado.
  - **Item 2:** grupo sem conteúdo nem campo visível não aparece ("Transgressão anterior 1…5" antes do "Sim").
- **Verificação:**
  - vitest 35/35 (3 testes novos) e `vue-tsc` + `vite build`;
  - `go test` em pipeline e workflow (teste novo do item 1), `GOOS=windows go vet`, `rulescov -estrito`;
  - licenças com 45 módulos Go e 40 npm; golden sem diff;
  - capturas da galeria revisadas: cadastro SR, configurações e entrada, sem regressão.

## Fases 2 e 3, adaptadas aos ajustes de tela do usuário (11/11/2026)
- **Pedido** (capturas do PADS): telas longas na vertical; ocupar mais a horizontal, mantendo a responsividade; tirar da tela principal o que alonga a página (modal ou painel lateral); loader no PDF; figura animada na espera (robô para a IA, engrenagem para o trabalho local); animação na etapa ativa. Ícones em SVG próprio (o IconScout foi descartado pela licença e pela dependência). Tudo numa entrega.
- **Diferença em relação ao plano:** em vez das abas Análise / Suas respostas / Autos, botões no topo que abrem **gavetas** (Análise, Suas respostas) e **modais** (Autos lidos); a memória de cálculo, as explicações longas e os textos de ajuda do cadastro também abrem em modal; a prévia da minuta tem rolagem própria e tela cheia.
- **Componentes novos:** `ui/Janela.vue` (modal/gaveta), `ui/Figura.vue` (robô e engrenagens), `ui/Carregando.vue` (loader).
- **Telas alteradas:** `Processamento`, `CartaoPergunta`, `PainelAnalise`, `PainelAutos`, `MinutaPrevia`, `VisualizadorFontes`, `Montagem`, `Entrada`, `ArquivosPeca`, `EditorEnquadramento`; `estilo.css` (barra de ação, botão de ajuda, trava de rolagem). O detalhe está no guia, §5.3.
- **Não feito:** Início, `PainelSEI`, Configurações de IA e `FaixaTipo` (não foram citados no pedido).
- **Verificação:** vitest 40/40 (3 novos: gavetas e modal dos autos, robô × engrenagens, memória de cálculo em modal), `vue-tsc` + `vite build`, licenças com 40 npm, golden sem diff. Capturas da galeria (sem GPU) revisadas em 1440 e 900 px; corrigidos na revisão: grupos com buraco (alvenaria), opções empilhadas, campos estreitos nos depoimentos e texto curto truncado.
- **Situação:** superada em parte pelo protótipo aprovado (abaixo); o robô da espera e a minuta desta rodada foram aprovados pelo usuário e ficaram.

## Protótipo aprovado (11/11/2026)
- **Como:** protótipo de 10 telas num canvas de design privado do usuário no claude.ai, com dados sintéticos, revisado em três rodadas e aprovado. Depois, implementado no Vue.
- **Entrada:** trilha "Tipo › Origem dos autos › Processamento", dois cartões com ícone, vantagens e botão de ação, nota de cifra. "Escolher outro caminho" e "Trocar o tipo de minuta" ficam no rodapé (`Entrada.vue`).
- **Cadastro** (`Montagem.vue`): "Cadastre as peças do **processo**" (na SR, "do procedimento"); barra "x de y peças concluídas"; **cartões das peças** com ✓ verde, "Pendente" na próxima e "Opcional" em Outros; o cartão "Conferir e processar" leva à conferência. **Cada peça abre num modal** (`Janela`) com Anterior/Continuar. Depoimentos e outros documentos em **linhas** (papel em controle segmentado, posto, nome, data, PDF compacto). Conferência com peças e envolvidos em tabela e o painel "Antes de processar / O cadastro está completo". **Loader no anexo** (`ArquivosPeca`: linha "Anexando PDF…" e selo "Carregando PDF…").
- **Processamento** (`Processamento.vue`): **trilha horizontal** das etapas (✓ desenhado; arco girando quando o app trabalha; pulso e "sua vez" quando espera o usuário; "parou" em parada com erro), com Análise, Suas respostas, Autos lidos, Consultar os autos e Cancelar tarefa na mesma faixa. Cartão de parada com "O que fazer" e código; tela final centralizada com três blocos e "Iniciar outra minuta".
- **Perguntas** (`CartaoPergunta.vue`, novo `CampoPergunta.vue`): campos com borda mais forte (1,5 px, 44 px) e escolhas em **controle segmentado** (ou cartões de escolha, se longas); **imputações em tabela** (Imputação | Conduta | Dispositivo | Data | Incluir), com o acusado ao lado do título; grupos numerados ("Transgressão anterior n") em **linhas acrescentadas uma a uma** ("Acrescentar transgressão", até 5), com a pergunta de controle antes; pergunta curta numa linha; **dosimetria em painéis** (sanção, dias com − e +, comportamento proposto) e circunstâncias em selos.
- **Sem mudança** em Go, regras, golden ou V4; nenhum pacote npm novo (40); `data-teste` preservados.
- **Verificação:** vitest 43/43 (3 testes novos: modal da peça, histórico linha a linha, tabela das imputações), `vue-tsc` + `vite build`, licenças, golden sem diff; capturas da galeria em 1440 px revisadas.
- **Situação:** testado e aprovado pelo usuário no `.exe` (PADS) em 11/11/2026 e commitado. A SR com o visual novo ainda não foi testada no `.exe`.
