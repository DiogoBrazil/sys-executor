# RELATÓRIO — MIGRAÇÃO DA INFERÊNCIA LOCAL PARA A GEMINI API (sys-executor)

- **Data:** 09/11/2026 · **Branch:** `add-mode-apikey-oauth` (sem push nem merge)
- **Autorização:** plano aprovado pelo usuário; texto da emenda V4 §47 aprovado ("Aprovo o texto").
- **Decisões:** D-10 (`REGISTRO_DUVIDAS_DECISOES.md`); arquitetura no ADR 009 (substitui o ADR 003).

## 1. O que mudou

| Área | Antes | Agora |
|---|---|---|
| Inferência | `llama-server` local (Qwen3.5 GGUF), sidecar em Job Object | `MotorGemini` (SDK `google.golang.org/genai` v1.77.0) na mesma porta `inference.Motor` |
| Modelos | catálogo GGUF, download com SHA-256, hardware, GPU, orçamento de memória | lista da chave (modelos de texto `gemini-*`), estágio e limites; o usuário testa e escolhe; nada fixo no código |
| Credencial | — | chave da Gemini API cifrada com DPAPI (`credenciais\gemini.dpapi`), validada antes de ser guardada |
| Envio | nada saía do computador | só blocos de texto de cada etapa, após **confirmação em cada tarefa**, com **pseudonimização** de dados pessoais; `store=false`, sem ferramentas, base fixa, transporte restrito ao host da API |
| Tela | "Modelos de IA local" | "Configurações de IA" (chave, modelos, teste, uso da última tarefa) |
| Erros | MOD-001…009 | IA-001…012 (MOD-* reservados, nunca reutilizados) |
| Preferências | v1 (modelo local, GPU) | v2 (modelo Gemini), com migração da v1 |

**Não mudou:** policy packs e catálogo das regras jurídicas (golden das políticas **sem diff**), motor determinístico, validadores (trecho literal, IDs, datas, citação legal, escopo PADS/SR), perguntas e DAG de revisão, SEI/chromedp, pdfium, OCR, sessão efêmera cifrada. A única pergunta nova é a confirmação de envio.

## 2. Governança

- V4: Emenda 1 (seção 47) e notas **[Emenda §47]** em §29.3 e §§34–38; `policies/V4.lock` e `policies/CHANGELOG.md` atualizados.
- Catálogo (193 regras): revogadas TRV-IA-001, 003…009, TRV-SEG-006/007 e TRV-OFF-001; reescritas TRV-OFF-002 e TRV-INS-003; novas TRV-IA-011…013, TRV-SEG-009…012 e TRV-OFF-003, todas com `enforced_by` e teste anotado. Matriz e plano de testes (§§8, 9, 11) atualizados.
- `go run ./cmd/rulescov -estrito`: GOV-LOCK-01 ok; 124/124 IDs de teste da V4 cobertos; 0 regras sem teste.

## 3. Verificação feita

| Verificação | Resultado |
|---|---|
| Linha de base antes da migração (`go test -race`, vet/build Windows, Vitest, rulescov, licenças) | Verde (o pacote `cmd/sys-executor` não compila no Linux sem GTK — esperado, o app é só Windows) |
| `go test -race` de todos os pacotes após a migração | **Verde** |
| Golden das políticas (`internal/policy/testdata/golden`) | **Sem alteração** |
| `GOOS=windows go vet ./...` e build `-H windowsgui` | Verde |
| Vitest (21) · `vue-tsc` · `vite build` | Verde |
| `cmd/licencas` (43 módulos Go, 40 npm) | Verde; nenhum exportador do OpenTelemetry é ligado ao executável (só a API, sem efeito) |
| Testes do `MotorGemini` com servidor falso | mapeamento do pedido, `store=false`, sem `tools`, sem temperatura; truncada/bloqueada/vazia; 400/401/403/404/429 (diária × por minuto)/503; cancelamento interrompe a requisição; tempo esgotado sem reenvio; host bloqueado; chave fora das mensagens; listagem filtrada e classificada |
| Pseudonimização | detecção por âncoras, CPF com dígito verificador, ida e volta byte a byte, JSON escapado, marcador deformado → correção → IA-010; nomes sentinela ausentes do pedido; trecho restaurado confere com os autos |
| Pseudonimização nos 10 casos sintéticos do benchmark (sem rede) | nenhum nome de pessoa restou no texto que seria enviado; ficaram títulos de peças, postos ("SGT PM") e "Porto Velho" |
| Pipeline | recusa do envio → IA-011, **nada enviado**, nova tentativa pergunta de novo; autorização oculta nomes detectados e informados; queda de conexão (IA-007) → parada recuperável e novo cliente na retomada |

## 4. Não verificado ainda (depende do usuário ou de outra fase)

1. **Rodada real com a Gemini API** (`cmd/bench` e teste de integração de `analise`): exige a chave de desenvolvimento do usuário; só casos sintéticos. Limiares do PLANO_TESTES §9 (delimitação ≥ 90%, JSON 1ª ≥ 90%, resultado ≥ 75%, SEG-INJ 5/5, **0/0** nos portões rígidos). Também calibra raciocínio, temperatura e janela de trabalho, e mede o efeito da pseudonimização. **Resultado: pendente.**
2. **DPAPI no Windows real:** o teste `internal/credencial` roda no job `go-windows` do CI; a execução local do binário de teste foi bloqueada pela política Device Guard da máquina (não contornada). Validar no CI ou na máquina do usuário.
3. **Validação manual no Windows:** executar o `.exe` **sem a pasta `llama\`**, cadastrar a chave, listar/testar/escolher o modelo, processar um PDF **sintético** até a etapa 9, testar sem rede e com proxy, se houver.
4. **Fases ainda não implementadas** (não afetadas, não verificáveis agora): biblioteca jurídica (F5), composição/redação (F6), DOCX/PDF, revisão e finalização (F7), instaladores NSIS/MSI (F8).

## 5. Limites e riscos

- **Pseudonimização é melhor esforço:** nomes sem âncora (posto, papel processual, rótulo, bloco de assinatura), grafias quebradas por OCR ou hifenização podem passar; o contexto pode permitir reidentificação. A confirmação permite informar nomes adicionais. Uma lista de prenomes aumentaria o alcance, mas só entra com licença verificada.
- **Retenção no provedor:** o app desliga o registro opcional (`store=false`), mas o provedor mantém registros para prevenção de abuso, e o nível gratuito pode usar o conteúdo para melhorar produtos (termos da Gemini API). O app não promete apagar registros externos (V4 §§1.1-9, 11, 47.3).
- **Proxy corporativo:** o Go usa as variáveis `HTTPS_PROXY`/`HTTP_PROXY`, não o proxy do sistema do Windows; em rede com proxy obrigatório, a falha aparece como IA-007.
- **Modelos mudam:** a lista vem da chave; um modelo escolhido que deixe de existir é sinalizado na tela.
- **Custo:** o app mostra tokens, não preço (não inventa valores).

## 6. Pendências registradas

- F5: busca semântica da biblioteca sem embeddings locais — escolher entre só lexical (FTS5 + tesauro) e embeddings do provedor para a legislação pública (V4 §47.7).
- Herdada da Fase 4: comparar a árvore de documentos do SEI com as peças do PDF para apontar mídias e arquivos fora do PDF (F5/F6, com aval do usuário).
- `internal/ingest/segment/carimbo_test.go` não está formatado pelo `gofmt` (anterior a esta migração).
