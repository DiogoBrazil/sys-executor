# Relatório — etapa 6 do plano D-11: cadastro de peças (V4 §48.4)

Data: 09/11/2026. Branch `add-mode-apikey-oauth`. Autorização: "Pode prosseguir com as demais etapas" (etapas 5 a 8).

## 1. O que mudou

### Entrada dos autos
O cartão "Enviar o PDF do processo" saiu da tela de entrada. No lugar dele há **"Cadastrar as peças"**, que abre o assistente (`frontend/src/views/Montagem.vue`).

**Passos do PADS:**
1. Memorando e acusado: arquivos, acusado, fatos com os dispositivos do catálogo de enquadramento.
2. Defesa prévia, ou "não apresentada".
3. Depoimentos: papel, depoente e data.
4. Alegações finais, ou "não apresentadas".
5. Outros documentos.
6. Conferência: peças, envolvidos com outras grafias e a marca "menor de 13 anos", pendências e "Concluir e processar".

**Passos da SR:**
1. Portaria: identificação, arquivos, sindicados e ofendidos, ou "não há ofendido individualizado".
2. Depoimentos: testemunha, declarante ou sindicado.
3. Outros documentos.
4. Conferência.

**Arquivos:**
- Cada peça aceita vários PDFs, que podem ser reordenados e removidos.
- O diálogo nativo é de seleção múltipla.
- Os bytes são lidos no Go e gravados cifrados na sessão.
- A interface recebe só o nome (sem a pasta), o número de páginas e um resumo do hash.

### Backend
- **`internal/workflow/cadastro.go`** guarda o cadastro na tarefa, só em memória e na sessão.
  - Comandos: `SalvarCadastro`, `AnexarArquivos`, `RemoverArquivo`, `MoverArquivo` e `ConcluirCadastro`.
  - Validação:
    - pessoas, fatos, dispositivos do catálogo, papéis por modalidade e datas;
    - IDs provisórios da interface (`NOVO-n`) remapeados para IDs definitivos, com as referências reescritas;
    - ID inexistente nunca é aceito como definitivo.
  - Arquivos e pendências:
    - arquivo repetido (SHA-256) → **ARQ-004**;
    - limite de 300 arquivos ou 1 GB → **ARQ-005**;
    - pendências impedem a conclusão (TAR-006).
  - Voltar ao cadastro (por uma pergunta ou por um PDF ilegível) o deixa **editável e preservado**.
- **`pdf.Juntador`:** junta os arquivos em memória numa única instância do pdfium (`FPDF_ImportPagesByIndex` + `SaveAsCopy`). O resultado é gravado como `autos.pdf` da sessão. Extração, índice, evidências e visualizador continuam com a numeração global, sem mudança.
- **Ordem processual da junção:** peça central, defesa, oitivas, outros documentos e alegações. Dentro de cada grupo vale a ordem do cadastro.
- **`segment.DeDeclaracao`:** cada peça declarada vira uma peça dos autos com a função do usuário (`Base USUARIO`). O nº SEI vem do carimbo dominante quando o usuário não o informa. A IA não reclassifica essas peças.
- **`segment.VerificarDeclaradas`** substitui a heurística das "3 primeiras peças" quando há cadastro:
  - peça central ilegível → pedir versão legível;
  - central com conteúdo do outro procedimento → incompatível, sem trocar o tipo;
  - primeira página com título forte de outra peça → pergunta "Voltar ao cadastro" / "Continuar como cadastrado".
- **Pseudonimização:**
  - `Mapa.CadastrarPessoa` registra cada envolvido e as outras grafias como variantes da mesma pessoa, inclusive sem âncora no texto;
  - pessoa marcada como criança bloqueia o ChatGPT (IA-021) antes de qualquer envio.
- **Delimitação PADS (E5/E9):**
  - o acusado do cadastro é conferido com o lido no Memorando, e a divergência gera alerta;
  - o dispositivo do cadastro preenche a imputação **cujo dispositivo citado é o mesmo**;
  - os dispositivos sem correspondência viram imputações extras, já incluídas e com o alerta e a **confirmação expressa da §22.1**;
  - imputação da IA sem dispositivo correspondente no cadastro fica sem preenchimento, com alerta;
  - nada é sugerido pelo aplicativo.
- **Delimitação SR:** os sindicados do cadastro são conferidos com os lidos na Portaria, com alerta nos dois sentidos.
- **App:** métodos `SalvarCadastro`, `AnexarArquivos`, `RemoverArquivo`, `MoverArquivo`, `ConcluirCadastro` e `CatalogoEnquadramento` no `TarefaService`; bindings regenerados.

## 2. Governança

| Item | Mudança |
|---|---|
| TRV-ING-001 | v2: a função declarada vale; a classificação pelo conteúdo é conferência. `enforced_by` em `segment.DeDeclaracao` e `segment.VerificarDeclaradas`. A `instrucao_ia` não mudou |
| TRV-ING-002 | v2: conteúdo que contradiz a declaração → pergunta, nunca troca |
| TRV-ING-007 | **Nova:** cadastro peça por peça |
| TRV-PRV-005 | v2: a origem fica registrada por arquivo |
| TRV-UI-001 a TRV-UI-004 | v2: o texto passa de "upload" para cadastro de peças; compatibilidade pela peça central declarada |

- **Testes novos:** CAD-PEC-01, CAD-PEC-02 e CAD-PEC-03 (plano de testes e matriz).
- **Verificação:**
  - `rulescov -estrito` verde;
  - V4.lock sem alteração;
  - **golden sem diff.**

## 3. Verificação
- `go test ./...` verde em todos os pacotes, exceto `cmd/sys-executor`, que no Linux não compila sem GTK (é um app Windows, e isso já acontecia antes).
- `go test -race` verde nos pacotes alterados.
- `GOOS=windows go vet ./...` verde.
- Vitest 30/30, `vue-tsc` e `vite build` verdes.
- **Teste de pipeline com pdfium real** (`TestCadastroPADSAteDelimitacao`):
  - 3 PDFs sintéticos anexados fora de ordem;
  - autos juntados em 4 páginas na ordem processual;
  - divergência "declarante × termo de inquirição" perguntada;
  - "Voltar ao cadastro" preserva o cadastro; corrigido o papel, a tarefa fica compatível e para em IA-001, sem IA.

## 4. Pendências e decisões em aberto
- **Interrogatório do acusado no PADS:** pendência §8.1 do plano. O assistente não oferece o acusado como testemunha ou declarante, e a orientação é cadastrar o interrogatório em "Outros documentos". Decidir se ele deve ter tipo próprio.
- **Conexão por fato:** o agrupamento por fato do cadastro ainda não pré-seleciona os grupos na pergunta de conexão existente. A conexão continua pela regra da seção 5, confirmada pelo usuário.
- **`ReceberUpload` (PDF único):** fica no backend para o SEI e os testes legados. A interface não oferece mais esse caminho (E6).
- **Teste manual no Windows:** cadastro PADS e SR sintéticos até TAR-004, com ChatGPT ou Gemini.
- **Etapa 7 (SEI pelos selecionados):** antes, é preciso o mapeamento real da tela "Gerar PDF" com `poc/sei-mapa`, feito pelo usuário.
