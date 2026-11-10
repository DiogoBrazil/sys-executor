# PLANO — ChatGPT OAuth, cadastro de peças, enquadramento manual e barreira de privacidade (sys-executor)

Branch: `add-mode-apikey-oauth` (limpa; só `logo-cabecalho-docs.jpeg` e `logo-cabecalho-docs.jpeg:Zone.Identifier` não rastreados, e eles **não** serão tocados). Sem push nem merge sem autorização. Commits **sem coautor**. Após a aprovação, este plano é copiado para `docs/planning/PLANO_CHATGPT_PECAS_PRIVACIDADE.md`.

---

## 1. Contexto

- **Origem:** a IA local foi trocada pela Gemini API (D-10, Emenda V4 §47). A Gemini funciona em parte: a delimitação acerta, mas a análise deu 400, e a correção ainda não foi testada pelo usuário.
- **O que o usuário pede agora:**
  1. **Sign in with ChatGPT** (assinatura Plus), implementado em Go;
  2. **cadastro peça por peça** no lugar do PDF monolítico;
  3. **enquadramento manual** das transgressões a partir do RDPM e do Estatuto;
  4. **barreira obrigatória** de privacidade antes do envio.
- **Restrições:** nenhuma regra jurídica muda. O Gemini é preservado.

### Decisões do usuário nesta sessão

| # | Decisão |
|---|---|
| E1 | **Escopo até a análise.** ChatGPT, entrada por peças, enquadramento e pseudonimização/restauração em todas as saídas de IA que já existem (delimitação e análise). O contrato de restauração fica pronto para as F6/F7. Biblioteca (F5), composição, DOCX/PDF, revisão e instalador continuam fora, cada fase com autorização própria. |
| E2 | **SEI:** o app lista os documentos (título e nº SEI, só localmente). O usuário escolhe, o app gera o PDF por "Apenas selecionados", que já está liberado na guarda, e o separa pelo carimbo SEI nas peças declaradas. **Nenhuma ação nova na guarda.** |
| E3 | **Sem rascunho persistente.** A V4 §29 continua valendo: o cadastro vive só na sessão efêmera. |
| E4 | **Pseudonimização:** a política da §47 continua (nº SEI, nº do processo, datas dos fatos e posto/graduação em claro). Acrescentam-se uma **barreira obrigatória** e uma **tela de revisão item a item**. |
| E5 | **Imputações digitadas no assistente servem de conferência.** A IA continua extraindo do Memorando o fato/conduta e a data; o formulário de congelamento (PADS-ESC-001) continua sendo a fonte. |
| E6 | **Entrada monolítica removida.** O assistente aceita só peça por peça. A segmentação por carimbo fica só para o PDF que o próprio SEI gera com os selecionados. |
| E7 | **Enquadramento manual obrigatório.** Botão de opção RDPM/Estatuto e seleção com filtro "artigo/inciso – descrição".<br>• RDPM arts. 15 (leve), 16 (média) e 17 (grave): a natureza é fixa.<br>• Estatuto (DL 09-A/1982) arts. 29 e 32: o usuário escolhe a natureza, por semelhança de intensidade com o RDPM.<br>Um **fato cometido** pode ter vários dispositivos, e **cada dispositivo vira uma imputação** com natureza própria. Imputações do mesmo fato são candidatas a **conexas**: vale a mais grave e as outras viram agravantes; se empatar, o usuário escolhe a que mais se amolda. Isso já é feito por `engine.SugerirPrincipal`/`AplicarConexao` (`internal/rules/engine/conexao.go`), sem nenhuma mudança no motor. |
| E8 | **Divergência** entre o dispositivo escolhido e o citado no Memorando: o app mostra os dois lado a lado, com o trecho e a §22.1, e **exige confirmação explícita**. O app nunca sugere um artigo. |
| E9 | **Momento do enquadramento:** no assistente (etapa do Memorando). Depois, no formulário de confirmação, ele é cruzado com os fatos extraídos pela IA. |

---

## 2. Diagnóstico do código real (resumo da auditoria)

### Preservado e não tocado
- **Motor determinístico:** `internal/rules/engine` e `internal/rules/civil`.
- **Domínio:** `internal/domain/pads` (`ImputacaoSet` congelado, um acusado) e `internal/domain/sr`.
- **Catálogo de regras e prompts:** catálogo de 193 regras, `internal/policy` com os golden em `internal/policy/testdata/golden` (devem ficar **sem diff**).
- **Validadores** da `analise`: conferência literal, `EV-nnnn`, citações legais.
- **Sessão e SEI:** sessão efêmera AES-GCM (`internal/session`); guarda do SEI (`internal/sei/guarda.go`).
- **Gemini:** `internal/inference/gemini*.go`.

### Dependências do PDF único
- `workflow.NomeAutos="autos.pdf"` (`gerenciador.go:20`) e `ReceberUpload([]byte)`.
- `pipeline.classificar` → `segment.Segmentar` (`pipeline.go:134`), único produtor de `d.Pecas`.
- `compat.pecaInicialIlegivel` (procura a peça central entre as "3 primeiras").
- `Entrada.vue` e o seletor `*.pdf` de uma única seleção (`main.go:390`).
- Visualizador e evidências com numeração global de páginas: `app/fontes.go`, `analise/evidencias.go`.

### Camada de IA
- **Porta síncrona** `inference.Motor` (`motor.go`), implementada pelo `MotorGemini`.
- **Gemini fixo em vários pontos:**
  - `FonteGemini` ligada direto em `main.go:80`;
  - `IAService` 100% Gemini;
  - textos dos erros IA-001…013;
  - preferências v2 com regex `^gemini-`;
  - `credencial` com nome de arquivo e entropia fixos;
  - `TransporteRestrito` de origem única.
- **Pseudonimização** (`internal/pseudonimo`): regex com âncoras, mapa só em memória, marcadores `[PESSOA-NN]`, embrulho do `Motor` em `pipeline/ia.go:236`.

### Problemas compartilhados (afetam qualquer provedor; corrigidos neste plano)
1. **Não há barreira antes do envio.** Os campos `Sistema` e `Schema` não passam pelo `Ocultar`.
2. **Formas escapadas em JSON** (`\n`, `\"`) dentro das `Secoes` e no histórico restaurado não casam no `Ocultar`.
3. **Diferenças de caixa e acento:** texto digitado pelo usuário com caixa ou acento diferente vai em claro.
4. **Caixa alta:** a forma em CAIXA ALTA só é aprendida se aparecer na mesma página.
5. **Limite de palavra:** quando a forma longa é rejeitada pelo limite de palavra, a forma curta não é tentada.
6. **Marcador deformado:** `reDeformado` não cobre `CPF-01`, `[pessoa-01]` etc.
7. **Autorização de envio solta:** não fica presa a provedor e modelo, e sobrevive à troca. O consentimento dado ao Google valeria para a OpenAI.
8. **Preferências:** subir para a v3 sem `migrarV2` volta aos valores padrão em silêncio.

### Problemas só do Gemini (registrados; ficam para depois)
- O 400 da análise ainda não foi testado depois de `6dfac51`.
- `Pensamento` nunca é preenchido, o que leva `MaxOutputTokens` ao limite do modelo.
- A classificação de erros depende do texto da mensagem (créditos, "limit: 0").
- O filtro de modelos usa nomes comerciais.
- Não foram calibrados nem a janela nem o nível gratuito.
- O `gofmt` acusa `carimbo_test.go`.
- Há pânico nos workers de extração.

### Lacunas que limitam o escopo
- O motor de cálculo não é chamado por nenhuma etapa.
- A tarefa para em `CONSULTANDO_FONTES` (TAR-004).
- As fases F5–F8 não existem; o DOCX/PDF só existe em `poc/docmodel`.

---

## 3. Sign in with ChatGPT: o que a documentação oficial confirma (09/11/2026)

**Autorização**
- Endpoint: `https://auth.openai.com/api/accounts/authorize`.
- No 1º acesso, `client_id=dynamic_agent_client` com `agent_name_hint=sys-executor`. Nos seguintes, o `client_id` emitido (`oaiapp_…`), com `id_token_hint`/`login_hint`.
- `ext_agent_host_id` obrigatório (`urn:uuid:<v4>`, estável por instalação).
- `redirect_uri=http://127.0.0.1:<porta>/auth/callback` (nunca `localhost`).
- `scope=openid profile email offline_access resource.invoke chatgpt.tokens.use.direct`, `resource=https://api.openai.com/v1`, `state`, `nonce`, PKCE S256.

**Retorno e troca do código**
- O retorno traz `code`, `state`, `client_id` e `scope`. Se não vier `client_id` no registro, o registro está incompleto. Um `client_id` diferente do salvo é rejeitado.
- Troca: POST de formulário para `https://auth.openai.com/api/accounts/oauth/token` com `client_id`, `code`, `code_verifier`, `redirect_uri` e `resource`.

**Validação do ID token**
- Assinatura pelo JWKS de `https://auth.openai.com/.well-known/openid-configuration`.
- `iss == https://auth.openai.com`; `aud` contém o `client_id`; `exp`/`iat`; `nonce` conferido à mão. A identidade é o `sub`.
- **É obrigatório que `chatgpt.tokens.use.direct` esteja entre os escopos concedidos.**

**Tokens**
- Access token: 1 h. Refresh token: 30 dias, **com rotação**. Os refreshes são feitos em série.
- No refresh, mandar `resource` e omitir `scope`.
- Revogação pelo `revocation_endpoint` do discovery.
- Erros de refresh (`invalid_grant`, `refresh_token_reused`…): apagar os tokens, manter o `client_id` e refazer o login.

**Modelos**
- `GET https://api.openai.com/v1/models` → `{models:[{slug, display_name, visibility}]}`. Ficam só os `visibility=="list"`, na ordem do servidor.

**Inferência**
- `POST https://api.openai.com/v1/responses` com Bearer, `store:false` e `stream:true`.
- Instruções vão em `instructions` (mensagens com papel `system` são recusadas).
- **É obrigatório omitir** `max_output_tokens`, `temperature`, `top_p`, `metadata`, `truncation`, `user`, `prompt`, `background`, `conversation` e `previous_response_id`.

**Streaming (SSE)**
- `response.output_text.delta` traz o texto.
- **Só `response.completed` conta como sucesso.**
- `response.failed` e `response.incomplete` são tratados à parte. Fluxo encerrado sem `completed` é falha.

**Erros da API**
- `subscription_sharing_usage_limit_exceeded` (429). **No Plus, o limite de 5 h é compartilhado com todos os apps.**
- `…_user_not_eligible` (403), `…_usage_unavailable` (503), `…_unsupported_capability` (400, com `error.param`), `…_invalid_user` (401), `chatpass_v2_*` (403).
- Antes de o fluxo abrir, 401/403/503 podem vir com corpo `{"detail":…}`.

**Ferramentas e limites**
- Não suportados: Files API, file search, code interpreter e MCP hospedado.
- Function tools são suportadas. **Saída estruturada (`text.format` json_schema) não está documentada**: será testada na PoC.

**Elegibilidade**
- "Disponível a todos os parceiros open source e a clientes privados selecionados". A cartilha cita projetos pessoais locais.
- Os **termos** (`openai.com/policies/sign-in-with-chatgpt-terms`) responderam 403 à leitura automática. **O usuário precisa lê-los.**
- Uso e retenção de dados para treinamento nesse modo **não estão documentados**: registrado como risco.

---

## 4. Arquitetura proposta (componentes e justificativas)

### 4.1 Governança: D-11 e Emenda 2 (V4 §48)

Isto vem **primeiro** e exige a aprovação do texto pelo usuário.

- **D-11 no `REGISTRO_DUVIDAS_DECISOES.md`:** registra E1–E9 e **reabre a parte da D-10** que excluía OAuth e outros provedores.
- **Emenda 2 (§48):**
  1. O ChatGPT (SIWC, plano do usuário) é um 2º provedor. A escolha é explícita, **sem fallback**, e a confirmação de envio fica vinculada ao provedor e ao modelo.
  2. Entrada por peças declaradas (prevalece sobre §§1.1, 2, 2.1, 2.2, 16.1, 17.4, 30.1–30.2, 31.4, 39 e 45-T02…T06). A declaração do usuário define a função da peça, e o conteúdo é cruzado com ela, sem troca silenciosa (§30.2 adaptada).
  3. Enquadramento manual pelo catálogo curado (E7/E8). A PADS-ESC-001 passa a ser: "fatos extraídos pela IA + dispositivos do usuário, conferidos e congelados".
  4. Barreira obrigatória antes do envio (E4).
- **Catálogo** (`internal/rules/catalog/*.yaml`, com `versao+1` ou `status: revogada`; IDs nunca são reutilizados):
  - generalizar TRV-SEG-012 (lista de origens por provedor) e TRV-IA-011 (`enforced_by` de cada motor);
  - TRV-ING-001/002 viram "conferência da função declarada";
  - TRV-UI-001…004;
  - regras novas para OAuth, barreira, cadastro e enquadramento.
- **Artefatos a atualizar:** `MATRIZ_REQUISITOS_V4.md`, `PLANO_TESTES_ACEITACAO.md`, `policies/CHANGELOG.md`, `go run ./cmd/rulescov -atualizar-lock` e `-estrito`. **Os golden não mudam**: a barreira não altera o `Sistema`.

### 4.2 Provedores de IA (o ChatGPT é a prioridade; o Gemini fica intacto)

**Porta.** `inference.Motor` continua síncrona. O adaptador SSE acumula os deltas. Não há streaming na interface (TRV-IA-010).
*Por quê:* a análise, o validador e a pseudonimização já dependem dessa porta, e trocá-la não traz ganho.

**`internal/inference/openai.go`, `openai_sse.go`, `openai_erros.go` (cliente próprio, sem o SDK openai-go)**
- O corpo é montado à mão: só `model`, `instructions`, `input`, `store:false`, `stream:true` e, se a PoC aprovar, `text.format`/`reasoning`.
- *Por quê:* garante que os campos proibidos nunca saiam. Evita as repetições automáticas do SDK. Exige pouco código e nenhuma dependência nova.
- `Truncada` ← `response.incomplete`. `Pedido.MaxTokens` e `Temperatura` são ignorados de propósito.
- Os códigos de erro são mapeados para IA-*. **Nada de repetir** em `usage_limit_exceeded`, `not_eligible` e `unsupported_capability`.
- 503 e erros de rede: até 3 tentativas com espera crescente, limitada.
- Prazo por **inatividade do fluxo**, em vez de prazo total.

**Saída estruturada (decidida na PoC)**
- **(1)** `text.format` json_schema strict, com o schema transformado só para a OpenAI: sem `maxLength`, e `enum:[""]` tratado. Os tamanhos e enums continuam sendo conferidos em Go.
- **(2)** Se (1) for recusado: JSON pedido no prompt, com a decodificação estrita e os reenvios que já existem (`analise/chamada.go`).
- Uma function tool forçada **não** entra sem decisão do usuário, porque a §47.3 diz "sem ferramentas".

**Rede: `internal/rede`**
- `TransporteRestrito`/`novoClienteHTTP` saem de `gemini_rede.go` para um pacote próprio.
- Um cliente de **origem única** para cada destino: `api.openai.com` e `auth.openai.com`. Sem redirecionamento, TLS 1.2 ou superior.
- Cada transporte ganha um gancho `Inspecionar(corpo)`, que é a camada 2 da barreira (§4.5).

**Generalização**
- `pipeline.FonteMotor` ganha `Alvo() (provedor, modelo string, err error)`.
- `FonteGemini` e a nova `FonteChatGPT` ficam em `internal/app/ia.go`. O provedor é escolhido em `main.go`, conforme as preferências.

### 4.3 OAuth: `internal/chatgpt`

- **`login.go`**
  - Servidor de retorno em `net.Listen("tcp","127.0.0.1:0")`, com um único `GET /auth/callback`.
  - Confere `Host`, `state`, e recusa um segundo retorno.
  - Encerra após o retorno, após 5 min, ao cancelar e em `OnShutdown`.
  - O navegador do sistema é aberto por `wapp.Browser.OpenURL`, **nunca na WebView**.
  - A página de retorno não carrega nada de fora.
- **`tokens.go`**
  - PKCE com `crypto/sha256`, sem `x/oauth2`: o refresh precisa de `resource` e de rotação controlada, coisas que o `TokenSource` não oferece.
  - Troca e refresh com um `sync.Mutex` por sessão. A gravação do novo refresh token é atômica e acontece **antes** do uso.
- **`idtoken.go`**
  - `github.com/coreos/go-oidc/v3` com `oidc.ClientContext(ctx, clienteRestrito)`: assinatura via JWKS, `iss`, `aud` e `exp`.
  - O `nonce` é conferido à mão.
  - Os escopos concedidos são exigidos.
  - Numa relogin, o `sub` precisa coincidir com o anterior; outra conta exige "desconectar" antes.
- **Armazenamento**
  - `credencial.NovaLojaNomeada(dir, "chatgpt.dpapi", entropia própria)`, com DPAPI no escopo do usuário e ACL da pasta.
  - O blob contém `{versao, host_id, client_id, sub, email, scopes, refresh_token, id_token}`.
  - O **access token fica só em memória**.
  - O `ext_agent_host_id` fica nesse blob, e não nas preferências.
  - "Desconectar" revoga o token e apaga o arquivo. Uma falha de rede **não** apaga credenciais.
- **Proteções**
  - Os tokens nunca vão ao Vue nem a URLs de páginas.
  - `sanear` também mascara `Bearer …` e `refresh_token` nas causas de erro (`SYSEXEC_DIAGNOSTICO`).

### 4.4 Interface de IA (`frontend/src/views/ConfiguracoesIA.vue`, `api.ts`, `IAService`)

- **Abas "ChatGPT (plano)" e "Gemini (chave)"**, com a escolha explícita do provedor ativo.
- **Na aba ChatGPT:**
  - botão "Continuar com ChatGPT", com cancelamento;
  - estado da conta (e-mail; permissão de uso do plano: sim/não);
  - lista de modelos, teste e escolha;
  - aviso "Usando seu plano ChatGPT", com link "Gerenciar uso" (Configurações › Uso do ChatGPT) e a observação sobre o limite de 5 h compartilhado;
  - botão "Desconectar".
- **Assincronia:** `IniciarLogin`/`CancelarLogin` retornam na hora e o resultado chega pelo evento `ia:visao`. Nenhum método devolve `error` (padrão de `tarefa.go`).
- **Preferências v3** (`internal/prefs`):
  - campos `provedor_ia` (`""|GEMINI|CHATGPT`) e `modelo_chatgpt`;
  - migração **`migrarV2`** explícita (v2 com modelo → `GEMINI`) e `migrarV1` → v3;
  - testes de cada caminho.
- **Erros**
  - IA-001…013 ganham texto que depende do provedor.
  - Códigos novos (números definidos na implementação; nada é reutilizado):
    - conta ChatGPT não conectada;
    - login cancelado ou negado;
    - permissão de uso do plano não concedida;
    - sessão expirada ou revogada (reconectar);
    - limite do plano atingido;
    - conta ou plano não elegível;
    - envio bloqueado pela barreira de privacidade.

### 4.5 Barreira de privacidade e restauração (`internal/pseudonimo`, `pipeline/envio.go`, `pipeline/ia.go`)

**Semente do mapa**
- Pessoas cadastradas no assistente: acusado, sindicados, ofendidos, depoentes, pessoas ligadas a documentos, com "outras grafias/apelidos" opcionais.
- Mais o que a detecção atual aprende de todas as páginas.
- O formato `[PESSOA-NN]` é mantido, porque já é testado e restaurado byte a byte.
- **O mapa nunca sai da memória**: não é gravado, não vai a log nem a mensagens de erro.

**`Ocultar` robusto**
- Casamento que ignora caixa e acento, via `texto.Alinhado`, com `\s+` entre os tokens.
- Cada grafia real nova vira uma variante `[PESSOA-NN.n]`, para a restauração continuar exata.
- Formas escapadas em JSON também são registradas.
- A forma curta é tentada quando a longa falha no limite de palavra.
- A forma em CAIXA ALTA é aprendida no documento inteiro.
- `reDeformado` passa a cobrir todos os tipos e as minúsculas.

**Camada 1 (`pseudonimo.Motor.Gerar`, depois de `Ocultar` e antes do envio)**
- `Usuario` e `Historico`: identificador conhecido que sobrou → **bloqueio sem exceção**. Detectores (CPF válido, e-mail, telefone, RG/matrícula rotulados) e nomes com âncora fora do mapa → `ErrRevisao{itens}`.
- `Sistema` e `Schema`: só a busca por identificadores conhecidos.
- `ErrRevisao` vira uma pergunta com **decisão por item**, usando `Pergunta.Grupos` + `CampoEscolha` + `Fontes`: "é pessoa → ocultar" / "não é dado pessoal". Não existe "ignorar tudo".

**Camada 2 (gancho `Inspecionar` do transporte, nos dois provedores)**
- Decodifica o corpo JSON exato que vai sair e percorre todas as strings, procurando as formas conhecidas normalizadas e verificando a integridade dos marcadores.
- Só bloqueia, sem revisão. É a garantia final sobre os bytes enviados.

**Revisão na autorização**
- Varredura das páginas já ocultadas, uma vez, com uma tela que lista os suspeitos (sem despejar o texto).
- Depois disso, `perguntaEnvio` mostra provedor, modelo, peças, páginas, tokens estimados e o resumo da pseudonimização.

**Autorização vinculada**
- Fica presa a `(provedor, modelo, versão do mapa)`. Se qualquer um mudar, `Esquecer` e perguntar de novo.

**Restauração**
- Mantém o caminho atual: a resposta é restaurada no embrulho, um marcador desconhecido gera erro (IA-010) e **nunca há adivinhação**.
- Teste-contrato: nenhum marcador pode sobrar em `Dados.IA`. É a base que as F6/F7 vão consumir para montar a minuta com os dados reais.

### 4.6 Catálogo de enquadramento: `internal/rules/enquadramento` (novo)

**Arquivo `dispositivos.yaml`, embutido no executável**
- Curado a partir de `docs/law_library/RDPM-RO.pdf` (arts. 15, 16 e 17, todos os incisos) e de `DECRETO-LEI, Nº 09-A…pdf` (arts. 29 e 32, incisos e alíneas).
- Cada item: `{id, diploma, artigo, inciso, alinea, texto_vigente, natureza (fixa no RDPM; vazia no Estatuto), fonte: arquivo+página, redacao_dada_por}`.
- **Só a redação vigente.** Exemplo: art. 29, XVIII, alínea e, com a redação da Lei 1.781/2007.
- O usuário revisa a curadoria, que fica registrada no CHANGELOG.

**Testes**
- IDs únicos; contagem de incisos por artigo; natureza coerente com o artigo do RDPM; nenhum item do Estatuto com natureza fixa.

**Busca e entrega à interface**
- `Buscar(diploma, termo)` normalizado (`texto.Termos`).
- Exposto à interface pela visão do assistente.

### 4.7 Cadastro de peças e assistente (`internal/workflow`, `internal/pipeline`, `internal/ingest`, frontend)

**Modelo em memória (Go tipado), em `workflow.tarefa`**
- Exposto por `Contexto.Cadastro()`. **Não** fica em `pipeline.Dados`, porque `liberarDados` o apagaria.

```
Cadastro{Tipo PADS|SR; Pessoas[]{ID PES-nn, Nome, Grafias[], PostoGrad, Militar bool, Papeis[]};
  PADS: Acusado PES-ID; Fatos[]{ID FATO-n, Resumo?, Data?, Enquadramentos[]{DispositivoID, Natureza}};
  SR: Portaria{Numero, Ident}; Sindicados[]PES-ID; Ofendidos[]PES-ID | SemOfendidoIndividualizado;
  Pecas[]{ID PEC-nn, Tipo, Nome, Descricao, PessoaIDs[], Data?, SEI?, NaoApresentada bool,
          Arquivos[]{ID ARQ-nn, Nome, SHA256, Paginas, Origem UPLOAD|SEI}}}
```

**Correspondência tipo → `segment.Funcao`** (o enum atual basta)

| Tipo declarado | `segment.Funcao` |
|---|---|
| memorando | `MemorandoDisciplinar` |
| portaria | `PortariaSR` |
| defesa prévia | `DefesaPrevia` |
| alegações finais | `AlegacoesFinais` |
| depoimento de testemunha | `TermoInquiricao` |
| depoimento de declarante | `TermoDeclaracoes` |
| depoimento de sindicado | `Interrogatorio` |
| outros | `Anexo` |

- O papel e a pessoa ficam no cadastro, fora do enum.
- Cada oitiva é uma peça própria, ligada a uma `PES-ID`, o que permite várias oitivas da mesma pessoa.

**Workflow**
- Novo estado **`MONTANDO_AUTOS`** no lugar de `RECEBENDO_UPLOAD`. Pontos a trocar:
  - `estados.go:41/70/83/85/132`;
  - `gerenciador.go:335/356/384/498/649`;
  - `visao.go:100/123`;
  - `tarefa.go:73`;
  - os destinos "Enviar outro PDF" em `pipeline.go:104/173` e `delimitar.go:90`, que passam a voltar ao assistente **preservando o cadastro**.

**Comandos** (no `Gerenciador` e expostos pelo `TarefaService`)
- `AdicionarPeca`, `AtualizarPeca`, `RemoverPeca`, `MoverPeca`, `AnexarArquivos(peca)`, `RemoverArquivo`, `MoverArquivo`, `SubstituirArquivo`.
- Pessoas e fatos com os comandos equivalentes.
- `ConcluirCadastro`, com validação das obrigatórias: memorando ou portaria; acusado ou sindicados; defesa e alegações apresentadas ou marcadas como "não apresentada".
- O seletor nativo é de múltiplos arquivos (`PromptForMultipleSelection`), só PDF na 1ª versão.
- Os bytes são lidos no Go e gravados cifrados na sessão. O frontend vê só nome, nº de páginas e situação.
- Duplicata detectada por SHA-256, com alerta.

**Junção e extração**
- **Junção em `VERIFICANDO`:** `pdf.Juntador` (interface nova, para não quebrar os falsos de teste). Uma única instância do pdfium abre os N arquivos e chama `FPDF_ImportPages`, depois `FPDF_SaveAsCopy` em memória, gravando `autos.pdf`.
  - *Por quê:* `extrair`, `Indexar`, `Blocos`/`Ponto`, `FontesService` e o PDF.js ficam **inalterados**, e a numeração global continua contígua. As funções existem no go-pdfium v1.21.4 wasm. Há um limite de tamanho **total** (memória wasm de 32 bits).
- **`classificar`:** `segment.DeDeclaracao(autos, cadastro)` com `Base=USUARIO`, em vez de `Segmentar`. `ClassificarResiduais` já ignora essas peças, porque só pega `Indeterminada`+`BaseNenhuma`. O `Peca.SEI` vem do carimbo dominante, quando houver.
- **`compatibilidade`:** `segment.VerificarDeclaradas` substitui a heurística das "3 primeiras". Ela mede a legibilidade da peça central declarada e cruza o conteúdo (`pistas`/carimbo → `classificar`). Divergência vira pergunta, nunca troca silenciosa.

**Delimitação PADS** (`pipeline/delimitar.go` `perguntaImputacoes`/`congelarImputacoes`)
- A IA continua extraindo conduta, data e o dispositivo **citado** (`DelimitarPADS`, que não muda).
- O formulário gera **um grupo por dispositivo do usuário**, com:
  - conduta extraída da IA, editável, com a fonte;
  - dispositivo e natureza do catálogo (natureza só editável no Estatuto);
  - data;
  - o `FATO-n` de origem.
- O pareamento usa o dispositivo citado.
  - Dispositivo do usuário sem correspondência → alerta da §22.1 e confirmação explícita.
  - Dispositivo citado e não cadastrado → alerta com escolha "incluir pelo catálogo" ou "não incluir", com confirmação.
  - Acusado divergente → alerta.
- O congelamento continua por `pads.ImputacaoSet`, uma imputação por dispositivo.
- O agrupamento por fato vira **sugestão** de grupo de conexão na confirmação de conexão que já existe. O usuário confirma; o motor não muda.

**Delimitação SR**
- Sindicados e ofendidos cadastrados são cruzados com os extraídos (`DelimitarSR`) e as divergências aparecem como alertas no formulário de `montarObjeto`.
- Ofendido é dado do cadastro e do mapa de pseudônimos. O domínio `sr` não muda.

**Frontend**
- `views/Montagem.vue` (assistente com passos por modalidade, sem router, no padrão de `telaAtual()`) e componentes `PassoPeca.vue`, `SeletorDispositivo.vue` (botão de opção + caixa com filtro + natureza) e `ListaConferencia.vue`, usando os tokens de `estilo.css`.
- Passos do PADS: Memorando (+ acusado + fatos/enquadramentos) → Defesa prévia → Depoimentos → Alegações finais → Outros → Conferência.
- Passos da SR: Portaria (+ sindicados/ofendidos) → Depoimentos → Outros → Conferência.
- Situações reais por arquivo: pendente / lido / erro / duplicado. **Não há percentuais fictícios.**
- `Entrada.vue` perde o cartão "Enviar o PDF do processo"; ficam "Cadastrar peças" e "Buscar no SEI".

### 4.8 SEI (`internal/sei`, `pipeline/sei.go`, `workflow/pedido.go`, `testutil/seisim`)

> **[Revisado em 10/11/2026 — E2 revisado]** O mapeamento real mostrou que "Apenas selecionados" exige a tela `documento_selecionar`, fora da guarda. Foi implementada a opção A: gerar "Todos", separar pelo carimbo no computador e o usuário escolher os documentos no cadastro. Ver `RELATORIO_ETAPA7_SEI.md`. Os itens abaixo ficam como registro do plano original.

1. **Mapeamento pelo usuário, antes de codificar.** Rodar `poc/sei-mapa` no SEI real (só estrutura) para mapear a tabela de documentos da tela "Gerar Arquivo PDF do Processo" (`procedimento_gerar_pdf`, já liberada na guarda).
2. **`jsDocumentos`:** devolve JSON `{sei, titulo, tipo}`. Lê a tabela da própria tela de gerar, e não a árvore, porque é a mesma tela da seleção e evita carregar pastas da árvore com ações fora da lista permitida. **Isso quebra o invariante "só booleanos" de `telas.go:128-132`**: os títulos ficam só no app e não vão à IA. Isso é registrado no ADR e no catálogo.
3. **`PedidoSelecao` com conteúdo** em `PedidoVisao`. O usuário marca os documentos; o app marca as caixas pela página (o JS do SEI preenche `hdnDocumentosApenas`), gera, baixa e **fecha o navegador**.
4. **Separação e classificação depois.** O PDF baixado é separado pelo carimbo (`autos.LerCarimbos`) em arquivos virtuais por nº SEI. O usuário agrupa e classifica no assistente sem pressa, porque o navegador já fechou e a sessão SEI não pode expirar no meio. O tipo vem pré-preenchido pelo título (`segment.classificar`, exportado).
5. **Simulador:** `seisim` passa a aceitar `rdoTipo=A`, a tabela de documentos e um PDF só com os selecionados.

### 4.9 Desempenho e observação

- **Sem módulo de logs.** A V4 §1.2/§33 proíbe logs técnicos.
- **Métricas só em memória**, na tela de Configurações (uso da última tarefa) e no `cmd/bench`: tempo de extração por arquivo, de junção, de pseudonimização e da barreira, páginas, peças, tokens estimados, chamadas e tempo de inferência por provedor.
- **`cmd/bench`** ganha `-provedor gemini|chatgpt` e uma **métrica de vazamento**: nenhum nome dos casos sintéticos pode aparecer em nenhum pedido.
- **Comparação monolítico × peças**, com os mesmos casos sintéticos, antes de afirmar qualquer ganho.

---

## 5. Ordem de implementação

Cada etapa é **autorizada separadamente** e termina com CI verde (`go test -race`, `GOOS=windows go vet`/build, Vitest/vue-tsc/vite, `rulescov -estrito`, `licencas`). A cada etapa também se atualizam o relatório e o `CONTINUIDADE_*.md`.

| Etapa | Conteúdo | Arquivos principais | Portão |
|---|---|---|---|
| **0** | D-11 + **texto da Emenda 2 (§48)** para aprovação; atualização do catálogo, da matriz e do lock depois do aval. Leitura dos termos do SIWC pelo usuário | `docs/rules/…V4.md`, `REGISTRO_DUVIDAS_DECISOES.md`, `internal/rules/catalog/*.yaml`, `policies/{CHANGELOG.md,V4.lock}` | Usuário aprova o texto |
| **1** | **PoC isolada `poc/chatgpt`** (módulo Go separado, dados sintéticos). Itens: registro dinâmico → `client_id`; login; ID token/JWKS/nonce; escopos; refresh com rotação; revogação; `/v1/models`; inferência SSE até `response.completed`; `text.format` strict com os schemas reais de `analise/schema.go`; `reasoning`; porta aleatória no relogin | `poc/chatgpt/*`, `docs/planning/RELATORIO_POC_CHATGPT.md` | Usuário aprova a PoC; nada no pipeline antes disso |
| **2** | **Barreira + correções compartilhadas da pseudonimização** (protege também o Gemini) | `internal/pseudonimo/{mapa,motor,detector,barreira}.go`, `internal/pipeline/{envio,ia}.go`, `internal/rede` | Testes de vazamento verdes |
| **3** | **Generalização multiprovedor:** `FonteMotor.Alvo`, autorização vinculada, prefs v3 + `migrarV2`, `credencial.NovaLojaNomeada`, erros neutros e novos | `internal/{pipeline,prefs,credencial,erros,app}` | Gemini sem regressão |
| **4** | **Provedor ChatGPT:** `internal/chatgpt`, `inference/openai*.go`, `FonteChatGPT`, `IAService` com abas, `ConfiguracoesIA.vue`, bindings | idem + `cmd/sys-executor/main.go`, `frontend/src/{api,estado}.ts` | Login e teste de modelo no Windows (usuário) |
| **5** | **Catálogo de enquadramento** + formulário PADS-ESC-001 com pareamento e alertas da §22.1 (funciona também no fluxo atual) | `internal/rules/enquadramento/*`, `internal/pipeline/delimitar.go`, `SeletorDispositivo.vue` | Usuário revisa a curadoria |
| **6** | **Cadastro de peças (upload)** + junção + `DeDeclaracao` + `VerificarDeclaradas` + semente do mapa | `internal/workflow/*`, `internal/ingest/{pdf,segment}`, `internal/pipeline/pipeline.go`, `internal/app/tarefa.go`, `Montagem.vue`, `Entrada.vue` | Casos sintéticos PADS/SR até TAR-004 |
| **7** | **SEI pela lista de selecionados** (depois do mapeamento feito pelo usuário) | `internal/sei/{cliente,telas}.go`, `internal/pipeline/sei.go`, `internal/workflow/pedido.go`, `internal/testutil/seisim` | Simulador verde + teste real do usuário |
| **8** | **Bench e regressão** nos dois provedores; ADR 010; README; continuidade | `cmd/bench`, `docs/planning/*` | Limiares do PLANO_TESTES §9 |

**Por que esta ordem**
- A barreira (2) vem antes do provedor novo: é independente de provedor e reduz o risco já no Gemini.
- O enquadramento (5) vem antes do assistente (6): pode ser validado no fluxo atual e o assistente só passa a usá-lo.
- O SEI (7) depende de um mapeamento real que só o usuário pode fazer.
- As etapas 5 e 6 podem ser feitas em paralelo com as 3 e 4.

---

## 6. Verificação

### Testes automatizados (no padrão que já existe)
- **`inference/openai_test.go`** (servidor `httptest` SSE, como `gemini_test.go`):
  - deltas → completed;
  - incomplete → `Truncada`;
  - cada `error.code` → IA-*;
  - fluxo encerrado sem completed → erro;
  - 401 → refresh + 1 nova tentativa;
  - corpo sem `max_output_tokens`/`temperature`/`top_p` e com `store:false`;
  - host fora da origem → bloqueio.
- **`chatgpt/*_test.go`** (discovery, JWKS e servidor de token falsos):
  - PKCE;
  - `state`/`nonce`/`aud` errados;
  - escopo `chatgpt.tokens.use.direct` ausente;
  - `sub` diferente;
  - retorno duplicado;
  - prazo esgotado e cancelamento;
  - 2 refresh concorrentes → 1 POST;
  - `refresh_token_reused` → apaga os tokens e mantém o `client_id`;
  - tokens ausentes de qualquer `ErroVisao`.
- **`credencial`:** entropias diferentes não se decifram (job `go-windows`).
- **`prefs`:** v3, v2 com e sem modelo, v1, chave desconhecida.
- **`pseudonimo`:**
  - nome completo, abreviado, apelido/grafia cadastrada, caixa e acento, quebra de linha;
  - CPF, telefone, e-mail, matrícula;
  - homônimos parciais (pessoas diferentes com nomes parecidos não se fundem);
  - forma escapada em JSON;
  - barreira em `Sistema`/`Schema`/`Historico`;
  - camada 2 sobre os bytes;
  - `ErrRevisao`;
  - marcador deformado/desconhecido → erro;
  - nenhum marcador restante em `Dados.IA`.
- **`pdf`:** junção de 2 ou 3 PDFs `pdfsint` → páginas, texto e numeração; PDF corrompido ou protegido.
- **`segment`:** `DeDeclaracao`/`VerificarDeclaradas` (central ilegível, conteúdo divergente, `BaseUsuario` ignorado).
- **`workflow`:**
  - comandos do cadastro;
  - duplicata, substituição, reordenação;
  - defesa ou alegações "não apresentadas";
  - várias oitivas da mesma pessoa;
  - vários sindicados, sem ofendido;
  - voltar ao assistente preservando o cadastro.
- **`pipeline`** (com `casos`): fluxo completo com um `Motor` falso que **registra todos os pedidos** e confirma que nenhum nome dos casos aparece neles; troca de provedor ou modelo → nova pergunta de envio; formulário PADS com pareamento e alertas da §22.1.
- **`enquadramento`:** integridade do catálogo e busca.
- **`seisim`:** lista, seleção, PDF parcial, separação por carimbo.
- **`telas.test.ts`:** abas de provedor, login assíncrono, assistente PADS/SR, seletor de dispositivo, revisão da barreira.
- **Regressão jurídica:**
  - golden **sem diff**;
  - `rulescov -estrito`;
  - testes A18–A30, PADS-E01…E05, SR01–SR36 e R1–R5 (motor) inalterados e verdes;
  - casos sintéticos com o resultado esperado (ex.: `pads-02`: IMP-1 procedente e IMP-2 improcedente).

### Manual no Windows (usuário; só dados sintéticos)
- Login ChatGPT: 1º acesso, relogin, cancelamento, negar a permissão do plano, desconectar.
- Escolha de modelo; tarefa PADS e SR sintéticas até TAR-004 com ChatGPT e com Gemini.
- Sem rede (IA-007); limite do plano.
- SEI sintético ou real com "selecionados".

---

## 7. Riscos

| Risco | Tratamento |
|---|---|
| **Elegibilidade e termos do SIWC:** repositório ainda privado; termos não lidos automaticamente; uso institucional | PoC antes de tudo; o usuário lê os termos; o app solicita a autorização real e confere os escopos. Publicar o código **não** autoriza enviar autos da PMRO a terceiros: isso depende de autorização institucional, que é do usuário |
| Retenção e treinamento de dados no modo SIWC não documentados | Registrar na Emenda; confirmação por tarefa; barreira; nunca prometer apagamento externo (V4 §1.1-9) |
| `text.format` strict recusado ou com limites (enums grandes de `EV-nnnn`) | Plano B: JSON pelo prompt com decodificação estrita e reenvios; medir no bench |
| Limite de 5 h do Plus numa tarefa com dezenas de chamadas | Erro próprio sem nova tentativa, tarefa preservada e retomável; medir chamadas no bench |
| Ausência de `max_output_tokens` (respostas longas) | Instrução de concisão (já existe); `incomplete` → `Truncada` → reenvio |
| Memória do wasm do pdfium na junção | Limite total e teste com arquivo grande |
| Tabela do SEI desconhecida | Mapeamento real antes da etapa 7; a guarda não é ampliada |
| Pseudonimização continua sendo de melhor esforço (identificação indireta: local, cargo, datas) | Revisão item a item; o texto da interface não promete anonimato |
| Curadoria dos dispositivos (redações sucessivas) | Revisão humana registrada no CHANGELOG; fonte com arquivo e página por item |

---

## 8. Decisões que ficam para depois (não bloqueiam as etapas 0–2)

1. **PADS sem oitiva do acusado:** o assistente do PADS lista testemunha e declarante. O interrogatório ou qualificação do acusado entra como "Outros documentos" ou deve ter um tipo próprio? (Confirmar na etapa 6.)
2. **Imagens (JPG/PNG) como peça:** a 1ª versão aceita só PDF. Converter imagens em páginas de PDF (`FPDFImageObj_LoadJpegFileInline`) fica como evolução, se o usuário quiser.
3. **Function tool**, se a saída estruturada falhar e o JSON pelo prompt ficar abaixo do limiar: exige decisão sobre a §47.3.
4. **Pendências do Gemini** (§2): continuam registradas, sem prioridade nesta tarefa.

---

## 9. Critérios de aceitação deste plano (escopo E1)

- **OAuth e provedor:**
  - OAuth oficial em Go, sem auth2api, Node, Codex CLI nem endpoints privados;
  - escopos conferidos e tokens protegidos por DPAPI, fora do Vue e de logs;
  - inferência pela Responses API com `store:false`/`stream:true` e sucesso **só** com `response.completed`;
  - modelos dinâmicos, com aviso quando o modelo escolhido sumir.
- **Gemini e IA local:**
  - Gemini preservado e sem regressão; troca de provedor só explícita, sem fallback;
  - nenhum LLM local reintroduzido.
- **Cadastro e SEI:**
  - cadastro PADS/SR peça por peça, com várias peças e vários arquivos, IDs únicos e vínculo com pessoas;
  - SEI por selecionados, sem download manual.
- **Enquadramento:** por catálogo, natureza fixa no RDPM e escolhida no Estatuto, divergência da §22.1 confirmada; uma imputação por dispositivo, com a conexão pelo motor existente.
- **Privacidade:**
  - barreira bloqueia qualquer identificador conhecido que sobrar; o mapa nunca sai do computador;
  - restauração determinística, com ID desconhecido gerando erro e nenhum marcador sobrando.
- **Regressão:** regras jurídicas, cálculos e golden inalterados; `rulescov -estrito` verde.
- **Fora deste plano (E1):** DOCX/PDF, composição da minuta e instalador seguem para as F5–F8.
