# ADR 010 — ChatGPT como segundo provedor, cadastro de peças e barreira de privacidade

- **Status:** Aceita (10/11/2026). **Complementa o ADR 009** (Gemini API), que continua valendo.
- **Origem:** D-11 (decisões E1 a E9), Emenda 2 da V4 (§48), plano `PLANO_CHATGPT_PECAS_PRIVACIDADE.md`, etapas 0 a 8.
- **Contexto:** o usuário pediu quatro mudanças:
  - usar o próprio plano ChatGPT (Plus) como alternativa à chave do Gemini;
  - entregar os autos peça por peça, no lugar do PDF monolítico;
  - informar o enquadramento das transgressões a partir do RDPM e do Estatuto;
  - ter a garantia de que nenhum dado pessoal conhecido sai do computador.
  
  Nenhuma regra jurídica muda.

## Decisão

### 1. Dois provedores, escolha explícita, sem troca automática
- A porta `inference.Motor` continua **síncrona** e única. `MotorGemini` (ADR 009) e `MotorChatGPT` a implementam.
- `pipeline.FonteMotor.Alvo()` devolve **(provedor, modelo)**. A autorização de envio de cada tarefa fica presa a esse par e à versão do mapa de pseudônimos. Se qualquer um mudar, o app pergunta de novo (V4 §48.1).
- Não existe fallback. Se o provedor escolhido falha, a parada é recuperável, com a tarefa preservada (V4 §47.6).
- Preferências v3 (`provedor_ia`, `modelo_chatgpt`), com migração explícita da v2 (que tinha só o modelo do Gemini) para `GEMINI`.

### 2. ChatGPT pelo *Sign in with ChatGPT*, em Go, sem intermediários
- **Login:** OAuth 2.0 com PKCE S256, `state` e `nonce`. O retorno chega só em `127.0.0.1:<porta aleatória>`, num único GET com o `Host` conferido.
  - registro dinâmico (`dynamic_agent_client` → `oaiapp_…`), com o nome verdadeiro **sys-executor**;
  - navegador do sistema, nunca a WebView.
- **ID token:** validado com `go-oidc/v3` (JWKS, emissor, destinatário, validade) e com o `nonce` conferido à parte.
  - a permissão `chatgpt.tokens.use.direct` é exigida para gerar texto;
  - no relogin, outra conta é recusada.
- **Credencial:**
  - DPAPI com entropia própria (`credenciais\chatgpt.dpapi`);
  - o access token fica só em memória;
  - a renovação é feita em série, e o refresh novo é gravado antes do uso;
  - falha de rede não apaga a credencial;
  - desconectar revoga a sessão e apaga a credencial.
- **Inferência:** Responses API com **cliente HTTP próprio**, sem o SDK da OpenAI. O corpo tem só `model`, `instructions`, `input`, `store:false`, `stream:true`, `text.format` (json_schema strict com os schemas inteiros, aceitos na PoC) e `reasoning`, quando informado.
  - Os campos proibidos pelo modo SIWC nunca saem.
  - A resposta só vale com `response.completed`.
  - Limite do plano e conta não elegível **não** são repetidos (IA-019, IA-020).
  - O destino é só `api.openai.com`, sem redirecionamento.
- **Termos do SIWC:** aviso de privacidade antes do primeiro login; botão só com texto, sem logotipo; pergunta sobre **criança menor de 13 anos** em toda tarefa, com bloqueio do ChatGPT (IA-021).

### 3. Barreira de privacidade em duas camadas, para os dois provedores
- **Camada 1** (`pseudonimo.Motor`, sobre o pedido já ocultado):
  - um identificador conhecido que sobre bloqueia sem exceção (IA-014);
  - possível nome não cadastrado gera `ErrRevisao`, que vira a **revisão item a item**. Não há "ignorar tudo".
- **Camada 2** (gancho `Inspecionar` do transporte, nos dois provedores): percorre todas as strings do **corpo exato** que vai sair e só bloqueia. É a garantia final sobre os bytes.
- **Ocultação mais robusta:**
  - ignora caixa e acento;
  - reconhece formas escapadas em JSON e a caixa alta aprendida no documento inteiro;
  - tenta a forma curta quando a longa falha;
  - detecta marcadores deformados.
- **Restauração:** determinística. Um marcador desconhecido interrompe a etapa (IA-010), e nenhum marcador pode sobrar nas saídas de IA. Esse é o contrato para as F6/F7.
- **Mapa:** a tabela de correspondência só existe na memória da tarefa. Nunca vai ao provedor, ao disco, a logs ou a mensagens.

### 4. Entrada por peças cadastradas
- Assistente PADS/SR (`Montagem.vue`). O cadastro vive na tarefa, só em memória e na sessão cifrada (E3, V4 §29).
- **Junção:** `pdf.Juntador` junta os arquivos em memória numa única instância do pdfium e grava `autos.pdf` da sessão. Extração, índice, evidências e visualizador continuam **inalterados**, com a numeração global contígua.
- **Separação:** `segment.DeDeclaracao` dá a cada peça a função declarada pelo usuário. `segment.VerificarDeclaradas` confere o conteúdo; se houver divergência, o app pergunta e nunca troca a função em silêncio (V4 §30.2 adaptada).
- **Pseudonimização:** os envolvidos cadastrados, com as outras grafias, semeiam o mapa de pseudônimos antes da detecção nas páginas.
- **Enquadramento manual:** catálogo curado embutido (`internal/rules/enquadramento`):
  - RDPM arts. 15 a 17, com a natureza fixa;
  - Estatuto arts. 29 e 32, com a natureza escolhida pelo usuário;
  - uma imputação por dispositivo, com a divergência em relação ao Memorando confirmada pela §22.1;
  - o motor de conexão não muda.
- **Monolítico:** a separação por carimbo fica só para o PDF que o próprio SEI gera (E6).
- **SEI (etapa 7, E2 revisado em 10/11/2026):**
  - o app gera no SEI o PDF de **todos** os documentos, a opção já permitida;
  - separa-o no computador pelo carimbo (`autos.LerTextoNativo`, sem OCR, e `pdf.Separador`);
  - os documentos vão ao cadastro, onde o usuário escolhe os de cada peça, com o tipo sugerido pelo título;
  - os não escolhidos não entram nos autos nem vão à IA.
  
  A geração "Apenas selecionados" foi descartada porque exige `documento_selecionar`, fora da guarda (mapeamento real com o `poc/sei-mapa`).

### 5. Medição sem dados reais (`cmd/bench`)
- `-provedor gemini|chatgpt|falso`.
  - O ChatGPT usa a **mesma** `chatgpt.Conta` e o mesmo `MotorChatGPT` do app. A credencial de desenvolvimento fica em `~/.config/sys-executor/chatgpt-dev.json` (0600), porque fora do Windows não há DPAPI. Decisão do usuário de 10/11/2026.
  - O provedor `falso` não usa rede.
- `-entrada memoria|monolitica|pecas` compara as entradas com os mesmos casos sintéticos.
- **Métrica de vazamento:** confere o corpo exato de cada requisição contra os nomes declarados nos casos (`casos.Caso.Pessoas`/`Identificadores`), independentemente do mapa. É portão (código de saída 3) e teste automatizado, sem rede, nas três entradas.

## Alternativas consideradas
- **SDK `openai-go`:** tem repetições automáticas e campos que o modo SIWC proíbe. O cliente próprio garante o corpo restrito com pouco código e nenhuma dependência nova.
- **`x/oauth2` (`TokenSource`):** não envia `resource` no refresh nem controla a rotação em série. Por isso, PKCE e troca de tokens foram implementados à mão; o `go-oidc` cuida só do ID token.
- **Codex CLI, proxy local ou reaproveitar tokens de outros apps:** proibidos pela V4 §48.1 e pelos termos.
- **Function tool forçada para a saída estruturada:** desnecessária, porque o json_schema strict foi aceito. Usá-la exigiria rever a §47.3 ("sem ferramentas").
- **Barreira só sobre o pedido (camada 1):** não cobre o que o transporte serializa (histórico, schema, instruções). A camada 2 sobre os bytes cobre.
- **Juntar as peças só na leitura (sem `autos.pdf`):** obrigaria a mudar extração, índice, evidências e visualizador. A junção em memória mantém tudo como estava.
- **SEI por "Apenas selecionados" ou montando a lista pela árvore:** a primeira exige liberar `documento_selecionar` na guarda e emendar a §48.4. A segunda imita um formato interno do SEI (`id±texto¥…`) e não enxerga documentos de pastas fechadas. A seleção no computador sobre o PDF completo não exige ação nova no SEI.
- **Bench reaproveitando a credencial do app (DPAPI) ou da PoC:** a do app exige o app fechado, porque dois processos renovando ao mesmo tempo invalidam a sessão. A da PoC acopla o bench a um módulo descartável.

## Consequências
- Há duas rotas de envio a terceiros. As ressalvas das §§1.1(9), 11, 47.3 e 48.7 valem para as duas, e a confirmação do usuário não substitui a autorização institucional.
- O limite de 5 h do Plus é compartilhado com os outros apps da conta. O bench consome esse limite (há o `-max-chamadas`).
- O ChatGPT trouxe dependências novas: `go-oidc/v3`, `go-jose/v4` (Apache-2.0) e `x/oauth2` (BSD-3-Clause).
- A interface não aceita mais o PDF único (E6). `ReceberUpload` continua no backend só para os testes legados. O SEI passou ao cadastro na etapa 7.
- Pelo SEI, o processo inteiro é baixado e separado, o que é mais lento em processos grandes. Os documentos não escolhidos ficam só na sessão cifrada até o fim da tarefa.
- A barreira pode pedir revisão de textos que não são nomes. O bench registra esses itens. Nas rodadas sem rede da etapa 8, nenhum item apareceu depois que o eco do provedor falso passou a citar a evidência, como faz um modelo real (relatório da etapa 8).
