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

### D-10 — Migração da inferência local para a Gemini API (09/11/2026)
- **Contexto:** os modelos locais (Qwen3.5 2B/4B em CPU) levaram de 1 a 6 minutos por caso no hardware disponível (RELATORIO_FASE4 §3). O usuário decidiu substituir a inferência local pela **Gemini API**, com chave cadastrada pelo próprio usuário, chamada pelo backend Go com o SDK oficial `google.golang.org/genai`. A V4 recebeu a **Emenda 1 (seção 47)**.
- **Decisões do usuário nesta sessão:**
  1. **Nível da conta:** qualquer nível de conta, **sem aviso** na interface. Em vez do aviso, o usuário pediu que se avaliasse substituir dados pessoais por apelidos antes do envio; avaliada como viável em melhor esforço e adotada (item 5).
  2. **Confirmação de envio em cada tarefa**, antes da primeira etapa de IA, com modelo, peças/páginas, estimativa aproximada de tokens e resumo da pseudonimização.
  3. **Emendar a V4** (seção 47) em vez de registrar só aqui; regras afetadas do catálogo revogadas ou reescritas, com novas regras para o provedor.
  4. **Documentos — estratégia A:** extração local mantida (pdfium/OCR/segmentação/conferência literal); só os blocos de texto de cada etapa são enviados; sem envio de PDF, sem Files API e sem armazenamento de contexto no provedor.
  5. **Pseudonimização:** nomes de pessoas, CPF, RG, matrícula, telefone, e-mail, CEP, endereço e data de nascimento detectados localmente são trocados por marcadores e restaurados na resposta; datas dos fatos, números SEI/processo e links não são trocados.
- **Limites registrados (pseudonimização):** não é anonimização garantida. Nomes sem âncora reconhecível (posto/graduação, papel processual, rótulos), grafias quebradas por OCR ou hifenização podem passar; o contexto (datas, unidade, fatos) pode permitir reidentificação. O usuário pode informar outros nomes na confirmação.
- **Fora do escopo:** OAuth, Vertex AI, outros provedores, fallback local.
- **Impacto:** remoção de `llama-server`, GGUF, gerenciador de modelos, hardware/GPU e tela Modelos; nova tela de configurações de IA; chave guardada com DPAPI (conta do usuário); `store=false` e nenhuma ferramenta em toda requisição; nova pergunta `envio-ia`; novos códigos de erro IA-*; ADR_009 substitui o ADR_003. Regras jurídicas, motor determinístico, validadores, perguntas e SEI inalterados.
- **Pendência:** busca semântica da biblioteca (V4 §§40.4, 47.7) — decidir no início da Fase 5.

### D-11 — ChatGPT (Sign in with ChatGPT), cadastro de peças, enquadramento manual e barreira de privacidade (09/11/2026)
- **Contexto:** o usuário pediu: (1) um 2º provedor de IA, o ChatGPT, pelo fluxo oficial *Sign in with ChatGPT* (uso do plano Plus), implementado em Go; (2) entrada por peças cadastradas uma a uma, no lugar do PDF monolítico; (3) pseudonimização obrigatória com restauração local. O Gemini (D-10) é preservado. Plano aprovado: `PLANO_CHATGPT_PECAS_PRIVACIDADE.md`. As mudanças na V4 serão feitas pela **Emenda 2 (seção 48)**, cujo texto está em `PROPOSTA_EMENDA2_V4.md` e **aguarda aprovação**.
- **Reabre parcialmente a D-10:** deixa de valer "Fora do escopo: OAuth… outros provedores". Continuam fora: Vertex AI, fallback automático entre provedores e retorno à IA local.
- **Decisões do usuário nesta sessão:**
  1. **Escopo (E1):** este trabalho vai até a análise: ChatGPT, entrada por peças, enquadramento, e pseudonimização/restauração em todas as saídas de IA que já existem. Biblioteca (F5), composição, DOCX/PDF, revisão e instalador (F6–F8) ficam fora, cada um com autorização própria. O contrato de restauração fica pronto para a F6/F7.
  2. **SEI (E2):** o app lista os documentos do processo (título e nº SEI, só localmente). O usuário escolhe; o app gera o PDF pela opção "Apenas selecionados" (já permitida pela guarda) e o separa pelo carimbo SEI. Nenhuma ação nova na guarda.
  3. **Rascunhos (E3):** **não há** rascunho persistente; a V4 §29 continua valendo.
  4. **Pseudonimização (E4):** a política da §47.3 é mantida (nº SEI, nº de processo, datas dos fatos e posto/graduação ficam em claro). Somam-se uma **barreira obrigatória** antes do envio (identificador conhecido que sobrar bloqueia sem exceção) e uma **revisão item a item** dos nomes suspeitos não cadastrados.
  5. **Imputações do assistente (E5):** servem para conferência. A IA continua extraindo do Memorando o fato/conduta e a data; o formulário de congelamento (PADS-ESC-001) continua sendo a fonte do que será julgado.
  6. **Sem entrada monolítica (E6):** só peça por peça. A segmentação por carimbo fica apenas para o PDF que o próprio SEI gera com os documentos selecionados.
  7. **Enquadramento manual (E7):** os dispositivos são escolhidos pelo usuário num catálogo curado:
     - **RDPM** arts. 15 (leve), 16 (média) e 17 (grave), com natureza fixa;
     - **Estatuto** (DL 09-A/1982) arts. 29 e 32, com natureza atribuída pelo usuário por semelhança de intensidade com o RDPM.
     
     Um **fato cometido** pode ter vários dispositivos, e **cada dispositivo é uma imputação** com natureza própria. Imputações do mesmo fato são candidatas a conexas: a principal é a mais grave e as demais viram agravantes; em empate, o usuário escolhe a que mais se amolda. É a regra já vigente (V4 §5) e já implementada (`engine.SugerirPrincipal`/`AplicarConexao`), **sem mudança no motor**.
  8. **Divergência com o Memorando (E8):** dispositivo escolhido ≠ dispositivo citado no Memorando → o app mostra os dois lado a lado, com o trecho e a §22.1, e exige confirmação explícita. O app nunca sugere artigo.
  9. **Momento do enquadramento (E9):** no assistente (etapa do Memorando), cruzado depois com os fatos extraídos pela IA no formulário de confirmação.
10. **Emenda 2 aprovada** ("Tudo certo"), com acréscimos decorrentes dos Termos do SIWC fornecidos pelo usuário:
    - aviso de privacidade antes do primeiro login (Termos §2.3);
    - botão só com texto, sem logotipo nem sugestão de endosso (Termos §5);
    - **crianças menores de 13 anos** (Termos §4): o usuário escolheu **pergunta em cada tarefa + bloqueio**. Se os autos contiverem dados de criança, ou se alguma pessoa do cadastro estiver marcada como criança, o ChatGPT fica bloqueado para a tarefa, e a análise só segue com outro provedor escolhido explicitamente.
    
    Inserida na V4 como seção 48; `V4.lock` atualizado.
- **Termos do SIWC (lidos pelo usuário):** o plano já atende a:
  - identidade própria;
  - tokens guardados só no computador;
  - requisições só da máquina do usuário;
  - consentimento expresso para processamento automático (a confirmação por tarefa);
  - nenhum acesso de uso geral, nenhuma alternância de contas e nenhuma cobrança.
- **Fontes oficiais consultadas (09/11/2026):** documentação *Sign in with ChatGPT — token sharing for open source* (developers.openai.com/siwc). Em resumo:
  - OAuth com PKCE e registro dinâmico (`dynamic_agent_client`), retorno em loopback `127.0.0.1`;
  - escopo obrigatório `chatgpt.tokens.use.direct`;
  - Responses API com `store:false` e `stream:true`, sem `max_output_tokens`/`temperature`;
  - sucesso só com `response.completed`;
  - limite de 5 h do Plus compartilhado entre apps.
  
  Os termos (openai.com/policies/sign-in-with-chatgpt-terms) não puderam ser lidos automaticamente (HTTP 403); o próprio usuário os forneceu em 09/11/2026 (item 10).
- **Riscos registrados:**
  - **elegibilidade:** a documentação diz "parceiros open source e clientes privados selecionados", e o repositório ainda não é público;
  - **dados no modo SIWC:** retenção e uso para treinamento não estão documentados;
  - **saída estruturada:** suporte a `text.format` json_schema não documentado (decidido na PoC).
  
  A confirmação do usuário não substitui autorização institucional para tratar dados da PMRO em serviço externo.
- **Ficam para depois:**
  - tipo próprio para o interrogatório do acusado no PADS;
  - imagens como peça;
  - uso de *function tool* (exigiria rever a §47.3);
  - pendências específicas do Gemini.

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
| P-09 | PDF consolidado real do SEI e mapeamento das telas | V4 §2.1 | Telas mapeadas (F0-08). Na F2: botão `btnGerar` informado pelo usuário; `acao_ajax` **resolvido** (a geração não usa ajax); segmentação **validada** num PDF real com o `autos-mapa` (carimbo do SEI em toda página; 30 peças na ordem da árvore). **RESOLVIDA**; fluxo completo validado no SEI/RO real em 08/11/2026 | Alta (fecha a F2) |
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
| PT-F3-01 | Caso de 29/02 material: calcular com 28/02 e com 01/03; perguntar só se os resultados diferirem | PADS-CMP-011: não impor critério e não perguntar sem necessidade | F3 |
| PT-F3-02 | Catálogo com `rotulo` e `tipo` em lista; tipo "PT" → `ARQUITETURA`, "Processo" → `PROCESSO`; `modulo` (planejado) separado de `enforced_by` (símbolos reais conferidos) | Preservar a matriz sem perder combinações e impedir referência a código inexistente | F3 |
| PT-F3-03 | Anotações `// Cobre: <caso>` e `// Regra: <ID>` nos testes | Casos de ponta a ponta não são dados como cobertos por teste de unidade | F3 |
| PT-F3-04 | Reabrir o conjunto de imputações exige motivo, sobe a versão e invalida o julgamento da versão anterior | Versão auditável (PADS-ESC-001) | F3 |
| PT-F3-05 | SR: indícios exigem enquadramento com fonte; ausência não tem; inconclusivo pode ter **[A VALIDAR na F6]** | SR-SOL-004/005 | F3 |
| PT-F3-06 | Dependência `go.yaml.in/yaml/v3` v3.0.5 (MIT/Apache-2.0), já indireta | Leitura estrita do catálogo | F3 |
| PT-F4-01 | Catálogo de modelos: Qwen3.5-4B Q4_K_M (padrão), Qwen3.5-2B Q4_K_M (básico) e Qwen3-Embedding-0.6B Q8_0; URL do Hugging Face fixada no commit e SHA-256 conferido | Reprodutibilidade e integridade; Qwen3-4B-Instruct-2507 fora (sem vantagem na F0) | F4 |
| PT-F4-02 | Orçamento = pesos + pico privado medido (KV ajustado ao contexto, lido do GGUF, inclusive atenção híbrida) + 1,5 GB de margem; contexto = o maior da faixa da V4 §38.3 que cabe na memória livre, reduzido até 4.096; limite do Job = memória livre − pesos | D-09.2; V4 §38.3 (memória efetivamente disponível) | F4 |
| PT-F4-03 | Trecho citado pela IA é conferido literalmente no documento (normaliza caixa, acentos, aspas, travessões, espaços e hifenização); omissão aceita só de frases inteiras, em ordem e próximas, marcada com "[…]" | Teste com o modelo real: o modelo pula a linha intermediária e mantém hífen de quebra; omitir palavra dentro da frase nunca é aceito | F4 |
| PT-F4-04 | Itens que conferem são acumulados entre os reenvios; os demais são descartados, com a limitação visível | Reenvio em CPU custa ~1 min; nunca se aceita trecho não verificado | F4 |
| PT-F4-05 | Na análise, a IA não pode citar dispositivo legal que não esteja nas evidências ou nas imputações | O modelo citou "art. 37 da Lei 8.112/90" no teste real; a fundamentação jurídica vem com a biblioteca (Fases 5–6) | F4 |
| PT-F4-06 | Dispositivo da imputação conferido no memorando: artigo e inciso (romano) presentes, alínea só se escrita como alínea, diploma só com forma de diploma | O modelo repetiu o inciso na alínea e pôs a natureza no diploma no teste real | F4 |
| PT-F4-07 | Natureza só é aceita se estiver escrita no memorando; data do fato só pelo trecho literal (D-09.4) e nunca de página OCR; faltando, o campo fica vazio e obrigatório (T08) | Não inventar; o usuário informa | F4 |
| PT-F4-08 | Perguntas com campos validados no Go; respostas versionadas e revisáveis; o DAG de invalidação refaz só o que depende da resposta e reabre com os valores anteriores; autorização de fato/militar novo (SR) não é revisável nesta fase | F4-05; decisão de autorização é definitiva no domínio | F4 |
| PT-F4-09 | Erro do motor de IA (MOD-00x) pára a etapa sem destruir a tarefa; "Tentar novamente" e "Abrir modelos de IA" | O usuário instala/troca o modelo sem perder a leitura dos autos | F4 |
| PT-F4-10 | GPU nunca é usada sem a sonda manual (em "avançado", com confirmação e sentinela); com a sonda aprovada, a GPU é priorizada e falha de carga volta à CPU | D-08, D-09.1, T14; validação real da GPU fica para outra máquina | F4 |
| PT-F4-11 | GGUF importado: com hash do catálogo é reconhecido como modelo testado; fora do catálogo, só por seleção manual | V4 §37 (recomendação só entre perfis testados) | F4 |
| PT-F4-12 | Fim de linha LF fixado no repositório (`.gitattributes`) | Packs, catálogo e golden entram no hash do policy_bundle; CRLF no Windows mudava o hash | F4 |
| PT-F4-13 | Um servidor de IA por tarefa, carregado na primeira etapa de IA, mantido enquanto a tarefa aguarda respostas e reservado contra remoção do modelo | V4 §37 (um gerador por tarefa); evita recarregar 2,7 GB a cada pergunta | F4 |
| PT-F4-14 | Condutas diversas do PADS ficam só em registro interno da tarefa; não aparecem no painel nem viram pergunta | I-05; PADS-E05 | F4 |
| PT-F4-15 | Peças de trâmite (capa, notificação, ciência, juntada, remessa, procuração) não são lidas pela IA; a peça central é lida na delimitação | Tempo de CPU; nenhuma prova nelas | F4 |

## 4. Histórico
- 08/11/2026: criado o registro e respondidas D-01…D-07 durante o planejamento. Plano aprovado como planejamento, sem autorização de código.
- 08/11/2026: Fase 0 executada (autorizada pelo usuário). Resultados em `RELATORIO_FASE0.md`. P-08 resolvida; P-10 parcial (WER → F1-02). Ajustes do §5 do relatório e pendências do §7 **aguardam decisão do usuário**. Duas quedas da máquina no dia: a 1ª durante o teste Vulkan (GPU), a 2ª por falta de energia, sem perda de artefatos.
- 08/11/2026 (após a Fase 0): D-08 (não repetir o teste Vulkan) e D-09 (11 ajustes aprovados) registradas; plano e ADRs atualizados. F0-08 (SEI) segue com o usuário.
- 08/11/2026 (~14:45): F0-08 executada pelo usuário (login, 2FA, sigiloso, gerar PDF). Reautenticação de sigiloso segue a V4 §§13/33: pedir a senha de novo, sem reter. Fase 0 encerrada.
- 08/11/2026: Fase 1 (Fundação) executada com autorização do usuário. P-10 resolvida (WER). Decisões técnicas PT-F1-01…05 registradas. Resultados em `RELATORIO_FASE1.md`.
- 08/11/2026: Fase 2 implementada e testada com Edge, Tesseract e pdfium reais (`RELATORIO_FASE2.md`). PT-F2-01…05 registradas. Fechamento aguarda `acao_ajax`, `autos-mapa` sobre PDF real e teste com o SEI real.
- 08/11/2026: Fase 2 concluída. Fluxo completo validado pelo usuário no SEI/RO real (login, 2FA, sigiloso, geração e download do PDF) e com PDF real (30 peças conferidas com a árvore). Próxima: Fase 3, que exige autorização expressa.
- 08/11/2026: Fase 3 (motor determinístico) concluída com autorização do usuário (`RELATORIO_FASE3.md`). Catálogo YAML semeado da matriz (185 regras); `rulescov -estrito` verde. PT-F3-01…06 registradas. Próxima: Fase 4, que exige autorização expressa.
