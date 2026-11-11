# Plano — novo visual do sys-executor ("capa dos autos refinada") — aprovado em 10/11/2026

## Contexto
Nos testes A (SEI real) e B (SR), o usuário achou as telas bagunçadas.
- **Processamento:** uma coluna longa empilha pergunta, painel, "Suas respostas" e autos.
- **Cartões de pergunta:** são enormes, com 15 botões "Ver no PDF" soltos e títulos de grupo vazios.
- **Cadastro:** blocos todos iguais e muito texto de ajuda.
- **Base:** os estilos de campos estão duplicados em cada componente.

Ele quer um visual moderno, agradável, mais fácil de usar e com animações suaves.

Decisões do usuário (10/11/2026):
1. **Direção visual:** "capa dos autos refinada". A identidade atual fica: pasta com aba bordô no PADS e azul-petróleo na SR. Entram mais espaço, cartões com profundidade suave, ícones, campos padronizados e animações. **Sem tema escuro.**
2. **Layout do processamento:** etapas fixas à esquerda. No centro, só a pergunta atual, com a barra de ação sempre visível. Análise, Suas respostas e Autos viram **abas**.
3. **Ajustes funcionais pendentes** entram **na 1ª fase**: P1, P2, P3 e os itens 1 e 2.
4. **Conferência visual:** uma **galeria local** de telas com dados sintéticos e capturas num Chromium sem GPU (`--disable-gpu`). Nada é enviado para fora e não se usa Superdesign.

Restrições do projeto:
- **Sem dependência npm nova.** O `cmd/licencas` continua em 40 npm. Ícones em SVG embutido.
- **Preservar todos os `data-teste`** e os textos verificados em `src/telas.test.ts`.
  - Os painéis de aba usam `v-show`, e não `v-if`, para os elementos continuarem existindo.
  - "Cancelar tarefa" continua visível, com o mesmo `data-teste`.
- **Sem mudança** em regras jurídicas, golden, V4 e backend, salvo P1 e o item 1.
- **Testes:**
  - só dados sintéticos;
  - não rodar bench;
  - verificação local rápida;
  - ao fim de cada fase, `.exe` + roteiro curto para o usuário;
  - commit sem coautor, só quando ele pedir.
- **Acessibilidade:** foco visível, `prefers-reduced-motion` (já existe em `estilo.css`), contraste AA e navegação por teclado nas abas.

## Sistema visual (tokens em `frontend/src/estilo.css`)
- **Cores:**
  - base: `--papel #eceee9` (fundo), `--superficie #ffffff` (cartões), `--superficie-2 #f5f6f2` (áreas recuadas e campos desabilitados), `--tinta #1b2430`, `--tinta-suave #56616c`, `--linha #d8ddd5`, `--linha-forte #a3ada4`;
  - tipo: `--pads #7a2e3a` e `--sr #2f5d6b`, com as versões claras;
  - `--acento` é definido no shell pelo tipo da tarefa e vale para o botão principal, a etapa ativa e a aba do cartão;
  - estados: alerta, erro e ok, como hoje. O foco continua azul (`--foco`) por acessibilidade.
- **Tipografia:**
  - mantém **IBM Plex Sans** (interface) e **Source Serif 4** (títulos), já locais;
  - escala 1.25 sobre 15px;
  - rótulos em caixa normal (sem CAIXA ALTA espaçada);
  - linhas de texto com no máximo cerca de 72 caracteres.
- **Forma:**
  - raios: `--r-sm 6px` (campos e chips), `--r-md 10px` (cartões), `--r-pasta 4px 14px 14px 14px` (cartão da pergunta);
  - sombras: `--sombra-1` (cartões comuns), `--sombra-2` (só o cartão da pergunta ativa e os modais);
  - espaçamento em múltiplos de 4px.
- **Movimento:**
  - tokens `--dur-1 120ms`, `--dur-2 200ms`, `--dur-3 320ms` e `--ease cubic-bezier(.2,.8,.2,1)`;
  - **um único momento orquestrado:** quando chega uma pergunta nova, o cartão "sai da pasta" (desliza e esmaece), e na linha do tempo o ✓ da etapa concluída é desenhado e o conector se preenche;
  - **o resto só responde à ação do usuário:**
    - seções recolhíveis abrem com altura animada (`grid-template-rows: 0fr → 1fr`);
    - o sublinhado das abas desliza;
    - o modal do PDF entra com fade e escala;
    - os botões têm um leve afundamento ao clicar;
    - erro de validação faz rolagem suave até o primeiro campo inválido;
  - **sem** fade-in em todas as seções e **sem** hover animado em todo cartão.
- **Elemento marcante (só um):** o **cartão da pergunta como folha presa à pasta**. Ele tem uma aba no canto superior esquerdo, na cor do tipo, com o nome da etapa ("Analisando provas"). A linha do tempo à esquerda funciona como a lombada da pasta. Todo o resto fica sóbrio.

## Componentes novos (reutilizáveis, `frontend/src/components/ui/`)
- `Icone.vue`: cerca de 16 ícones SVG embutidos (check, alerta, info, erro, arquivo-pdf, pasta, pessoa, buscar, seta, chevron, lixeira, subir/descer, engrenagem, fechar, cadeado, relógio), com `aria-hidden` e `currentColor`.
- `Recolhivel.vue`: seção que abre e fecha com animação. Tem título, contador opcional e `aberto` inicial, e usa `<button aria-expanded>`.
- `Abas.vue`: `role=tablist` com setas do teclado, sublinhado deslizante e contadores nas abas. Os painéis usam `v-show`.
- **Callouts globais** `.callout.info|alerta|erro` (com ícone), usados pelo `AvisoErro.vue` e pelos alertas dos grupos.
- **Estilos globais de formulário** (input, select, textarea, radio, checkbox), botões (`.botao`, `.primario` com `--acento`, `.perigo`, `.discreto`, `.icone`), chips (`.chip`) e cartões (`.cartao`). Os estilos duplicados saem do `CartaoPergunta.vue` e do `Montagem.vue`.

## Fases

### Fase 1: fundação, galeria e ajustes pendentes
1. Tokens, estilos globais e os componentes `ui/*` acima. `AvisoErro.vue` passa a ser callout.
2. **Galeria de telas (só desenvolvimento, fora do `.exe`):**
   - arquivos: `frontend/galeria.html`, `frontend/src/galeria/{main.ts, cenarios.ts, api-falsa.ts}` e `frontend/vite.galeria.config.ts`;
   - um plugin `resolveId` troca `src/api.ts` por `api-falsa.ts`;
   - cada cenário fixa `estado.visao`/`estado.ia` com dados sintéticos e é escolhido por `#cenario`. As fixtures copiam a forma das de `src/telas.test.ts`;
   - cenários: início, entrada, cadastro PADS (cada passo), cadastro SR, conferência, autorização de envio com revisão de nomes, confirmação de imputações, resultado com muitos fundamentos e fontes, histórico disciplinar, parada TAR-004, erro, configurações de IA e visualizador;
   - script `npm run galeria` (servidor de desenvolvimento);
   - as capturas ficam no scratchpad, por um script local: `vite build -c vite.galeria.config.ts` para a pasta do scratchpad, `vite preview` e `chrome-headless-shell --headless --disable-gpu --window-size=1440,900 --screenshot` (binário em `~/.cache/ms-playwright/chromium_headless_shell-1248`);
   - o `index.html`, o `vite.config.ts` e o `embed.go` não mudam.
3. **Ajustes funcionais:**
   - **P1:** em `internal/pipeline/analisar.go` (pergunta "Conclusão sobre o fato", cerca da linha 680), o título passa a levar o sindicado, como em "Conclusão sobre o fato F-1: 3º SGT PM HÉLIO FICTÍCIO", usando o mesmo rótulo do título do grupo. Muda também o resumo de "Suas respostas".
   - **P2:** em `Montagem.vue`, seção Ofendidos:
     - texto de orientação ("Se o ofendido é a Administração ou uma instituição, não a cadastre: marque 'Não há ofendido individualizado'");
     - aviso no campo quando o nome tiver palavra institucional (administração, estado, fazenda, polícia, município, prefeitura, secretaria, união);
     - a opção "Menor de 13 anos" fica escondida para esses nomes na conferência.
   - **P3:** em `Montagem.vue`, nos depoimentos:
     - se o nome digitado for igual, ignorando caixa e acentos, ao de uma pessoa já cadastrada, aparece "Esta pessoa já está cadastrada como sindicado/ofendido/depoente. [Usar a mesma]", reaproveitando `trocarDepoente()`;
     - na conferência, aviso quando um sindicado depõe com papel diferente de "Sindicado".
   - **Item 1:** em `internal/pipeline/cadastro.go`, `conferirComCadastro()`, os casos 0 e "vários" também usam `nomeComPosto(ps)`. O teste vai em `conferencia_cadastro_test.go`.
   - **Item 2:** no `CartaoPergunta.vue`, um grupo sem descrição, alertas, fontes e campos visíveis não é mostrado. Isso resolve os títulos vazios "Transgressão anterior 1…5".
4. Entregar o `.exe` com o novo visual base, que já se espalha pelos estilos globais, e os ajustes.

### Fase 2: processamento e perguntas (o centro do uso)
- `Processamento.vue`:
  - grade `[lombada 248px sticky | conteúdo até 780px]`;
  - a lombada tem as etapas (`v.etapas`), "Consultar os autos" e "Cancelar tarefa", com o fluxo de confirmação atual;
  - o conteúdo tem o título da etapa, o cartão da pergunta ou o cartão de estado (barra de progresso e atividade) e as **Abas** Análise (`PainelAnalise`), Suas respostas (com o contador) e Autos (`PainelAutos`);
  - abaixo de 900px, a lombada vira uma faixa de progresso horizontal no topo.
- `CartaoPergunta.vue`:
  - aba da etapa no canto, título e contexto (contexto com mais de 3 linhas recolhe em "Mostrar explicação");
  - cada grupo vira seção. "Fundamento:" e "Tese da defesa:" viram lista estruturada, com os 2 primeiros visíveis e "Mostrar mais (n)";
  - as fontes viram "Trechos citados (n)" num `Recolhivel`, agrupadas por peça, com chips "pág. N". O `data-teste` `fonte-…` e o texto "Ver no PDF" no `title`/`aria-label` ficam;
  - alertas "Lacuna:" e "Diligência sugerida:" ganham rótulo e ícone próprios;
  - SIM/NÃO e escolhas de até 4 opções viram controle segmentado ou cartões de escolha (o input radio nativo continua, por acessibilidade e testes);
  - **barra de ação fixa** (`position: sticky; bottom: 0`) com o botão principal na cor do tipo;
  - rolagem até o primeiro erro;
  - transição de entrada do cartão a cada pergunta nova (`<Transition>` com `:key="pergunta.id"`).
- `PainelAnalise.vue`: `details` viram `Recolhivel`, e os itens com situação viram selos ("confirmado", "congelada", "admitido").
- `PainelAutos.vue`: lista de peças com ícone de PDF, páginas e selo OCR.
- `VisualizadorFontes.vue`: modal com entrada animada, cabeçalho fixo e fechar com Esc. A lógica do PDF.js não muda.

### Fase 3: cadastro, entrada, início e configurações
- `Montagem.vue`:
  - **o stepper mostra a situação de cada passo** (✓ completo, ponto de pendência, contagem de arquivos), calculada a partir do `local`;
  - blocos viram `.cartao`;
  - o texto longo do topo e das seções vira "Como funciona?" num `Recolhivel`;
  - navegação Anterior/Próximo/Concluir em barra fixa no rodapé;
  - a conferência vira um resumo em tabela (peças e envolvidos).
- `ArquivosPeca.vue`:
  - linhas de arquivo com ícone, chip de páginas e botões-ícone (subir, descer, remover) com `aria-label`; os `data-teste` ficam;
  - área "Anexar PDF(s)" com aparência de zona de anexo (o diálogo do Wails continua);
  - "Documento do SEI" com o mesmo select, estilizado.
- `EditorEnquadramento.vue` e `CampoBusca.vue`: estilo dos campos e da lista de sugestões, com transição ao abrir.
- `Inicio.vue`: as pastas ganham leve elevação no hover e a aba sobe 2px. É a única animação de hover do app. O cartão de primeiro uso da IA vira callout.
- `Entrada.vue` e `PainelSEI.vue`: duas opções em cartões com ícone. O SEI mostra o progresso com o mesmo cartão de estado do processamento.
- `ConfiguracoesIA.vue`, `ConfigChatGPT.vue` e `ConfigGemini.vue`:
  - escolha do serviço como cartões de opção;
  - abas com o `Abas.vue`;
  - estados conectado/sem chave com selo e ícone.
- `FaixaTipo.vue`: faixa mais fina, título e peça central à esquerda e, à direita, botão-ícone "Configurações de IA".

## Arquivos principais
- `frontend/src/estilo.css` (tokens e base), `frontend/src/components/ui/{Icone,Recolhivel,Abas}.vue` (novos).
- `frontend/src/views/{Processamento,Montagem,Inicio,Entrada,ConfiguracoesIA}.vue`.
- `frontend/src/components/{CartaoPergunta,PainelAnalise,PainelAutos,ArquivosPeca,EditorEnquadramento,CampoBusca,AvisoErro,FaixaTipo,PainelSEI,VisualizadorFontes,ConfigChatGPT,ConfigGemini}.vue`.
- `frontend/galeria.html`, `frontend/src/galeria/*`, `frontend/vite.galeria.config.ts`, script `galeria` no `package.json` (só desenvolvimento).
- Fase 1 (Go): `internal/pipeline/analisar.go` (P1), `internal/pipeline/cadastro.go` e `conferencia_cadastro_test.go` (item 1).
- Docs: `GUIA_CONTINUIDADE.md` e um relatório curto `docs/planning/RELATORIO_INTERFACE.md`, atualizados a cada fase.

## Verificação (em cada fase)
1. `cd frontend && npx vitest run` (32+ testes, todos verdes, com testes novos para item 2, P3 e abas) e `npm run build` (vue-tsc + vite).
2. Se mexer em Go: `gofmt`, `GOOS=windows go vet ./...` e `go test ./internal/pipeline ./internal/workflow`.
3. **Capturas da galeria** (Chromium headless, `--disable-gpu`, 1440×900 e 900×800) de cada cenário, revisadas por mim com o Read das imagens antes da entrega. Conferir alinhamento, hierarquia, contraste, ausência de cortes e o comportamento com texto longo.
4. `go run ./cmd/licencas` (continua com 40 npm) e `git diff --stat -- internal/policy/testdata` (vazio).
5. Gerar o `sys-executor.exe` em `C:\Users\diogo\Documents\testes-sys-executor\` e passar ao usuário um roteiro curto do que abrir (um PADS por upload ou SEI até TAR-004, e uma SR na fase 3).
