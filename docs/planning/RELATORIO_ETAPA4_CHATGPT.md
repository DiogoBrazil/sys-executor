# RELATÓRIO — Etapa 4 do plano D-11: provedor ChatGPT (Sign in with ChatGPT)

- **Data:** 09/11/2026 · **Branch:** `add-mode-apikey-oauth`
- **Autorização:** usuário, 09/11/2026 ("sim").
- **Regras:** V4 §§48.3, 48.6, 48.7.
  - Catálogo: TRV-SEG-012 v2; novas TRV-SEG-014, 015 e 016 e TRV-IA-015.
  - Erro novo: IA-021.
- **Base:** a PoC (`RELATORIO_POC_CHATGPT.md`) definiu os detalhes: schema strict aceito, erros `{"detail":…}`, SSE sem `Content-Type`, `invalid_grant` para conta sem plano.

## 1. Componentes

| Componente | O que faz |
|---|---|
| `internal/chatgpt` (`Conta`) | **Login:** navegador do sistema (`wapp.Browser.OpenURL`, nunca a WebView); Authorization Code com PKCE S256, `state` e `nonce`; retorno só em `127.0.0.1:<porta aleatória>` (um GET, `Host` conferido, segundo retorno → 409); registro dinâmico como **sys-executor**; relogin com o `client_id` salvo, `id_token_hint`/`login_hint` e o mesmo `ext_agent_host_id`.<br>**ID token:** `go-oidc/v3` (JWKS, emissor, destinatário, validade) e `nonce`. Outra conta no relogin é recusada sem trocar a credencial.<br>**Permissão do plano:** sem ela, a conta fica conectada e **não gera texto** (IA-017).<br>**Credencial:** DPAPI em `credenciais\chatgpt.dpapi`, com entropia própria; **access token só em memória**.<br>**Renovação:** em série, com o refresh novo gravado antes do uso. Refresh morto apaga só os tokens e pede reconexão (IA-018). Falha de rede não apaga nada.<br>**Desconectar:** revoga pelo `revocation_endpoint` da descoberta e apaga a credencial; se a revogação não for confirmada, a tela avisa. |
| `inference.MotorChatGPT` | Responses API com corpo restrito (`model`, `instructions`, `input`, `store:false`, `stream:true`, `text.format` json_schema **strict com o schema inteiro**, `reasoning` opcional); SSE próprio, sem SDK.<br>Só `response.completed` vale; `incomplete` por tamanho → `Truncada`; por filtro → IA-009.<br>Erros pelos códigos documentados: limite do plano IA-019 e conta não elegível IA-020, **sem nova tentativa**; capacidade não suportada → INT-001; 401 → renovação e uma nova tentativa.<br>Inatividade → IA-008; barreira de bytes → IA-014; destino só `api.openai.com`. |
| `inference.ListarModelosChatGPT` / `TestarModeloChatGPT` | Lista da conta (`visibility=list`, `supported_in_api`), na ordem do servidor, com contexto e níveis de raciocínio. O teste envia uma frase curta, sem autos. |
| `app.FonteChatGPT` | Alvo: conta conectada, plano autorizado e modelo escolhido, conferidos sem rede (IA-015/017/018/004). Abre o motor exatamente para o alvo autorizado. |
| `app.IAService` (ChatGPT) | `SelecionarProvedor`, `EntrarChatGPT` (assíncrono; o resultado vai pelo evento `ia:visao`), `CancelarEntradaChatGPT` (também no fechamento do app), `DesconectarChatGPT`, `AtualizarModelosChatGPT`, `SelecionarModeloChatGPT` (grava o provedor), `TestarModeloChatGPT`, `AbrirUsoChatGPT`. **A visão nunca contém tokens.** |
| Pipeline (§48.6) | Confirmação de envio ao ChatGPT com a pergunta **obrigatória** sobre dados de criança menor de 13 anos. Resposta diferente de "não" → IA-021; o ChatGPT fica **bloqueado na tarefa**, sem envio, e a análise só segue com outro serviço, com nova confirmação. |
| Interface | Configurações de IA com o **serviço usado nas análises** (escolha explícita e selo "Usando seu plano ChatGPT") e abas **ChatGPT** e **Gemini API**. Na aba ChatGPT, o **aviso de privacidade** precisa ser confirmado antes do botão "Entrar com ChatGPT" (só texto, sem logotipo). Ela traz ainda: espera com "Cancelar", conta, permissão do plano, modelos (usar, testar), "Gerenciar uso no ChatGPT", "Desconectar" com confirmação e uso da última tarefa. O aviso de primeiro uso vale para o serviço escolhido. |

## 2. Verificação

| Verificação | Resultado |
|---|---|
| `go test -race` (internal, bench, rulescov) | Verde |
| `GOOS=windows go vet ./...`, build `-H windowsgui` | Verde |
| `rulescov -estrito` | 199 regras; 124/124 IDs da V4 |
| `cmd/licencas` | 45 módulos Go e 40 npm; entram `go-oidc/v3`, `go-jose/v4` (Apache-2.0) e `x/oauth2` (BSD-3-Clause) |
| Golden | Sem diff |
| Vitest (26), `vue-tsc`, `vite build` | Verde |
| Testes novos | **chatgpt** (emissor OIDC falso com JWKS RSA): login completo; relogin e outra conta; negado, state forjado, nonce errado, invalid_grant, sem client_id, sem plano; retorno duplicado; cancelamento; renovação concorrente com 1 POST e rotação gravada; refresh morto; desconectar com revogação.<br>**inference** (SSE falso): corpo restrito; fluxo (sem completed, incompleto, filtro, limite, indisponível, vazio); erros HTTP nos dois formatos; 401 → renovação; inatividade e cancelamento; barreira e endpoint privado bloqueado; listagem.<br>**app:** alvo do ChatGPT; tela do ChatGPT; entrada que falha; montagem sem ChatGPT.<br>**pipeline:** dados de criança bloqueiam o ChatGPT.<br>**frontend:** aviso de privacidade antes de entrar; botão só texto; modelos; desconectar; escolha do serviço; plano não autorizado. |

## 3. Teste manual no Windows (09/11/2026, conta Plus do usuário)

- **Conexão e modelo:** conta conectada; uso do plano autorizado; modelo `gpt-6-luna` escolhido e testado.
- **Tarefa PADS** com `pads-sintetico-texto.pdf`:
  - a confirmação mostrou "ChatGPT (OpenAI), pelo seu plano, modelo gpt-6-luna" e a pergunta sobre crianças, respondida "Não";
  - delimitação correta;
  - **IMP-1 procedente** e **IMP-2 improcedente**, com fundamentos e fontes conferidos e lacunas apontadas;
  - parada em **TAR-004**, como previsto.
- **Caixa diferente:** o nome do acusado digitado em minúsculas na conduta ("Carlos Inventado") foi ocultado como variante do conhecido ("CB PM CARLOS INVENTADO") e restaurado, sem revisão.
- **Defeito encontrado e corrigido:** a IA comentou os marcadores na redação ("O marcador CARLOS INVENTADO identifica a mesma pessoa que Carlos Inventado"). A `InstrucaoMarcadores` passou a proibir comentar, explicar ou mencionar marcadores, inclusive dizer que dois são a mesma pessoa. É só a instrução de pseudonimização; os golden não mudam.

## 4. Pendente
- Confirmação real do 400 do Gemini (etapa 2, §3-A), quando a cota gratuita renovar.
- Etapas 5 a 8 do plano: enquadramento, cadastro de peças, SEI por selecionados, bench e regressão.
