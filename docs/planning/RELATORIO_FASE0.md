# RELATÓRIO DA FASE 0 — VIABILIDADE (sys-executor)

- **Data:** 08/11/2026
- **Escopo:** PoCs isoladas em `poc/`, descartáveis e **sem dados reais**. Usam a fixture sintética PADS-01 e a biblioteca jurídica pública de `docs/law_library`.
- **Máquina de medição:** AMD Ryzen 5 5600X (6C/12T, AVX2, sem AVX-512), 32 GB de RAM (16 GB visíveis no WSL2), GPU AMD Radeon RX 6600 XT 8 GB (driver 32.0.21045.16010), Windows 11 Pro build 26300, WebView2 157.
- **Artefatos pesados** (modelos, binários, fixtures geradas, logs de avaliação) ficam em `~/.cache/sys-executor-dev/`, fora do repositório. Os hashes SHA-256 estão em `bin/SHA256SUMS` e `models/SHA256SUMS`.

## 1. Situação por tarefa

| Tarefa | Resultado | Decisão |
|---|---|---|
| F0-01 Wails 3.1 + Vue (Windows x64) | **OK** | Mantém Wails v3.1.0 |
| F0-02 Sidecar llama-server, Job Object, json_schema, RAM | **OK em CPU**; **Vulkan derrubou a máquina** (ver §2) | Mantém sidecar; GPU só com sonda segura |
| F0-03 Avaliação de modelos | Linha de base registrada; **nenhum modelo passa em tudo** | Ajustes de arquitetura (§5) |
| F0-04 go-pdfium wasm | **OK** (806 págs em 0,67 s) | Mantém wasm; contingência CGO desnecessária |
| F0-05 Tesseract | **OK**, com alerta sobre datas | Mantém; datas de OCR exigem confirmação |
| F0-06 Typst + DOCX | **OK** (Word abre o DOCX; TNR no PDF) | Mantém ADR-007 |
| F0-07 SQLite FTS5 + embeddings | **OK** como stack (busca em 2 ms); qualidade 14–16/20 no top-5 | Mantém ADR-006; biblioteca inicial pré-indexada |
| F0-08 PoC SEI (usuário) | **OK**: executada pelo usuário com login real, 2FA, processo sigiloso e geração do PDF (§3) | Mantém ADR-008; navegação só por cliques (URLs com `infra_hash`) |
| F0-09 Licenças e SBOM | **OK**, com 2 pontos de atenção no pacote do Tesseract (§4.9) | — |

## 2. Incidente: desligamento inesperado da máquina (08/11/2026)

**Evidências:**
- Evento 6008 do Windows: "o desligamento do sistema … não era esperado". Não há minidump nem BugCheck 1001, ou seja, foi um travamento/desligamento duro, sem tela azul registrada.
- O último arquivo escrito antes da queda foi `logs/eval-win-vulkan-qwen35-4b.txt` (11:53:30, UTC−4), **vazio**. A máquina caiu ao carregar ou executar o Qwen3.5-4B no `llama-server.exe` **Vulkan** na RX 6600 XT.
- Todas as execuções anteriores, em CPU, terminaram normalmente.

**Causa mais provável (não comprovada, por falta de dump):** travamento do driver de vídeo AMD ou instabilidade de energia sob carga Vulkan de compute. Sem minidump não dá para separar as duas hipóteses. **O teste Vulkan não foi repetido nesta sessão**, para não derrubar a máquina de novo.

**Segunda queda (08/11/2026, entre 13:59:41 e 14:18, falta de energia elétrica relatada pelo usuário):** ocorreu **depois** de concluído este relatório (gravado às 13:59:26) e sem nenhuma carga de GPU em execução. Nenhum artefato da Fase 0 se perdeu: todas as PoCs compilam (`go vet`, incl. `GOOS=windows`) e os logs/bancos em `~/.cache/sys-executor-dev/` estão íntegros. Diferença útil entre os dois registros do Windows: nesta queda há o evento **Kernel-Power 41** (típico de perda de energia), enquanto a queda das ~11:53, durante o teste Vulkan, registrou **apenas o 6008**, sem o 41. Isso não prova a causa da primeira queda, mas mantém a hipótese de travamento ligado à GPU no lugar de falta de energia. (O horário do 6008 é o último carimbo horário "last alive", e não o instante exato da queda.)

**Consequências para o produto (incorporar na Fase 2/4):**
1. **GPU nunca é o padrão no primeiro uso.** O app começa em CPU, e a GPU é oferecida depois de uma **sonda curta**: modelo pequeno, `-ngl` parcial, poucos tokens, timeout.
2. **Sentinela de falha:** antes da sonda, gravar a preferência `gpu_sonda_em_andamento=true` (preferência persistente mínima, sem dados de processo); apagar ao concluir. Se o app reiniciar com a sentinela presente, marcar a GPU como **instável** e usar CPU, com uma mensagem amigável. Assim, uma queda como a de hoje não se repete em loop.
3. Seleção manual de GPU continua disponível em "avançado", com aviso.
4. O teste Vulkan na RX 6600 XT fica **pendente de decisão do usuário** (ver §7).

## 3. Achados por PoC

### F0-01 — Wails v3.1.0 + Vue/TS (`poc/wails`)
- Compilação cruzada WSL → Windows **sem CGO** (exe de 18,8 MB). Bindings gerados com `GOOS=windows wails3 generate bindings`. Sem o `GOOS`, o gerador analisa o alvo Linux, que exige GTK4/WebKitGTK, e não encontra os serviços.
- Autoteste automático: front chama o binding Go com acentos ("Olá, Fase 0 — ação, atenção, Rondônia!"), recebe o evento `time` emitido pelo Go e devolve o resultado. **Front pronto em 652 ms; ciclo completo em 1,05 s.**
- **Atenção V4 §33 (sem logs):** o Wails escreve logs `INF` no stderr por padrão, incluindo URLs servidas e informações de build. Na Fase 1, o logger da aplicação precisa ser silenciado e o binário compilado com `-H windowsgui`.
- O exe não abriu a partir do caminho UNC do WSL (`\\wsl.localhost\…`). Sem impacto no produto, que roda instalado. Testes manuais devem copiar o exe para uma pasta local.

### F0-02 — Sidecar `llama-server` b12207 (`poc/llm`, `poc/sidecar-win`)
- **Segurança:** porta aleatória em 127.0.0.1, `--api-key` aleatória (requisição sem chave → **HTTP 401**), `--no-webui` e `--log-disable` aceitos na b12207 (P-10: `--no-webui` **confirmado**).
- **`response_format: json_schema` (strict)** respeitado no Windows e no Linux. JSON 100% válido em todas as chamadas que terminaram (a única falha foi truncamento por `max_tokens`, com o Qwen3.5-2B na T3 da 1ª rodada).
- **Job Object (Windows), 3 testes:**
  1. O app sai sem matar o sidecar → o sidecar morre (`KILL_ON_JOB_CLOSE`). **OK.**
  2. O app é morto à força (`taskkill /F`, simulando crash) → o sidecar morre. **OK.** Contraste: no Linux, sem esse mecanismo, um `llama-server` **ficou órfão** nesta sessão quando o processo pai foi morto.
  3. Limite `JOB_OBJECT_LIMIT_JOB_MEMORY` = 800 MB → o sidecar aborta na carga com **`0xC0000409`** (fail-fast), e o app detecta. **Funciona, mas a falha não é "limpa"**: o gerenciador precisa mapear esse código para "memória insuficiente" (V4 §38.4).
- **Medição de memória:** o pico do Job (`PeakJobMemoryUsed`) mede **memória privada** (commit). Os pesos carregados via **mmap não entram** na conta: Qwen3.5-2B deu 1.210 MB no Job contra 2,58 GB de RSS no Linux. Consequência: o **orçamento de memória (plano §9) não pode usar só o Job**. Ele precisa somar o tamanho do GGUF (pesos mapeados) ao pico privado.
- **Hardware:** `x/sys/cpu` (AVX2) e `GlobalMemoryStatusEx` (total/livre) funcionam no binário Windows.
- **Pico de RSS (Linux, ctx 8192):** Qwen3.5-2B 2,6 GB · Qwen3.5-4B 5,8 GB · Qwen3-4B-Instruct-2507 6,6 GB.

### F0-03 — Avaliação dos modelos (fixture PADS-01, CPU, Q4_K_M, temperatura 0,1, sem "thinking")

Tarefas: **T1** delimitar as 2 imputações do memorando (artigo, inciso, natureza, data, trecho literal); **T2** classificar a função das 6 peças só por IA; **T2b** classificação híbrida (título inequívoco → regra Go; ambíguo → IA, um documento por chamada); **T3** analisar a IMP-1 com injeção de prompt nos autos, mídia ausente e conduta fora do memorando; **T4** redigir um trecho citando SEI.

| Modelo | T1 | T2 | T2b | T3 | T4 | tok/s | Pico RSS |
|---|---|---|---|---|---|---|---|
| Qwen3.5-2B | OK | 3/6 | 5/6 | FALHA: copiou o documento bruto em `fundamentos` | OK | ~24 | 2,6 GB |
| Qwen3.5-4B | FALHA: `mes_fato=0` | 3/6 | 5/6 | **OK** (detectou injeção e mídia, não obedeceu, propôs improcedência) | OK | ~11 | 5,8 GB |
| Qwen3-4B-Instruct-2507 | OK | 2/6 | 5/6 | FALHA: não listou a mídia ausente | OK | ~11 | 6,6 GB |

Tempo por tarefa em CPU: T1 22–50 s, T3 65–85 s (só **uma** imputação).

**Thinking × non-thinking (Qwen3.5-4B, `--reasoning on`, `max_tokens` ×4):** **inviável em CPU.** O raciocínio consumiu o orçamento de tokens e T1, T2, T3 e T4 terminaram **sem JSON** (saída truncada). Cada tarefa levou de 4 a 11 min, contra 25 a 70 s sem raciocínio. A T2b ficou em 4/6, com duas classificações vazias. **Decisão:** perfis CPU usam non-thinking. O thinking só será reavaliado em perfis GPU, com orçamento de raciocínio limitado (`--reasoning-budget`) e medido por etapa.

**Conclusões:**
1. **Datas como texto livre falham:** na 1ª rodada, todos devolveram `"1008-20-20"` e similares. Com dia/mês/ano como inteiros no schema, ficou melhor, mas ainda houve `mes_fato=0`. **Regra:** o Go extrai a data por regex do `trecho_literal` (que é conferido literalmente contra o documento) e só aceita a da IA se coincidir.
2. **Função documental:** o erro remanescente do T2b é o mesmo nos três modelos. A peça 6 ("Manifestação", após a instrução) sai como DEFESA_PREVIA. **A cronologia (antes/depois dos termos de inquirição) deve ser decidida pelo Go**, deterministicamente, e não pela IA. A IA fica apenas com a dúvida residual.
3. **Injeção de prompt:** só o Qwen3.5-4B passou na T3. O isolamento por delimitadores não basta; o validador Go (citações só de `evid_refs` permitidos, bloqueio de condutas fora do memorando) continua obrigatório.
4. **Recomendação inicial de catálogo (a confirmar com o benchmark da Fase 4):** **Qwen3.5-4B** como perfil padrão (melhor em T3) e **Qwen3.5-2B** como perfil básico para máquinas de 8 GB, rápido mas exigindo validação mais forte. O Qwen3-4B-Instruct-2507 não trouxe vantagem.
5. **Licenças (P-08 resolvida):** os quatro modelos testados são **Apache-2.0**, conforme o metadado GGUF e a API oficial do Hugging Face. O **Qwen2.5-3B-Instruct** está sob a **Qwen Research License** ("Non-Commercial … research or evaluation purposes only"; uso comercial exige licença da Alibaba) e **fica fora do catálogo**.

### F0-04 — go-pdfium v1.21.4 (WebAssembly/wazero, sem CGO) (`poc/pdfium`)
- PDF sintético de **806 páginas** (2,13 M caracteres): texto estruturado com posições em **2,30 s com 1 worker** (2,9 ms/página) e **0,67 s com 4 workers** (0,8 ms/página).
- Inicialização do runtime: 2,2 s. Render de 200 dpi: 10–14 ms/página. RSS: 330 MB (1 worker) / 470 MB (4 workers).
- **A contingência CGO da ADR-004 não é necessária.**

### F0-05 — Tesseract 5.4.0 (Windows, `por`) (`~/.cache/.../fixtures`)
- **stdin → stdout funciona no Windows** (`tesseract.exe stdin stdout -l por --tessdata-dir …`): não é preciso arquivo temporário para a imagem (P-10 confirmado).
- Fixture sintética escaneada: **CER 0,08% / WER 0,51%** (pág. 1) e **0% / 0%** (pág. 4).
- Resoluções reais escaneadas (R148/R208): confiança média de 92% (r148-1), 92% (r208-1) e 84% (r208-3).
- **Alerta crítico:** em r208-3, o ano **"2017"** (conferido visualmente no recorte da imagem) foi lido como **"2047" com confiança 92**. Outra versão do Tesseract leu "2077" em r208-1. **A confiança do OCR não protege datas e números.** Regra para a Fase 2: datas, números SEI, matrículas e artigos vindos de páginas OCR entram como **"não confirmados"** e exigem verificação cruzada (outra ocorrência no texto, coerência cronológica) ou confirmação humana antes de alimentar o motor determinístico.
- A saída TSV exige `-c tessedit_create_tsv=1`, porque o alias `tsv` depende de `tessdata/configs`, que não vamos empacotar.
- **Tamanho:** o runtime mínimo (tesseract.exe + 26 DLLs) cai de 116 MB para **22 MB** com `strip --strip-debug`, com resultado idêntico (mesmo CER). Somar ~19 MB de `por` + `osd`.

### F0-06 — DocModel → DOCX + PDF (`poc/docmodel`)
- **DOCX** por writer OOXML próprio (`archive/zip` + XML): **o Word abriu sem reparo** (via COM, invisível), com 2 páginas, Times New Roman 12, brasão no cabeçalho, rodapé com campo `PAGE` numerando 1 e 2, e todos os acentos e aspas tipográficas preservados.
- **PDF** pelo **Typst 0.15.1**: no Windows embute **TimesNewRomanPSMT** do sistema. São 2 páginas, iguais ao DOCX neste caso. Cabeçalho (§32.1), margens 2/2/3/1,5 cm, recuo de 2,5 cm, justificado e alínea vazia só com rótulo (SR), como esperado.
- **Fallback de fonte:** sem TNR (Linux), o Typst caiu em Libertinus Serif, a fonte embutida dele. **Liberation Serif (OFL) precisa ser empacotada** para o fallback métrico (ADR-007).
- O texto extraído do PDF contém hífens suaves (U+00AD) da hifenização. A comparação golden DOCX × PDF deve normalizá-los.
- Sem metadados de autor no DOCX (`core.xml` só com título genérico).

### F0-07 — Biblioteca jurídica: SQLite FTS5 + Qwen3-Embedding (`poc/rag`)
- **Stack viável:**
  - `modernc.org/sqlite` v1.62.0 (sem CGO) com **FTS5** (`unicode61 remove_diacritics 2`) funcionando;
  - 12 PDFs com camada de texto → **1.688 chunks** por artigo (1.779 no modo contextual);
  - extração e chunking via go-pdfium em 33 s;
  - banco de **9,5 MB** com vetores float32 de 1024 dimensões;
  - vetores carregados em 16 ms e **busca vetorial exata em 1,8–2,0 ms/consulta**. A ADR-006 (força bruta em Go, sem banco vetorial) está **confirmada**.
  - As duas resoluções escaneadas (R148/R208) ficaram de fora deste índice e dependem do OCR (F0-05).
- **Custo dos embeddings em CPU:** Qwen3-Embedding-0.6B Q8_0 processou ~300–375 tokens/s (`llama-bench` pp512), ou seja, **~1,1 chunk/s e 25–28 min** para a biblioteca inicial. O tamanho do lote não muda a vazão. `-c/-b/-ub` muito grandes (8192) pioram, porque a atenção roda contra o cache inteiro. RSS ~970 MB. **Proposta:** a biblioteca inicial vai **pré-indexada no instalador** (embeddings gerados no build, ~10 MB), e só documentos incluídos pelo usuário são embutidos localmente, em segundo plano e de forma retomável.
- **Qualidade:** 20 consultas parafraseadas com gabarito no RDPM real (`poc/rag/consultas.json`); acerto = artigo certo no RDPM avulso ou dentro da Coletânea.

| Configuração | BM25 hit@1 / @5 | Vetor hit@1 / @5 | RRF hit@1 / @5 |
|---|---|---|---|
| Base, instrução da consulta em pt | 3 / 7 | 9 / 10 | 7 / 13 |
| Base, instrução em **inglês** | 3 / 7 | 10 / 14 | 6–7 / 14 |
| + chunk **contextual** (documento — capítulo — artigo) | 3 / 7 | 10 / 13 | 8–9 / 14 |
| + **sinônimos do domínio** na consulta BM25 | 8 / 12 | 10 / 13 | **12 / 16** |
| + prefixo como stemming (sem sinônimos) | 8 / 10 | 10 / 13 | 8 / 13 |

- **Leitura dos resultados:**
  - O BM25 puro falha em paráfrases ("falta" × "transgressão", "graves" × "grave"), porque o FTS5 não tem stemming em português.
  - O vetor 0,6B resolve ~70% no top-5.
  - Sinônimos jurídicos curados são a alavanca mais forte. **Ressalva:** foram escritos olhando as mesmas 20 consultas, então o 16/20 é otimista e precisa ser confirmado num conjunto separado na Fase 5.
  - Houve variação de ±1 entre execuções idênticas no RRF (empates e não determinismo numérico).
- **Falhas persistentes** (arts. 17, 18, 29, 33, 37, 74):
  - a Coletânea é composta, e o mesmo número de artigo aparece em vários diplomas internos;
  - artigos-lista longos (art. 17) são partidos em pedaços sem o caput;
  - números de página de rodapé vazam para o texto.
  Tudo isso é escopo do **parser estrutural de normas** já previsto (plano §10).
- **No produto, a recuperação semântica não é o caminho principal das citações.** O catálogo de regras resolve a maioria por **referência exata** (diploma + artigo → `dispositivo`, etapa 1 da recuperação híbrida), e o validador bloqueia qualquer citação que não venha de `LegalPassage` recuperada. Os números acima medem o apoio à redação e à pesquisa livre.

### F0-08 — Mapeamento do SEI (`poc/sei-mapa`)
- Ferramenta **somente de observação** (chromedp v0.23 + Edge do sistema, perfil descartável, downloads bloqueados, dígitos mascarados, nada gravado). Instruções em `poc/sei-mapa/LEIA-ME.md`.
- Teste de fumaça na página pública de login (sem credenciais):
  - formulário POST com `txtUsuario`, `pwdSenha` (mais um campo texto homônimo), `selOrgao` e o oculto `hdnAcao`;
  - botão "ACESSAR";
  - links **"Entrar com gov.br"** e **"Autenticação em dois fatores"**;
  - **um único host**: `sei.sistemas.ro.gov.br`.
- **Observação técnica:** o chromedp v0.23 mudou a API (comandos CDP genéricos `chromedp.Call`/`CallBrowser`, `Evaluate[T]`, `Listen`). A versão precisa ser fixada no `go.mod` do produto.
- **Execução real pelo usuário (08/11/2026, ~14:45):** login + 2FA, pesquisa rápida de um processo **sigiloso**, reautenticação, "Gerar Arquivo PDF do Processo" → "Todos os documentos disponíveis" → "Gerar". **Download bloqueado como previsto.** Nenhum dado do processo foi registrado (dígitos mascarados; os prints compartilhados na conversa não foram copiados para nenhum arquivo). Na 1ª tentativa, a tela pós-login demorou a carregar (lentidão do SEI; na 2ª tentativa veio `HTTP 200` normalmente).
- **Fluxo observado** (todas as páginas em `sei.sistemas.ro.gov.br`; 747 requisições, **um único host**; nenhum diálogo JavaScript):

| # | Etapa | Requisição (`acao` ou caminho) | Tela / campos (estrutura) |
|---|---|---|---|
| 1 | Login | `GET/POST /sip/login.php` (`sigla_orgao_sistema`, `sigla_sistema`) | `txtUsuario`, `pwdSenha`, `selOrgao`, oculto `hdnAcao`; botão "ACESSAR" |
| 2 | 2FA | `POST /sip/login.php` (mesma URL) | `txtCodigoAcesso`, checkbox `chkLiberar` ("Não usar 2FA neste dispositivo e navegador"); botões "Validar" e "Desativar 2FA" |
| 3 | Entrada | `GET /sei/inicializar.php` → `procedimento_controlar` | "SEI - Controle de Processos"; formulário de pesquisa rápida `protocolo_pesquisa_rapida` (`txtPesquisaRapida`) no quadro `top` |
| 4 | Pesquisa | `POST protocolo_pesquisa_rapida` → `GET procedimento_trabalhar` | Título "SEI - <nº do processo>" |
| 5 | **Reautenticação (sigiloso)** | `GET/POST usuario_validar_acesso` (`acao_destino`, `acao_negado`, `id_procedimento`…) → `procedimento_trabalhar` com `acesso=1` | Modal em iframe `modal-frame`, título "SEI - Identificação de Acesso": `txtUsuario` (já preenchido), `pwdSenha`, botão "Acessar". Processo não sigiloso abre direto, sem esta etapa (informação do usuário) |
| 6 | Processo | `procedimento_visualizar` (iframe `ifrArvore`, "SEI - Árvore Montar"), `arvore_visualizar` (iframe `ifrConteudoVisualizacao`), `arvore_processar_html` (iframe aninhado `ifrVisualizacao`) | Barra de ícones com `img` "Gerar Arquivo PDF do Processo"; árvore com ícones de nível de acesso (hipótese legal no `title`) |
| 7 | Gerar PDF | `XHR POST /sei/controlador_ajax.php` (`acao_ajax`) e `GET procedimento_gerar_pdf` (em `ifrVisualizacao`) | Rádios `rdoTipo` ("Todos os documentos disponíveis", "Todos exceto selecionados", "Apenas selecionados"), `selDocumentosExceto`, `selDocumentosApenas`, ocultos `hdnDocumentosExceto`, `hdnDocumentosApenas`, `hdnFlagGerar`, `hdnInfraTipoPagina`; botões "Gerar" e "Fechar"; iframe `ifrDownload` |
| 8 | Download | `POST procedimento_gerar_pdf` → `GET exibir_arquivo` (`nome_arquivo`, `nome_download`) no iframe `ifrDownload` | Nome sugerido no padrão `SEI_<nº do processo com _>.pdf`. O `net::ERR_ABORTED` é efeito do bloqueio. Depois, o quadro `top` mostra um botão "Cancelar" (provável aviso de processamento) |

- **Consequências para a Fase 2 (cliente `sei`):**
  1. **Toda URL do `controlador.php` carrega `infra_hash`** (além de `infra_sistema` e `infra_unidade_atual`). O app **não pode montar URLs**: ele navega apenas preenchendo campos e clicando em elementos encontrados por rótulo, como previsto nos detectores por conteúdo.
  2. **Allowlist de leitura (deny-by-default)** confirmada: `/sip/login.php`, `/sei/inicializar.php`, `procedimento_controlar`, `protocolo_pesquisa_rapida`, `procedimento_trabalhar`, `usuario_validar_acesso`, `procedimento_visualizar`, `arvore_visualizar`, `arvore_processar_html`, `procedimento_gerar_pdf`, `exibir_arquivo` e `controlador_ajax.php`. **O valor de `acao_ajax` não foi capturado** (a ferramenta registra só nomes de parâmetros); ele precisa ser fixado na Fase 2, porque a guarda deve permitir só esse valor.
  3. **Reautenticação de sigiloso:** conforme a V4 (§13, "pedir credenciais/autorização novamente", e §33), o app **pede a senha de novo** ao usuário quando aparecer "Identificação de Acesso". Ela não é retida da tela de login. O usuário vem preenchido pelo SEI, então só a senha é pedida.
  4. **2FA:** o app **nunca** marca `chkLiberar` e **nunca** aciona "Desativar 2FA", que devem entrar na lista de bloqueio do cliente. O perfil do Edge é descartável, então nenhum "lembrar dispositivo" sobrevive à tarefa.
  5. **Frames aninhados:** o fluxo atravessa `top` → `ifrConteudoVisualizacao` → `ifrVisualizacao` → `ifrDownload`, e os detectores e cliques precisam operar dentro dos iframes (mesma origem; funciona com CDP).
  6. **Timeouts generosos com progresso visível** (a tela pós-login chegou a demorar mais de um minuto na 1ª tentativa), sem reenviar formulários: a página de 2FA não pode ser recarregada, porque reenviaria o código.
  7. **Download:** `Browser.setDownloadBehavior` = `allowAndName` para o diretório efêmero; o evento de download confirma o nome. O nome sugerido contém o número do processo e **não deve ser usado** como nome em disco (nome opaco, plano §13).
- **Complemento do usuário (08/11/2026):** o botão "Gerar" da tela "Gerar Arquivo PDF do Processo" é `<button type="button" name="btnGerar" value="Gerar" onclick="gerar();" class="infraButton">` (atalho G), e a opção "Todos os documentos disponíveis" já vem marcada por padrão. O `controlador_ajax.php` foi chamado ao clicar no **ícone PDF** da barra do processo, antes da tela de geração; o valor de `acao_ajax` será informado pelo usuário (DevTools → Rede).
- **Ainda pendente (P-09, parte 2):** observar um **PDF consolidado real**, localmente e sem copiar dados, para fixar o padrão de segmentação (cabeçalho e rodapé de cada documento). Necessário antes da Fase 2.

### F0-09 — Licenças e SBOM
Ver §4.

## 4. Licenças e SBOM (componentes efetivamente usados nas PoCs)

### 4.1 Go (dependências compiladas)
| Módulo | Versão | Licença |
|---|---|---|
| github.com/wailsapp/wails/v3 | v3.1.0 | MIT |
| github.com/go-ole/go-ole | v1.3.0 | MIT |
| github.com/klippa-app/go-pdfium | v1.21.4 | MIT (embute PDFium em wasm: BSD-3 + terceiros, incl. FreeType sob FTL) |
| github.com/tetratelabs/wazero | v1.12.0 | Apache-2.0 |
| github.com/jolestar/go-commons-pool/v2 | v2.1.2 | Apache-2.0 |
| modernc.org/sqlite, libc, mathutil, memory | v1.62.0 / v1.80.1 / v1.7.1 / v1.12.1 | BSD-3-Clause |
| github.com/chromedp/chromedp, cdproto | v0.23.0 / v0.159.0 | MIT |
| golang.org/x/sys, x/text | v0.48–0.49 / v0.43.0 | BSD-3-Clause |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause |
| github.com/remyoudompheng/bigfft | 2023-01-29 | BSD-3-Clause |
| dustin/go-humanize, mattn/go-isatty, ncruces/go-strftime | — | MIT |

**Nenhuma dependência Go copyleft.**

### 4.2 Frontend (build)
Vue 3 (MIT), @wailsio/runtime 3.1.0 (MIT), Vite 8 / TypeScript 5.9 / vue-tsc (MIT/Apache-2.0, só no build).

### 4.3 Binários de terceiros (empacotados)
| Componente | Versão | Licença | Tamanho mínimo |
|---|---|---|---|
| llama.cpp `llama-server` CPU x64 | b12207 | MIT (+ LLVM OpenMP: Apache-2.0 with LLVM exception) | 47 MB (server + 7 DLLs + 14 backends CPU) |
| llama.cpp Vulkan x64 | b12207 | MIT | 89 MB |
| Typst CLI | 0.15.1 | Apache-2.0 (+ NOTICE) | 52 MB (exe) |
| Tesseract | 5.4.0.20240606 (UB Mannheim) | Apache-2.0 | 22 MB após strip |
| tessdata `por` + `osd` | — | Apache-2.0 | 19 MB |

### 4.4 Pontos de atenção (Tesseract para Windows)
Das 26 DLLs que o `tesseract.exe` carrega:
- **`libjbig-0.dll` (JBIG-KIT, GPL-2.0+)**, puxada pela libtiff;
- **`libiconv-2.dll` (LGPL-2.1+)**;
- `libgcc_s_seh-1`/`libstdc++-6` (GPL-3 com GCC Runtime Library Exception, sem impacto prático).

O Tesseract roda como **processo separado** (agregação), então o app não é contaminado. Mesmo assim, distribuir a DLL GPL exige oferecer o código-fonte. **Recomendação:** na Fase 8, compilar o Tesseract sem libtiff/jbig/libarchive/OpenSSL, já que só recebemos PNG via stdin. Isso elimina GPL e LGPL do pacote. [A VALIDAR com a TI/jurídico da instituição]

### 4.5 Modelos (download no primeiro uso, não empacotados)
Qwen3.5-2B, Qwen3.5-4B, Qwen3-4B-Instruct-2507 e Qwen3-Embedding-0.6B: **Apache-2.0** (metadado GGUF + API do Hugging Face, 08/11/2026). Qwen2.5-3B-Instruct: **Qwen Research License**, excluído.

### 4.6 Sistema (não redistribuído)
WebView2 Runtime (Microsoft), Microsoft Edge (automação SEI), Times New Roman (fonte do Windows), drivers Vulkan.

## 5. Ajustes propostos ao plano (para aprovação do usuário)
1. **GPU segura** (§2): CPU por padrão, sonda curta com sentinela de falha, GPU como opção.
2. **Orçamento de memória** = tamanho do GGUF (mmap) + pico privado medido (Job) + KV + margem. Mapear `0xC0000409` sob limite do Job para "memória insuficiente".
3. **Datas e números de OCR** sempre "não confirmados" até verificação cruzada ou confirmação humana.
4. **Datas extraídas pela IA** conferidas por regex Go sobre o trecho literal.
5. **Cronologia das peças** (defesa prévia × alegações finais) decidida pelo Go, pela posição relativa aos termos de instrução.
6. **Wails:** logger silenciado e `-H windowsgui` (V4 §33).
7. **Tesseract** recompilado sem GPL/LGPL e com strip (§4.4).
8. **Liberation Serif** empacotada como fallback.
9. **Biblioteca inicial pré-indexada** no instalador (FTS5 + embeddings gerados no build). Só documentos novos do usuário são embutidos localmente, em segundo plano e de forma retomável.
10. **Recuperação:**
    - instrução da consulta do Qwen3-Embedding em inglês;
    - chunk com cabeçalho hierárquico (diploma — capítulo/seção — artigo);
    - **tesauro jurídico curado** (coloquial → normativo) para expandir a consulta lexical;
    - Coletânea dividida por diploma interno;
    - conjunto de avaliação separado (held-out) na Fase 5.
11. **Modo thinking** desligado nos perfis CPU.

## 6. Linha de base do benchmark (para os limiares da Fase 4)
- Os portões rígidos do plano continuam valendo: 0 violações após validação e 0 citações inventadas aprovadas.
- Linha de base CPU (Ryzen 5 5600X): ver tabelas da F0-03 e da F0-07. Os limiares de qualidade serão definidos na Fase 4 com o `cmd/fixturegen` gerando várias fixtures, porque uma única fixture (PADS-01) é pouco para fixar números.

## 7. Pendências que dependem do usuário (atualizado 08/11/2026: itens 2 e 3 decididos — D-08 e D-09)
1. ~~**F0-08:** executar `poc/sei-mapa` com login real e colar o RESUMO.~~ **Feito em 08/11/2026** (§3).
2. **Teste Vulkan na RX 6600 XT:** repetir ou não, e como. Opções: (a) não repetir, adotar a GPU segura do §5.1 e validar a GPU em outra máquina; (b) repetir com sonda mínima (modelo 2B, `-ngl 4`, 16 tokens), com o usuário presente e trabalho salvo.
3. Aprovar os ajustes do §5 antes da Fase 1.
