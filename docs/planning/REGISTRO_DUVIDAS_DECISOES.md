# REGISTRO DE DÚVIDAS E DECISÕES — SYS-EXECUTOR

> Registro vivo. Contém apenas **perguntas que surgiram de fato**, as respostas do usuário e o impacto de cada uma. Nada aqui é resposta presumida. Data das decisões D-01…D-07: **08/11/2026**, na sessão de planejamento.
> A V4 continua sendo a fonte da verdade. Estas decisões **complementam** a V4 em pontos que ela não resolvia literalmente. Elas não reabrem regras já aprovadas.

## 1. Decisões tomadas pelo usuário

### D-01 — Art. 41, § 3º do RDPM (execução de medidas privativas/restritivas de liberdade)
- **Contexto:** o arquivo `docs/law_library/RDPM-RO.pdf`, consolidado até o Dec. 24.965/2020, traz o § 3º do art. 41 como vigente. O parágrafo extingue a execução de prisão e detenção disciplinar com base na Lei 13.967/2019. O STF declarou essa lei inconstitucional na ADI 6595 (https://portal.stf.jus.br/noticias/verNoticiaDetalhe.asp?idConteudo=487619). Nenhum ato estadual revogando o § 3º foi localizado na biblioteca nem nas buscas web de 08/11/2026. A V4 (§§ 4, 10, 44) já registrava que o ato revogador mencionado pelo usuário não estava identificado e mandava manter a modelagem de detenção/prisão e a cláusula residencial.
- **Alternativas apresentadas:**
  1. Nota curada + alerta na revisão (era a recomendação).
  2. Nota curada sem alerta.
  3. Usuário fornece o ato revogador.
- **Decisão:** **Nota curada, sem alerta.**
- **Impacto:**
  - A biblioteca terá o registro `nota_vigencia` (`RDPM art. 41 §3º = CONSIDERADO_REVOGADO_PELO_USUARIO`, `fonte_revogadora = null`, origem `V4 §4/§10/§44 + D-01`).
  - O trecho não é oferecido à redação.
  - Uma regra de núcleo proíbe citá-lo como vigente (`TRV-LIB-012`), e o validador bloqueia a citação (`TRV-VAL-014`).
  - A interface **não** mostra alerta.
  - A cláusula do §10 continua sendo usada quando houver detenção/prisão.
  - Se o usuário trouxer o ato revogador, a nota passa a citá-lo, sem nenhuma outra mudança.
- **Natureza:** decisão funcional do usuário. **Não é comprovação documental independente de vigência** (V4 §44).

### D-02 — Arquitetura Windows
- **Decisão:** **somente x64** na V1. ARM64 fica fora do escopo e pode ser reavaliado depois.
- **Impacto:** instaladores `.exe` e `.msi` x64; um único conjunto de binários de terceiros (llama.cpp CPU/Vulkan, Tesseract, Typst); matriz de testes de instalação reduzida.

### D-03 — Motor de PDF da minuta
- **Alternativas apresentadas:** Typst embutido; LibreOffice headless (~350 MB); Word por automação COM com fallback Typst.
- **Decisão:** **Typst embutido.**
- **Impacto:**
  - DOCX (writer OOXML próprio) e PDF (Typst) saem do **mesmo DocModel**.
  - A paginação pode diferir entre os dois formatos; as regras de formatação são as mesmas.
  - O PDF é a referência visual da prévia.
  - Golden files dos dois formatos.
  - Times New Roman vem do sistema e não é redistribuída. Fallback métrico Liberation Serif: [A VALIDAR].

### D-04 — Materiais ausentes
- **Itens:** `logo-pmro.png`; as duas imagens de referência do cabeçalho; os modelos de julgamento de PADS (absolvição e procedência parcial); os modelos A e B de relatório de SR; o PDF técnico "Especificação Técnica - Go Sidecar llama-server e RAG Local".
- **Decisão:** o usuário **fornece até a Fase 7**.
- **Impacto:**
  - As Fases 0–6 usam fixtures fictícias.
  - Nenhum visual institucional será imitado.
  - A Fase 7 (fidelidade institucional) não fecha sem esses arquivos.
  - Os modelos servem apenas de referência de forma e estilo, nunca de fonte de fatos (V4 §§3, 27.1).
  - Localização sugerida: `docs/assets/` (brasão, cabeçalhos) e `docs/models/` (modelos de minuta).

### D-05 — Contagem de agravantes do art. 39, VII (alíneas a–e) na heurística de dias
- **Pergunta:** havendo várias alíneas do VII reconhecidas, quantas agravantes entram na comparação com as atenuantes?
- **Decisão:** **cada alínea reconhecida conta como uma agravante.** Exemplo do usuário: 2 atenuantes × (VII-a + VII-d) = 2×2, resultado **2 dias**.
- **Complemento já existente na V4:** a reincidência (inciso III) conta **no máximo uma vez**, mesmo com várias punições anteriores (V4 §15.5, R9). A conexão (inciso II) **não** se multiplica em várias agravantes (V4 §5.1).
- **Impacto:** `engine.Dosimetria` conta circunstâncias reconhecidas por (artigo, inciso, alínea); para o inciso III aplica `min(1, …)`; para o inciso II, uma única ocorrência. Testes `DOS-D05-*`.

### D-06 — Regressão especial bom/ótimo: a janela precisa conter fato atual?
- **Decisão:** **sim.** A janela estritamente inferior a 1 ano que soma 2 graves equivalentes precisa conter **ao menos uma transgressão procedente do PADS atual**.
- **Comportamento adicional:** se o histórico reconhecido sozinho já formar o gatilho, o app alerta que pode haver inconsistência com a classificação atual informada e **pergunta** ao usuário. Nada é recalculado silenciosamente.
- **Impacto:** `engine.GatilhoBomOtimo`; testes `CMP-D06-*`.

### D-07 — Reincidência no limite exato do prazo
- **Pergunta:** o art. 39, parágrafo único, do RDPM só desconsidera a punição anterior quando o tempo decorrido é **superior** a 1, 2 ou 4 anos. No dia exato do aniversário, há reincidência?
- **Decisão:** **sim.** Exatamente 1, 2 ou 4 anos ainda conta. A partir do dia seguinte ao aniversário, não conta mais.
- **Distinção preservada:** a regra "inferior a um ano" da regressão especial (V4 §15.2) continua **excluindo** o dia do aniversário. São funções diferentes no motor (`DentroPrazoReincidencia` × `MenosDeUmAno`), cada uma com sua própria bateria de testes.
- **Impacto:** testes `REI-D07-*` (ex.: término 10/01/2025 × fato leve 10/01/2026 = reconhecer; × 11/01/2026 = não reconhecer).

### D-08 — Teste de GPU (Vulkan) na máquina do usuário
- **Contexto:** em 08/11/2026, na F0-02, o `llama-server` Vulkan com Qwen3.5-4B na RX 6600 XT coincidiu com um desligamento duro do Windows (6008, sem Kernel-Power 41 e sem dump). Ver RELATORIO_FASE0 §2.
- **Alternativas apresentadas:** (a) não repetir e adotar a GPU segura; (b) repetir com sonda mínima (2B, `-ngl 4`, 16 tokens).
- **Decisão:** **(a) não repetir.** A GPU será validada em outra máquina.
- **Impacto:** nada roda em GPU na máquina do usuário. O produto começa em CPU; a GPU só é oferecida após uma sonda curta com sentinela de falha (D-09, item 1). Os testes de instalação com GPU AMD/NVIDIA/Intel (plano §14) precisam de máquina de teste separada.

### D-09 — Ajustes ao plano vindos da Fase 0
- **Decisão:** o usuário **aprovou todos os 11 ajustes** do §5 do `RELATORIO_FASE0.md`:
  1. GPU segura: CPU por padrão, sonda curta com sentinela `gpu_sonda_em_andamento`, GPU instável → CPU com mensagem amigável; seleção manual em "avançado", com aviso.
  2. Orçamento de memória = GGUF (mmap) + pico privado (Job) + KV + margem; `0xC0000409` sob limite do Job → "memória insuficiente".
  3. Datas, números SEI, matrículas e artigos vindos de OCR entram como **não confirmados** até verificação cruzada ou confirmação humana.
  4. Datas extraídas pela IA conferidas por regex Go sobre o `trecho_literal`.
  5. Cronologia das peças (defesa prévia × alegações finais) decidida pelo Go.
  6. Wails com logger silenciado e `-H windowsgui`.
  7. Tesseract recompilado sem libtiff/jbig/libarchive/OpenSSL (sem GPL/LGPL) e com strip.
  8. Liberation Serif (OFL) empacotada como fallback.
  9. Biblioteca inicial pré-indexada no instalador; só documentos novos do usuário são embutidos localmente, em segundo plano e de forma retomável.
  10. Recuperação: instrução da consulta em inglês, chunk com cabeçalho hierárquico, tesauro jurídico curado, Coletânea dividida por diploma interno e conjunto held-out na Fase 5.
  11. Modo thinking desligado nos perfis CPU.
- **Impacto:** plano (§§7, 9, 10, 12, 14) e ADRs 003–007 atualizados em 08/11/2026.

## 2. Pendências reais (não bloqueiam o plano)

| ID | Pendência | Origem | Tratamento | Prioridade |
|---|---|---|---|---|
| P-01 | Materiais institucionais ausentes (D-04) | V4 §§3, 27.1, 32.1, 46.2 | Usuário fornece até a F7 | Alta para a F7 |
| P-02 | Ato revogador do art. 41 § 3º; texto da Lei 2.484/2011 (citada na cláusula §10, não localizada) | V4 §§4, 10, 44 | Verificação opcional; sem efeito na V1 (D-01) | Baixa |
| P-03 | Aniversário de 29/02 em ano não bissexto | V4 §§12.2-8, 15.2 | Motor retorna `ErrCasoLimite29Fev` e o app pergunta ao usuário **quando o caso ocorrer e for material** | Runtime |
| P-04 | Saldo de pontos negativo após descontos | Não tratado na V4 | Calcular, sinalizar na memória de cálculo e perguntar se for material | Runtime |
| P-05 | Processos distintos julgados em momentos diferentes | V4 §15.4 (limite da regra) | Não unificar; perguntar se ocorrer | Runtime |
| P-06 | Certificado Authenticode para assinar instaladores | V4 §34 | Instituição providencia; sem ele, o SmartScreen avisa | Média (F8) |
| P-07 | Taxa OSMF do WiX Toolset v6/v7 para gerar `.msi` | Pesquisa de 08/11/2026 | Verificar se se aplica a uso institucional; contingência `msitools/wixl` | Média (F8) |
| P-08 | Licença do Qwen2.5-3B-Instruct (provável "Qwen Research", diferente de Apache-2.0) | V4 §37 | **RESOLVIDA na F0:** Qwen Research License (não comercial) → **fora do catálogo**. Qwen3.5-2B/4B, Qwen3-4B-Instruct-2507 e Qwen3-Embedding-0.6B são Apache-2.0 (RELATORIO_FASE0 §4.5) | — |
| P-09 | PDF consolidado real do SEI e mapeamento das telas | V4 §2.1 | Telas mapeadas (F0-08). Na F2: botão `btnGerar` informado pelo usuário; `acao_ajax` **resolvido** (a geração não usa ajax); **falta** a saída do `autos-mapa` sobre um PDF real (calibra a segmentação) | Alta (fecha a F2) |
| P-10 | Desempenho do pdfium-wasm em autos longos; Tesseract via stdin; flag `--no-webui`; exclusão do WER; fallback de fonte | Técnico | **RESOLVIDA:** pdfium-wasm OK (806 págs em 0,67 s); Tesseract stdin OK; `--no-webui` OK; fallback de fonte exige empacotar Liberation Serif (F0). **Exclusão do WER validada na F1** (`WerAddExcludedApplication` por usuário, conferida no registro e reversível) | — |
| P-11 | Duplicidade real RDPM-RO.pdf × RDPM dentro da Coletânea (2015, versão anterior ao Dec. 24.965/2020) | V4 §40.6 | Tela de decisão manter/substituir no bootstrap; o usuário escolhe | Média (F5) |

## 3. Interpretações funcionais registradas pelo planejamento (sem pergunta, por decorrerem diretamente da V4)

| ID | Interpretação | Base | Observação |
|---|---|---|---|
| I-01 | Equivalências calculadas em **unidades** (L=1, M=2, G=4), com graves equivalentes = ⌊u/4⌋ e o resto descrito | V4 §§8.4, 15.1 | Reproduz exatamente todos os exemplos da V4 (2M+2L = 1G+1M; 3L = 1M+1L; 4L+2M = 2G) |
| I-02 | Gatilho do excepcional: unidades válidas anteriores < 12 **e** anteriores + atuais ≥ 12 | V4 §§8.6, 15.3 | Se as anteriores já forem ≥ 12, há inconsistência e o app pergunta |
| I-03 | Detenção limitada a 10 dias | RDPM art. 58, II, b (conferido) | A heurística 1/2/3 nunca ultrapassa esse limite; o validador confere |
| I-04 | Principal entre conexas = maior gravidade **confirmada pelo usuário**; a IA só sugere | V4 §5.1 | — |
| I-05 | Classificação PADS de condutas fora do memorando: registrada só internamente, **nunca** levada à minuta nem sugerida como nova imputação | V4 §22.1, PADS-E05 | — |

Se o usuário discordar de alguma interpretação I-xx, ela será corrigida neste registro e no catálogo antes da Fase 3.

## 3-A. Decisões técnicas da implementação (propostas técnicas, sem efeito jurídico)

| ID | Decisão | Motivo | Fase |
|---|---|---|---|
| PT-F1-01 | Os arquivos da sessão são cifrados com **AES-256-GCM e chave aleatória que só existe na memória** do processo, em vez de DPAPI | Com DPAPI, resíduos de uma queda continuariam legíveis pela conta do usuário; com a chave só em memória, eles ficam ilegíveis para sempre, e a varredura da próxima abertura os apaga (T12). A ACL restrita ao usuário continua valendo | F1 |
| PT-F1-02 | Serviços expostos à interface **nunca devolvem `error`**; devolvem `Resposta{visao, erro}` com o erro do catálogo | O Wails envia `err.Error()` ao JavaScript, o que exporia a causa técnica (V4 §33) | F1 |
| PT-F1-03 | Pasta de dados do WebView2 própria, restrita ao usuário, esvaziada na abertura e, no encerramento, por uma cópia do executável em modo "faxina" que espera o app terminar | Os processos do WebView2 só soltam os arquivos depois que o app encerra (SEG-LOG-01) | F1 |
| PT-F1-04 | Escolher "Acessar o SEI/RO" mostra o aviso TAR-004 e mantém a tarefa na escolha de entrada até a Fase 2 | Não simular funcionalidade inexistente (V4 §33) | F1 |
| PT-F1-05 | Etapa de processamento sem implementação encerra a tarefa com TAR-004 e descarta os temporários | Nunca simular sucesso | F1 (substituída por PT-F2-04) |
| PT-F2-01 | Índice de busca dos autos só em memória (SQLite `:memory:`) | Um arquivo SQLite guardaria o texto dos autos em claro, contornando a cifra da sessão | F2 |
| PT-F2-02 | A guarda do SEI aplica a lista de ações a qualquer tipo de recurso nos controladores | Uma imagem apontando para `controlador.php?acao=…` também executaria a ação | F2 |
| PT-F2-03 | Falha no SEI devolve à escolha da entrada com aviso, mantendo o tipo | T03 e V4 §30.3: oferecer o envio do PDF sem reiniciar | F2 |
| PT-F2-04 | Etapa de fase futura pára a tarefa com o aviso TAR-004, sem destruí-la | Permite consultar autos, cobertura e fontes; nada é simulado | F2 |
| PT-F2-05 | Edge sem janela (`--headless=new`), sem disfarçar o agente do navegador | Credenciais digitadas no app (V4 §33); janela visível só para diagnóstico | F2 |

## 4. Histórico
- 08/11/2026: criado o registro e respondidas D-01…D-07 durante o planejamento. Plano aprovado como planejamento, sem autorização de código.
- 08/11/2026: Fase 0 executada (autorizada pelo usuário). Resultados em `RELATORIO_FASE0.md`. P-08 resolvida; P-10 parcial (WER → F1-02). Ajustes do §5 do relatório e pendências do §7 **aguardam decisão do usuário**. Duas quedas da máquina no dia: a 1ª durante o teste Vulkan (GPU), a 2ª por falta de energia, sem perda de artefatos.
- 08/11/2026 (após a Fase 0): D-08 (não repetir o teste Vulkan) e D-09 (11 ajustes aprovados) registradas; plano e ADRs atualizados. F0-08 (SEI) segue com o usuário.
- 08/11/2026 (~14:45): F0-08 executada pelo usuário (login, 2FA, sigiloso, gerar PDF). Reautenticação de sigiloso segue a V4 §§13/33: pedir a senha de novo, sem reter. Fase 0 encerrada.
- 08/11/2026: Fase 1 (Fundação) executada com autorização do usuário. P-10 resolvida (WER). Decisões técnicas PT-F1-01…05 registradas. Resultados em `RELATORIO_FASE1.md`.
- 08/11/2026: Fase 2 implementada e testada com Edge, Tesseract e pdfium reais (`RELATORIO_FASE2.md`). PT-F2-01…05 registradas. Fechamento aguarda `acao_ajax`, `autos-mapa` sobre PDF real e teste com o SEI real.
