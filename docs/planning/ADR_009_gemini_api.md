# ADR 009 — Inferência pela Gemini API com chave do usuário

- **Status:** Aceita (09/11/2026). **Substitui o ADR 003** (sidecar llama-server).
- **Contexto:** os modelos locais (Qwen3.5 2B/4B em CPU) levaram de 1 a 6 minutos por caso no hardware disponível (RELATORIO_FASE4 §3). Decisão do usuário: usar a Gemini API (D-10; emenda V4 §47).

## Decisão
- **SDK oficial** `google.golang.org/genai` (v1.77.0, Apache-2.0), backend Gemini Developer API, **chave de API do usuário**. Sem OAuth, Vertex AI, outros provedores ou fallback local.
- **Porta inalterada:** `inference.Motor`. O `MotorGemini` só troca o transporte; montagem do pedido, reenvios do validador, conferência literal, regras e motor determinístico continuam como estavam (golden das políticas sem diff).
- **Envio mínimo (estratégia A):** extração local (pdfium/OCR/segmentação) mantida; só os blocos de texto de cada etapa vão ao provedor. Sem PDF, Files API ou context caching.
- **Proteções em toda requisição:** base fixa `https://generativelanguage.googleapis.com/` (o SDK aceitaria `GOOGLE_GEMINI_BASE_URL` do ambiente), transporte que recusa qualquer outro destino e não segue redirecionamento, `store=false` (projetos novos registram requisições por padrão), nenhuma ferramenta, continuação automática desligada (cada continuação é cobrada).
- **Integridade:** só vale resposta com `FinishReason=STOP` e texto; `MAX_TOKENS` vira "truncada" (o validador pede concisão); bloqueio → IA-009; vazio → IA-010. Partes de raciocínio são descartadas.
- **Falhas:** 429 por minuto, 5xx e rede são repetidos até 3 vezes, com backoff exponencial, jitter e `RetryInfo` (≤ 60 s); cota diária, 4xx e tempo esgotado não são repetidos (custo) e viram parada recuperável. Cancelamento interrompe a requisição HTTP.
- **Parâmetros:** temperatura no padrão do modelo (a documentação do Gemini 3 desaconselha valores abaixo de 1,0); nível de raciocínio configurável (padrão do modelo); janela de trabalho de 32.768 tokens (limitada pelo modelo) para manter o fatiamento; contagem de tokens estimada localmente (sem rede).
- **Chave:** DPAPI (escopo do usuário, entropia própria) em `%LOCALAPPDATA%\sys-executor\credenciais\gemini.dpapi`, pasta com ACL só do usuário; validada pela listagem de modelos antes de ser guardada; nunca vai à interface.
- **Modelos:** listados da chave (`generateContent`, prefixo `gemini-`, sem variantes de imagem/áudio/voz/embeddings), com estágio (estável, prévia, experimental, apelido "latest"); o usuário testa e escolhe; nada fixo no código.
- **Privacidade adicional:** confirmação de envio em cada tarefa e pseudonimização local de dados pessoais (pacote `pseudonimo`), transparente para a conferência literal.

## Alternativas consideradas
- **PDF inteiro ao modelo (multimodal / Files API):** quebra a conferência literal com página e posição, envia tudo e exige limpeza remota.
- **Híbrido (imagem das páginas de OCR ruim):** melhora digitalizados, mas o trecho lido na imagem não pode ser conferido literalmente; fica para decisão futura.
- **Cliente HTTP próprio (sem SDK):** menos dependências, mas o SDK oficial acompanha a API (campos novos, como `store`) e foi pedido pelo usuário.
- **Streaming:** respostas são JSONs pequenos; o progresso real já é por etapa e contagem; TRV-IA-010 (sem streaming na interface) mantida.

## Consequências
- A análise por IA exige internet; as demais funções seguem offline (parada recuperável IA-007).
- Dependências novas no executável: o SDK e suas dependências (Apache/BSD; inclui a API do OpenTelemetry sem nenhum exportador).
- Fica pendente a busca semântica da biblioteca (F5): embeddings locais não existem mais (V4 §47.7).
- GGUFs baixados em versões anteriores (`%LOCALAPPDATA%\sys-executor\modelos`) não são apagados pelo app; o usuário pode removê-los manualmente.
