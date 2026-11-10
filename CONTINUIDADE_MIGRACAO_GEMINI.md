# Continuidade — migração da IA local para a Gemini API

Atualizado em 09/11/2026. Branch: **`add-mode-apikey-oauth`** (nada foi enviado ao remoto nem mesclado).

> **Atualização de 09/11/2026 — D-11:** o usuário aprovou o plano `docs/planning/PLANO_CHATGPT_PECAS_PRIVACIDADE.md`. Ele trata do ChatGPT pelo *Sign in with ChatGPT* como 2º provedor, do cadastro peça por peça, do enquadramento manual (RDPM/Estatuto) e da barreira obrigatória de privacidade. A D-11 reabre parcialmente o item 1 da §1 abaixo, que excluía o OAuth e outros provedores; o Gemini continua preservado. **Etapa 0 concluída:** a Emenda 2 foi aprovada e inserida na V4 (seção 48), com `V4.lock`/CHANGELOG atualizados; o catálogo de regras muda junto com o código de cada etapa. **Etapa 1 concluída (PoC `poc/chatgpt`):** login, tokens, renovação, revogação, modelos e inferência SSE funcionaram. Os 8 schemas reais foram aceitos em strict sem nenhuma violação. Resultados em `docs/planning/RELATORIO_POC_CHATGPT.md`; a sessão de teste foi revogada. **Etapa 2 concluída (barreira de privacidade):**
- casamento sem diferenciar caixa nem acentos;
- camada 1 no pedido e camada 2 nos bytes da requisição (IA-014);
- revisão item a item na autorização e no meio da etapa.

Relatório: `docs/planning/RELATORIO_ETAPA2_BARREIRA.md`. O teste no Windows confirmou a barreira e revelou três correções:
- ordem das telas;
- **400 do Gemini**: `maxItems` recusado; o schema vai sem limites de lista e é conferido no Go;
- cota diária classificada como IA-005.

O 400 só pode ser reconfirmado ponta a ponta quando a cota diária gratuita (20/dia) renovar. **Etapas 2 e 3 commitadas** (`ab14035`, `0c5335c`). **Etapa 4 implementada** (provedor ChatGPT; relatório em `docs/planning/RELATORIO_ETAPA4_CHATGPT.md`). O `.exe` de teste está em `Documents\testes-sys-executor\sys-executor-chatgpt.exe`. **Próximo passo:** o usuário testa com a conta Plus; depois, etapa 5 (enquadramento), que depende de autorização expressa. **Nenhum código** foi alterado; cada etapa do plano depende de autorização expressa.

Leia junto: `docs/planning/RELATORIO_MIGRACAO_GEMINI.md` (relatório técnico), `docs/planning/ADR_009_gemini_api.md` (arquitetura), V4 §47 (`docs/rules/sys-executor_especificacao_consolidada_V4.md`) e D-10 (`docs/planning/REGISTRO_DUVIDAS_DECISOES.md`).

---

## 1. Decisões já tomadas (não reabrir sem o usuário)

1. A inferência local (llama-server/GGUF) foi **removida**. A IA agora é a **Gemini API**, com a chave de API do próprio usuário, chamada pelo backend Go com o SDK oficial `google.golang.org/genai` v1.77.0. O escopo é só a chave de API: **não há OAuth, Vertex AI, outros provedores nem retorno à IA local**, apesar do nome da branch.
2. A conta Gemini pode ser **de qualquer nível**, e a interface não mostra aviso sobre o nível. No lugar do aviso, há a **pseudonimização** de dados pessoais antes do envio (melhor esforço).
3. O usuário **confirma o envio em cada tarefa**, antes da primeira etapa de IA.
4. A mudança entrou como **emenda na V4** (§47), com CHANGELOG, `V4.lock`, catálogo de regras e matriz de requisitos atualizados.
5. **Documentos (estratégia A):** a extração continua local. Só os blocos de texto de cada etapa são enviados; o PDF não é enviado, e não se usa a Files API nem o cache de contexto do provedor.
6. As regras jurídicas, os cálculos, as perguntas, os validadores e a integração com o SEI **não foram alterados**. Os arquivos de referência dos prompts (golden) continuam sem nenhuma diferença.

---

## 2. O que está feito (commits)

| Commit | Conteúdo |
|---|---|
| `eb3afc9` | Emenda V4 §47, D-10, `V4.lock`, CHANGELOG |
| `0f901cf` | Backend: `MotorGemini`, `credencial` (DPAPI), `pseudonimo`, confirmação `envio_ia`, `IAService`/`FonteGemini`, preferências v2, erros IA-001…012, catálogo de regras e matriz, `cmd/bench` adaptado; remoção do sidecar, GGUF, hardware e GPU |
| `8b42d0c` | Interface: tela "Configurações de IA", que substitui a tela Modelos; bindings regenerados |
| `c533908` | Documentação: ADR 009, relatório, plano de testes, README; exclusão do antigo `docs/ESTADO_E_CONTINUIDADE.md` |
| `e265925` | A lista de modelos passa a excluir os de geração de imagem (`nano-banana`) e de transcrição |
| `9366170` | Créditos ou faturamento esgotados viram **IA-013**; os erros também são classificados pelo `status` da API; o teste de modelo mostra o diagnóstico |
| `86c25e5` | Prazo esgotado durante a espera devolve o erro real, não mais "context canceled"; espera de no máximo 5 s nas configurações; cota com "limit: 0" vira IA-004 |
| `6bbc0f7` | **Defeito anterior à migração:** Tesseract ausente derrubava o app (interface não nula guardando um `*Tesseract` nulo). Agora vira OCR-001 |
| `00f177f` | **Defeito anterior à migração:** uma parada recuperável chegava à interface como não retomável, escondendo o erro e o botão "Tentar novamente" |
| `6dfac51` | Reenvio com resposta vazia (cortada durante o raciocínio) gerava parte `{}` e era recusado com **400 INVALID_ARGUMENT**. Agora vai como texto explicativo; o limite de saída passa a ser o do modelo quando ele raciocina; o diagnóstico do 400 mostra `fieldViolations` e a etapa |

**Verificação automatizada, verde no último commit:**
- `go test -race` de `internal/...` e `cmd/...`;
- `GOOS=windows go vet ./...` e build Windows;
- Vitest (22), `vue-tsc` e `vite build`;
- `go run ./cmd/rulescov -estrito` e `go run ./cmd/licencas`.

---

## 3. Onde paramos (teste manual no Windows)

| Passo | Situação |
|---|---|
| Abrir o `.exe` | OK, depois de desligar o **Controle Inteligente de Aplicativos** (Smart App Control) do Windows 11, que bloqueia executáveis sem assinatura. A máquina não é gerenciada pela organização |
| Cadastrar a chave e listar os modelos | OK |
| Testar modelos (chave do **nível gratuito**) | 3.5 Flash, 3.5 Flash Lite e 3.6 Flash responderam. **3.8 Flash** falhou com `DEADLINE_EXCEEDED` do próprio Google (servidor sobrecarregado). Uma chave anterior, de projeto pré-pago, estava **sem créditos** (agora IA-013) |
| Tarefa PADS com `pads-sintetico-texto.pdf` | Extração, segmentação, confirmação de envio e **delimitação OK**. A delimitação acertou o gabarito: acusado CARLOS INVENTADO; IMP-1 no art. 17, V (grave); IMP-2 no art. 15, III (leve); fato em 05/04/2026 |
| Etapa "Analisando provas" | Falhou com **INT-001 / 400 INVALID_ARGUMENT**. A correção provável (`6dfac51`) foi feita mas **ainda não foi testada pelo usuário** |

**Próximo passo imediato:** abrir a versão de `6dfac51` com diagnóstico, enviar de novo o PDF sintético e verificar se a análise chega às perguntas de resultado. O esperado é **IMP-1 procedente e IMP-2 improcedente** (procedência parcial); a tarefa deve então parar em "Consultando fontes" com TAR-004, que é o fim previsto nesta fase. Se o 400 persistir, o diagnóstico novo mostra o campo exato que o Google recusou.

---

## 4. O que falta

1. **Rodar a Gemini API real a partir da máquina de desenvolvimento.** Hoje as falhas aparecem uma por rodada de teste manual. Faltam:
   - o teste de integração: `SYSEXEC_GEMINI_API_KEY=… SYSEXEC_GEMINI_MODELO=gemini-3.6-flash go test -run Integracao -v ./internal/analise/`;
   - o benchmark: `SYSEXEC_GEMINI_API_KEY=… go run ./cmd/bench -modelo gemini-3.6-flash [-pseudonimizar=false] -saida rel.md`.
   
   Guarde a chave de desenvolvimento num arquivo fora do repositório (ex.: `~/.config/sys-executor/gemini-dev.key`, com permissão 600) e use `SYSEXEC_GEMINI_API_KEY=$(cat …)`. **Use só casos sintéticos, nunca autos reais.**
2. **Benchmark contra os limiares** (PLANO_TESTES_ACEITACAO §9): delimitação ≥ 90%; JSON válido na 1ª tentativa ≥ 90%; resultado ≥ 75%; SEG-INJ 5/5; **0 violações aceitas e 0 citações inventadas**. Comparar com e sem pseudonimização e registrar o resultado no `RELATORIO_MIGRACAO_GEMINI.md` §4.
3. **Calibrar** a janela de trabalho (hoje 32.768 tokens), o nível de raciocínio (hoje o padrão do modelo) e o comportamento com os limites do nível gratuito, que uma tarefa completa (dezenas de chamadas) pode atingir.
4. **Fluxo SR no Windows:** ainda não foi testado manualmente. Não existe PDF sintético de SR só com texto; gere um como descrito no §5.
5. **Teste da DPAPI no Windows** (`internal/credencial`): roda no job `go-windows` do CI. Na máquina do usuário, o binário de teste foi bloqueado pelo Smart App Control. Com ele desligado, pode ser executado.
6. **Teste sem rede no Windows:** espera-se IA-007, com a tarefa preservada e retomada por "Tentar novamente".
7. **Pendências que vêm de antes da migração:**
   - Fase 5: decidir entre busca só lexical e embeddings Gemini para a legislação pública (V4 §47.7);
   - Fases 6 a 8: composição da minuta, DOCX/PDF, revisão e instaladores;
   - comparar a árvore de documentos do SEI com as peças do PDF (pendência da Fase 4).
8. **CI:** o passo `gofmt` acusa `internal/ingest/segment/carimbo_test.go`, arquivo anterior à migração e não alterado por ela; provavelmente é diferença de versão do formatador.
9. **Melhoria sugerida, ainda não feita:** os pânicos dentro dos workers da extração (fora de `executarSeguro`) ainda derrubam o processo. Um `recover` ali transformaria esse tipo de falha em INT-001.

---

## 5. Ambiente e rotina de teste

- **Desenvolvimento:** WSL, Go 1.26+ (o CI usa 1.27.x), Node 24 e `wails3` (v3.1.0) em `~/go/bin`.
- **Bindings** (depois de mudar serviços em `internal/app`): `GOOS=windows wails3 generate bindings -ts -d frontend/bindings ./cmd/sys-executor`.
- **Build:** `cd frontend && npm run build && cd .. && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o sys-executor.exe ./cmd/sys-executor`.
- **Pasta de teste no Windows:** `C:\Users\diogo\Documents\testes-sys-executor\` (no WSL: `/mnt/c/Users/diogo/Documents/testes-sys-executor/`). Feche o app antes de copiar um `.exe` novo: o Windows bloqueia a substituição e o app só permite uma instância aberta.
- **Modo diagnóstico** (mostra a causa técnica, com os dígitos mascarados):
  ```powershell
  cd C:\Users\diogo\Documents\testes-sys-executor
  $env:SYSEXEC_DIAGNOSTICO = "1"; .\sys-executor.exe
  ```
- **OCR:** a pasta de teste não tem o Tesseract (`componentes\tesseract\tesseract.exe` + `tessdata\por.traineddata`). Sem ele, PDFs com página digitalizada (ex.: `testdata/autos/sei-sintetico.pdf`) terminam com OCR-001.
- **PDF sintético só com texto:** foi gerado com `internal/testutil/pdfsint.Gerar` a partir do caso `pads-02-duas-imputacoes` de `internal/testutil/casos`: linhas de até 72 caracteres, rodapé `SEI nº <SEI da peça>`, uma página por peça. O arquivo não está no repositório; regenere-o se precisar (ou um equivalente para SR, a partir de `sr-01-extravio`).
- **Dados no Windows:** `%LOCALAPPDATA%\sys-executor\credenciais\gemini.dpapi` guarda a chave cifrada; `preferencias.json` guarda o modelo escolhido, nunca a chave. A pasta `modelos\` de versões anteriores pode ser apagada manualmente.
- **Modelos com a chave gratuita:** usar o **Gemini 3.6 Flash**, que é Estável e respondeu. O 3.8 Flash estava sobrecarregado. Não usar Prévia, Experimental, apelidos `-latest` nem Flash Lite para análise jurídica.

---

## 6. Cuidados

- **Nunca** enviar autos reais à API em desenvolvimento ou teste.
- Não pôr a chave no código, em commits, em logs, no chat nem em argumentos de linha de comando (use a variável de ambiente lida de um arquivo).
- Não fazer push nem merge sem autorização do usuário.
- Qualquer mudança de regra jurídica exige decisão do usuário e passa pelo fluxo da V4 (CHANGELOG + `rulescov -atualizar-lock`).
