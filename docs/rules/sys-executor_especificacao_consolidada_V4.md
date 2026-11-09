# SYS-EXECUTOR — ESPECIFICAÇÃO CONSOLIDADA — V4

> **Estado:** levantamento colaborativo de requisitos; **não é o prompt final de implementação**.  
> **Atualizado:** 08/11/2026 — **V4**: documento único que reúne a íntegra da V3 (seções 1–27) e as decisões transversais e de inteligência artificial aprovadas posteriormente (seções 28–46). A V3 já incluía a definição concluída das regras do Relatório de Sindicância Regular e a vinculação estrita do julgamento de PADS às imputações do Memorando Disciplinar.  
> **Finalidade deste arquivo:** transferir integralmente para uma nova conversa o contexto, as decisões, os critérios jurídicos e técnicos, as dúvidas e as próximas etapas do projeto, evitando recomeçar a discussão.  
> **Hierarquia de decisões:** as regras posteriores incluídas nesta versão substituem explicitamente disposições anteriores incompatíveis. As seções 15–20 detalham o PADS; as seções 21–27 consolidam as regras de SR, as diferenças entre os procedimentos e a revisão humana; **as seções 28–46 (V4) consolidam as decisões transversais e de IA e prevalecem sobre trechos das seções 1–27 que ainda tratavam esses assuntos como pendentes**. Os pontos das seções 1–27 alcançados por essas decisões estão sinalizados com **[Atualização V4]**, mantido o texto original para histórico. Preservar o restante.  
> **Emenda 1 (09/11/2026) — seção 47:** a inferência por modelos locais foi substituída pela **Gemini API** (Google), por decisão expressa do usuário. A seção 47 prevalece sobre as seções 28–46 **somente** quanto ao mecanismo de inferência, à conectividade exigida pela IA, aos modelos e à guarda da chave de API; os pontos alcançados estão sinalizados com **[Emenda §47]**, mantido o texto original para histórico. **Nenhuma regra jurídica, de cálculo, de perguntas, de validação, de revisão ou do SEI foi alterada.**  
> **Emenda 2 (09/11/2026) — seção 48:** por decisão expressa do usuário, o **ChatGPT** (*Sign in with ChatGPT*, plano do usuário) é admitido como segundo provedor de inferência, ao lado da Gemini API, à escolha explícita do usuário e sem troca automática; a entrada dos autos passa a ser por **peças cadastradas**; o **enquadramento** das imputações do PADS passa a ser informado pelo usuário a partir de catálogo curado; e passa a existir uma **barreira obrigatória de privacidade** antes de cada envio. A seção 48 prevalece **somente** nesses pontos; os pontos alcançados estão sinalizados com **[Emenda §48]**, mantido o texto original para histórico. **Nenhuma regra de conexão, dosimetria, reincidência, comportamento, pontuação, regressão, resultados do PADS, vinculação ao Memorando (seção 22) ou de fatos novos na SR (seção 24.3) foi alterada.**  
> **Legenda (V4):** **[APROVADO]** decisão explícita do usuário; **[PROPOSTA TÉCNICA]** detalhamento recomendado, não imposição normativa nem escolha técnica validada; **[VALIDAR EM TESTES]** solução/valor provisório dependente de prova de desempenho ou compatibilidade.  


---

## 1. Objetivo e limites da aplicação

**sys-executor** será um aplicativo **desktop para Windows** destinado a auxiliar a análise de processos administrativos disciplinares da Polícia Militar do Estado de Rondônia (PMRO), em especial:

1. **PADS — Processo Apuratório Disciplinar Sumário:** ler o processo, examinar acusações e provas, avaliar defesa, enquadramento e circunstâncias, e elaborar **minuta fundamentada de julgamento/decisão**, com cálculo proposto de sanção, pontos e comportamento disciplinar.
2. **Sindicância Regular (SR):** ler os autos e elaborar **minuta de relatório de sindicância**, segundo as regras funcionais consolidadas nas seções 21–27. **Não gerar homologação nem avocação.**

O sistema deverá automatizar a coleta do **PDF consolidado dos autos no SEI/RO**, facilitar a análise de documentos textuais e permitir revisão humana iterativa. A integração específica da IA será discutida **somente após definir todas as regras de julgamento de PADS e de relatório de Sindicância Regular**. É **ferramenta de apoio**: não substitui a autoridade competente, não profere decisão definitiva por conta própria e não pratica atos processuais de publicação ou punição autonomamente.

> **[Atualização V4]** As regras de PADS e de SR foram concluídas e, em seguida, a integração de IA foi definida: **IA local prioritária**, modelos GGUF, motor determinístico em Go, RAG jurídico local, pesquisa web desabilitada por padrão e validação obrigatória das minutas (seções 36–42). O aplicativo continua sendo ferramenta de apoio.

### 1.1. Ciclo geral de uso

1. Usuário inicia o aplicativo, autentica-se no SEI e informa o número do processo.
2. Aplicativo localiza o processo, reúne os documentos disponíveis e gera/baixa o PDF consolidado.
3. Extrai texto (inclusive com OCR quando necessário), identifica peças, provas, referências a anexos, cronologia e lacunas; alerta sobre evidências não capturadas. **Não tenta baixar automaticamente mídias/anexos externos**; solicita descrição textual ou transcrição fornecida pelo usuário conforme seção 17.
4. IA realiza análise preliminar e **faz perguntas complementares somente quando necessárias**, reaproveitando dados já identificados nos autos, solicitando confirmação se houver dúvida.
5. Usuário confirma as informações, o resultado proposto para cada imputação e os elementos necessários à dosimetria.
6. Sistema calcula consequências jurídicas propostas, pontos e comportamento; apresenta **memória de cálculo e fundamentação auditável**.
7. Gera minuta editável e versão para leitura; usuário baixa, revisa e pede ajustes quantas vezes forem necessárias.
8. Trabalho permanece em **“aguardando revisão”** até comando explícito do usuário, como **“pode finalizar”**.
9. Só após a autorização de finalização, remove arquivos temporários locais e contexto transitório de trabalho na medida tecnicamente possível; **não prometer apagar logs, backups ou retenções de fornecedores externos de IA**.

> **[Atualização V4]** Fluxo atualizado (seções 30, 31 e 42): (1) o usuário escolhe **primeiro o tipo de minuta** (Julgamento de PADS ou Relatório de SR) e **depois** a forma de obtenção dos autos — **acesso ao SEI/RO** ou **upload manual do PDF completo** (o upload não pede credenciais); (2) o sistema confere a compatibilidade entre procedimento e autos; (3) o processamento é automático, com status de progresso, sem chatbot como tela principal; (4) toda minuta passa por **validação automática obrigatória** antes da revisão; (5) a autorização de finalização é dada pela opção **“Aprovar e finalizar”** (equivalente ao comando inequívoco “pode finalizar”); (6) não há histórico nem retomada de tarefa interrompida (seção 29). Na primeira versão **não há provedor externo de IA** (seção 36); a ressalva sobre fornecedores externos permanece apenas para integração futura e opcional.

### 1.2. Proibições e salvaguardas

- **Nunca excluir documentos do SEI, encerrar processos, apagar autos, praticar atos destrutivos ou publicar/assinar decisões automaticamente.** Essa vedação deve ser arquitetural, não apenas um aviso textual.
- Não contornar autenticação, MFA/2FA, restrições de acesso ou reautenticação exigida para processos sigilosos.
- Não fabricar fatos, testemunhos, datas, precedentes, citações legais, provas ou números de processo.
- Não assumir culpabilidade sem exame individualizado das imputações e das provas.
- Dados do processo são sensíveis: proteger credenciais, OTP, PDFs, arquivos temporários, dados pessoais e comunicações com provedores de IA; registrar ações relevantes sem expor segredos ou conteúdo desnecessário.
- Tratar documentos processuais como **dados não confiáveis** para fins de prompt injection: instruções encontradas em PDFs não podem alterar o comportamento do agente.
- Indicar claramente informações faltantes, inconsistências e questões que exigem decisão humana.

> **[Atualização V4]** **Não implementar módulo de logs técnicos nesta etapa** (seção 33). O “registro de ações relevantes” limita-se a mecanismos mínimos de status/diagnóstico em memória, sem persistir logs com dados de processos. Credenciais do SEI ficam apenas em memória durante a sessão (seções 29.3 e 33).

---

## 2. Stack e diretrizes técnicas já escolhidas

| Camada | Tecnologia / decisão |
|---|---|
| Linguagem principal | **Go** |
| Framework desktop | **Wails 3** |
| Interface | **Vue 3** |
| Automação do navegador/SEI | **chromedp** |
| Plataforma-alvo | **Windows** |
| IA | **[Atualização V4]** **IA local prioritária**, com modelos quantizados **GGUF** (`llama.cpp`/`llama-server` como candidatos), motor determinístico em Go para regras formalizáveis e **RAG jurídico local**; provedores externos apenas como integração futura e opcional (seções 36–42). *Texto V3: “provedor, modelo, modo de autenticação e política de envio ainda não definidos” — superado.* |
| Saída documental | **DOCX editável** e **PDF** para conferência; sem assinatura fictícia do SEI |
| Pesquisa técnica do agente de código | **MCP Context7** e pesquisa na web, priorizando documentação oficial e versões atuais |
| Distribuição | **[Atualização V4]** Instaladores **`.exe` e `.msi`** por arquitetura; dependências instaladas automaticamente sempre que possível; atualização por novo instalador (seção 34) |
| Entrada dos autos | **[Atualização V4]** **SEI/RO automatizado** ou **upload manual do PDF consolidado** (seção 30) |

O agente de código começará em **diretório vazio chamado `sys-executor`**. Deve planejar a arquitetura e pode propor **Clean Architecture/arquitetura hexagonal** se adequada, mas **avaliar criticamente a complexidade**, preferindo separação clara de responsabilidades, testes e manutenibilidade sem abstrações excessivas. **Não presumir arquitetura fechada**.

O agente deverá consultar documentação oficial e, diante de decisões relevantes não resolvidas, **parar e perguntar ao usuário antes de concluir o plano ou implementar**. Também deverá verificar compatibilidade das versões das ferramentas. Não foram definidas neste projeto versões exatas de Go, Wails, Vue ou chromedp.

### 2.1. Automação do SEI/RO

> **[Emenda §48]** O SEI passa a fornecer só os documentos selecionados pelo usuário, gerados pela própria opção do SEI e separados pelo carimbo, sem ampliar as ações permitidas (seção 48.4).

- Tela de login conhecida: `https://sei.sistemas.ro.gov.br/sip/login.php?sigla_orgao_sistema=RO&sigla_sistema=SEI`.
- Autenticação com **usuário, senha e instituição “PM”**.
- Se o SEI solicitar **código OTP/2FA**, a interface deve solicitar ao usuário e continuar após autorização; nunca escolher desativar ou burlar 2FA.
- Processos com acesso restrito/sigiloso podem solicitar **nova autenticação**; suportar essa etapa legitimamente.
- Navegar pelo número do processo, sem depender de IDs ou links internos dinâmicos fixados no código.
- Na função **“Gerar Arquivo PDF do Processo”**, escolher **“Todos os documentos disponíveis”**, acionar **“Gerar”** e obter o PDF consolidado.
- Fazer download do **PDF consolidado produzido pela função autorizada do SEI** e processá-lo; verificar falhas de geração, PDFs incompletos, páginas não textuais e peças ausentes. Isso não autoriza coleta automática de anexos audiovisuais externos.
- Alertar sobre anexos externos, vídeos, mídias, links, provas ou documentos não incorporados ao PDF; não afirmar que foram analisados quando não foram.
- A automação deve ser **restrita a operações de consulta e obtenção** dos autos, com logs operacionais seguros e tratamento de mudanças da interface.

> **[Atualização V4]** A expressão “logs operacionais seguros” deve ser lida conforme a seção 33: **sem módulo de logs técnicos persistentes nesta etapa**. A automação do SEI é uma das duas modalidades de entrada; a alternativa é o **upload manual do PDF** (seção 30). Sem internet, o aplicativo avisa e oferece o upload, sem reiniciar a configuração (seções 30.3 e 35). As credenciais são solicitadas **a cada nova tarefa** e mantidas somente em memória (seção 33).

### 2.2. Processamento e rastreabilidade

> **[Emenda §48]** O tipo e a função de cada peça passam a ser declarados pelo usuário no cadastro de peças; a identificação automática passa a servir de conferência, e a página de referência é a dos autos montados a partir das peças (seções 48.2 e 48.4).

- Reconhecer tipos de peças: memorando disciplinar, portaria, notificações, defesa prévia, termos de oitiva, documentos probatórios, alegações finais, relatórios, decisões e anexos.
- Construir linha do tempo e identificar fatos controvertidos.
- Associar afirmações importantes à **peça de origem, identificador SEI quando disponível, página do PDF e passagem de apoio**.
- Para páginas ilegíveis, tentar extração/OCR; se documento essencial continuar ilegível, solicitar versão legível. Para contradições, examinar e fundamentar quando possível, consultando o usuário quando a divergência relevante persistir. Para provas audiovisuais inacessíveis, aplicar seção 17; nunca completar lacunas por suposição.
- Quando houver pesquisa externa de legislação/jurisprudência, utilizar fontes oficiais e **não enviar dados sigilosos do caso em buscas públicas**.

> **[Atualização V4]** Processamento de autos extensos em etapas, preservando texto integral e vínculos às fontes durante a sessão: seção 39. Pesquisa externa: seção 41.

---

## 3. Estrutura da minuta de julgamento de PADS

As duas decisões de exemplo anteriormente apresentadas pelo usuário (uma de improcedência/absolvição e outra de procedência parcial com punição) servem como **referência estrutural e de estilo**, nunca como fonte de fatos para outro caso.

### 3.1. RELATÓRIO

- Identificar procedimento, acusado, imputações do **Memorando Disciplinar** e contexto processual.
- Resumir objetivamente a acusação, defesa preliminar, provas colhidas, alegações finais e cronologia processual.
- Relatar o que as partes sustentam, sem antecipar juízo definitivo de culpa.

### 3.2. FUNDAMENTAÇÃO

- Analisar **cada transgressão imputada separadamente**, seus elementos, autoria, materialidade, documentos, testemunhos e argumentos defensivos.
- Contrapor elementos favoráveis e desfavoráveis ao acusado, avaliar coerência e suficiência probatória.
- Examinar as imputações, a defesa e os elementos probatórios, **sem implementar auditoria autônoma de nulidades nem análise/cálculo de prescrição**. O usuário é responsável pela regularidade da instrução e pela prescrição. Argumentos efetivamente constantes da defesa podem ser relatados/analisados conforme os autos, sem criar um checklist de nulidades.
- Quando não houver prova suficiente, não afirmar procedência; avaliar o **in dubio pro reo** quando cabível.
- Explicitar enquadramento normativo, justificantes, atenuantes, agravantes, conexão entre transgressões, dosimetria, pontos e comportamento.
- Manter cálculos e fundamentos auditáveis e não afirmar ter visualizado prova audiovisual não acessada. **Não exigir rotular ou registrar internamente, como requisito adicional, a origem ‘informação prestada pelo usuário’ para descrições de mídias**: seguir as instruções do usuário, sem inventar observação direta.

### 3.3. DECISÃO

- Propor resultado **procedente, parcialmente procedente ou improcedente**, por imputação.
- Quando houver sanção, indicar sua natureza, dias se cabíveis, fundamentos e consequências administrativas.
- Incluir providências de publicação, ciência, recurso, nota de punição e arquivamento, respeitando a sequência jurídica aplicável.
- **Somente a autoridade competente decide efetivamente**; o documento produzido é minuta sujeita à revisão.

### 3.4. Providências finais

**Improcedência:** propor ciência ao acusado, publicação em boletim interno e arquivamento, conforme caso.

**Procedência total ou parcial:** propor ciência, publicação do ato pertinente, observância dos recursos cabíveis, elaboração/publicação de nota de punição **somente após irrecorribilidade da decisão condenatória**, conforme redação alterada do art. 32 do RDPM, e arquivamento quando cabível. Não confundir publicação da decisão com publicação da nota de punição.

**Ponto ainda pendente de validação:** compatibilizar no texto final a proposta de regressão especial de comportamento já calculada na minuta com o marco legal de publicação e os recursos, sem efetivar atos automaticamente.

---

## 4. RDPM/RO — normas e classificação das transgressões

Norma-base: **Decreto Estadual nº 13.255/2007 (RDPM/RO)**, com alterações. O usuário disponibilizou anteriormente PDF de 28 páginas com alterações até o Decreto nº 24.965/2020. **Este Markdown não substitui o texto oficial consolidado**; o próximo agente deverá receber/consultar o regulamento e conferir sua vigência e alterações.

Classificação e sanções previstas na modelagem acordada:

| Natureza | Dispositivo de referência | Punição | Desconto ordinário |
|---|---|---|---:|
| Leve | Art. 15 | Repreensão | −1 ponto |
| Média | Art. 16 | Detenção | −2 pontos |
| Grave | Art. 17 | Prisão | −4 pontos |

**Diretriz funcional reafirmada em 07/11/2026:** o usuário **confirmou** que deseja manter a modelagem de detenção/prisão disciplinar e a cláusula de cumprimento residencial. **Situação da comprovação documental:** o documento de origem e esta conversa não identificam o ato revogador que o usuário menciona para o art. 41, § 3º. Registrar a orientação do usuário como **aprovada no projeto**, sem afirmar que a revogação foi independentemente verificada. Em futura preparação da implementação, conferir o texto oficial vigente e sinalizar conflito normativo comprovado; jamais executar/aplicar restrição de liberdade automaticamente.

---

## 5. Transgressões conexas e não conexas — art. 58, VI, e § 1º

### 5.1. Conexas

Quando uma transgressão for **causa ou consequência de outra**, examinar a conexão e escolher a transgressão **principal** conforme a regra jurídica aplicável (em geral, a de maior gravidade demonstrada). As conexas secundárias de menor gravidade podem funcionar como circunstâncias agravantes nos termos do regulamento, em vez de gerar punições autônomas. A natureza da **principal** orienta a sanção, a verificação de reincidência e o tratamento da regressão especial, sem duplicar a mesma infração no histórico.

**Cuidado:** não contar automaticamente uma conexão como múltiplas agravantes independentes nem duplicar a mesma transgressão em cálculo de equivalência. Fundamentar a escolha da principal e a incidência das circunstâncias.

### 5.2. Não conexas

Quando as transgressões forem independentes, **cada transgressão procedente deverá receber a punição correspondente**, com análise própria de prova, natureza, circunstâncias, reincidência e dosimetria. Os cálculos de pontos e comportamento devem considerar o conjunto de sanções aplicadas sem dupla contagem, respeitando os gatilhos de regressão especial. Se houver imputações improcedentes, não computá-las.

### 5.3. Conduta de perguntas

O sistema deve identificar e explicar a conexão ou ausência dela com base nos autos, pedir confirmação quando necessário e **não repetir perguntas** cujas respostas já estejam comprovadas. Para infrações não conexas, perguntar por cada infração/natureza relevante, reutilizando informação de histórico quando aplicável.

---

## 6. Circunstâncias do julgamento e dosimetria

### 6.1. Art. 36 — sete fatores obrigatórios quando houver punição

Examinar individualmente os fatores dos incisos I a VII do art. 36, com apoio nos autos, incluindo antecedentes/personalidade, motivos, natureza, danos/riscos, tempo/lugar/meios, arrependimento e consequências, respeitando a redação exata vigente do artigo. **Não omitir análise só porque o fator não altera o resultado**; indicar ausência de elementos quando pertinente.

### 6.2. Justificativas, atenuantes e agravantes

- Verificar **causas de justificação** do art. 37.
- Verificar **circunstâncias atenuantes** do art. 38.
- Verificar **circunstâncias agravantes** do art. 39, incluindo reincidência.
- Examinar circunstâncias com fundamentação específica, sem presumir sua existência.
- **Comportamento atual:** perguntar/confirmar classificação; **ótimo ou excepcional** corresponde à hipótese de atenuante do art. 38, I; **insuficiente**, à agravante do art. 39, I. Comportamento bom não preenche essas hipóteses específicas.
- **Relevantes serviços prestados à Corporação (art. 38, II):** perguntar **sim/não**; se sim, solicitar justificativa sucinta e factual, sem inventar condecorações, elogios ou feitos. Essa é uma apreciação contextual do usuário, fundamentada na minuta.
- Usar comportamento **anterior à punição atual** para examinar essas circunstâncias, não o comportamento calculado após a sanção.

### 6.3. Regra de dias proposta pelo usuário

Para punições de **detenção ou prisão**, o usuário escolheu uma **heurística inicial de dosimetria**, não apresentada como fórmula literal do RDPM:

| Comparação de circunstâncias reconhecidas | Dias sugeridos |
|---|---:|
| Nenhuma atenuante ou agravante | 2 |
| Quantidades iguais | 2 |
| Mais agravantes do que atenuantes | 3 |
| Mais atenuantes do que agravantes | 1 |

A diferença de quantidades **não multiplica** o ajuste: três agravantes e nenhuma atenuante resultam, nessa proposta, em **3 dias**, e não 5. Exibir memória de cálculo e fundamentos; permitir revisão pelo usuário; verificar compatibilidade com o art. 58, III, e com a norma vigente. Para **repreensão**, não se calcula duração em dias.

---

## 7. Reincidência — art. 39, III e parágrafo único

### 7.1. REGRA FUNCIONAL FINAL, CORRIGIDA EXPRESSAMENTE PELO USUÁRIO

**Comparar somente transgressões de natureza EQUIVALENTE: leve com leve, média com média, grave com grave.** A versão anterior, que admitia punição anterior leve e infração atual grave para reconhecer reincidência em quatro anos, foi **rejeitada e substituída**. Não reutilizá-la.

| Natureza da infração atual procedente | Natureza exigida da punição anterior | Prazo de referência |
|---|---|---|
| Leve | Leve | 1 ano |
| Média | Média | 2 anos |
| Grave | Grave | 4 anos |

O prazo compara:

1. **Data de término do cumprimento da punição anterior** (não a data do fato anterior); e
2. **Data de cometimento da transgressão atual** (não a data do julgamento).

Segundo o texto transcrito do art. 39, parágrafo único, a anterior não é considerada se houver transcorrido período **superior** ao prazo; verificar corretamente datas e casos de limite exato.

**Exemplo correto:** punição anterior **leve** encerrada em 10/01/2025 e infração atual **grave** em 10/07/2026: **não reconhecer reincidência**, porque as naturezas não coincidem. Se a anterior fosse **grave**, o intervalo de 1 ano e 6 meses estaria dentro dos 4 anos.

### 7.2. Perguntas condicionais

**Somente depois** de definir a natureza da transgressão atual procedente:

1. Perguntar: **“O acusado possui punição anterior por transgressão de natureza [LEVE/MÉDIA/GRAVE]?”**.
2. Se **não**, não exigir data nem fazer pergunta irrelevante.
3. Se **sim**, solicitar **data de término do cumprimento da punição anterior em DD/MM/AAAA**; aproveitar informação documental se já disponível e confirmar se incerta.
4. Comparar com a data do fato atual, extraída dos autos, e demonstrar cálculo e conclusão.
5. Se houver múltiplas punições anteriores da mesma natureza, permitir informar as relevantes, sem transformar automaticamente várias punições em várias agravantes.

**Conexas:** somente natureza da transgressão principal. **Não conexas:** avaliar individualmente cada transgressão procedente, conforme sua natureza, reutilizando dados sem perguntas repetidas.

Esta é a **interpretação funcional escolhida pelo usuário**, sujeita à conferência jurídica do RDPM e de orientações vigentes; não a apresentar como interpretação oficialmente pacificada sem fonte.

---

## 8. Pontuação e comportamento disciplinar — arts. 69 a 71

### 8.1. Classificação pelo saldo de pontos — art. 70

| Comportamento | Pontos positivos |
|---|---|
| Excepcional | **Mais de 40** |
| Ótimo | **30 a 40** |
| Bom | **10 a 29** |
| Insuficiente | **Menos de 10** |

A pontuação atual da ficha individual será **perguntada ao usuário** se não estiver documentalmente disponível. A classificação atual também deverá ser perguntada/confirmada; se pontos e classificação divergirem, **alertar e pedir esclarecimento** antes de calcular. No texto final, **não inventar pontuação ou comportamento**: se não forem fornecidos, usar **`[A PREENCHER]`** e sinalizar que o cálculo não pôde ser concluído.

### 8.2. Desconto ordinário — art. 71

- **Repreensão:** −1 ponto.
- **Detenção:** −2 pontos.
- **Prisão:** −4 pontos.
- O art. 71, I, também prevê **+2 pontos por ano contínuo de efetivo serviço sem punição**; a rotina de julgamento não deve presumir acréscimos não registrados na ficha. O usuário informará a pontuação atual, já refletindo o histórico administrativo.
- Quando não houver regressão especial, calcular `saldo final = saldo atual − soma dos descontos aplicáveis` e reclassificar conforme art. 70.
- Quando houver mais de uma punição independente **e não houver regressão especial acionada pelo conjunto do mesmo PADS**, considerar todos os descontos ordinários aplicáveis, sem duplicação. **Se houver regressão especial decorrente do conjunto das transgressões procedentes no mesmo julgamento, prevalece a pontuação fixa de destino, sem descontos ordinários extras das punições desse mesmo PADS** (seção 15.4).

### 8.3. Regressão especial — arts. 70, §§ 2º, 3º, 4º e 5º

A regressão especial **não é um desconto comum**. Se ocorrer, a norma atribui diretamente uma pontuação de destino, independentemente do saldo anterior. **Não descontar novamente os pontos da punição que desencadeou a regressão especial**.

| Comportamento atual | Gatilho funcional definido | Resultado especial |
|---|---|---|
| **Bom** | Equivalente a **2 transgressões graves cometidas em menos de 1 ano** | **Insuficiente, 9 pontos** |
| **Ótimo** | Equivalente a **2 transgressões graves cometidas em menos de 1 ano** | **Bom, 20 pontos** |
| **Excepcional** | A **3ª transgressão grave equivalente**, sem janela de 1 ano, pertinente ao período atual no excepcional | **Ótimo, 30 pontos** |
| **Insuficiente** | Não há nova regressão especial nessa regra | **Desconto ordinário** |

**Regra específica para insuficiente:** o § 2º ressalva quem já ingressou no insuficiente por outro motivo. Exemplo confirmado: acusado atualmente insuficiente com 7 pontos recebe uma punição grave; aplica-se −4, resultando **3 pontos, ainda insuficiente**, e não reposição automática para 9.

### 8.4. Equivalências do art. 70, § 5º

- **2 transgressões leves equivalem a 1 média**.
- **2 transgressões médias equivalem a 1 grave**.
- Consequentemente, **4 leves equivalem a 1 grave**, **8 leves a 2 graves** e **4 médias a 2 graves**.
- **Combinações mistas são permitidas** na regra funcional escolhida: por exemplo, **4 leves + 2 médias = 2 graves equivalentes**.
- O agente deve apresentar conversão e somatório, sem arredondamentos implícitos indevidos e sem contar duas vezes a mesma transgressão.
- A aplicação dessas equivalências ao gatilho de terceira grave do comportamento excepcional é **interpretação funcional aceita pelo usuário por ora**, sujeita a validação jurídica.

### 8.5. Comportamento BOM ou ÓTIMO — janela temporal

- Incluir a **data do cometimento da transgressão atual** e datas de fatos anteriores.
- Considerar apenas transgressões anteriores **reconhecidas em decisão disciplinar**, além das transgressões do processo atual consideradas procedentes.
- Para bom/ótimo, verificar o equivalente a **duas graves em intervalo inferior a um ano**, inclusive quando a infração atual for **leve ou média** e completar a equivalência com o histórico.
- Perguntar inicialmente se houve transgressões anteriores no período e suas **quantidades por natureza**; se as datas não estiverem comprovadas nos autos, solicitar **datas individuais**, para conferir corretamente a janela.
- Exemplo ilustrativo com infração atual em **06/11/2026**: consultar transgressões anteriores no período imediatamente anterior de um ano. **Não usar apenas uma pergunta “após 05/11/2025” como substituta de validação precisa do limite legal “menos de um ano”**; calcular a diferença exata e tratar a fronteira com cuidado.
- Exemplo confirmado: quatro médias cometidas em 20/12/2025, 12/02/2026, 23/07/2026 e 04/11/2026 (esta última sob julgamento) equivalem a duas graves no intervalo e acionam a regressão para quem esteja no bom ou ótimo.
- Se não atingir o gatilho, aplicar desconto ordinário e classificar pelo saldo.

### 8.6. Comportamento EXCEPCIONAL — terceira grave

- O § 4º menciona regressão **no ato de publicação da terceira transgressão grave**, sem impor a janela de um ano dos §§ 2º e 3º.
- Para a **minuta do julgamento**, o usuário determinou que, **se a transgressão atual procedente completar a terceira grave equivalente, o sistema já deverá calcular e propor a regressão para ótimo com 30 pontos**, mesmo que a publicação efetiva ocorra depois. Isso **não autoriza** o aplicativo a publicar ou alterar registros do SEI automaticamente.
- O sistema pergunta se existem transgressões anteriores reconhecidas equivalentes a graves e quantas; se houver **duas equivalentes** e a atual for uma grave equivalente, será a terceira. Para misturas de leves/médias, coletar dados suficientes para a conversão.
- **Não aplicar prazo de um ano** para essa contagem.
- **Regra funcional adicional expressamente aprovada:** contar somente as transgressões pertinentes ao **período ATUAL em que o militar está classificado como excepcional**. Não reutilizar transgressões que já ocasionaram regressão anterior de excepcional para ótimo, mesmo que depois ele tenha recuperado o excepcional.
- Se não completar o requisito da terceira grave equivalente, aplicar o desconto ordinário e recalcular comportamento.
- **Ponto jurídico a validar:** compatibilizar o cálculo antecipado na minuta com a referência legal à publicação; não afirmar que a mudança já ocorreu administrativamente antes do ato competente.

### 8.7. Transgressões conexas e não conexas no cálculo de comportamento

- **Conexas:** respeitar a eleição da principal e a absorção das secundárias como agravantes; **não duplicar infrações conexas** como se fossem punições independentes na pontuação ou equivalências.
- **Não conexas:** cada infração procedente gera sua punição e deve entrar na análise do conjunto de fatos, das equivalências e da pontuação final. **Regra posterior aprovada:** se o conjunto procedente do mesmo PADS acionar regressão especial, aplicar uma única vez a pontuação fixa correspondente, sem descontos ordinários adicionais do mesmo julgamento (seção 15.4).
- Histórico de fatos anteriores: apenas transgressões com prática já reconhecida em **decisão disciplinar**, não simples acusações pendentes. Fatos do processo atual entram somente se considerados procedentes.

### 8.8. Apresentação da memória de cálculo

O usuário deverá ver, antes de aprovar a minuta:

1. Pontuação e comportamento **atuais** informados/confirmados.
2. Imputações atuais consideradas procedentes, sua natureza, conexão e **datas dos fatos**.
3. Histórico disciplinar anterior utilizado, com natureza, datas e comprovação.
4. Equivalências aplicadas e janela temporal quando cabível.
5. Identificação de regressão especial ou desconto ordinário.
6. Pontuação e comportamento **propostos após a punição**.
7. Fundamento legal e ressalvas sobre dados ausentes ou interpretações jurídicas.

**Nunca aplicar simultaneamente a pontuação fixa de 9/20/30 pontos e descontos ordinários das punições procedentes integrantes do mesmo PADS que acionou essa regressão especial.** Ver exemplos detalhados na seção 15.4.

---

## 9. Perguntas inteligentes e interface de análise

Diretriz **expressamente aprovada pelo usuário**: **a interface deve perguntar apenas o necessário, aproveitar os dados já identificados nos autos e solicitar confirmação quando houver dúvida**, sem repetir perguntas.

Ordem sugerida:

1. Extrair e apresentar imputações, datas, provas e classificação preliminar.
2. Confirmar procedência/improcedência de cada imputação e conexão entre elas.
3. Determinar a natureza da infração principal (conexas) ou de cada infração independente.
4. Perguntar **comportamento atual**, **pontuação atual** e eventual divergência entre ambos.
5. Perguntar sobre **relevantes serviços prestados à Corporação** (sim/não; se sim, breve justificativa).
6. Para **reincidência**, perguntar somente se há punição anterior da **mesma natureza** da infração atual relevante e, se sim, a **data do término do cumprimento**.
7. Para **regressão especial bom/ótimo**, perguntar sobre infrações anteriores reconhecidas no período de um ano, suas quantidades por natureza e datas individuais se necessário.
8. Para **regressão excepcional**, perguntar sobre histórico pertinente ao **período atual de comportamento excepcional**, com quantidades equivalentes a graves, **sem filtro de um ano**.
9. Apresentar dosimetria, cálculo de pontos, comportamento proposto e memória de cálculo para validação humana.
10. Gerar minuta e permitir revisões.

**Importante:** a classificação atual usada para atenuantes/agravantes deve ser a anterior à punição, mesmo quando a minuta já prevê novo comportamento final.

---

## 10. Cláusula de cumprimento residencial de detenção/prisão — redação atualizada

**Obrigatória somente quando a minuta propuser detenção ou prisão disciplinar.** Redação aprovada, com correções gramaticais pontuais e campos parametrizados:

> Com base no artigo 45 do RDPM e considerando as especificidades da OPM, em conformidade com a Lei nº 2.484, de 06 de junho de 2011, que inclui o § 4º ao artigo 2º da Lei nº 1.063/2002 — a qual impõe ao Estado o dever de fornecer alimentação ao militar em situação de prisão ou detenção administrativa —, o(a) **{posto/graduação, matrícula e nome do(a) acusado(a)}** deverá iniciar o cumprimento da punição disciplinar em sua residência, às 08h do dia seguinte ao da sua ciência da nota de punição, a ser elaborada após o esgotamento dos recursos cabíveis, sem prejuízo de suas atribuições.

- Ajustar apenas flexões gramaticais pertinentes à identificação do militar, sem alterar o sentido da cláusula. Se faltar posto/graduação, matrícula ou nome, solicitar ou marcar campo pendente; não inventar.
- A minuta **não emite nota de punição**, não agenda nem executa sanção. A nota deve observar a irrecorribilidade, conforme seção 3.4.
- A orientação material sobre cumprimento residencial foi **confirmada pelo usuário**; a comprovação documental independente das normas vigentes/ato revogador permanece sujeita à verificação indicada na seção 4.

---

## 11. Campos pendentes, validação e revisão humana

- **Comportamento e pontuação**: anteriormente definidos como `[A PREENCHER]` quando desconhecidos; com o novo módulo, **solicitar dados para cálculo**. Se não obtidos, manter marcadores e indicar impossibilidade de cálculo confiável, sem inventar.
- Evidências incompletas: alertar e permitir ao usuário fornecer documentos adicionais ou confirmar que a análise prosseguirá com limitação expressa.
- Citações na **minuta**: redação simples com o identificador SEI quando disponível, sem exigência de página; se não houver identificador, mencionar a peça nos autos. Na **interface**: permitir abrir página e, preferencialmente, realçar trecho. Ver seção 16.
- Revisão: usuário pode pedir mudanças na minuta, baixar novas versões e continuar em estado **“aguardando revisão”**.
- Finalização: apenas após comando inequívoco **“pode finalizar”**; então limpar dados temporários locais na medida possível, sem prometer exclusão de registros de provedores externos.
- Saídas: **DOCX editável** e **PDF** sem assinaturas simuladas ou indicação falsa de que houve publicação oficial.

> **[Atualização V4]** A minuta só é apresentada depois da **validação automática obrigatória** (seção 42). A revisão ocorre com visualização da minuta completa, download DOCX/PDF e as opções **“Aprovar e finalizar”** ou **“Solicitar ajustes”** (área simples de texto, sem histórico permanente de chatbot; seção 31.3). A limpeza após finalização segue a seção 29; a formatação, a seção 32.

---

## 12. Estado das questões: resolvidas, escolhas funcionais e pendências reais (08/11/2026)

### 12.1. Definido expressamente pelo usuário

1. **Não verificar prescrição**: sem cálculo, alerta automático obrigatório ou checklist de prescrição.
2. **Não auditar nulidades ou regularidade da instrução**: presumir instrução regular; responsabilidade do usuário.
3. **Rastreabilidade simplificada na minuta**: mencionar número SEI se existente; se inexistente, indicar documento nos autos. Permitir navegar à fonte na interface com destaque preferencial.
4. **Prova insuficiente ou contraditória**: examinar e fundamentar; aplicar *in dubio pro reo* quando persistir dúvida razoável impeditiva de responsabilização.
5. **Divergência de datas**: não escolher arbitrariamente; procurar solução fundamentada e perguntar se relevante e não resolvida.
6. **OCR**: tentar extração, solicitar versão legível de peça essencial se a tentativa falhar.
7. **Pesquisa jurídica**: base vetorial de fontes oficiais e jurisprudência, com pesquisa web complementar **condicional**; atualização da base só a pedido do usuário.
8. **Anexos externos audiovisuais**: não baixar nem transcrever automaticamente; solicitar descrição textual/transcrição; permitir arquivos complementares manuais.
9. **Múltiplas punições independentes com regressão especial no mesmo PADS**: pontuação fixa de destino, sem descontos adicionais pelas infrações integrantes daquele julgamento.
10. **Reincidência**: somente mesma natureza, anterioridade efetiva do término do cumprimento, janela correspondente, regra para coincidência de datas/horários, somente uma agravante, e ausência de reconhecimento quando falta dado necessário.
11. **Comportamento excepcional**: usar apenas histórico pertinente ao período atual de excepcional; não exigir obrigatoriamente data de ingresso/retorno; não reutilizar infrações consumidas em regressão passada.
12. **Intervalo inferior a um ano**: limite por aniversário de calendário, excluindo a data exata do aniversário.
13. **Cláusula residencial**: texto da seção 10.

> **[Atualização V4]** Decisões adicionais aprovadas na V4 (sessão efêmera, modalidades de entrada, interface, formatação, segurança, instalação, modo offline, IA local, modelos, hardware, autos extensos, biblioteca jurídica, pesquisa web, validação das minutas e preservação das regras jurídicas): seções 28–46.

### 12.2. Ainda NÃO definido — não presumir

> **[Atualização V4]** Vários itens abaixo foram resolvidos nas seções 28–46. O texto original foi mantido para histórico, com a situação atual indicada em cada item.

1. **Sindicância Regular**: **CONCLUÍDO o levantamento de regras funcionais**; consultar seções 21–27. Permanecem apenas validações e decisões gerais de formato/implementação, não um novo levantamento substantivo de SR.
2. **[RESOLVIDO NA V4 — seções 36–42; escolhas finas na seção 43]** **IA**: provedor, autenticação, modelos, custos, retenção, política de processamento e integração; **adiada expressamente até após concluir regras dos dois documentos**. → IA local prioritária, sem envio de autos a terceiros; modelos exatos, embeddings e banco vetorial seguem **[VALIDAR EM TESTES]**.
3. **[RESOLVIDO NA V4 — seção 32]** **Formato de DOCX/PDF**: papel timbrado, tipografia, margens, cabeçalhos, assinatura reservada, numeração, nomenclatura e versão final. → Padrão institucional único, Times New Roman 12, `logo-pmro.png`, cabeçalhos aprovados; margens/recuos como **[PROPOSTA TÉCNICA]** segundo a redação oficial.
4. **[RESOLVIDO NA V4 — seções 29 e 33]** **Persistência de sessões**: retomada após fechamento, múltiplos trabalhos, armazenamento e criptografia específicos. → Sem histórico e sem retomada; temporários restritos à conta do Windows, com proteção adicional quando adequada; limpeza ao finalizar e resíduos removidos na próxima abertura. *Não houve decisão expressa sobre várias tarefas simultâneas; o desenho aprovado trata cada execução como tarefa independente (seção 29.1) e os limites de concorrência ficam para avaliação técnica (seção 43).*
5. **[RESOLVIDO NA V4 — seção 31]** **Interação/correção**: como apresentar e preservar revisões parciais de hipóteses, fatos, circunstâncias e recálculos em UI. → Perguntas pontuais durante a análise; ajustes por área simples de texto; regeneração preservando partes válidas e decisões confirmadas; revalidação antes de reapresentar.
6. **[RESOLVIDO NA V4 — seções 29, 33 e 34; arquitetura fina na seção 43]** **Segurança operacional e arquitetura fina**: gerenciamento de credenciais, armazenamento local, logs técnicos, empacotamento e atualização do software. → Credenciais só em memória; sem módulo de logs técnicos nesta etapa; instaladores `.exe`/`.msi`; atualização por novo instalador.
7. **Validação jurídica documental externa**: redação oficial vigente sobre cumprimento residencial/privação de liberdade e confirmação do ato revogador mencionado pelo usuário; também revisão da fundamentação de equivalências e aplicação conjunta do art. 70, com o usuário decidindo em caso de conflito comprovado. Isso **não** significa reabrir as escolhas funcionais aprovadas sem motivo. → **[Atualização V4 — seção 44]** O usuário confirmou que as interpretações aprovadas serão mantidas; a conferência normativa continua prevista apenas como salvaguarda, e só um conflito efetivamente demonstrado é levado à decisão do usuário.
8. **Casos-limite ainda não discutidos em profundidade**: data de 29/02 em aniversário anual, históricos incompletos complexos, processos com fatos distribuídos em várias janelas temporais, e possíveis efeitos de conjuntos processuais distintos. A regra geral é conhecida, mas não se devem inventar exceções. → **[Permanece pendente na V4]**: perguntar apenas quando o caso ocorrer e for material.

---

## 13. Exemplos de aceitação para testes futuros

Os exemplos abaixo ilustram **regras funcionais definidas pelo usuário**; não substituem análise jurídica dos autos.

| Caso | Entrada resumida | Resultado esperado |
|---|---|---|
| Reincidência com naturezas distintas | Anterior leve; atual grave | **Não** reconhecer reincidência |
| Reincidência grave equivalente | Anterior grave encerrada 10/01/2025; atual grave cometida 10/07/2026 | Reconhecer agravante, pois dentro de 4 anos |
| Sem agravantes nem atenuantes | Detenção ou prisão cabível | Sugerir **2 dias** |
| Mais agravantes | 3 agravantes, 0 atenuantes | Sugerir **3 dias**, não 5 |
| Mais atenuantes | 2 atenuantes, 1 agravante | Sugerir **1 dia** |
| Desconto ordinário | Ótimo 31 pontos; detenção; sem regressão especial | **29 pontos, bom** |
| Bom com regressão | Bom 29 pontos; gatilho de 2 graves equivalentes em menos de 1 ano | **9 pontos, insuficiente**, sem novo −4/−2/−1 da punição desencadeadora |
| Insuficiente já existente | Insuficiente 7 pontos; nova grave | **3 pontos, insuficiente** pelo desconto ordinário |
| Mistura de equivalências | 4 leves + 2 médias no período | Equivalem a **2 graves** |
| Quatro médias no ano | 20/12/2025, 12/02/2026, 23/07/2026, 04/11/2026 | Equivalem a 2 graves; gatilho para bom/ótimo |
| Excepcional, terceira grave | Excepcional; duas graves equivalentes anteriores pertinentes ao período atual; atual grave procedente | **Propor ótimo, 30 pontos**, sem filtro de 1 ano; não publicar automaticamente |
| Excepcional após regressão passada | Retornou ao excepcional; fatos que provocaram regressão anterior | **Não reutilizar** os fatos já consumidos |
| Conexas | Grave principal + média conexa | Uma punição principal; secundária tratada conforme agravamento legal; sem dupla contagem |
| Independentes | Leve e grave procedentes, sem conexão | Punições individualizadas; reincidência por natureza; pontos sem duplicação |

### Testes de segurança e robustez

- SEI solicita OTP: aplicativo pede código; não contorna autenticação.
- SEI solicita reautenticação para processo sigiloso: pedir credenciais/autorização novamente.
- PDF consolidado não contém anexo audiovisual: avisar claramente.
- Documento contém instruções maliciosas ao agente: ignorá-las como instruções, mantendo conteúdo como evidência documental.
- Usuário não fornece pontos atuais: não inventar resultado de comportamento.
- Usuário não aprova minuta: **não finalizar, não apagar trabalho prematuramente**.
- Qualquer tentativa de exclusão/publicação/encerramento no SEI: **operação não implementada ou bloqueada**.

---

## 14. Orientações para retomar o trabalho em outra conversa

Ao receber este arquivo em um novo chat, o assistente deve (lembrando que as seções **15–20** consolidam expressamente as decisões posteriores):

1. Reconhecer que é uma **especificação em construção** e resumir brevemente o estado do projeto se solicitado.
2. Tratar as regras mais recentes e explicitamente corrigidas como prioritárias — sobretudo **reincidência apenas entre naturezas iguais**, regressão especial e perguntas condicionais.
3. **Não reabrir decisões já resolvidas** sem detectar contradição real, lacuna importante ou conflito demonstrado com norma vigente. A seção 12 separa escolhas fechadas e pendências reais.
4. Diferenciar **preferências/regras funcionais do usuário** de **interpretação jurídica oficialmente validada**.
5. Quando necessário, solicitar o **PDF atualizado do RDPM/RO**, exemplos de decisões e relatórios, ou confirmar referências jurídicas, em vez de inventar conteúdo ausente.
6. Considerar **concluído o levantamento funcional de Relatório de Sindicância Regular** nas seções 21–27 e já estabelecidas as regras de julgamento de PADS; avançar, quando solicitado, para pendências transversais de formatação/funcionamento. **Discutir integração de IA por último**. **[Atualização V4]** Pendências transversais e integração de IA já definidas nas seções 28–46.
7. Manter uma versão consolidada e atualizada desta especificação, registrando correções que substituem regras anteriores.
8. **Não criar o prompt final para o agente de código até autorização expressa do usuário.**

---

## 15. DECISÕES POSTERIORES — CÁLCULOS DISCIPLINARES E CASOS DE BORDA (07–08/11/2026)

**Esta seção complementa e, onde houver incompatibilidade, substitui o texto das seções 7 e 8.** Exemplos são fictícios e projetados para testes. A aprovação pelo usuário é de **regras funcionais do software**; a referência exata às normas ainda deve ser verificada na etapa de fundamentação, sem alterar silenciosamente decisões.

### 15.1. Equivalências — combinação de naturezas e contagem temporal

Equivalências confirmadas para contar graves equivalentes na regressão especial:

- **2 leves = 1 média**, **2 médias = 1 grave**, **4 leves = 1 grave**.
- É permitido **somar naturezas diferentes**. Não agrupar isoladamente por natureza quando a soma válida fechar o gatilho; por exemplo, **2 médias + 4 leves = 2 graves equivalentes**.
- Também: 1 grave + 2 médias = 2 graves; 1 grave + 4 leves = 2 graves; 8 leves = 2 graves; 4 médias = 2 graves.
- Restos parciais não devem ser arredondados para uma grave inteira: 2 médias + 2 leves = 1 grave e 1 média (não duas graves). 3 leves = 1 média e 1 leve (não uma grave).
- Não usar a equivalência para modificar a **natureza da punição** ou para comparar reincidência: uma leve não se torna grave apenas por integrar uma combinação de equivalência de comportamento.
- Para **BOM e ÓTIMO**, as transgressões somadas devem integrar um conjunto de fatos cujas **datas de cometimento** estejam em **intervalo estritamente inferior a um ano**. Considerar histórico anterior reconhecido em decisão disciplinar e fatos do PADS atual quando considerados procedentes. Não contar mera acusação pendente.
- Para **EXCEPCIONAL**, manter regra autônoma da terceira grave equivalente **sem janela de um ano**, usando apenas infrações válidas para o período atual em excepcional (seção 15.3).
- Não duplicar uma mesma infração em múltiplas equivalências para alcançar artificialmente um gatilho.

**Teste E1 — quatro médias no período:** histórico médio em 20/12/2025, 12/02/2026, 23/07/2026; média atual em 04/11/2026. Quatro médias = duas graves; para militar ótimo, propor **BOM, 20 pontos**; para militar bom, **INSUFICIENTE, 9 pontos**, desde que demais condições se cumpram.

**Teste E2 — mistura confirmada:** médias em 10/01 e 15/03/2026; leves em 20/05, 10/07, 15/09 e 07/11/2026 (última no PADS atual). Dois fatos médios = uma grave; quatro leves = outra grave; duas graves equivalentes no período; militar ótimo → **BOM, 20 pontos**.

**Teste E3 — abaixo do limiar:** duas médias e duas leves no período = uma grave e uma média. **Não** ativar regressão especial bom/ótimo apenas por esse conjunto; considerar desconto(s) ordinário(s) das punições do processo atual, conforme classificação original.

### 15.2. Datas do intervalo «inferior a um ano» — aniversário exclusivo

A regra aprovada é: **um intervalo de um ano completo não é inferior a um ano**. A comparação se faz por aniversário calendário (data de mesmo dia/mês no ano seguinte), e não por um número fixo de dias como 365.

Exemplo aprovado pelo usuário:

| Fato inicial | Fato posterior | Resultado |
|---|---|---|
| 07/11/2026 | 05/11/2027 | Inferior a um ano (válido) |
| 07/11/2026 | **06/11/2027** | **Inferior a um ano (última data válida)** |
| 07/11/2026 | **07/11/2027** | **Exatamente um ano (não válido)** |
| 07/11/2026 | 08/11/2027 | Mais de um ano (não válido) |

**Distinção:** na contagem inclusiva de dias, 06/11/2027 seria o 365º dia contando 07/11/2026 como dia 1; na contagem de **tempo transcorrido**, um ano é completado no aniversário de 07/11/2027. A regra adotada usa este segundo conceito. Nunca excluir 06/11/2027 por confundir com a data do aniversário.

**Conjuntos de vários fatos:** verificar se os fatos usados para compor a equivalência podem ser reunidos num intervalo inferior a um ano, tomando suas datas relevantes e **não** somando fatos dispersos ao longo de mais de um ano sem janela válida. Ex.: fatos de 01/01/2025, 30/12/2025 e 02/01/2026 não formam todos uma única janela inferior a um ano. O agente deve demonstrar que subconjunto foi contado e não usar uma única data de referência sem comprovar as datas individuais quando necessário.

**29 de fevereiro:** o uso do calendário, e não contagem fixa, está confirmado; entretanto **não foi escolhido um critério específico para aniversário de 29/02 em ano não bissexto**. Levar à confirmação antes de aplicar esse caso-limite, não impor 28/02 ou 01/03 como escolha jurídica silenciosa.

### 15.3. Excepcional — delimitação do período atual

- Regra: a **terceira transgressão grave equivalente** pertinente ao **período atual em excepcional** aciona proposta de regressão para **ÓTIMO, 30 pontos**, sem janela anual.
- O sistema **não exige obrigatoriamente** a data em que o militar alcançou ou recuperou comportamento excepcional; solicita ao usuário confirmar quais infrações anteriores reconhecidas pertencem ao período atual nessa classificação. Usa documentos dos autos quando suficientes; pergunta somente quando necessário.
- Infrações anteriormente usadas para provocar regressão excepcional → ótimo **não podem ser reutilizadas**, mesmo após o militar retornar ao excepcional.
- O usuário aprovou a contagem antecipada **na minuta** se a sanção atual completar o gatilho, embora a alteração efetiva dependa do ato administrativo competente e da regra de publicação. O software não altera ficha funcional ou SEI.

**Teste EX1:** militar excepcional; duas graves anteriores reconhecidas e pertinentes ao período excepcional atual; uma grave procedente no PADS: **propor ótimo com 30 pontos**, sem filtro de um ano.

**Teste EX2:** militar regrediu de excepcional para ótimo em 2022 por terceira grave, recuperou excepcional em 2025, cometeu uma grave em 2026. As três graves consumidas em 2022 **não** contam novamente. Se nenhuma outra grave equivalente foi reconhecida no período atual, **não** disparar terceira grave apenas por histórico antigo.

### 15.4. Várias transgressões NÃO CONEXAS procedentes no mesmo PADS

**Decisão final, expressamente confirmada:** avaliar cada imputação e sanção de modo individualizado. Porém, **se o conjunto de transgressões procedentes do mesmo julgamento acionar regressão especial, a pontuação final do julgamento será o valor fixo da regressão: 9 (bom), 20 (ótimo) ou 30 (excepcional), conforme hipótese. Não efetuar descontos ordinários adicionais de nenhuma das sanções integrantes desse mesmo PADS, mesmo que alguma delas não seja estritamente necessária para completar o limiar.**

**Motivação funcional da decisão:** o resultado não deve variar segundo a ordem artificial de processamento de infrações dentro do mesmo julgamento; a regressão especial substitui, **naquele cálculo de comportamento**, a soma de descontos ordinários do conjunto processual. Isso não absolve ou transforma punições independentes: cada transgressão permanece individualmente procedente e recebe a sanção cabível.

| Situação com datas no período válido e fatos independentes | Resultado esperado |
|---|---|
| Ótimo, 35 pontos; 2 graves procedentes no mesmo PADS | **Bom, 20 pontos** (não 12) |
| Ótimo, 35 pontos; 3 graves procedentes no mesmo PADS | **Bom, 20 pontos** (não 16 nem 8) |
| Ótimo, 35 pontos; 2 graves + 1 leve procedentes no mesmo PADS | **Bom, 20 pontos** (não 19) |
| Bom, 29 pontos; 2 graves procedentes no mesmo PADS | **Insuficiente, 9 pontos** (não 1) |
| Ótimo, 35 pontos; 1 grave isolada sem histórico suficiente ao gatilho | **31 pontos, ótimo** (desconto ordinário −4) |
| Ótimo, 35 pontos; 1 grave + 1 leve independentes, sem histórico suficiente ao gatilho | **30 pontos, ótimo** (descontos −4 e −1) |

**Justificativa de referência para uma minuta com regressão especial decorrente do conjunto:**

> Considerando que as transgressões disciplinares reconhecidas no presente procedimento, analisadas conjuntamente, satisfazem os requisitos para a regressão especial de comportamento prevista no art. 70 do RDPM, propõe-se a reclassificação do acusado para o comportamento BOM, com atribuição de 20 (vinte) pontos. A pontuação decorre da aplicação da regra especial de regressão, que estabelece diretamente a pontuação correspondente ao novo comportamento. Assim, para fins do cálculo proposto, não se acumulam descontos ordinários referentes às punições integrantes do mesmo julgamento, evitando-se a incidência simultânea de critérios distintos sobre o conjunto de transgressões considerado.

Adaptar «BOM»/«20 pontos» para a hipótese concreta e evitar usar essa fundamentação quando o gatilho não estiver satisfeito. **A justificativa é uma proposta redacional aprovada em princípio, não jurisprudência nem citação literal do regulamento.** Fundamentar o gatilho com datas, naturezas, equivalências e identificação dos fatos, sem afirmar alteração funcional já publicada.

**Limite da regra:** refere-se a **um mesmo PADS**, na mesma proposta de julgamento. O texto não decide tratamento automático de processos distintos julgados em momentos diferentes, nem autoriza unificar decisões passadas ficticiamente.

### 15.5. Reincidência: condições cumulativas, anterioridade e horário

**Regra funcional final.** Para cada imputação procedente (principal nas conexas; individual nas independentes), considerar reincidência somente se:

1. A natureza da punição anterior for **idêntica** à da infração atual: leve/leve, média/média, grave/grave. Equivalências do comportamento **não** tornam infrações de naturezas distintas reincidência.
2. Estiver conhecida/comprovada (documentos ou informação do usuário) a **data do término do cumprimento da punição anterior**.
3. O cumprimento anterior **tiver terminado antes de ser cometida** a infração atual. Término posterior ao fato atual impede reincidência, mesmo que anteceda a decisão final do PADS.
4. O lapso entre término do cumprimento anterior e data de cometimento atual esteja dentro do prazo: **leve 1 ano; média 2 anos; grave 4 anos**, mantendo a ressalva normativa de que intervalo **superior** ao limite não conta. Caso de limite temporal exato deve respeitar texto legal vigente e não ser confundido com o «inferior a um ano» da regressão de bom/ótimo.
5. Se as **datas forem iguais**, verificar horário de término do cumprimento e horário de cometimento do fato, quando disponíveis. Só reconhecer quando a anterioridade temporal for efetivamente demonstrada; com horários iguais ou indisponíveis, **não reconhecer**.

**Falta de data:** perguntar ao usuário somente quando necessário. Se não souber a data de término, **não reconhecer agravante de reincidência**; registrar na análise que a informação necessária não pôde ser confirmada, sem inventá-la. Se ambas as datas forem iguais e os horários não constarem dos autos, perguntar horários; se não fornecidos, não reconhecer.

**Múltiplas punições anteriores válidas da mesma natureza:** mesmo duas ou mais punições dentro do prazo geram **uma única circunstância agravante de reincidência (art. 39, III)**, jamais uma agravante por ocorrência. O histórico pode ser relevante para antecedentes, mas não replicar artificialmente a mesma circunstância na dosimetria.

| Caso | Término punição anterior | Fato atual | Naturezas | Resultado |
|---|---|---|---|---|
| R1 | 10/05/2024 | 07/11/2026 | grave/grave | Reconhecer (antes e dentro de 4 anos) |
| R2 | 15/12/2026 | 07/11/2026 | grave/grave | **Não** reconhecer: cumprimento terminou depois do fato |
| R3 | desconhecida | 07/11/2026 | grave/grave | **Não** reconhecer: data não comprovada |
| R4 | 10/01/2025 | 10/07/2026 | leve/grave | **Não** reconhecer: natureza diferente |
| R5 | 07/11/2026, 08h | 07/11/2026, 16h | grave/grave | Reconhecer, se demais requisitos satisfeitos |
| R6 | 07/11/2026, 18h | 07/11/2026, 08h | grave/grave | Não reconhecer: sem anterioridade |
| R7 | 07/11/2026, sem hora | 07/11/2026, sem hora | grave/grave | Não reconhecer: sem comprovação da anterioridade |
| R8 | 07/11/2026, 08h | 07/11/2026, 08h | grave/grave | Não reconhecer: eventos simultâneos, não anterioridade |

**Caso R9 — várias punições:** anterior grave A encerrada 10/01/2020 (fora do prazo); graves B e C encerradas 10/05/2024 e 15/03/2026 (dentro dos quatro anos) e nova grave em 07/11/2026. Reconhecer **somente uma agravante** por reincidência. Se houver uma atenuante e nenhuma outra agravante, contagem 1 × 1 sugere **2 dias** pela heurística; não computar duas agravantes para sugerir 3 dias.

### 15.6. Fronteiras importantes entre cálculos

- **Reincidência** olha a **data de término do cumprimento da punição anterior** e a **data do fato atual**, e exige mesma natureza e anterioridade estrita.
- **Regressão especial bom/ótimo** olha as **datas de cometimento** das transgressões reconhecidas/procedentes, com **janela inferior a um ano** e equivalências mistas.
- **Regressão excepcional** usa transgressões do **período atual em excepcional**, sem limite de um ano, e gatilho da terceira grave equivalente.
- **Dosimetria** conta circunstâncias reconhecidas individualmente, inclusive reincidência no máximo uma vez por transgressão relevante, observando conexidade, sem contar punições anteriores múltiplas como agravantes distintas do mesmo inciso.
- **Pontuação** usa a classificação **anterior** ao PADS para atenuantes/agravantes, e só depois calcula o comportamento **proposto** após sanções. Se houver regressão especial no mesmo PADS, aplicar a regra fixa da seção 15.4; caso contrário, descontos ordinários.

---

## 16. REGRAS APROVADAS — EXTRAÇÃO DOCUMENTAL, PROVAS E MINUTAS

### 16.1. Classificação de peças pelo conteúdo e ordem processual

> **[Emenda §48]** A função de cada peça passa a ser declarada pelo usuário no cadastro (seção 48.4). A classificação pelo conteúdo e pela ordem processual descrita aqui serve de **conferência**: divergência gera pergunta, nunca troca silenciosa.

O aplicativo deve identificar automaticamente **peça e função**, sem depender apenas do título: memorando disciplinar, portaria, defesa prévia, oitivas, termos, provas, alegações finais, relatório, anexos etc. Nomes genéricos («Manifestação», «Petição», «Informação») não bastam para decidir sua função.

**Defesa prévia versus alegações finais:** examinar posição no fluxo, data, contexto, conteúdo e sequência instrutória. A defesa prévia ocorre antes das alegações finais; estas contêm, em regra, a **última argumentação defensiva antes do julgamento**. Ambas podem apresentar pedidos como arquivamento, absolvição ou improcedência; **esses pedidos não discriminam sozinhos a classificação**.

**Exemplo D1:** peça intitulada «Manifestação» apresentada depois do memorando disciplinar, antes das oitivas e da instrução: provável **defesa prévia**, considerando elementos dos autos.

**Exemplo D2:** peça «Manifestação» apresentada após a instrução, respondendo aos depoimentos e requerendo absolvição: provável **alegações finais**.

Se o conteúdo e a cronologia não permitirem classificação segura, sinalizar dúvida ao usuário; não atribuir tipo definitivo com base em nome ambíguo.

### 16.2. Divergência de datas e fatos

Se memorando citar fato em 10/08/2026, testemunha em 11/08/2026 e defesa em 09/08/2026:

1. Preservar as versões divergentes e respectivas fontes.
2. Examinar documentos, contexto e outros elementos que possam resolver discrepância.
3. Se houver fundamento suficiente, apresentar a conclusão fundamentada.
4. Se a divergência persistir e for relevante, perguntar ao usuário.
5. **Jamais escolher arbitrariamente** uma data ou preencher prova inexistente.

### 16.3. OCR, ilegibilidade e peça essencial

Tentar extração textual normal e, em páginas digitalizadas ou sem texto útil, **OCR/reconhecimento de texto**. Sinalizar baixa confiança ou falha de leitura. Se uma **peça essencial ao julgamento** não puder ser interpretada, solicitar ao usuário **versão legível**; não concluir como se a peça tivesse sido lida. Para peças não essenciais, não presumir conteúdo e avaliar a limitação no contexto da prova. Não foi aprovado reconhecimento automático de áudio/vídeo.

### 16.4. Rastreabilidade — formato na minuta e na interface

**Na minuta:** texto jurídico simples, sem sobrecarregar com marcações de páginas técnicas. Exemplo com identificador disponível:

> Conforme declarou o 3º SGT FULANO DE TAL, em seu termo de oitiva (SEI nº 0065432100), o acusado encontrava-se no local dos fatos no momento da ocorrência.

Sem identificador SEI:

> Conforme declarou o 3º SGT FULANO DE TAL, em seu termo de oitiva incluído nos autos, o acusado encontrava-se no local dos fatos no momento da ocorrência.

**Não criar número SEI fictício nem exigir mencioná-lo quando inexistente.** Não exigir número de página na redação da minuta.

**Na interface:** manter relação entre afirmações, peça, número SEI se disponível, página do PDF consolidado e passagem de apoio, para **abrir diretamente a página correspondente** e, **preferencialmente, destacar o trecho** pertinente. A limitação técnica de destaque não autoriza inventar citações; disponibilizar ao menos navegação à página.

### 16.5. Contradições e insuficiência de prova

A fundamentação escrita deve confrontar as versões e elementos favoráveis/desfavoráveis ao acusado, ponderando credibilidade e suficiência com base nos autos. **Não é exigido quadro comparativo prévio de convergências e divergências na interface.** Se permanecer dúvida razoável sobre autoria/materialidade ou responsabilidade disciplinar, aplicar *in dubio pro reo* quando cabível e propor improcedência fundamentada, sem presumir culpa.

**Exemplo P1:** testemunha A diz ter visto o acusado abandonar posto; testemunha B afirma que ele permaneceu. Não escolher uma versão automaticamente. Analisar contexto e outras provas. Se o conjunto não permitir conclusão segura, redigir fundamentação de insuficiência, com resultado compatível.

### 16.6. Prescrição e nulidades — fora do escopo de verificação automática

**Decisão inequívoca:** não implementar rotina de cálculo/verificação de prescrição ou checklist/auditoria de nulidades processuais. Presumir que o usuário conduziu instrução conforme RDPM; sua avaliação de regularidade processual e prescrição é responsabilidade humana. Isso não obriga omitir a existência de argumento defensivo expresso nos autos, mas o programa não deve gerar de ofício uma revisão processual completa nem concluir automaticamente pela nulidade/prescrição.

---

## 17. ANEXOS, VÍDEOS, DEPOIMENTOS POR VIDEOCONFERÊNCIA E FONTES ADICIONAIS

### 17.1. Escopo da automação do SEI

Gerar/baixar o **PDF consolidado** por fluxo do SEI já escolhido na seção 2.1. **Não tentar obter automaticamente anexos externos, gravações de vídeo/áudio, mídias de links ou documentos não incorporados ao PDF**, mesmo se referidos em atas. Apenas identificar e alertar sobre tais referências.

Isso **não** proíbe o uso normal de arquivos PDF do processo que já venham incorporados ao consolidado. A restrição específica recai sobre aquisição/análise automatizada de mídias externas.

### 17.2. Mídia ou depoimento não contido no consolidado

**Vídeo de ocorrência:** se o PDF afirma que «as imagens da câmera demonstram o fato», mas o vídeo não está disponível, o sistema deve **pedir ao usuário descrição textual do conteúdo e confirmação dos fatos**. Não acessar automaticamente a URL ou a mídia.

**Oitiva por videoconferência:** se houver ata com link para gravação, sem transcrição, **solicitar ao usuário a descrição do que foi dito ou upload de uma transcrição textual**. Não clicar no link para tentar transcrever automaticamente a sessão; não alegar que assistiu ao depoimento.

**Fonte das descrições do usuário:** o usuário esclareceu que **não precisa de registro interno especial rotulando a informação como fornecida por ele**; seguir suas instruções e utilizar o conteúdo fornecido para elaborar a fundamentação de modo natural. **Salvaguarda remanescente:** não apresentar falsamente o aplicativo como tendo observado pessoalmente arquivo não acessado; não inventar conteúdo ou fala.

**Exemplo V1:** usuário informa que uma testemunha declarou na gravação que viu o acusado afastar-se por 20 minutos. A fundamentação pode referir-se a esse depoimento segundo informação confirmada pelo usuário, mas sem dizer «o sistema examinou a gravação e verificou».

### 17.3. Vídeo inacessível e não descrito

Se o usuário não puder descrever conteúdo nem fornecer transcrição, o aplicativo deverá:

1. Avisar claramente qual prova foi citada mas não examinada.
2. **Não presumir** que a gravação comprova inocência ou culpa.
3. Examinar o restante do conjunto probatório.
4. Redigir **minuta com ressalva sobre a limitação**, fundamentando suficiência ou insuficiência probatória do que foi efetivamente acessível.
5. Empregar *in dubio pro reo* quando a dúvida razoável remanescente impedir afirmar a responsabilidade.

**Exemplo V2:** defesa diz «a gravação demonstra que o militar permaneceu no local», porém nem vídeo nem relato/transcrição estão disponíveis. Não presumir que a gravação existe com esse exato conteúdo; apresentar a alegação como tal e fundamentar a análise com as provas acessíveis, consignando limitação.

### 17.4. Inclusão manual de arquivos complementares

> **[Emenda §48]** A distinção entre peça de origem SEI e material fornecido pelo usuário se faz pela origem registrada no cadastro de peças (seção 48.4).

Permitir anexação manual de **documentos complementares** ao trabalho (PDF, DOCX, imagens e arquivos textuais/transcrições, conforme suporte técnico a definir). A decisão de permitir arquivos adicionais foi confirmada. Arquivos de áudio/vídeo podem ser armazenados como referências/arquivos complementares somente conforme decisão técnica posterior, **mas isso não significa autorização para sua transcrição ou análise automática**: a regra aprovada é solicitar **texto ou transcrição**. Distinguir a peça de origem SEI de material complementar fornecido pelo usuário na organização do trabalho, sem impor etiqueta obrigatória de autoria do usuário dentro da fundamentação.

---

## 18. FONTES JURÍDICAS, BASE VETORIAL E PESQUISA WEB

### 18.1. Corpus previsto

O usuário escolheu **tentar manter banco vetorial com diversas fontes oficiais**: RDPM/RO, legislação pertinente, atos normativos institucionais e **jurisprudência**. A tecnologia do banco, modelo de embeddings, indexação, atualização, custo, hospedagem e demais escolhas de IA **ainda não estão definidos** e serão avaliados por último.

Usar **somente fontes oficiais** como conteúdo jurídico de apoio. Priorizar fonte primária/normativa vigente para afirmar texto de lei ou decreto, e origem oficial para decisões judiciais/administrativas. Não tratar resultados da busca vetorial como prova de vigência ou aplicabilidade; abrir/conferir o documento de origem antes de citar artigo, julgado ou interpretação.

> **[Atualização V4]** Tecnologia e funcionamento definidos na seção 40: biblioteca jurídica **permanente** e local, separada dos autos temporários; conteúdo inicial na pasta do projeto **`docs/law_library`**; busca textual exata combinada com busca semântica; embeddings em modelo separado, gerenciado automaticamente; categorias, duplicidades, versões e exclusão com confirmação. `chromem-go`, SQLite FTS5 e `Qwen3-Embedding-0.6B` são **candidatos**, não escolhas fechadas.

### 18.2. Política de pesquisa

**Modelo B aprovado — condicional:** pesquisar jurisprudência e informação jurídica complementar quando houver **controvérsia relevante** ou necessidade de fundamentação adicional. Não fazer varredura jurisprudencial indiscriminada em todo PADS; não depender de pesquisa para reproduzir fato do processo.

**Consulta habitual à base existente** é permitida e normal. **Pesquisa web externa** pode ser usada de modo complementar quando necessária e tecnicamente viável, **somente em fontes oficiais** e sem expor detalhes sigilosos dos autos em consultas públicas. A forma operacional de proteção dos dados será definida com a arquitetura da IA.

> **[Atualização V4]** Forma operacional definida na seção 41: pesquisa na internet **desabilitada por padrão na primeira instalação**, habilitável nas configurações (preferência persistente); quando habilitada, pesquisa automática e condicional em fontes oficiais, sem confirmação por consulta e **sem transmitir nomes, dados pessoais, identificadores ou informações sigilosas** dos autos; resultados não entram automaticamente no corpus. Offline ou com a opção desligada, usa-se só a base local.

### 18.3. Atualização normativa (diferente de consulta)

A base jurídica deve distinguir:

- **Consultar:** recuperar e conferir dados já indexados, como parte normal da elaboração da fundamentação.
- **Pesquisar pontualmente na web:** buscar fonte oficial em controvérsia relevante, conforme modelo B, sem implicar por si só atualização sistemática do banco.
- **Atualizar base/corpus:** incorporar novos textos, versões, atos e jurisprudência à base — **somente por solicitação expressa do usuário**. Não promover atualização automática no início da aplicação, periodicamente ou ao consultar.

Em caso de informação divergente sobre vigência, indicar a divergência e sua origem, sem sobrescrever silenciosamente as premissas do usuário. Registrar data/versão da base de maneira tecnicamente viável a definir.

> **[Atualização V4]** Controle de versões, duplicidades, substituição e rastreabilidade: seção 40.6.

---

## 19. MATRIZ ESTENDIDA DE ACEITAÇÃO E SITUAÇÕES DE TESTE

Esta tabela é uma extensão da seção 13 e contém hipóteses para testes funcionais, **não atos jurídicos automaticamente válidos**.

| ID | Situação de entrada | Saída/conduta obrigatória |
|---|---|---|
| A01 | Peça «Manifestação» anterior às oitivas, com pedidos de absolvição | Avaliar pelo conteúdo e cronologia; provável defesa prévia; não classificar como alegações finais só pelo pedido |
| A02 | «Manifestação» após instrução, comentando provas | Prováveis alegações finais, sujeitas à conferência contextual |
| A03 | Datas de fato 09, 10 e 11/08/2026 em peças diferentes | Expor divergências; não escolher arbitrariamente; perguntar quando relevante e irresoluta |
| A04 | Digitalização essencial ilegível | Tentar OCR; solicitar versão legível se OCR não resolver |
| A05 | Oitiva com SEI 0065432100 | Citação natural com número SEI; interface abre página original |
| A06 | Oitiva sem número SEI disponível | Citação «termo de oitiva incluído nos autos», sem ID inventado |
| A07 | Testemunhas se contradizem sobre presença do acusado | Fundamentar confronto; *in dubio pro reo* se persistir dúvida relevante |
| A08 | PDF cita vídeo não incorporado | Alertar e solicitar descrição textual; não baixar/acessar automaticamente vídeo |
| A09 | Ata de videoconferência com link e sem transcrição | Solicitar descrição ou upload de transcrição; não transcrever automaticamente |
| A10 | Gravação mencionada, não disponível nem descrita | Não pressupor conteúdo; fundamentar a partir das provas acessíveis e ressalvar limitação |
| A11 | Usuário complementa relato de depoimento | Seguir instrução; sem obrigação de registrar etiqueta «informação do usuário»; não afirmar observação direta da gravação |
| A12 | Oitiva, foto ou documento adicional manual | Aceitar inclusão manual conforme tipos suportados; distinguir origem operacional no trabalho |
| A13 | Jurisprudência relevante à suficiência da prova | Pesquisar condicionalmente em fontes oficiais; conferir decisão original antes de citar |
| A14 | Aplicativo iniciado com acesso à internet | **Não** atualizar automaticamente base normativa |
| A15 | Usuário solicita atualizar corpus jurídico | Permitir fluxo específico de atualização futura, com consulta a fontes oficiais |
| A16 | Suspeita de prescrição | Não calcular nem checar autonomamente prescrição |
| A17 | Instrução do PADS aparentemente irregular | Não iniciar auditoria automática de nulidades, preservada leitura dos argumentos efetivamente presentes |
| A18 | Ótimo 35, 2 graves + 1 leve independentes, mesma decisão e datas válidas | Bom **20 pontos**, sem −1 adicional |
| A19 | Ótimo 35, 3 graves independentes, mesma decisão e datas válidas | Bom **20 pontos**, sem −4 extra |
| A20 | Bom 29, 2 graves equivalentes no período | Insuficiente **9 pontos**, sem descontos adicionais do mesmo PADS |
| A21 | Leve e grave anteriores/atuais de naturezas distintas | Reincidência **não** caracterizada |
| A22 | Término anterior após fato atual | Reincidência **não** caracterizada |
| A23 | Datas iguais, anterior encerra às 08h, fato às 16h | Reconhecer reincidência se mesma natureza e prazo correto |
| A24 | Datas iguais, horários faltantes ou simultâneos | **Não** reconhecer reincidência |
| A25 | Várias punições anteriores da mesma natureza dentro do prazo | Uma agravante de reincidência, não uma por punição |
| A26 | Término da punição anterior desconhecido | Perguntar se possível; se não fornecido, não aplicar reincidência |
| A27 | Fato 07/11/2026 e outro 06/11/2027 | Intervalo ainda inferior a um ano |
| A28 | Fato 07/11/2026 e outro 07/11/2027 | Intervalo de exatamente um ano; não válido para gatilho «menos de um ano» |
| A29 | Excepcional recuperado após regressão pretérita | Não reutilizar transgressões consumidas na regressão passada |
| A30 | Excepcional com duas graves reconhecidas no período atual + terceira grave atual | Propor ótimo **30 pontos**, sem aguardar publicação para cálculo da minuta; não alterar ficha/SEI |
| A31 | Usuário pede gerar documento final sem autorizar encerramento do trabalho | Pode gerar versão DOCX/PDF para revisão; manter «aguardando revisão» até comando explícito |

---

## 20. HISTÓRICO DE DECISÕES E ORDEM DE CONTINUIDADE

### 20.1. Histórico resumido da consolidação

- **Até 06/11/2026:** especificação inicial de 14 seções, stack desktop, SEI, PADS, art. 36–39, reincidência por mesma natureza, comportamento e regressões, entrega de minutas, salvaguardas.
- **07/11/2026 — comportamento:** confirmado valor fixo da regressão especial para múltiplas independentes no mesmo PADS; equivalências mistas autorizadas; intervalo «menos de um ano» por aniversário calendário; histórico atual no excepcional pode ser confirmado sem data obrigatória de reingresso.
- **07/11/2026 — escopo jurídico:** sistema não analisará prescrição nem auditará nulidades; usuário assumirá avaliação de regularidade da instrução; ausência de prova deve ser fundamentada e não presumida.
- **07/11/2026 — cumprimento residencial:** aprovado parágrafo da seção 10 com posto/graduação, matrícula e nome, ajustando gramática; orientação de detenção/prisão reafirmada.
- **07/11/2026 — reincidência:** exige término da punição anterior *anterior ao fato atual*; igualdade de datas exige horários se disponíveis; sem prova da anterioridade, não reconhecer; várias punições válidas geram uma agravante apenas.
- **07/11/2026 — prova e SEI:** identificação semântica/cronológica de peças, OCR e versão legível de peça essencial, citações simples na minuta, página/trecho na interface, fundamentação direta das contradições, *in dubio pro reo* cabível.
- **07/11/2026 — mídia externa:** não obtenção nem transcrição automática; solicitar descrição ou transcrição; anexação manual permitida; conteúdo não acessado não deve ser inventado.
- **07/11/2026 — pesquisa jurídica:** base vetorial proposta com fontes oficiais e jurisprudência, web complementar condicional, atualização somente por pedido; arquitetura da IA adiada até finalizar regras dos documentos.
- **08/11/2026:** compilação documental das decisões acima, **sem produzir prompt de implementação**.
- **08/11/2026 — V3:** conclusão das regras funcionais de Sindicância Regular e vinculação estrita do PADS ao Memorando Disciplinar (seções 21–27).
- **08/11/2026 — V4:** decisões transversais e de IA (sessão efêmera, duas modalidades de entrada, interface sem chatbot, formatação institucional, segurança, instalação, modo offline, IA local, modelos, hardware, autos extensos, biblioteca jurídica, pesquisa web, validação obrigatória e preservação das regras jurídicas) incorporadas nas seções 28–46, **sem produzir prompt de implementação**.

### 20.2. Próximo ponto de discussão

**Etapa de definição funcional do RELATÓRIO DE SINDICÂNCIA REGULAR CONCLUÍDA em 08/11/2026**; consultar seções 21–27, fundamentadas nos dois modelos trazidos pelo usuário. Não transpor automaticamente regras de julgamento de PADS para SR.

Após regras dos dois documentos, discutir pendências de formato, fluxo de revisão e persistência e, **por último**, provedor/arquitetura de IA. **Não gerar prompt definitivo de implementação sem autorização expressa.**

> **[Atualização V4]** As pendências de formato, revisão, persistência e IA foram decididas (seções 28–46). O levantamento de requisitos está concluído. **Próximo passo:** o usuário revisa esta V4 como fonte da verdade e, somente quando autorizar expressamente, elabora-se o prompt definitivo de implementação.

---

## ESTADO FINAL DESTE DOCUMENTO

> **[Atualização V4]** Registro histórico do estado anterior às seções 21–46. O estado vigente está ao final do documento (“ESTADO FINAL — V4”).

**V3 (08/11/2026): REGRAS FUNCIONAIS DE PADS E DE RELATÓRIO DE SINDICÂNCIA REGULAR CONSOLIDADAS. PERMANECEM PENDÊNCIAS TRANSVERSAIS, DE VALIDAÇÃO JURÍDICA DOCUMENTAL E DE ARQUITETURA/INTEGRAÇÃO COM IA, EXPRESSAMENTE ADIADA PARA A ETAPA POSTERIOR. O USUÁRIO REVISARÁ E APROVARÁ TODA MINUTA. ESTE ARQUIVO CONTINUA SENDO UMA ESPECIFICAÇÃO EM CONSTRUÇÃO, NÃO UM PROMPT DE IMPLEMENTAÇÃO. O PROMPT DEFINITIVO SOMENTE PODERÁ SER GERADO APÓS AUTORIZAÇÃO EXPRESSA DO USUÁRIO.**


---

# CONSOLIDAÇÃO V3 — REGRAS FUNCIONAIS DA SINDICÂNCIA REGULAR E LIMITE DO JULGAMENTO DE PADS

> **Estado:** estas seções registram as decisões **expressamente aprovadas pelo usuário** na continuação de 08/11/2026. Integram a especificação e substituem declarações anteriores de que a estrutura/regras de SR ainda precisavam ser definidas. **Não são o prompt de implementação.**
>
> **Critério de precedência:** ao tratar de PADS, aplicar as seções anteriores e especialmente a seção 22, sobre a vinculação às acusações iniciais. Ao tratar de SR, aplicar integralmente as seções 21 e 23–26. As regras transversais são aplicáveis a ambas, quando compatíveis com a natureza específica de cada uma.

## 21. DIFERENÇAS ESTRUTURANTES ENTRE PADS E SINDICÂNCIA REGULAR

### 21.1. Definições aprovadas — natureza, partes e finalidade

| Aspecto | PADS | Sindicância Regular (SR) |
|---|---|---|
| Nome | **Processo Apuratório Disciplinar Sumário** | **Sindicância Regular** |
| Natureza | **Processo** disciplinar de julgamento | **Procedimento** de investigação sumária, de natureza inquisitorial |
| Objeto | **Acusações delimitadas no Memorando Disciplinar** | Fatos investigados a partir da portaria, com possibilidade excepcional de fatos supervenientemente revelados, mediante consulta prévia ao usuário |
| Envolvidos principais | **Um acusado por PADS** | **Um ou mais sindicados**, podendo existir vários ofendidos |
| Defesa | Contraditório e ampla defesa; acusado pode defender-se e produzir provas na instrução | Não há contraditório/ampla defesa próprios de processo acusatório nessa fase inquisitorial; os sindicados apresentam sua versão quando ouvidos em termo |
| Documento produzido | **Minuta de julgamento de PADS**, com relatório, fundamentação e decisão | **Minuta de Relatório de Sindicância Regular**, com as quatro seções fixadas na seção 23 |
| Finalidade da conclusão | Procedência, procedência parcial ou improcedência **das acusações iniciais**, com consequências disciplinares quando cabíveis | Apontar, para cada fato e sindicado, existência/ausência de indícios de crime comum, crime militar e transgressão disciplinar, ou caráter inconclusivo devidamente fundamentado |
| Sanção e comportamento | Punição, dosimetria, reincidência, pontos e regressão quando cabíveis, conforme regras aprovadas | **Não** aplicar punição, dosimetria, reincidência, comportamento, pontuação ou regressão |
| Etapas posteriores | Atos da autoridade competente, recursos, nota de punição e execução fora da automação | **Homologação e avocação não serão geradas**; atos posteriores são de competência externa ao aplicativo |

O sistema não deve confundir **acusado** (PADS) com **sindicado** (SR), nem chamar a SR de processo disciplinar julgador. A distinção não é apenas terminológica: altera os limites da apuração, a suficiência probatória e o tipo de documento.

### 21.2. Limites de atuação e autoridade

- A ferramenta produz **minutas para revisão humana**, não decisões administrativas ou judiciais definitivas.
- **SR:** o relatório é elaborado na perspectiva do encarregado; não produzir documentos de homologação ou avocação, não decidir punição e não implementar práticas posteriores a cargo da autoridade.
- **PADS:** o julgamento só pode versar sobre as imputações formais do Memorando Disciplinar, observadas todas as demais regras já definidas de fundamentação, decisão e dosimetria.
- **Ambos:** ao concluir a elaboração de uma minuta, informar explicitamente que o usuário deve revisar integralmente fatos, provas, fundamentos, enquadramentos e conclusões; a decisão final de aprovar ou utilizar a minuta é do usuário/autoridade competente. A versão DOCX/PDF pode ser gerada para conferência antes da finalização. O trabalho continua **aguardando revisão** até autorização expressa.

## 22. REGRA DE ESCOPO OBRIGATÓRIA DO PADS: VINCULAÇÃO ÀS IMPUTAÇÕES DO MEMORANDO DISCIPLINAR

### 22.1. Regra expressa e prevalente

> **[Emenda §48]** Regra **mantida integralmente**. Os dispositivos e a natureza das imputações passam a ser informados pelo usuário a partir do catálogo curado (seção 48.5), conferidos contra o texto do Memorando; divergência exige confirmação expressa, e o aplicativo nunca sugere troca de dispositivo.

**O julgamento de PADS está estritamente vinculado à acusação ou às acusações descritas no Memorando Disciplinar inicial.** Não ampliar, substituir, alterar ou criar imputação nova no julgamento para aproveitar transgressão diversa identificada durante a instrução. O Memorando Disciplinar é a referência delimitadora do objeto acusatório.

A ocorrência de outra conduta potencialmente transgressional durante a instrução **não autoriza** o sys-executor a tratá-la como nova acusação, julgá-la procedente ou impor sanção por ela no PADS em análise. Não inserir no documento uma indicação autônoma de nova transgressão fora do objeto inicial, nem encaminhamento automático para responsabilização por esse fato. Não propor ao usuário uma simples troca do artigo ou descrição para transformar acusação não demonstrada em outra distinta.

**Atenção à distinção:** esta decisão é sobre transgressão **não contida nas imputações originais**, e não implica deixar de examinar o suporte jurídico e probatório das acusações efetivamente formalizadas. Quando uma questão de classificação for materialmente ambígua, não inovar na imputação; tratar a dúvida com o usuário em vez de ampliar silenciosamente o objeto.

### 22.2. Resultados admitidos

- **Procedência:** todas as acusações originais efetivamente comprovadas, conforme avaliação das provas e fundamentos pertinentes.
- **Procedência parcial:** parte das acusações originais comprovada e parte não comprovada. Somente imputações originalmente formuladas e consideradas procedentes podem fundamentar sanção.
- **Improcedência/absolvição:** nenhuma acusação original comprovada. Mesmo se a instrução sugerir outra transgressão **fora do Memorando**, isso **não altera o resultado** nem legitima punição neste PADS.

### 22.3. Exemplos de aceitação

**PADS-E01 — nova conduta descoberta, acusação original refutada.** Memorando: abandono de posto. Instrução demonstra permanência no posto, mas revela outra possível transgressão não acusada. **Esperado:** improcedência da acusação original e absolvição; não adicionar outra infração ou punição.

**PADS-E02 — duas imputações iniciais, terceira encontrada.** Memorando: imputações A e B. A comprovada, B não comprovada; surge C, não descrita. **Esperado:** procedência parcial restrita a A, sem julgar nem punir C.

**PADS-E03 — acusações iniciais não comprovadas.** Nenhuma imputação formal do memorando se sustenta, ainda que existam indícios de outra conduta. **Esperado:** improcedência/absolvição de todas as acusações originárias.

**PADS-E04 — duas acusações comprovadas.** Ambas constam do memorando e ambas têm suporte probatório suficiente. **Esperado:** procedência de ambas, aplicando conexidade/não conexidade e consequências já aprovadas; sem fatos adicionais.

**PADS-E05 — tentativa de adicionar item novo.** Extrator/documento aponta potencial infração distinta, sem previsão no memorando. **Esperado:** não acrescer imputação na minuta de julgamento; respeitar a delimitação original, independentemente do fato de ter sido relatada por testemunha, sindicado/acusado ou documento posterior.

## 23. RELATÓRIO DE SINDICÂNCIA REGULAR — ESTRUTURA FIXA E PREENCHIMENTO

### 23.1. Quatro seções obrigatórias

A estrutura **aprovada e obrigatória** é exatamente:

```text
I – DADOS DA SINDICÂNCIA
    a. Portaria:
    b. Sindicados:
    c. Ofendido:
    d. Testemunhas ouvidas:
    e. Perícias realizadas:
    f. Outras informações:

II – DA SÍNTESE DOS FATOS EM APURAÇÃO

III – ANÁLISE DAS PROVAS

IV – SOLUÇÃO
```

Variações de numeração presentes em modelos históricos (por exemplo, relatório que chama a síntese de III, a análise de IV e a solução de V) **não substituem** a estrutura de quatro seções acima. Preservar os seis rótulos e sua ordem, mesmo quando alguma alínea permanecer vazia. A organização jurídica interna da fundamentação pode ser contínua, sem subtítulos artificiais obrigatórios.

### 23.2. Fontes de preenchimento e regra de segurança

- Extrair informações dos autos e de complementações disponíveis; **preencher automaticamente apenas quando houver certeza suficiente** sobre o dado e sua correspondência com o campo.
- Se informação faltar ou houver divergência relevante não resolvida, **manter somente o rótulo da alínea sem conteúdo**, informar ao usuário para conferir/preencher manualmente e não inventar informação.
- Quando identificado, mencionar **identificador SEI** junto a testemunhas, termos, perícias ou peças relevantes, conforme exemplos. Sem identificador comprovado, omitir o número, sem fabricar código ou referência.
- Registrar a condição da pessoa **por ato e contexto**, e não apenas pelo nome ou pela primeira aparição. Uma pessoa pode ser ofendido, declarante, testemunha ou sindicado em atos diferentes; não duplicar indevidamente nem atribuir categoria equivocada.
- Aceitar **múltiplos sindicados e múltiplos ofendidos**.

### 23.3. Conteúdo exato das alíneas

**a. Portaria:** número e identificação da portaria de instauração; data, quando suficientemente identificada e pertinente; não confundir a portaria com peça de outra sindicância apensada/juntada.

**b. Sindicados:** policiais militares formalmente relacionados na investigação e outros cuja inclusão na análise/solução tenha sido **expressamente autorizada pelo usuário** segundo a seção 24.3. Identificações pessoais/funcionais somente quando seguras. Podem ser vários.

**c. Ofendido:** pessoa(s) ou ente ofendido(s) pelo fato investigado, quando identificados com segurança. Pode haver vários. Não atribuir automaticamente a qualidade de ofendido a toda pessoa que narra um fato.

**d. Testemunhas ouvidas:** relacionar **pessoas efetivamente ouvidas como testemunhas**, conforme Termos de Inquirição e conteúdo/condição do ato, incluindo SEI quando disponível. Não listar indiscriminadamente sindicados, ofendidos ou declarantes apenas por terem prestado depoimento.

**e. Perícias realizadas:** relacionar os exames e laudos periciais efetivamente constantes dos autos, com número do laudo/exame e SEI se disponíveis. Se não identificados com segurança, deixar `e. Perícias realizadas:` sem preenchimento. Resultado ou teor da perícia deve ser discutido na seção III, não inferido só de sua menção.

**f. Outras informações:** **seleção relevante e sucinta**, como Termos de Declarações de pessoas não ouvidas como testemunhas, relatórios técnicos importantes que não sejam perícias e documentos de outros procedimentos com influência direta na investigação. Não transformar a alínea em inventário completo de BOs, ofícios, certidões e demais peças, que poderão ser examinados/citados na seção III se relevantes. Evitar repetir itens já listados em outras alíneas.

### 23.4. Exemplos de preenchimento e pendência

```text
I – DADOS DA SINDICÂNCIA

a. Portaria: Portaria nº 00/2026/PM-EXEMPLO, de 10 de outubro de 2026.
b. Sindicados: 3º SGT PM [IDENTIFICAÇÃO CONFIRMADA] (SEI nº 0000000000).
c. Ofendido:
d. Testemunhas ouvidas: [NOME CONFIRMADO] (SEI nº 0000000001).
e. Perícias realizadas:
f. Outras informações: Termo de Declarações de [NOME CONFIRMADO] (SEI nº 0000000002).
```

**Atenção:** nomes e códigos acima são **exclusivamente ilustrativos, fictícios e não utilizáveis como dados reais**; em implementação, não preencher exemplos fictícios em campos faltantes. Quando o dado for duvidoso, a saída correta é manter **somente** `c. Ofendido:` ou `e. Perícias realizadas:`, por exemplo, com aviso de revisão na interface.

### 23.5. II – DA SÍNTESE DOS FATOS EM APURAÇÃO

- Elaborar prioritariamente a partir da **Portaria de Instauração** e documentos que deram origem à investigação.
- Resumir de modo objetivo a notícia inicial, condutas supostas, datas, locais e pessoas envolvidos, quando seguros.
- Preservar a natureza preliminar das informações: empregar, quando pertinente, fórmulas como «segundo informações iniciais», «conforme a portaria», «supostamente», sem afirmar como já demonstrada a imputação.
- **Não antecipar** indícios, culpa, subsunção definitiva ou conclusão.
- Reservar fatos **novos efetivamente identificados na instrução** e a avaliação das provas para a seção III, respeitando a autorização prévia para inclusão de fato novo fora do escopo (seção 24.3).
- Não transformar a síntese em reprodução integral da portaria ou em relatório de toda a instrução.

## 24. III – ANÁLISE DAS PROVAS: ORGANIZAÇÃO, PROVAS, PARTICIPANTES E ALCANCE

### 24.1. Organização aprovada — fatos primeiro, pessoas em cada fato

A fundamentação será organizada **prioritariamente pelos fatos investigados**. Dentro de cada fato, examinar as condutas individuais dos envolvidos, os depoimentos, as provas documentais/periciais e eventuais contradições. Não é obrigatório criar subtítulos para cada militar ou fato quando o texto contínuo for mais claro.

- Diferenciar fatos investigados e hipóteses relevantes; evitar tratar a sindicância como um bloco indistinto de culpa coletiva.
- Examinar a participação **de cada sindicado** em cada fato, inclusive quando o conjunto probatório é comum.
- Fatos independentes podem produzir resultados distintos para a mesma pessoa.
- Um mesmo fato pode ter resultado diferente para sindicados diversos, segundo os elementos de cada conduta.
- Não exigir prova para condenação disciplinar: o relatório busca **indícios fundados**, não julgamento definitivo.

### 24.2. Tratamento de provas e qualidade argumentativa

**Depoimentos e termos:**
1. Sintetizar fielmente o que testemunhas, declarantes e sindicados disseram, **sem inventar ou alterar o significado** das falas.
2. Empregar transcrição literal **pontual** quando indispensável/útil à fundamentação; identificar depoente e peça/SEI. Evitar transcrições integrais desnecessárias.
3. Avaliar as declarações no contexto de toda a prova, não presumindo automaticamente veracidade ou falsidade.
4. Confrontar divergências relevantes, indicando se fortalecem, enfraquecem ou são neutras para a hipótese investigada.
5. **Não interpretar automaticamente omissão** em depoimento como tentativa de ocultação, reconhecimento de culpa ou comprovação do fato.
6. Atribuir precisamente cada informação ao declarante e ao fato/sindicado relacionado.

**Contradições:**
- Nem toda contradição equivale a indício de irregularidade. Examinar materialidade, relevância, contexto, coerência, apoio externo e influência no objeto investigado.
- Se divergência relevante persistir, expor a limitação e seus efeitos sobre a suficiência dos indícios; não escolher arbitrariamente uma versão.
- Divergências de datas: registrar as versões e fontes, examinar possibilidade de resolução fundamentada e consultar usuário apenas quando a questão for relevante e não solucionada (seções 16.2 e 17).

**Perícias:**
- Relacionar laudos na alínea `e` da seção I e analisar seu conteúdo na seção III, em conexão com demais provas.
- **Ausência de vestígios periciais não prova, isoladamente, que o fato alegado não ocorreu**. Não inferir verdade absoluta a partir de laudo negativo. Verificar tipo de fato narrado, limites técnicos, data do exame, depoimentos e demais elementos.
- Se o laudo for inconclusivo ou limitado, reproduzir corretamente suas limitações sem atribuir conclusões que o perito não sustentou.
- A inexistência de perícia não é, por si, constatação de que nenhum fato aconteceu.

**Documentos de outros procedimentos:**
- É permitido examinar inquéritos policiais, outras sindicâncias e peças correlatas **quando incorporados aos autos da SR analisada**.
- Identificar, quando possível, **procedimento e documento de origem**, SEI e contexto.
- Avaliar a pertinência real dos elementos ao fato e sindicado; **não importar automaticamente** conclusões de inquérito ou outra sindicância como verdade definitiva.
- Distinguir fatos demonstrados de suspeitas, indiciamentos e conclusões preliminares de investigações diversas.

**Mídias e peças ausentes:**
- Aplicar as mesmas salvaguardas aprovadas para PADS (seções 16–17): identificar vídeo/gravação/documento ausente do PDF consolidado; **não obter nem transcrever automaticamente** áudio/vídeo externo; pedir ao usuário descrição textual/transcrição ou arquivo complementar pertinente.
- Na indisponibilidade, não presumir conteúdo favorável ou desfavorável; examinar provas restantes e consignar limitação quando relevante.
- Não exigir etiqueta interna obrigatória de «informação fornecida pelo usuário» no corpo da fundamentação; **não afirmar que o aplicativo assistiu/escutou mídia não acessada**.
- Aplicar OCR a peças não textuais quando necessário, pedir versão legível de peça essencial se não for possível ler.

### 24.3. Escopo da portaria, fatos novos e novos militares — consulta prévia obrigatória

**Fatos novos não abrangidos pelo objeto original:** se a instrução revelar, por exemplo, uma irregularidade distinta da suposta agressão que motivou a portaria, o sistema pode **propor** sua inclusão na análise e solução da SR, mas **nunca incluí-la automaticamente**. Deve primeiro:

1. Identificar o novo fato e explicar sucintamente por que parece diferente do objeto inicial.
2. Mostrar **os elementos concretos dos autos** que sustentam a identificação, com referências quando disponíveis.
3. **Perguntar expressamente ao usuário** se deseja incluí-lo na seção III e na seção IV.
4. Se autorizado, analisar o fato novo e as condutas relevantes, **sem presumir dispensadas eventuais providências formais** necessárias à ampliação da investigação.
5. Se não autorizado, manter o objeto definido pelo usuário e não ampliar a investigação nem a minuta automaticamente.

**Novo militar potencialmente envolvido:** se surgir possível irregularidade de policial militar **não relacionado inicialmente como sindicado** (seja da mesma OPM, seja de **outra OPM**), apresentar sucintamente indícios e documentos de apoio e **perguntar antes** de incluí-lo no rol de sindicados para análise individualizada e solução. Uma inclusão aprovada no relatório não implica alteração automática de portaria, publicação, notificação, penalidade ou encaminhamento.

**Diferença crítica do PADS:** esta flexibilidade controlada existe **somente em SR**. Em PADS, vale a vinculação estrita ao Memorando Disciplinar (seção 22), **sem** possibilidade de incluir nova acusação na minuta por mera autorização informal.

### 24.4. Participação de civis e limite da solução

- **Seção III:** analisar participação de civis, depoimentos, documentos e eventuais contribuições para reconstrução dos fatos, quando relevantes ao objeto.
- Civis **não são sindicados militares** e não devem ser enquadrados em transgressões do RDPM.
- **Seção IV – SOLUÇÃO: mencionar como sujeitos das conclusões exclusivamente os policiais militares sindicados**. Não formular conclusão autônoma de responsabilização criminal ou disciplinar de civil, nem criar item de solução relativo a civil. A referência contextual a fatos da análise não transforma civil em sindicado.
- Não ampliar automaticamente escopo nem gerar medidas administrativas padronizadas por mera presença de civis.

### 24.5. Situação funcional de cada militar na data de cada fato

- Identificar, quando possível, se cada sindicado estava **na ativa, na reserva remunerada, reformado ou convocado para atividade na data do fato**.
- Avaliar a norma aplicável àquela **situação concreta e à legislação pertinente no período**, sem transpor automaticamente condição funcional atual a evento anterior.
- Não presumir que a mesma transgressão do RDPM é aplicável de modo indistinto a militares da ativa, reserva e reforma; conferir hipótese normativa relevante.
- Em sindicância com fatos em **datas diferentes** e mudança de situação funcional (por exemplo, fatos na convocação e após encerramento), analisar cada fato separadamente e, se cabível, produzir enquadramentos/conclusões distintos.
- Se a condição funcional for decisiva e não estiver demonstrada nos autos, perguntar ao usuário **somente pelo dado essencial**, sem instaurar uma sequência de perguntas marginais.

### 24.6. Proporcionalidade e qualidade da redação

- Não impor quantidade fixa de páginas ou parágrafos.
- Casos simples (poucos fatos, prova convergente, controvérsia pequena) admitem fundamentação objetiva.
- Casos complexos (diversos fatos/sindicados, prova técnica, versões conflitantes, questões jurídicas substanciais) requerem aprofundamento proporcional.
- **Nunca omitir elemento probatório relevante apenas para encurtar** a minuta.
- Evitar repetição de depoimentos, blocos documentais ou fundamentos; sintetizar fielmente e transcrever apenas trechos úteis.
- Manter texto jurídico claro, contínuo e compreensível, adequado aos dois modelos fornecidos, sem presumir que estilo pontual de um caso específico seja regra obrigatória universal.

## 25. IV – SOLUÇÃO: CONCLUSÕES, ENQUADRAMENTOS E EVENTUAIS PROVIDÊNCIAS

### 25.1. Finalidade, formato e densidade

A seção **IV – SOLUÇÃO** consolidará, em texto **objetivo mas suficientemente fundamentado**, as conclusões da análise probatória e jurídica da seção III. Não repetir toda a prova, não reescrever integralmente depoimentos e não antecipar sanção. Indicar os resultados sobre os **militares sindicados**, com enquadramento de cada possível infração quando houver suporte.

- **Vários sindicados, mesmo resultado:** pode haver **conclusão conjunta**, **desde que a análise da conduta de cada um tenha sido realizada individualmente** na seção III.
- **Resultados diferentes:** produzir conclusões individualizadas por sindicado.
- **Vários fatos em relação a um mesmo sindicado:** distinguir a conclusão **por fato** (por exemplo, indícios em um, ausência de indícios noutro, terceiro inconclusivo).
- **Vários sindicados, vários fatos:** organizar conclusão de forma legível por combinação fato/sindicado, sem transpor um resultado para outro.
- **Somente militares sindicados figuram como sujeitos na solução** (seção 24.4). Policial não previsto na portaria somente se o usuário autorizou sua inclusão conforme seção 24.3.

### 25.2. Critério aprovado de suficiência indiciária

A SR é investigativa e **não exige prova suficiente para condenação ou imposição disciplinar**, mas não admite apontar infração com base em conjectura destituída de suporte. Distinguir:

1. **Existência de indícios suficientes:** elementos concretos dos autos sustentam razoavelmente ocorrência de possível crime comum, crime militar e/ou transgressão disciplinar e, quando couber, vínculo do sindicado com a conduta; fundamentar qual fato e qual suporte.
2. **Ausência de indícios suficientes:** os elementos examinados não fornecem suporte razoável para imputar irregularidade na investigação, sem afirmar necessariamente que o fato é impossível ou comprovadamente inexistente.
3. **Conclusão inconclusiva:** faltam elementos essenciais, ou existem limitações graves, de modo que **não é possível formar conclusão suficientemente segura nem em nível indiciário**; explicitar as lacunas e o que permanece sem esclarecimento. O usuário pode autorizar a emissão da minuta ainda assim, desde que a conclusão seja qualificada e fundamentada, sem mascarar insuficiência.

**Distinção entre ocorrência e autoria:** é admissível reconhecer **indícios da ocorrência da irregularidade sem indícios suficientes para individualizar o responsável**. Não atribuir autoria por mera presença, função ou vínculo circunstancial; explicar separadamente os elementos da ocorrência/materialidade e os da autoria na seção III e, quando relevante, na solução.

### 25.3. Enquadramento jurídico — específico, contextual e sujeito a conferência

- **Crime comum:** indicar possível delito, **artigo e diploma legal** correspondente, quando houver indícios e suporte para seus requisitos.
- **Crime militar:** indicar possível delito, **artigo e diploma legal** correspondente, quando houver indícios e suporte para seus requisitos.
- **Transgressão disciplinar:** indicar **artigo/inciso do RDPM/RO** ou outra norma disciplinar pertinente, vinculados à conduta e à condição funcional da época.
- **Pluralidade:** permitir dois ou mais possíveis crimes/transgressões do mesmo sindicado, fundamentando cada enquadramento; não somar itens irrelevantes.
- **Dúvida sobre artigo/norma:** quando há fato sugestivo de irregularidade, mas não é seguro escolher entre normas, apresentar alternativas fundamentadas ao usuário e pedir confirmação **antes de concluir**; não inventar artigo, literal normativo ou pena.
- **Subsunção:** primeiro examinar fatos/provas, depois confrontar requisitos de cada possível infração. Não eleger artigo apenas por semelhança temática.
- **Data e condição funcional:** usar legislação aplicável ao período de cada fato e situação funcional do sindicado; não aplicar automaticamente status presente ao passado.
- **Portaria com classificação preliminar:** o relatório não está preso à capitulação jurídica originalmente aventada na portaria. Pode afastá-la ou propor outro enquadramento para **os mesmos fatos investigados**, desde que exista suporte concreto, sem incluir fato independente fora do escopo sem a autorização prévia da seção 24.3.
- **Estilo:** tratar enquadramentos como **possíveis infrações indiciadas**, sem declaratória de condenação ou aplicação de sanção.

### 25.4. Providências: excepcionalidade, não automatismo

**Decisão final do usuário:** o relatório deve **se limitar, por padrão, a indicar fundamentadamente os indícios ou sua ausência**. Sugestões de providências **somente quando existir necessidade específica identificada nos autos**, com justificativa concreta e redação compatível com o papel do encarregado.

- **Não recomendar automaticamente** instauração de PADS, arquivamento, encaminhamento criminal, comunicação à Corregedoria-Geral ou qualquer outro ato como consequência padronizada de toda conclusão.
- Se uma providência específica for efetivamente relevante, **pode sugeri-la**, sem tratá-la como obrigação automática e sem transformar a minuta em homologação ou avocação.
- Sugestões são do encarregado, **não determinações** nem atos executados pelo aplicativo.
- A presença de conclusão pela ausência de indícios **não implica anexar obrigatoriamente** fórmula «sugiro arquivamento», embora alguns modelos históricos a contenham; seguir regra posterior e mais específica.
- O mesmo vale para indícios de crime/transgressão: **não anexar por padrão** «instaurar PADS» ou «encaminhar à autoridade», salvo necessidade concreta demonstrada e pertinente.

### 25.5. Exemplos fictícios de redação — não substituem documentos reais

**S1 — ausência de indícios, dois sindicados, conclusão conjunta admitida:**

> Diante da análise dos elementos constantes dos autos, conclui-se que não foram identificados indícios suficientes de crime comum, crime militar ou transgressão disciplinar atribuíveis ao 3º SGT PM [MILITAR A] e ao CB PM [MILITAR B], quanto ao fato investigado. As condutas de ambos foram examinadas individualmente na análise das provas.

**S2 — resultados distintos, mesma investigação:**

> a) CB PM [MILITAR A]: quanto ao fato [FATO 1], identificam-se indícios de transgressão disciplinar prevista no art. [DISPOSITIVO CONFIRMADO] do [DIPLOMA CONFIRMADO], em razão de [FUNDAMENTO PROBATÓRIO SUCINTO]. Não foram identificados indícios de crime comum ou militar relacionados a esse fato.
>
> b) 3º SGT PM [MILITAR B]: quanto ao mesmo fato, os elementos examinados não fornecem indícios suficientes para atribuir-lhe crime ou transgressão disciplinar, conforme fundamentado na análise das provas.

**S3 — mesmo militar, fatos diferentes:**

> Quanto ao fato 1, há indícios de [INFRAÇÃO A, ARTIGO E LEI CONFIRMADOS]. Quanto ao fato 2, não foram identificados indícios suficientes de irregularidade. Quanto ao fato 3, as provas são inconclusivas, sobretudo pela ausência de [ELEMENTO ESSENCIAL], não sendo possível conclusão segura até mesmo em nível indiciário.

**S4 — ocorrência indiciada, autoria não definida:**

> Os elementos analisados permitem apontar indícios da ocorrência de [IRREGULARIDADE], mas não fornecem suporte suficiente para atribuir sua autoria individualmente ao(s) sindicado(s) [IDENTIFICAÇÕES], conforme as limitações probatórias examinadas.

**S5 — nova irregularidade não descrita na portaria:** se o sistema detectar novo fato, **não gerar de imediato** parágrafo da solução. Primeiro apresentar resumo da descoberta e pedir autorização; **somente se autorizado** incluir análise e conclusão fundamentadas, respeitando eventual necessidade de atos formais.

Todos os marcadores acima são **ilustrações para testes** e não autorizações para usar nomes/artigos fictícios em relatório real.

## 26. REGRAS TRANSVERSAIS DE INTERAÇÃO, REVISÃO E TRATAMENTO DE INSUFICIÊNCIAS

### 26.1. Lacunas relevantes e sugestões de diligências

- Identificar documentos, testemunhos, perícias ou outros elementos **relevantes ausentes**, incompletos, ilegíveis ou inacessíveis.
- Avaliar se o restante do acervo permite conclusão indiciária fundamentada; não sugerir diligências por formalismo, se os dados são suficientes.
- Quando a lacuna impedir uma conclusão suficientemente segura, **alertar o usuário e sugerir diligências complementares concretas**, como obter uma peça mencionada ou realizar uma oitiva, com explicação do que a diligência poderia esclarecer.
- **O usuário decide** se prossegue sem a diligência. Se optar por prosseguir, permitir minuta com resultado compatível com o conjunto probatório, incluindo **conclusão inconclusiva** quando necessária (seção 25.2).
- **Não confundir ausência de indícios com prova positiva de inexistência da irregularidade**; evitar redação categórica incompatível com lacunas conhecidas.
- O sistema não realiza autonomamente diligências externas, atos de investigação, intimações ou alterações de processo no SEI.

### 26.2. Política de perguntas: apenas o essencial

**Diretriz expressa de eficiência aprovada ao encerrar o levantamento funcional de SR:** não prolongar análise com perguntas repetitivas ou sobre casos periféricos cujo tratamento já decorra de princípios aprovados. O usuário fará revisão final.

Perguntar principalmente quando:
1. É necessário decidir se será incluído **fato novo fora do objeto inicial da SR**.
2. É necessário decidir se será incluído **novo militar não relacionado inicialmente** como sindicado, inclusive de outra OPM.
3. Falta informação **essencial e material** para identificar envolvido, data, condição funcional, enquadramento, conteúdo de peça relevante ou suficiência da conclusão.
4. Há dúvida jurídica substancial entre alternativas de enquadramento ou divergência factual relevante que não foi resolvida pelos autos.
5. Há prova essencial ausente ou lacuna impeditiva de conclusão, para o usuário optar por complementar ou prosseguir.

**Não** perguntar detalhes de estilo repetidamente, não reabrir definições aprovadas por mera hipótese remota, não exigir confirmação para dados claramente demonstrados, não impor checklists suplementares de prescrição/nulidade fora do escopo acordado.

### 26.3. Revisão humana obrigatoriamente sinalizada — PADS e SR

**Decisão expressa posterior e transversal:** ao final da geração de **qualquer minuta de julgamento de PADS ou Relatório de SR**, sempre informar ao usuário que deve **revisar integralmente** o documento e confirmar sua adequação. A **aprovação e decisão final pertencem ao usuário competente**, não ao aplicativo.

**Texto de aviso proposto, sujeito apenas a ajustes de interface/redação sem alterar o sentido:**

> **Atenção — revisão obrigatória:** esta minuta é um documento de apoio. Confira integralmente os fatos, as provas, a fundamentação, os enquadramentos jurídicos, os cálculos quando aplicáveis e as conclusões antes de utilizá-la. A revisão, aprovação e decisão final cabem ao usuário/autoridade competente.

A apresentação deste aviso:
- ocorre **ao concluir a minuta e disponibilizar DOCX/PDF para revisão**;
- **não bloqueia** download/visualização de versões para conferência;
- **não equivale a autorização de finalizar** o trabalho;
- não altera a regra de manter o status **«aguardando revisão»** e possibilitar ajustes iterativos até comando explícito de finalização, como «pode finalizar»;
- não autoriza assinatura, publicação, punição, arquivamento administrativo ou exclusão de documentos do SEI.

## 27. MATRIZ DE TESTES DE ACEITAÇÃO DA SINDICÂNCIA REGULAR E CONTROLE DE CONTINUIDADE

Estes cenários complementam as seções 13 e 19. São exemplos funcionais fictícios, não declarações jurídicas sobre um caso real.

| ID | Cenário | Comportamento esperado |
|---|---|---|
| SR01 | PDF com portaria e três sindicados seguros | Seção I lista portaria e todos os sindicados identificados, sem nomes inventados |
| SR02 | Nome do ofendido não identificado com segurança | Manter apenas `c. Ofendido:` e alertar usuário para conferir/preencher |
| SR03 | Nenhuma perícia nos autos | Manter apenas `e. Perícias realizadas:`; não inventar laudo |
| SR04 | Testemunha em Termo de Inquirição com SEI comprovado | Listar em `d. Testemunhas ouvidas:` com SEI |
| SR05 | Pessoa ouvida em Termo de Declarações, não como testemunha | Considerar para `f. Outras informações:` se relevante; não classificar automaticamente como testemunha |
| SR06 | Pessoa aparece como ofendida em um fato e declarante em outro | Respeitar condição em cada ato, sem duplicação indevida e sem categoria presumida |
| SR07 | Alínea `f` com numerosos ofícios e BOs comuns | Não produzir inventário; selecionar apenas itens especialmente relevantes |
| SR08 | Portaria noticia fato inicial, instrução traz detalhes | Seção II sintetiza a notícia originária sem antecipar resultado; seção III analisa as provas |
| SR09 | Dois fatos com sindicados diferentes | Seção III prioriza cada fato e individualiza condutas em seu interior |
| SR10 | Três sindicados, resultados idênticos, análises individuais feitas | Seção IV pode redigir **conclusão conjunta** |
| SR11 | Três sindicados com resultados divergentes | Seção IV apresenta conclusões individuais |
| SR12 | Mesmo sindicado: fato A com indícios e fato B sem indícios | Solução distingue resultado por **fato** e **pessoa** |
| SR13 | Provas irreconciliáveis e essenciais ausentes | Advertir, sugerir diligências se necessário, permitir resultado inconclusivo fundamentado se usuário prosseguir |
| SR14 | Contradição secundária sem relação com fato | Não converter automaticamente em indício de irregularidade |
| SR15 | Laudo sem lesão, testemunha relata agressão | Não concluir automaticamente inexistência do fato pelo laudo negativo; avaliar conjunto e limites técnicos |
| SR16 | Documentos de outro IP incorporados à SR | Avaliar pertinência; distinguir conclusão/indiciamento de prova demonstrativa |
| SR17 | Vídeo referido mas ausente do PDF | Pedir descrição/transcrição; não baixar nem transcrever vídeo automaticamente; não inferir conteúdo |
| SR18 | Prova da ocorrência de irregularidade sem identificação de autor | Distinguir indícios do fato e insuficiência de elementos sobre autoria |
| SR19 | Fato novo diferente do da portaria | Apresentar resumo e fontes; perguntar **antes** de incluir nas seções III/IV |
| SR20 | Novo possível sindicado de outra OPM | Apresentar elementos; perguntar **antes** de incluir; sem alterar portaria automaticamente |
| SR21 | Civil é personagem relevante no fato | Examinar atuação na seção III; **não** apresentar civil como sindicado na seção IV |
| SR22 | Dois militares com mesmo comportamento, um da ativa, outro reserva | Conferir condição funcional na data e aplicabilidade dos dispositivos individualmente |
| SR23 | Militar convocado em fato A, não convocado em fato B | Enquadrar por condição funcional de cada data, com resultados potencialmente distintos |
| SR24 | Portaria sugere enquadramento X, fatos do mesmo objeto sustentam Y | Avaliar e fundamentar Y sem apego automático a X; não adicionar fato externo sem consulta |
| SR25 | Elementos apontam possível crime e transgressão do mesmo militar | Examinar separadamente e citar dispositivos/diplomas específicos quando confirmados |
| SR26 | Artigo sugerido mas requisitos normativos não comprovados | Não apontar indício só por semelhança; solicitar confirmação se ambiguidade substancial |
| SR27 | Indícios suficientes de transgressão | Solução aponta indícios e tipificação, **sem inserir por padrão instauração de PADS** |
| SR28 | Ausência de indícios | Solução indica ausência fundamentadamente, **sem sugerir arquivamento automaticamente** |
| SR29 | Necessidade específica de comunicação comprovada nos autos | Pode propor comunicação sucinta e fundamentada, sem torná-la regra fixa |
| SR30 | Relatório elaborado em caso simples | Texto objetivo, sem omissões relevantes ou extensão mínima artificial |
| SR31 | Relatório elaborado em caso complexo | Fundamentação mais profunda, proporcional aos fatos, pessoas e prova |
| SR32 | Fala de sindicado omite detalhe | Não presumir ocultação/culpa automaticamente; examinar contexto |
| SR33 | Usuário quer concluir sem diligência essencial sugerida | Permitir prosseguimento e qualificar limitações/conclusão inconclusiva quando necessário |
| SR34 | DOCX e PDF gerados para revisão | Exibir aviso de conferência obrigatória; manter **aguardando revisão** |
| SR35 | Usuário não autoriza inclusão de fato novo ou novo militar | Não incluir automaticamente na análise/solução; não inventar providência |
| SR36 | Dois sindicados com conclusão conjunta, mas provas individualmente diferentes | Conclusão conjunta admissível **só se** cada conduta foi examinada e resultado efetivamente igual |

### 27.1. Modelos fornecidos pelo usuário e limites de utilização

**Modelo A — relatório com indícios:** documento de seis páginas, identificado no arquivo enviado como `Relatorio de Sindicância Regular 2 - com indicação de crime ou transgressão(2).pdf`. Características observadas: múltiplos sindicados, investigação de fatos distintos, uso de depoimentos e provas provenientes de outros procedimentos juntados, análise individualizada da situação funcional e desfechos distintos; solução aponta indícios para determinados sindicados e ausência para outro, com encaminhamentos pontuais. **Usar como exemplo de forma e raciocínio, não como precedente obrigatório ou como fonte de fatos para outra SR.**

**Modelo B — relatório sem indícios:** documento de três páginas, `Relatorio de Sindicância Regular 1 - sem indicação de crime ou transgressão(1).pdf`. Características observadas: dois sindicados, análise de narrativas/depoimentos/BOs e laudos, conclusão conjunta de ausência de indícios e sugestão de arquivamento. A numeração das seções naquele documento difere da **estrutura aprovada de quatro seções**. A ausência de lesão em exame pericial **não deve ser convertida, como regra universal, em prova de inexistência de agressão**. A sugestão de arquivamento presente no modelo **não é obrigatória** em relatórios novos, por decisão posterior do usuário.

**O modelo não supre os autos:** nenhum fato, nome, artigo ou evidência dos modelos pode ser transportado para outro caso. Divergências, campos incompletos ou raciocínios específicos de exemplos históricos não obrigam a repetição de conclusões.

### 27.2. Registro cronológico de decisões posteriores à V2

1. **Natureza do trabalho:** PADS é *Processo Apuratório Disciplinar Sumário*; SR é procedimento inquisitorial; produzir julgamento de PADS e somente relatório de SR, nunca homologação/avocação.
2. **Estrutura SR:** quatro seções aprovadas com seis alíneas fixas na seção I; preencher alíneas apenas com dados seguros, senão manter rótulo vazio e avisar para conferência.
3. **Vários sindicados:** condutas avaliadas individualmente; conclusão conjunta admissível quando os resultados forem iguais e a análise individual tiver sido feita.
4. **Organização da seção III:** prioridade aos **fatos investigados**, avaliando em cada fato as condutas individuais e provas; contradições ponderadas, sem presunção automática de irregularidade.
5. **Suficiência:** distinguir indícios suficientes, ausência de suporte e inconclusividade quando lacunas impedirem resultado seguro mesmo em nível indiciário.
6. **Tipificação:** indicar diploma/artigo para crime comum, militar ou transgressão; múltiplas infrações por sindicado possíveis quando fundamentadas; conferir legislação aplicável à data e situação funcional.
7. **Providências:** somente quando houver necessidade específica nos autos, sem instauração/arquivamento/encaminhamento automáticos.
8. **Síntese:** usar portaria e notícia inicial; não antecipar conclusão. **Solução:** objetiva, sem repetir a fundamentação aprofundada.
9. **Seção I:** distinção por função documental entre testemunha, declarante, sindicado, ofendido; `f` seletiva; perícias listadas em `e` e examinadas em III.
10. **Escopo SR:** fatos novos e militares não relacionados inicialmente, inclusive de outras OPMs, **somente após consulta e autorização do usuário**, com resumo probatório.
11. **Mídia, laudos e lacunas:** não presumir conteúdo ausente; laudo negativo não exclui sozinho o fato; pedir complemento e sugerir diligências relevantes; permitir prosseguir com limitação explícita.
12. **Autoria:** separar ocorrência/materialidade de autoria, não atribuir responsabilidade por mera presença.
13. **Civis:** podem ser analisados em III; seção IV refere-se **somente aos militares sindicados**.
14. **Condição funcional:** analisar ativa/reserva/reforma/convocação **na época de cada fato**, inclusive quando o mesmo militar mudou de condição entre episódios.
15. **Provas de outros procedimentos:** aproveitar se incorporadas aos autos, sem importar conclusões automaticamente.
16. **Depoimentos:** sínteses fiéis, trechos pontuais, confronto contextual, sem atribuir ocultação automaticamente por omissão.
17. **Detalhamento:** proporcional à complexidade; sem limite fixo, evitando perguntas marginais e repetições.
18. **Regra crítica de PADS:** imputações do Memorando Disciplinar são **limite intransponível do julgamento**; infração diversa descoberta não vira acusação nova. Julgar procedente/parcial/improcedente exclusivamente o memorando.
19. **Revisão humana:** sempre informar ao usuário, ao fim da elaboração de minutas de PADS e SR, que deve revisar integralmente; decisão final é humana.
20. **Próxima etapa:** as regras funcionais de SR estão **encerradas por decisão expressa do usuário**. Não continuar perguntas periféricas; quando solicitado, abordar pendências transversais e, só mais tarde, a IA. **Não gerar prompt de implementação sem autorização expressa.**

### 27.3. Pendências que permanecem fora desta conclusão funcional

> **[Atualização V4]** Situação atual de cada item: **IA** → definida nas seções 36–42; modelos, embeddings e banco vetorial exatos ficam para testes (seção 43). **Formatação** → seção 32. **Persistência e segurança** → seções 29, 33 e 34. **Interface/revisão** → seção 31. **Validação normativa** → seção 44 (mantidas as interpretações aprovadas). **Casos-limite raros** → continuam pendentes, perguntar só quando ocorrerem e forem materiais.

Os itens ainda pendentes no núcleo anterior continuam válidos, exceto a afirmação de que a SR ainda não teria regras definidas:

- **Configuração técnica de IA:** provedor, modelo, autenticação, custos, embeddings, política de transmissão/armazenamento e arquitetura — **adiada** até etapa posterior, por ordem do usuário.
- **Formatação final DOCX/PDF:** papel timbrado, margens, tipografia, assinatura reservada, campos de versão, nomes de arquivos e detalhes da apresentação.
- **Persistência e segurança operacional:** retomada de sessões, armazenamento temporário, criptografia local, logs, empacotamento e atualização do desktop.
- **Detalhes finos de interface/revisão:** como corrigir parcialmente fatos, provas, cálculos, propostas e anexos sem perder o trabalho.
- **Validação documental de normas específicas já destacadas na V2:** conferir texto oficial vigente quando necessário, sem reabrir escolhas aprovadas sem motivo relevante; o documento registra critérios funcionais, não substitui fonte normativa primária.
- **Casos-limite raros** ainda não decididos: só perguntar quando ocorrerem de fato e forem materiais, sem reabrir hipoteticamente a especificação encerrada.

### 27.4. Orientação de continuidade para um novo chat ou agente

**Fonte da verdade:** a íntegra desta V3, incluídas seções 1–27, deve ser lida em conjunto. **[Atualização V4]** A fonte da verdade passa a ser a íntegra desta **V4, seções 1–46**. **Não descartar seções antigas**, exceto nos pontos expressamente corrigidos por decisões posteriores. As escolhas de PADS (reincidência de mesma natureza, regressão, conexidade, dosimetria, SEI e evidências) continuam intactas e são complementadas pela vinculação acusatória da seção 22.

**Não reabrir regras de SR:** sua definição funcional foi encerrada. **Não transformar relatório de SR em decisão de PADS, nem decisão de PADS em investigação aberta.** Manter perguntas apenas para lacunas essenciais do caso concreto ou decisões técnicas relevantes ainda pendentes.

**Saída da aplicação:** gerar minuta editável DOCX e PDF para revisão, solicitar conferência humana e aceitar ajustes até autorização de finalização. Não gerar documentos de homologação/avocação; não assinar, publicar, encerrar, excluir ou punir no SEI.

**Vedação explícita:** NÃO gerar o prompt definitivo de planejamento/implementação até que o usuário autorize expressamente sua elaboração.

---

## ESTADO FINAL — V3 (08/11/2026)

> **[Atualização V4]** Registro histórico. O estado vigente está em “ESTADO FINAL — V4”, ao final do documento.

**CONCLUÍDO O LEVANTAMENTO FUNCIONAL DO RELATÓRIO DE SINDICÂNCIA REGULAR; PRESERVADAS AS REGRAS APROVADAS DE JULGAMENTO DE PADS, COM EXPLICITAÇÃO DE QUE O JULGAMENTO É LIMITADO ÀS IMPUTAÇÕES DO MEMORANDO DISCIPLINAR. A APLICAÇÃO DEVERÁ SEMPRE SOLICITAR REVISÃO HUMANA FINAL DE AMBOS OS TIPOS DE MINUTA. PERMANECEM PENDÊNCIAS TRANSVERSAIS E A INTEGRAÇÃO DE IA, ADIADA. ESTE MARKDOWN É A ESPECIFICAÇÃO CONSOLIDADA EM CONSTRUÇÃO, NÃO O PROMPT DEFINITIVO DE IMPLEMENTAÇÃO.**

---

# CONSOLIDAÇÃO V4 — DECISÕES TRANSVERSAIS E DE INTELIGÊNCIA ARTIFICIAL

> **Estado:** seções aprovadas pelo usuário na continuação de 08/11/2026, inclusive as confirmações finais sobre formatação (sugestão segundo a redação oficial), compatibilidade técnica e validação normativa. Integram a especificação e **resolvem as pendências transversais e de IA** antes listadas nas seções 12.2 e 27.3. **Não são o prompt de implementação.**
>
> **Critério de precedência:** as regras funcionais e jurídicas de PADS e SR das seções 1–27 permanecem intactas. Nos assuntos transversais e de IA, prevalecem as seções 28–46. Neste bloco, referências à “V3” designam as seções 1–27 deste documento.
>
> **Legenda:** **[APROVADO]** decisão explícita do usuário; **[PROPOSTA TÉCNICA]** detalhamento recomendado que não deve ser confundido com imposição normativa ou escolha técnica já validada; **[VALIDAR EM TESTES]** solução/valor provisório dependente de prova de desempenho ou compatibilidade.

## 28. Princípios de continuidade e alcance

1. **[APROVADO]** Permanecem encerradas as definições funcionais de julgamento de PADS e de elaboração de Relatório de Sindicância Regular constantes da V3; este documento acrescenta definições transversais, sem reabrir interpretações já aprovadas.
2. **[APROVADO]** Nenhuma funcionalidade nova poderá ampliar as imputações do Memorando Disciplinar no PADS. Na SR, fatos novos observam a autorização humana prévia estabelecida na V3.
3. **[APROVADO]** O sistema é assistente de análise e redação de **minutas**. Não decide, assina, publica, homologa, altera ficha disciplinar, impõe sanção ou executa atos administrativos por conta própria.
4. **[APROVADO]** As decisões posteriores da própria V3 que corrigiram regras anteriores continuam prevalecendo. A futura fusão deve ser aditiva, sem truncar seções, exemplos ou casos-limite existentes.
5. **[APROVADO]** Não gerar o prompt definitivo de implementação sem autorização expressa do usuário.

## 29. Sessões de trabalho efêmeras e exclusão de dados

### 29.1. Ciclo de vida de cada tarefa — [APROVADO]

- Cada execução de PADS ou SR é uma tarefa temporária e independente.
- **Não haverá armazenamento permanente de julgamentos, relatórios, histórico de tarefas nem retomada de uma tarefa interrompida.**
- Se o usuário fechar o aplicativo, ocorrer falha abrupta ou houver perda irrecuperável do contexto daquela interação, a análise deverá ser iniciada novamente; não fornecer mecanismo de restauração de trabalho.
- Enquanto a tarefa estiver ativa, o usuário poderá revisar, pedir ajustes, regenerar e baixar diferentes versões da minuta sem encerrar a sessão.
- Após o comando inequívoco do usuário para **aprovar/finalizar**, descartar os autos temporários, análises, índices temporários, minutas e demais artefatos de execução controlados pelo aplicativo. O usuário poderá manter normalmente os DOCX/PDF que tenha baixado ou salvo intencionalmente fora do armazenamento temporário.
- Documentos da **biblioteca jurídica permanente** e arquivos dos modelos instalados **não** são apagados ao concluir uma tarefa.

### 29.2. Arquivos temporários e falhas — [APROVADO]

- Usar o mínimo necessário de arquivos temporários durante a execução, com acesso restrito à conta do usuário do Windows.
- Empregar proteção adicional quando tecnicamente apropriada; evitar conteúdo sensível desnecessário em arquivos auxiliares.
- Limpar os temporários no encerramento normal da tarefa/aplicativo.
- Na inicialização seguinte, identificar e **tentar remover resíduos deixados por encerramento abrupto**, sem transformar tal varredura em mecanismo de recuperação da tarefa.
- Não prometer apagamento forense garantido de dados já copiados em caches do sistema, backups ou locais externos ao controle do aplicativo.

### 29.3. Distinção obrigatória entre armazenamento permanente e transitório

> **[Emenda §48]** Acrescenta-se a linha "Credencial do ChatGPT" (seção 48.3). O cadastro de peças é trabalho transitório, somente na sessão (seção 48.4).

> **[Emenda §47]** A linha "Modelos locais" deixa de existir e é acrescentada a linha "Chave da Gemini API" (seção 47.4).

| Armazenamento | Conteúdo | Persistência |
|---|---|---|
| Biblioteca jurídica | Leis, regulamentos, normas, jurisprudência e respectivos índices | Permanente e gerenciável |
| Modelos locais | GGUF de geração, modelo de embeddings e componentes instalados | Permanente e gerenciável |
| Preferências | Configurações como habilitar/desabilitar pesquisa web | Permanente |
| Trabalho de PADS/SR | PDF dos autos, extrações, OCR, índices de autos, informações intermediárias, minutas | Apenas na sessão; limpeza ao final/falha |
| Credenciais SEI | Senha e dados secretos necessários à autenticação | Somente memória da sessão; não persistir |

## 30. Fluxo inicial da aplicação e modalidades de entrada — [APROVADO]

### 30.1. Ordem obrigatória das escolhas

> **[Emenda §48]** O "upload manual do PDF completo/consolidado" passa a ser o **cadastro de peças** (upload de um ou vários arquivos por peça), e a modalidade SEI passa a obter só os documentos selecionados (seção 48.4).

1. **Primeiro**, perguntar **qual minuta o usuário deseja elaborar**:
   - **Julgamento de PADS**; ou
   - **Relatório de Sindicância Regular**.
2. **Somente depois**, perguntar como serão obtidos os autos:
   - **Acesso ao SEI/RO**, com autenticação do usuário e geração/download do PDF consolidado; ou
   - **Upload manual do PDF completo/consolidado** previamente obtido no SEI/RO.
3. Processar os autos por **um pipeline documental compartilhado**; a escolha inicial do procedimento determina as **regras jurídicas e funcionais, validações, perguntas, estrutura e cabeçalho** aplicáveis.
4. A modalidade de upload **não deve solicitar credenciais do SEI**, abrir o SEI ou gerar/baixar novamente o PDF.
5. As duas modalidades resultam nas mesmas capacidades de extração, OCR, classificação, rastreabilidade, análise, geração, revisão e finalização da minuta.

### 30.2. Checagem de compatibilidade entre procedimento e autos

> **[Emenda §48]** A peça central é a declarada pelo usuário no cadastro, e seu **conteúdo é conferido**. Se indicar outra modalidade ou outra peça, o aplicativo pergunta; a troca nunca é silenciosa (seção 48.4).

**[APROVADO]** Após receber o PDF por qualquer modalidade, localizar e identificar a peça instauradora/delimitadora principal a partir de **conteúdo e função**, nunca somente pelo nome do arquivo:

| Procedimento escolhido | Peça central de identificação | Significado |
|---|---|---|
| PADS | **Memorando Disciplinar** | Descreve fatos e transgressão(ões) imputadas; delimita estritamente o que poderá ser julgado |
| Sindicância Regular | **Portaria de instauração** | Descreve o objeto e os fatos inicialmente submetidos à investigação |

- Considerar a organização e o contexto dos autos. **Uma Portaria de SR anexada como prova ou antecedente dentro de um PADS não transforma automaticamente o PADS em SR.**
- Se houver incompatibilidade real entre o procedimento selecionado e o processo apresentado, exibir aviso compreensível e solicitar correção ou esclarecimento.
- Se a peça central estiver ilegível, ausente ou classificada com incerteza, informar a limitação e perguntar quando necessário, em vez de presumir procedimento diferente.
- **Nunca trocar silenciosamente** de PADS para SR ou vice-versa.
- A validação de compatibilidade não autoriza reinterpretar ou ampliar imputações do PADS; manter todas as regras próprias de SR para fatos surgidos durante a apuração.

### 30.3. Sem internet ao escolher SEI — [APROVADO]

- Caso o usuário escolha SEI e não exista conexão suficiente, informar amigavelmente que a obtenção automatizada exige internet.
- Oferecer **upload do PDF completo** como alternativa para continuar a mesma configuração inicial, sem obrigar o usuário a reiniciar o aplicativo.

## 31. Interface, execução automática e revisão de minutas — [APROVADO]

### 31.1. Filosofia da interface

- Interface Windows responsiva, bem organizada, clara e visualmente agradável.
- **Não construir um chatbot generalista como tela principal.** O aplicativo é uma ferramenta de execução de tarefas.
- Depois da escolha do documento e da entrada dos autos, processar automaticamente a tarefa em segundo plano **dentro da sessão ativa do aplicativo**, mostrando ao usuário **status/progresso claros**.
- Exemplos de estados: obtendo autos, extraindo texto/OCR, classificando peças, analisando provas, consultando fontes jurídicas, elaborando minuta, validando minuta, aguardando esclarecimento e aguardando revisão.
- **Não mostrar o texto da IA sendo gerado token por token** nem exigir streaming de texto para a interface. Streaming interno é opção técnica, não funcionalidade de UI requerida.
- O progresso numérico só deve ser exibido quando houver métrica real; em etapas indeterminadas, usar indicação de atividade sem percentuais fictícios.

### 31.2. Perguntas pontuais durante a análise

- Se a V3 exigir confirmação ou dado faltante (por exemplo, pontuação, histórico disciplinar, dúvida de data, informação sobre gravação), pausar o ponto necessário e apresentar **pergunta objetiva e contextualizada**, aproveitando os dados já extraídos.
- Não insistir em perguntas desnecessárias; não inventar respostas ou transformar o sistema em conversa livre.
- Após resposta, prosseguir com a mesma tarefa ativa.

### 31.3. Visualização e revisão

- Ao concluir a elaboração **e passar pela validação obrigatória**, apresentar **a minuta completa** ao usuário, preferencialmente em visualização integrada fiel ao PDF final.
- Não é necessário um editor de texto visual para editar manualmente o documento.
- Disponibilizar **DOCX editável e PDF** para download/conferência.
- Disponibilizar duas decisões visíveis: **aprovar e finalizar** ou **solicitar ajustes**.
- Para ajustes, oferecer área simples de texto/chat breve, onde o usuário indica correções pontuais, complementações ou regeneração. A IA aplica as mudanças na mesma sessão, preservando as partes válidas e as decisões já confirmadas, valida a nova versão e retorna a minuta inteira para revisão.
- O usuário poderá repetir o ciclo até autorização inequívoca de finalização.
- O texto enviado para pedir ajustes **não** cria um histórico permanente de chatbot.

### 31.4. Exemplo de fluxo de aceitação

```text
Iniciar → Escolher [Julgamento de PADS | Relatório de SR]
        → Escolher [Acessar SEI | Enviar PDF]
        → Validar compatibilidade → Processar autos → Consultar fontes
        → Solicitar esclarecimentos pontuais quando necessários
        → Gerar minuta → Validar minuta
        → Visualizar minuta completa e disponibilizar DOCX/PDF
        → [Aprovar e finalizar] OU [Descrever ajustes → Regenerar → Revalidar → Revisar]
        → Finalização autorizada → Limpar dados temporários
```

## 32. Formatação institucional dos documentos — [APROVADO + PROPOSTA TÉCNICA]

### 32.1. Requisitos aprovados

- DOCX e PDF devem seguir um **padrão institucional único** para PADS e Sindicância Regular, respeitando as estruturas e títulos específicos.
- **Fonte Times New Roman, corpo 12**.
- Utilizar o brasão da Polícia Militar do Estado de Rondônia fornecido no projeto sob o nome exato **`logo-pmro.png`**.
- Cabeçalho de ambas as minutas deverá seguir as **duas imagens de referência fornecidas pelo usuário na conversa de 08/11/2026**:
  - brasão centralizado;
  - abaixo do brasão, em letras maiúsculas, negrito e centralizado:
    - `SECRETARIA DE SEGURANÇA, DEFESA E CIDADANIA`
    - `POLÍCIA MILITAR DO ESTADO DE RONDÔNIA`
  - título, centralizado, em maiúsculas e negrito, com espaçamento visual em relação ao cabeçalho:
    - **`JULGAMENTO DE PADS`**; ou
    - **`RELATÓRIO DE SINDICÂNCIA REGULAR`**.
- Os títulos específicos **não** se confundem: selecionar de acordo com a escolha inicial de procedimento.
- Nenhuma assinatura, publicação, homologação, validação SEI ou ato oficial deve ser ficticiamente atribuído a uma minuta.

### 32.2. Proposta de parâmetros de redação oficial (não afirmar ser norma específica da PMRO)

O usuário autorizou **sugestão técnica seguindo a redação oficial**. Recomenda-se utilizar como padrão inicial, sujeito a adequação por modelos institucionais reais fornecidos:

| Elemento | Sugestão de formatação |
|---|---|
| Papel | A4, orientação retrato |
| Margens | Superior e inferior de 2 cm; esquerda de 3 cm; direita de 1,5 cm, **como proposta**, a conferir com modelo institucional aplicável |
| Corpo do texto | Times New Roman 12, cor preta, alinhamento justificado |
| Entrelinhas | Simples (1,0), ou o espaçamento exigido por modelo institucional disponível |
| Parágrafos | Recuo de primeira linha de aproximadamente 2,5 cm se compatível com o documento institucional; sem recuos arbitrários em itens numerados |
| Espaçamento entre parágrafos | Uniforme e discreto; evitar quebras e espaços vazios excessivos |
| Seções | Títulos próprios de PADS/SR em destaque, consistentes e sem alterar sua ordem jurídica |
| Paginação | Numeração automática e consistente, evitando numeração manual no conteúdo |
| Assinatura | Espaço para assinatura/identificação da autoridade responsável, sem inserir assinatura digital ou simular assinatura do SEI |
| Exportação | PDF com aparência equivalente ao DOCX, prevenindo mudança de paginação, fonte ou cabeçalho |
| Nomes de arquivo | Identificar tipo de minuta e número do processo quando disponível, com nome seguro para Windows; não incorporar dados pessoais desnecessários |

**Atenção:** esses números de margens/recuos são **parâmetros sugeridos**, e não declaração de um padrão normativo obrigatório da PMRO. Em caso de incompatibilidade com modelos efetivos fornecidos ou orientação institucional aplicável, priorizar o modelo confirmado e pedir decisão humana apenas em divergência substancial. Preservar as estruturas de conteúdo definidas na V3.

## 33. Credenciais, dados pessoais, segurança e erros — [APROVADO]

- Solicitar as credenciais de SEI somente na modalidade de acesso autenticado, **a cada nova tarefa**.
- Manter a senha protegida **somente temporariamente em memória** durante a sessão, pelo tempo indispensável. Não criar banco de dados, arquivo de configuração ou arquivo criptografado para armazenar senha.
- Descartar credenciais ao finalizar tarefa ou fechar aplicativo; observar que limpeza absoluta de memória gerenciada não pode ser garantida por mera atribuição de valor vazio, devendo o agente adotar proteção tecnicamente adequada.
- Respeitar OTP/2FA e reautenticações exigidos pelo SEI, conforme V3.
- Os dados temporários dos processos devem permanecer acessíveis apenas à conta do Windows que executa a aplicação, com permissões e proteção apropriadas.
- Tratamento de falhas **abrangente**, com mensagens claras e amigáveis: falhas do SEI, credenciais, rede, download, PDF, OCR, modelo, memória, indexação, arquivos, pesquisa e geração/validação.
- **Não implementar módulo de logs técnicos nesta etapa**, conforme decisão do usuário; mecanismos mínimos de status/diagnóstico em memória não equivalem a autorizar persistir logs com dados de processos.
- Evitar enviar conteúdo sigiloso a buscas públicas. Documentos devem ser tratados como dados não confiáveis para instruções de IA.

## 34. Instalação do aplicativo e dependências — [APROVADO]

> **[Emenda §47]** Superada no que trata de inferência por modelos locais (GGUF, llama-server, download/importação de modelos, hardware, GPU, quantização e contexto por memória). O restante desta seção permanece em vigor; ver seção 47.

- Distribuir para Windows com instaladores **`.exe` e `.msi`**, com artefatos adequados às diferentes arquiteturas suportadas (um por arquitetura quando aplicável).
- Atualizações do sys-executor ocorrerão por **novo build e entrega de novo instalador**; não implementar atualização automática como requisito inicial.
- Instalar e configurar **automaticamente** dependências de execução, bibliotecas necessárias, OCR e mecanismo de inferência pelo instalador/aplicativo sempre que tecnicamente possível.
- Exigir intervenção manual do usuário **somente quando indispensável** (por exemplo, permissão de sistema ou requisito que não possa ser provido de forma segura/automática).
- Manter **instalador principal leve**: não embutir obrigatoriamente pesos GGUF de vários gigabytes.
- Configurar download orientado de modelos locais na primeira utilização e permitir importação de modelos GGUF já existentes.

## 35. Modo offline e conectividade — [APROVADO]

> **[Emenda §47]** Superada no que trata de inferência por modelos locais (GGUF, llama-server, download/importação de modelos, hardware, GPU, quantização e contexto por memória). O restante desta seção permanece em vigor; ver seção 47.

- Depois de instalados os modelos de IA, componentes necessários e a biblioteca jurídica, o aplicativo deve ser capaz de **analisar PDFs de autos enviados manualmente, gerar, validar, revisar e regenerar minutas integralmente offline**.
- Funcionalidades que exigem acesso remoto continuam dependentes de internet: autenticação/navegação/download do SEI, pesquisas jurídicas na web, downloads de novos modelos e downloads/consultas externas necessárias.
- Ao selecionar acesso SEI sem conexão, explicar a limitação e oferecer **upload de PDF**, preservando a seleção do procedimento e a configuração corrente.
- O aplicativo não deve tentar silenciosamente consultar fontes externas se a pesquisa web estiver desligada ou a máquina offline.

## 36. Estratégia geral da IA local — [APROVADO]

> **[Emenda §47]** Superada no que trata de inferência por modelos locais (GGUF, llama-server, download/importação de modelos, hardware, GPU, quantização e contexto por memória). O restante desta seção permanece em vigor; ver seção 47.

1. **IA local é prioritária** na primeira versão.
2. Arquitetura preparada para integração **futura e opcional** com provedores de IA externos, sem implementá-los como requisito inicial nem autorizar envio de autos a terceiros.
3. Utilizar modelo local quantizado **GGUF**; `llama.cpp`/`llama-server` como candidatos técnicos coerentes com o anexo de arquitetura enviado, **não escolha irrevogável**.
4. Não acoplar regras funcionais/jurídicas ao fornecedor ou arquivo específico do modelo.
5. Manter **motor determinístico em Go** para verificações, cálculos disciplinares, janelas temporais, equivalências, pontuação e demais regras formalizáveis. A IA interpreta documentos, ajuda a analisar evidências, recupera fontes e redige textos; a decisão final e a avaliação humana permanecem necessárias.
6. A proposta `Go + sidecar llama-server + RAG local`, apresentada no PDF de referência pelo usuário, é **insumo para avaliação técnica**, não especificação obrigatória a ser copiada integralmente. Corrigir problemas de gerenciamento de subprocessos, segurança e embeddings antes de implementar.

## 37. Gerenciamento e seleção de modelos de geração — [APROVADO]

> **[Emenda §47]** Superada no que trata de inferência por modelos locais (GGUF, llama-server, download/importação de modelos, hardware, GPU, quantização e contexto por memória). O restante desta seção permanece em vigor; ver seção 47.

- Manter **mais de um modelo local instalado**.
- Para cada tarefa, executar **apenas o modelo de geração selecionado**; modelos auxiliares de embeddings são componentes separados e podem ser usados sequencialmente.
- Detectar CPU, RAM, GPU e VRAM disponíveis e **selecionar automaticamente** um modelo apropriado dentre os disponíveis/compatíveis, considerando também qualidade aferida em testes, não só tamanho.
- Disponibilizar **seleção manual avançada** entre modelos instalados compatíveis.
- Na primeira utilização, verificar modelos instalados e oferecer **download orientado do modelo recomendado**, além de **importação de GGUF existente**.
- Verificar integridade e compatibilidade dos arquivos importados/baixados; permitir listar e remover modelos não utilizados.
- Modelo exato não foi definitivamente aprovado: avaliar Qwen 2.5 3B Instruct GGUF (referência original), Qwen 3 4B Instruct, Qwen 3.5 2B, Qwen 3.5 4B ou candidatos adequados, conforme documentação, licença, desempenho e testes reais.
- O aplicativo não precisa mostrar tokens gerados em tempo real; pode utilizar streaming **internamente** se vantajoso tecnicamente.

## 38. Hardware, GPU, quantização e contexto — [APROVADO COMO DIRETRIZES INICIAIS; VALIDAR EM TESTES]

> **[Emenda §47]** Superada no que trata de inferência por modelos locais (GGUF, llama-server, download/importação de modelos, hardware, GPU, quantização e contexto por memória). O restante desta seção permanece em vigor; ver seção 47.

### 38.1. Perfis de hardware

| Perfil | Requisitos pretendidos | Observações |
|---|---|---|
| Básico | **8 GB RAM**, CPU **x64 com ao menos 4 núcleos físicos**, AVX2 **preferencial**; SSD recomendado | Alvo mínimo, sem promessa de desempenho/qualidade jurídica até testes; modelo leve |
| Intermediário | 16 GB RAM, CPU moderna, GPU opcional | Maior margem para OCR, modelo e contexto |
| Avançado | 32 GB RAM ou mais, GPU compatível preferencialmente com 8 GB VRAM ou mais | Modelos/contextos mais exigentes, conforme testes |

- Detectar GPU compatível e **priorizá-la em relação à CPU** quando viável. Avaliar aceleração e offload parcial GPU/CPU quando apropriado.
- Uma GPU detectada pode ser insuficiente/incompatível. Nesse caso, avaliar uso da CPU com recursos adequados em vez de bloquear automaticamente a tarefa.
- O aplicativo deve proteger-se de falta de memória e monitorar falhas de carregamento/execução.

### 38.2. Quantização

- **Q4_K_M** será a **referência inicial** de quantização por equilíbrio entre uso de memória e qualidade; não proibir outras quantizações compatíveis.
- Quantização e tamanho do modelo são decisões distintas. Validar empiricamente os efeitos na fidelidade documental e qualidade de redação jurídica.
- **Não declarar Qwen 2.5 3B ou qualquer outro modelo como definitivo** antes de testes.

### 38.3. Janela de contexto adaptativa

| RAM total do perfil | Faixa inicial indicativa de contexto |
|---|---|
| 8 GB | 4.096–8.192 tokens |
| 16 GB | 8.192–16.384 tokens |
| 32 GB ou mais | 16.384–32.768 tokens |

- Ajustar contexto ao **modelo e à memória efetivamente disponível**, não apenas à RAM instalada.
- As faixas são **pontos de partida**, não limites fixos nem garantia de viabilidade.
- Contexto maior não substitui recuperação documental estruturada; considerar espaço para instruções, fontes, dados e resposta.

### 38.4. Falta de capacidade da máquina — [APROVADO]

- Se o modelo escolhido não puder executar adequadamente, apresentar **mensagem amigável e compreensível**, sem expor erro técnico cru.
- Se houver outro modelo instalado ou disponível comprovadamente compatível, **sugerir a alternativa**.
- Se nenhuma alternativa viável existir, informar que a tarefa não poderá ser realizada naquele equipamento devido às limitações de hardware.
- Não simular sucesso nem seguir com análise degradada sem conhecimento do usuário.

## 39. Processamento de autos extensos — [APROVADO]

- Processar autos extensos em **etapas**, sem pressupor que o PDF inteiro caiba na janela de contexto.
- Manter durante a sessão o **texto integral extraído**, suas peças, páginas, referências SEI e vínculos às fontes originais, evitando reduzir a única evidência disponível a resumos.
- Extrair e organizar **informações estruturadas**: fatos, datas, imputações, sindicados/acusados, oitivas, provas, alegações, contradições, limitações, informações faltantes e relações entre documentos.
- Criar **sínteses de trabalho** e recuperar os trechos relevantes diretamente da fonte original conforme cada etapa da análise ou revisão.
- Usar busca documental textual e, quando tecnicamente justificado, busca vetorial **temporária** para os autos; não tornar a indexação vetorial de toda página um requisito obrigatório.
- Verificar cobertura das peças/páginas e sinalizar falhas de OCR e documentos essenciais não interpretáveis; não presumir conteúdo ilegível nem ignorar peças silenciosamente.
- Não usar informações que expandam imputações do PADS. Na SR, observar regras de inclusão de fatos novos sob autorização.
- Dados, sínteses e índices relativos aos autos são **temporários** e sujeitos à limpeza prevista na seção 29.

## 40. Biblioteca jurídica permanente / RAG — [APROVADO]

### 40.1. Finalidade

- Armazenar **legislações, regulamentos, normas, jurisprudência e outras fontes jurídicas** para fundamentação, recuperação de dispositivos, esclarecimento de dúvidas e citações durante a elaboração das minutas.
- A biblioteca é **permanente** e separada de autos e minutas **temporários**.
- Priorizar fontes oficiais, versões identificáveis e rastreabilidade até a origem.
- Utilizar mecanismo RAG local, com recuperação de trechos e referência aos documentos originais.

### 40.2. Conteúdo inicial e diretório do projeto

- O usuário criará no **projeto** a pasta literal **`docs/law_library`**, depositando nela os documentos que deverão compor a biblioteca jurídica inicial.
- O agente deverá identificar e importar/indexar esses documentos no processo inicial de instalação/preparação da biblioteca, sem exigir acervo grande na primeira versão.
- Permitir que o usuário **amplie manualmente** a biblioteca ao longo do tempo, adicionando documentos posteriores.
- Não supor quais arquivos concretos estarão presentes na pasta até examiná-la durante a implementação; não inventar conteúdo, versão ou vigência.

### 40.3. Armazenamento e recuperação

- A biblioteca deverá guardar **documentos e metadados de origem** além dos vetores; embeddings não substituem textos normativos originais.
- Utilizar recuperação combinando, quando viável, **busca textual exata** (número de artigo, inciso, expressões) e **busca semântica** (conceitos e temas), evitando confiar apenas em similaridade vetorial.
- `chromem-go` para vetor e mecanismo de busca textual local como SQLite FTS5 são **candidatos técnicos**, não escolhas obrigatórias ou validadas.
- Ao indexar normas, preservar estruturas de artigo, parágrafo, inciso, alínea, contexto, número/identificação da fonte e localização.
- RAG fornece passagens de apoio; **não demonstra por si só vigência ou aplicabilidade jurídica**.

### 40.4. Embeddings — [APROVADO]

- Manter modelo de embeddings **separado** do modelo de linguagem usado para análise/redação.
- O aplicativo gerenciará **automaticamente** o modelo de embeddings recomendado, inclusive download inicial, sem configuração manual obrigatória pelo usuário.
- Trocar o modelo de **geração** não deve exigir reconstruir a base vetorial.
- Trocar versão/arquitetura do próprio modelo de **embeddings** pode exigir reindexação para manter compatibilidade, sempre preservando a integridade dos originais.
- `Qwen3-Embedding-0.6B` é candidato de avaliação, não seleção definitiva.
- Não usar automaticamente função de embeddings dependente de API externa se o modo local/offline for requerido.

### 40.5. Categorias — [APROVADO]

Classificar automaticamente por conteúdo e metadados, com **correção manual pelo usuário**. Categorias iniciais exemplificativas:

1. Legislação (leis, decretos, leis complementares);
2. Regulamentos disciplinares;
3. Normas institucionais (diretrizes, portarias, instruções);
4. Jurisprudência (decisões e acórdãos de fontes oficiais);
5. Outros documentos jurídicos (manuais, pareceres e referências pertinentes).

A classificação não deve impedir buscas que atravessem diferentes categorias.

### 40.6. Duplicidades, versões, inclusão e substituição — [APROVADO]

- Ao adicionar documento, detectar **possíveis duplicidades** e versões diferentes mediante identificação, conteúdo e metadados.
- Mostrar as versões encontradas e deixar o usuário escolher **substituir a versão anterior ou manter ambas**.
- **Nunca substituir silenciosamente** normas ou jurisprudência.
- A substituição deverá atualizar os **índices textual e vetorial** correspondentes e preservar metadados/rastreabilidade da operação e da fonte.
- Se versões distintas coexistirem, identificar a versão usada em cada recuperação/citação, sem presumir que a mais recente armazenada seja necessariamente a vigente ou aplicável.
- Atualização do **conteúdo** da base jurídica continua restrita a **solicitação expressa do usuário**, mesmo que o sistema consiga pesquisar juridicamente na web.

### 40.7. Exclusão de documentos — [APROVADO]

- Solicitar **confirmação** antes de excluir um documento da biblioteca jurídica.
- Remover o documento e seus **índices textuais e vetoriais** correspondentes; preservar outros documentos e índices.
- Não afetar pesos GGUF ou modelos auxiliares instalados.
- Se documento estiver sendo utilizado por **tarefa ativa**, bloquear exclusão até encerramento dessa tarefa.
- Exibir mensagem amigável de sucesso, falha ou impedimento.

## 41. Pesquisa jurídica externa — [APROVADO]

- A pesquisa na internet estará **DESABILITADA por padrão na primeira instalação**.
- O usuário poderá habilitá-la ou desabilitá-la nas **configurações**; a preferência persistirá entre sessões até alteração.
- Quando habilitada, permitir **pesquisa jurídica automática e condicional** em fontes oficiais, **sem confirmação por consulta**, quando houver controvérsia jurídica relevante, insuficiência das referências locais ou necessidade justificada de apoio adicional.
- A pesquisa externa deve consultar fontes oficiais, sem transmitir nomes, dados pessoais, identificadores ou informações sigilosas dos autos em consultas públicas.
- A internet é complemento; não exigir pesquisa web indiscriminadamente em toda tarefa.
- Quando a opção estiver desligada ou a conexão indisponível, usar **somente a base jurídica local**, sinalizando limitação relevante se houver.
- Resultado de pesquisa externa poderá apoiar a minuta, mas **não será incorporado automaticamente** ao corpus RAG persistente; incorporar/atualizar corpus exige solicitação explícita.
- Referências jurídicas devem ser conferidas na fonte de origem antes de citação; manter rastreabilidade.

## 42. Validação obrigatória das minutas — [APROVADO]

### 42.1. Porta de qualidade anterior à revisão humana

- **Toda minuta PADS ou SR deverá passar por validação automática obrigatória antes de ser disponibilizada para aprovação do usuário.**
- A validação poderá combinar verificações determinísticas em Go com chamadas adicionais à mesma IA, sem obrigar segundo modelo de geração.
- Verificar pelo menos:
  1. Fidelidade entre fatos, datas, depoimentos, afirmações e fontes dos autos;
  2. Referências jurídicas e jurisprudenciais verificáveis; ausência de fontes inventadas;
  3. Regras específicas do PADS (estrita vinculação ao Memorando) ou da SR (Portaria, fatos novos autorizados, estrutura própria);
  4. Coerência entre provas, argumentos, fundamentação e conclusões;
  5. Cálculos determinísticos de sanção, reincidência, pontos e comportamento quando aplicáveis;
  6. Presença das seções obrigatórias, campos e avisos sobre evidências essenciais ausentes;
  7. Coerência entre resultado confirmado pelo usuário e minuta escrita.
- A validação automatizada **não substitui revisão humana** e não representa homologação jurídica.

### 42.2. Quando uma falha for encontrada

1. Tentar **corrigir automaticamente** o problema por nova leitura/consulta de documentos, revisão de informações estruturadas e verificação das referências.
2. Não inventar fatos, provas, artigos ou jurisprudência para cobrir lacunas; não alterar silenciosamente decisões confirmadas pelo usuário.
3. Se a correção automática não resolver, **mostrar claramente a pendência ao usuário** e solicitar esclarecimento **somente quando necessário**, antes de disponibilizar a minuta para aprovação.
4. Limitações não impeditivas poderão ser explicitadas no texto da minuta, seguindo as regras já aprovadas para provas inacessíveis, contradições e insuficiência probatória.
5. Após ajustes do usuário, revalidar antes de apresentar novamente a minuta completa.

## 43. Documentação e escolhas técnicas em aberto — [VALIDAR EM TESTES / DELEGAR]

O usuário aprovou **avaliar tecnicamente** as seguintes escolhas, sem necessidade de interrogá-lo por detalhes triviais:

- Modelos definitivos de geração e embeddings, licenças, formatos GGUF, quantizações e consumo real de RAM/VRAM;
- Compatibilidade do `llama.cpp`/`llama-server` como sidecar em Windows, CPU, aceleração GPU e desligamento robusto;
- Banco vetorial local e mecanismo de busca lexical, indexação de dispositivos jurídicos e migração/reindexação;
- OCR e extração de PDFs, classificação documental, rastreabilidade e revisão por etapas;
- Valores efetivos de contexto, chunking, caching temporário, concorrência, limites de memória e desempenho em equipamentos com 8 GB;
- Integração Wails 3/Go/Vue 3, seleção de modelos, instaladores `.exe`/`.msi` e dependências automatizáveis.

**Regra de encaminhamento:** o agente de implementação deverá consultar documentação oficial atual, realizar testes comparativos, justificar escolhas e perguntar ao usuário **somente em decisões substanciais ou conflitos jurídicos/técnicos relevantes**; não tratar candidatos como tecnologias fechadas sem confirmação empírica.

## 44. Validação jurídica normativa e preservação das decisões da V3 — [APROVADO]

- O usuário confirmou que **as interpretações funcionais/jurídicas já aprovadas na V3 serão mantidas**, inclusive as relativas a detenção/prisão, cumprimento residencial, pontuação, regressão especial, equivalências e reincidência.
- A implementação deverá **verificar documentação normativa oficial vigente**, mas **não alterar silenciosamente** decisão aprovada por hipótese de conflito.
- Se surgir **conflito normativo efetivamente demonstrado**, apresentá-lo ao usuário com fonte e explicação, para decisão expressa; não reabrir questões sem fundamento relevante.
- A aprovação do usuário de uma regra funcional **não equivale a comprovação documental independente de vigência ou validade de interpretação**. Essa distinção deve ser mantida no consolidado final.

## 45. Matriz complementar de aceitação / testes

> **[Emenda §48]** T02 e T03 passam a ser verificados pelo cadastro de peças; T04–T06, pela conferência da peça central declarada (seção 48.2).

| ID | Cenário | Resultado esperado |
|---|---|---|
| T01 | Usuário inicia tarefa | Primeira escolha é PADS ou SR; segunda, SEI ou PDF |
| T02 | PDF completo por upload | Não solicita senha ou acesso ao SEI; entra no pipeline compartilhado |
| T03 | SEI sem internet | Mensagem amigável e opção de enviar PDF sem reiniciar configuração |
| T04 | Seleciona PADS; PDF é claramente SR | Alerta e pede correção/confirmação; não altera tipo automaticamente |
| T05 | PADS contém portaria de SR como peça anexa | Não confunde o procedimento principal |
| T06 | Peça instauradora ilegível | Solicita esclarecimento/versão legível; não infere tipo arbitrariamente |
| T07 | Tarefa em execução | Mostra status; não apresenta chatbot completo nem tokens em streaming |
| T08 | Informação disciplinar essencial ausente | Pergunta objetivamente, conforme V3, e retoma análise após resposta |
| T09 | Minuta concluída e validada | Oferece visualização e download DOCX/PDF, além de ajustes/finalização |
| T10 | Usuário pede ajuste | Regenera/revalida minuta na mesma sessão; preserva partes válidas |
| T11 | Usuário aprova e finaliza | Apaga artefatos temporários controlados, preserva biblioteca/modelos |
| T12 | App fecha ou falha durante tarefa | Trabalho não pode ser retomado; resíduos são limpos na próxima abertura |
| T13 | Novo acesso SEI | Solicita novamente credenciais; senha não persiste em disco |
| T14 | Computador tem GPU adequada | Prioriza GPU; aceita fallback CPU se necessário e viável |
| T15 | Máquina insuficiente | Mensagem clara, sugere outro modelo viável; se não houver, informa impossibilidade |
| T16 | Máquina com 8 GB RAM | Testar modelo pequeno Q4 e contexto adaptativo sem garantir previamente sucesso |
| T17 | Autos com centenas de páginas | Processa por etapas, guarda texto original e referências, detecta peças não lidas |
| T18 | Base jurídica e modelos pré-instalados, sem internet | Permite gerar/revisar minuta com upload PDF |
| T19 | Pesquisa web desabilitada inicialmente | Não realiza consultas jurídicas externas |
| T20 | Pesquisa web habilitada | Pesquisa condicional em fonte oficial, sem dados sigilosos e sem atualizar corpus |
| T21 | Alternância de modelo gerador | Não obriga reindexação do corpus jurídico |
| T22 | Inclusão de norma com versão similar | Detecta duplicidade, apresenta opções manter/substituir e atualiza índices conforme escolha |
| T23 | Tentativa de excluir documento usado em tarefa ativa | Bloqueia exclusão; preserva demais dados |
| T24 | IA inventa referência ou contradiz fontes | Validação detecta, tenta corrigir e pede esclarecimento se persistir |
| T25 | Exportação de PADS e SR | Times New Roman 12 e cabeçalho/título específicos com `logo-pmro.png` |
| T26 | Atualização do aplicativo | Distribui novo instalador; não depende de atualizador automático |
| T27 | Modelos não instalados | Assistente de download orientado ou importação GGUF |
| T28 | Usuário adiciona fonte a `docs/law_library` na preparação inicial | Documento é identificado e indexado quando incorporado ao conjunto inicial |

## 46. Registro da fusão V3 + complemento → V4 e continuidade — [APROVADO]

### 46.1. Como esta V4 foi montada

- As seções 1–27 da V3 foram **preservadas integralmente**, sem resumo nem supressão de regras, quadros, exemplos ou casos-limite.
- Os trechos das seções 1–27 que tratavam como pendente algo decidido nas seções 28–45 receberam nota **[Atualização V4]**, mantido o texto original para histórico (seções 1, 1.1, 1.2, 2, 2.1, 2.2, 11, 12, 14, 18, 20 e 27).
- O complemento de decisões transversais e de IA de 08/11/2026 foi incorporado como seções 28–45, com a numeração ajustada (1 → 28 … 18 → 45) e as remissões internas corrigidas.
- Nas seções 28–46, as referências à “V3” designam as regras das seções 1–27 deste documento.

### 46.2. Instruções originais de fusão (cumpridas nesta V4)

1. **Preservar integralmente** a V3 — seções 1 a 27, quadros, exemplos, textos aprovados e critérios de precedência; não resumir ou apagar regras antigas.
2. Incorporar este complemento como **novas seções de decisões transversais/IA** ou distribuir seu conteúdo entre capítulos equivalentes, sempre mantendo a distinção **[APROVADO] / [PROPOSTA TÉCNICA] / [VALIDAR EM TESTES]**.
3. Atualizar o quadro de pendências da V3: formatação institucional, sessão efêmera, duas modalidades de entrada, interface sem chatbot, biblioteca jurídica, estratégia de IA, modo offline, pesquisa web e validação de minuta **não devem continuar listados como indefinidos** no que foi aqui decidido.
4. Preservar como **não finalizados** a escolha exata dos modelos/embeddings/banco vetorial e os parâmetros calibrados por teste; o anexo `Especificação Técnica - Go Sidecar llama-server e RAG Local.pdf` é uma **referência de pesquisa**, não obrigação de copiar código.
5. As **imagens dos cabeçalhos** e o futuro arquivo `logo-pmro.png` devem ser tratados como insumos de referência do projeto; este Markdown descreve seu conteúdo, mas não os incorpora como bytes.
6. **Não gerar prompt definitivo de implementação** sem autorização expressa do usuário, ainda que a especificação tenha sido consolidada.

---

## ESTADO FINAL — V4 (08/11/2026)

**ESPECIFICAÇÃO CONSOLIDADA ÚNICA. CONCLUÍDO O LEVANTAMENTO DOS REQUISITOS FUNCIONAIS, JURÍDICOS, TRANSVERSAIS E DE ARQUITETURA DE IA DO SYS-EXECUTOR: REGRAS DE JULGAMENTO DE PADS (VINCULADO ÀS IMPUTAÇÕES DO MEMORANDO DISCIPLINAR) E DE RELATÓRIO DE SINDICÂNCIA REGULAR PRESERVADAS; SESSÕES EFÊMERAS; ENTRADA POR SEI OU UPLOAD DE PDF; INTERFACE DE EXECUÇÃO SEM CHATBOT; FORMATAÇÃO INSTITUCIONAL ÚNICA; IA LOCAL PRIORITÁRIA COM RAG JURÍDICO; FUNCIONAMENTO OFFLINE COM PDF ENVIADO PELO USUÁRIO; PESQUISA WEB DESABILITADA POR PADRÃO; VALIDAÇÃO AUTOMÁTICA OBRIGATÓRIA E REVISÃO HUMANA FINAL DE TODA MINUTA. PERMANECEM APENAS ESCOLHAS TÉCNICAS A VALIDAR EM TESTES (SEÇÃO 43), A CONFERÊNCIA NORMATIVA COMO SALVAGUARDA (SEÇÃO 44) E CASOS-LIMITE RAROS. ESTE MARKDOWN É A FONTE DA VERDADE DO PROJETO, NÃO O PROMPT DEFINITIVO DE IMPLEMENTAÇÃO. O PROMPT DEFINITIVO SOMENTE PODERÁ SER GERADO APÓS AUTORIZAÇÃO EXPRESSA DO USUÁRIO.**

---

## 47. EMENDA 1 — Migração da inferência para a Gemini API — [APROVADO] (09/11/2026)

### 47.1. Decisão e alcance

> **[Emenda §48]** Deixa de valer "OAuth… e outros provedores ficam fora do escopo": o ChatGPT é admitido como segundo provedor (seção 48). Continuam fora: Vertex AI, troca automática entre provedores e inferência local.

- Por decisão expressa do usuário, a inferência por modelos de linguagem executados no computador (GGUF, `llama.cpp`/`llama-server`) **é removida** e substituída pela **Gemini API** (Google), acessada pelo backend Go com o SDK oficial `google.golang.org/genai` e autenticada por **chave de API cadastrada pelo próprio usuário**. Motivo: desempenho insatisfatório dos modelos locais no hardware disponível.
- A troca é **exclusivamente de infraestrutura de inferência**. Permanecem integralmente em vigor, sem alteração: as regras jurídicas e de redação das seções 1–27; o motor determinístico em Go (seção 36, item 5); as perguntas, confirmações e autorizações (inclusive as da SR, seção 24.3); a vinculação do PADS às imputações do Memorando Disciplinar (seção 22); a validação obrigatória (seção 42); a revisão humana; a biblioteca jurídica local (seção 40); a pesquisa web desabilitada por padrão (seção 41); as sessões efêmeras (seção 29); as credenciais do SEI somente em memória (seção 33).
- Não há fallback para modelo local. OAuth, Vertex AI e outros provedores ficam fora do escopo desta emenda.

### 47.2. Seções alcançadas

| Seção | Efeito da emenda |
|---|---|
| 1, 2, 12.2 e "Estado final — V4" | Onde se lê "IA local prioritária" e "sem envio de autos a terceiros", vale a seção 47 |
| 29.3 | Sai "Modelos locais"; entra "Chave da Gemini API" (47.4) |
| 34 | Sem mecanismo de inferência, pesos GGUF ou download de modelos no instalador/primeiro uso; demais itens mantidos |
| 35 | A análise e a geração por IA **exigem internet**; as demais funções seguem offline (47.6) |
| 36, itens 1–3 | Provedor autorizado: Gemini API. Itens 4–6 mantidos |
| 37 | Substituída por 47.5 (seleção de modelo Gemini). Mantido: sem streaming de tokens na interface |
| 38 | Superada (perfis de hardware, GPU, quantização, contexto por memória). Mantido de 38.3: "contexto maior não substitui recuperação documental estruturada"; mantido de 38.4: não simular sucesso nem seguir com análise degradada sem conhecimento do usuário |

### 47.3. Envio de conteúdo ao provedor

> **[Emenda §48]** Somam-se a barreira obrigatória, a revisão item a item e o bloqueio de dados de crianças no ChatGPT (seção 48.6). A autorização de envio passa a valer para um provedor e um modelo (seção 48.1).

- **Somente o necessário:** o texto dos autos continua extraído localmente (pdfium/OCR), segmentado e conferido no computador; a IA recebe apenas os blocos de texto de cada etapa, como antes. **Não** se envia o PDF, **não** se usa a Files API nem armazenamento de contexto no provedor.
- **Confirmação em cada tarefa:** antes da primeira etapa de IA, o usuário vê o modelo, as peças e páginas que serão lidas, uma estimativa aproximada de tokens e o resumo da pseudonimização, e **autoriza ou recusa** o envio. Sem autorização, nada é enviado e a tarefa para de forma recuperável.
- **Pseudonimização (melhor esforço):** antes do envio, nomes de pessoas, CPF, RG, matrícula, telefone, e-mail, CEP, endereço e data de nascimento detectados localmente são substituídos por marcadores (ex.: `[PESSOA-03]`, `[CPF-01]`) e restaurados na resposta. Datas dos fatos, números SEI e de processo e links não são substituídos. Posto/graduação é mantido. O usuário pode informar, na confirmação, outros nomes ou dados a ocultar. **Não é anonimização garantida**: dados sem padrão reconhecível podem passar, e o contexto pode permitir reidentificação.
- **Sem retenção opcional:** toda requisição desliga o registro de requisições do provedor ("store" = falso); o modelo não recebe ferramentas (sem busca na web, URL, execução de código ou chamada de funções).
- **Nível da conta:** qualquer nível de conta da Gemini API é aceito, sem aviso na interface (decisão do usuário). Fica registrado que, pelos termos do provedor, o nível gratuito pode usar o conteúdo para melhoria de produtos, com revisão humana, e que o nível pago registra prompts e respostas por tempo limitado para prevenção de abuso.
- Valem, quanto ao provedor, as ressalvas já previstas nas seções 1.1 (item 9) e 11: o aplicativo **não promete apagar** registros, logs ou retenções do provedor externo. A confirmação do usuário **não substitui** eventual autorização institucional para tratar dados da PMRO em serviço externo.

### 47.4. Chave da Gemini API

| Armazenamento | Conteúdo | Persistência |
|---|---|---|
| Chave da Gemini API | Chave cadastrada pelo usuário | Permanente, protegida pelo Windows (DPAPI, conta do usuário); substituível e removível pelo usuário |

- A chave é lida e usada somente pelo backend; nunca é exibida de volta, enviada à interface, incluída em eventos, preferências, mensagens de erro ou logs, nem empacotada no instalador.
- A validação da chave usa a listagem de modelos, sem geração de texto.
- A regra da seção 33 sobre a **senha do SEI** (somente memória, a cada tarefa, sem arquivo cifrado) permanece inalterada; a chave da Gemini API não é credencial do SEI.

### 47.5. Escolha do modelo

- Nenhum modelo é fixado no código. O aplicativo lista os modelos Gemini de texto disponíveis para a chave, com nome, identificador, estágio (estável, prévia, experimental ou apelido "latest"), limites e capacidade de raciocínio; o usuário escolhe um e pode testá-lo.
- Um único modelo por tarefa. Atribuir modelos diferentes a etapas diferentes depende de nova decisão do usuário.
- Os parâmetros técnicos (janela de trabalho, raciocínio, temperatura) são calibrados no benchmark sintético, sem dados reais, mantidos os portões rígidos: **zero violações aceitas após validação e zero citações inventadas aprovadas**.

### 47.6. Conectividade e falhas

- Sem internet, chave ausente, inválida ou revogada, cota esgotada, modelo indisponível, tempo esgotado, resposta bloqueada ou incompleta: mensagem clara em português, **parada recuperável** com o estado da tarefa preservado e opção de tentar novamente; nunca sucesso simulado nem resposta interrompida tratada como concluída.
- Funcionam sem internet: abrir e ler PDF, OCR, segmentação, visualização das fontes, cálculos determinísticos e as demais funções que não dependem da IA. O acesso ao SEI continua dependendo de internet (seção 35).
- O cancelamento pelo usuário interrompe a requisição em andamento.

### 47.7. Pendência registrada

- A busca semântica da biblioteca jurídica (seção 40.4) previa embeddings locais. Com a remoção da inferência local, a escolha entre busca somente lexical e embeddings do provedor (apenas para a legislação pública, com consultas compostas pelo Go a partir do catálogo) será decidida pelo usuário no início da fase da biblioteca.

## ESTADO FINAL — V4 + EMENDA 1 (09/11/2026)

**INFERÊNCIA POR GEMINI API COM CHAVE DO USUÁRIO (SEÇÃO 47), EM SUBSTITUIÇÃO À IA LOCAL. ENVIO SOMENTE DO NECESSÁRIO, COM CONFIRMAÇÃO EM CADA TAREFA E PSEUDONIMIZAÇÃO DE DADOS PESSOAIS. TODAS AS DEMAIS REGRAS DA V4 PERMANECEM EM VIGOR.**

## 48. EMENDA 2 — ChatGPT como segundo provedor, cadastro de peças, enquadramento manual e barreira de privacidade — [APROVADO] (09/11/2026)

### 48.1. Decisão e alcance

- Por decisão expressa do usuário (D-11), a seção 47 passa a admitir **dois provedores de inferência**:
  - a **Gemini API**, com chave do usuário, inalterada;
  - o **ChatGPT**, pelo fluxo oficial *Sign in with ChatGPT*, que usa o plano ChatGPT do próprio usuário em requisições elegíveis à Responses API da OpenAI.
- A integração é implementada diretamente no backend Go do aplicativo, sem serviço intermediário, Node.js, Codex CLI, proxy local, reaproveitamento de cookies ou tokens de outros aplicativos, nem endpoints privados do ChatGPT.
- O provedor e o modelo são **escolhidos explicitamente** pelo usuário. **Não há troca automática** entre provedores.
- A autorização de envio de cada tarefa (47.3) vale para **um provedor e um modelo**. Se qualquer um dos dois mudar, ou se mudar o conjunto de dados a ocultar, o aplicativo pergunta de novo.
- A entrada dos autos passa a ser feita por **peças cadastradas individualmente** (48.4), e o **enquadramento** das imputações do PADS passa a ser **informado manualmente** pelo usuário a partir de um catálogo curado (48.5).
- Continuam **integralmente em vigor**:
  - as regras jurídicas e de redação das seções 1–27;
  - a vinculação do PADS às imputações do Memorando Disciplinar (seção 22);
  - o regime de fatos e militares novos na SR (seção 24.3);
  - as regras de conexão, dosimetria, reincidência, comportamento, pontuação e regressão, e o motor determinístico;
  - a validação obrigatória (seção 42), a revisão humana e as sessões efêmeras (seção 29);
  - as credenciais do SEI somente em memória (seção 33).
- Continuam fora do escopo: Vertex AI, troca automática entre provedores e retorno à inferência local.

### 48.2. Seções alcançadas

| Seção | Efeito da emenda |
|---|---|
| 1, 1.1, 2 (tabela), 2.1, 30.1, 30.3, 31.4, 35 e "Estado final" | Onde se lê "upload manual do PDF completo/consolidado", vale o cadastro de peças (48.4). A modalidade SEI passa a obter só os documentos selecionados pelo usuário |
| 2.2 e 16.1 (A01, A02, D1, D2) | O tipo e a função de cada peça são **declarados pelo usuário** no cadastro. A classificação automática por conteúdo e cronologia passa a servir de **conferência**: divergência gera pergunta, nunca troca silenciosa |
| 17.4 (A12) | A distinção entre peça de origem SEI e material complementar se faz pela origem registrada no cadastro (SEI ou envio pelo usuário) |
| 16.4, 39 | "Página do PDF consolidado" passa a ser a página dos autos montados a partir das peças, com o vínculo peça → arquivo → página preservado |
| 22 e PADS-ESC-001 | As imputações continuam vinculadas ao Memorando. A IA extrai fatos, condutas e datas; os **dispositivos e a natureza vêm do usuário** (48.5), conferidos e congelados como antes |
| 29.3 | Acrescenta a credencial do ChatGPT (48.3) |
| 30.2 (T04–T06) | A peça central (Memorando ou Portaria) é a declarada pelo usuário, e seu conteúdo é conferido. Se o conteúdo indicar outra modalidade ou outra peça, o aplicativo pergunta. A troca nunca é silenciosa |
| 45 (T02, T03) | Substituídos pelos testes do cadastro de peças |
| 47.1 | Deixa de valer "OAuth… e outros provedores ficam fora do escopo" |
| 47.3 | Somam-se a barreira obrigatória e a revisão item a item (48.6). As demais regras de envio valem para os dois provedores |
| 47.4, 47.5 | Valem para a Gemini; para o ChatGPT valem 48.3 e 48.7 |

### 48.3. Conta ChatGPT (Sign in with ChatGPT)

- O usuário conecta a própria conta pelo **navegador padrão do Windows**, com o fluxo oficial de autorização (OAuth 2.0 com PKCE e OpenID Connect).
- **Verificações obrigatórias:**
  - `state`, `nonce`, emissor, destinatário, assinatura e validade do token de identidade;
  - concessão expressa da **permissão de uso do plano**. Sem essa permissão, o provedor fica desabilitado para gerar texto, mesmo com o login feito.
- O aplicativo se identifica pelo seu nome verdadeiro, **sys-executor**.
- O retorno da autorização é recebido somente no endereço local do próprio computador (`127.0.0.1`).

| Armazenamento | Conteúdo | Persistência |
|---|---|---|
| Credencial do ChatGPT | Identificação do registro do aplicativo, identificador da instalação, identidade validada da conta, permissões concedidas e token de renovação | Permanente, protegida pelo Windows (DPAPI, conta do usuário); removida ao desconectar |

- O token de acesso fica **somente em memória**.
- Os tokens nunca são exibidos, enviados à interface, incluídos em eventos, preferências, mensagens de erro ou logs, nem colocados em endereços de página.
- A renovação substitui o token de renovação a cada uso e é feita **uma de cada vez**.
- Falhas de rede **não** apagam a credencial.
- **Desconectar** revoga a sessão no provedor, quando possível, e apaga a credencial local.
- Os limites de uso do plano são do provedor. O aplicativo **não tenta contorná-los** e informa o usuário quando são atingidos (48.7).
- A assinatura é pessoal e não é compartilhada entre usuários. O aplicativo não oferece acesso de uso geral à conta para outras ferramentas, não alterna contas para contornar limites e não cobra pelo uso do plano.
- **Aviso de privacidade antes do primeiro login.** Antes de conectar a conta, o aplicativo explica, em português:
  - quais dados vão ao provedor: só blocos de texto pseudonimizados das tarefas autorizadas;
  - que a tabela de correspondência e os arquivos dos autos não saem do computador;
  - onde ficam guardadas as credenciais;
  - como desconectar.
  
  Os Termos do *Sign in with ChatGPT* exigem esse aviso.
- **Marca.** O botão de conexão é só texto ("Entrar com ChatGPT"), sem logotipo da OpenAI e sem sugerir patrocínio ou endosso.

### 48.4. Cadastro de peças

- Depois de escolher a modalidade, o usuário cadastra as peças num assistente próprio.
  - **PADS:** Memorando Disciplinar (com o acusado e os fatos e dispositivos, 48.5); defesa prévia ou indicação de **não apresentada**; depoimentos (testemunha ou declarante, com o depoente); alegações finais ou indicação de **não apresentadas**; outros documentos.
  - **SR:** Portaria (com um ou vários sindicados e, quando houver, um ou vários ofendidos, ou a indicação de ausência de ofendido individualizado); depoimentos (testemunha, declarante ou sindicado, com o depoente); outros documentos.
- **Regras de identificação:**
  - cada peça tem identificador próprio e pode reunir **um ou vários arquivos**, com ordem e origem preservadas;
  - cada oitiva é uma peça própria, e a mesma pessoa pode ter várias;
  - arquivos repetidos são apontados.
- **Conferência antes da análise:** o usuário pode adicionar, excluir, substituir, renomear, reordenar e corrigir vínculos antes de iniciar a análise.
- **Pela modalidade SEI:**
  - o aplicativo apresenta a lista de documentos do processo (título e número SEI), lida somente no computador;
  - o usuário escolhe os documentos;
  - o aplicativo gera o PDF somente dos selecionados, pela própria opção do SEI, e o separa pelo carimbo SEI;
  - o usuário classifica e agrupa as peças no assistente.
  
  Não se ampliam as ações permitidas no SEI (seção 2.1 e guarda), e os títulos lidos do SEI não são enviados à IA.
- **Cruzamento com o conteúdo:** os dados declarados pelo usuário (tipo da peça, acusado, sindicados, ofendidos, depoentes) são cruzados com o conteúdo extraído. Divergência relevante gera pergunta. Os campos preenchidos pelo usuário **não ampliam** as acusações do Memorando nem o objeto da Portaria.
- A ausência de defesa prévia ou de alegações finais é registrada como tal e **não** é interpretada como confissão, concordância ou renúncia.
- O cadastro existe **somente na sessão efêmera** (seção 29). Não há rascunho persistente nem retomada após o fechamento do aplicativo.

### 48.5. Enquadramento manual no PADS

- O enquadramento das imputações é **informado pelo usuário** a partir de um catálogo curado, extraído da biblioteca jurídica, com a redação vigente de cada dispositivo:
  - **RDPM:** arts. 15 (natureza leve), 16 (média) e 17 (grave), com todos os incisos. A natureza é **fixa**, dada pelo próprio artigo;
  - **Estatuto dos Policiais Militares** (Decreto-Lei nº 09-A/1982): arts. 29 (preceitos da ética) e 32 (deveres). Contrariar um desses preceitos também é infração disciplinar. Como esses artigos não trazem a natureza, ela é **atribuída pelo usuário** por semelhança de grau de intensidade com as transgressões do RDPM.
- Um **fato cometido** pode ser enquadrado em mais de um dispositivo. **Cada dispositivo constitui uma imputação** com natureza própria.
- Imputações do mesmo fato são **candidatas a conexas** e seguem a regra já vigente (seção 5):
  - a principal é a de maior gravidade, e as demais funcionam como agravantes;
  - em empate de natureza, o usuário escolhe a que mais se amolda ao fato.
  
  A conexão é sempre confirmada pelo usuário.
- **Papel da IA:** extrai do Memorando os fatos, as condutas, as datas e os dispositivos citados, **apenas para conferência**.
- **Divergência com o Memorando:** se um dispositivo escolhido não corresponder ao citado no texto, o aplicativo mostra os dois lado a lado, com o trecho e a remissão à seção 22.1, e exige **confirmação expressa**.
- O aplicativo **nunca sugere** troca de dispositivo e nunca acrescenta imputação não contida no Memorando (seção 22).
- O catálogo de dispositivos é revisado pelo usuário e registrado no CHANGELOG das políticas, com a fonte (arquivo e página) de cada item.

### 48.6. Barreira de privacidade antes do envio

- A pseudonimização da seção 47.3 continua com o mesmo alcance: nº SEI, nº de processo, datas dos fatos, links e posto/graduação **não** são substituídos. Ela passa a contar também com as pessoas cadastradas (48.4), suas grafias informadas e as variações de caixa e acento.
- **Barreira obrigatória.** Antes de qualquer requisição, o aplicativo confere o conteúdo exato que será enviado: instruções, dados estruturados, textos, histórico de reenvio e formato de resposta.
  - Restou dado pessoal **já conhecido** (pessoa cadastrada ou detectada, CPF, RG, matrícula, telefone, e-mail, endereço)? A requisição é **bloqueada**, sem opção de ignorar.
  - Encontrou possível dado pessoal **não cadastrado**? A requisição só segue depois de uma decisão do usuário **item a item**, numa tela que lista os itens sem despejar o texto dos autos. Não existe confirmação genérica para ignorar tudo.
- A tabela de correspondência entre marcadores e valores reais existe somente na memória da tarefa. Ela **nunca** é enviada ao provedor, gravada em disco, registrada em logs ou mostrada em mensagens de erro.
- **Restauração.** As respostas são restauradas localmente de forma determinística. Marcador desconhecido ou alterado interrompe a etapa como resposta incompleta. O aplicativo nunca adivinha identidades.
- **Crianças.** Na confirmação de envio de cada tarefa com o ChatGPT, o aplicativo pergunta se os autos contêm dados pessoais de **criança menor de 13 anos**. Em caso afirmativo, ou se alguma pessoa do cadastro estiver marcada como criança, o ChatGPT fica **bloqueado** para a tarefa. Os Termos do *Sign in with ChatGPT* proíbem o envio desses dados, mesmo pseudonimizados. A análise só prossegue com outro provedor escolhido explicitamente pelo usuário, sem troca automática.
- **Limite.** Pseudonimização não é anonimização: o contexto (local, cargo, datas, circunstâncias) pode permitir reidentificação, e o aplicativo não promete anonimato. A confirmação do usuário não substitui autorização institucional para tratar dados da PMRO em serviço externo.

### 48.7. Provedor ChatGPT — requisições, modelos e falhas

- **Requisições:**
  - pela Responses API pública da OpenAI, com o registro de requisições desligado (`store` = falso) e resposta em fluxo, consumida internamente;
  - não há streaming de tokens na interface (seção 37, mantida);
  - o modelo não recebe ferramentas, como na 47.3;
  - não se enviam arquivos, imagens nem o PDF; só os blocos de texto de cada etapa.
- **Conclusão:** a resposta só é aceita com o evento de **conclusão** do provedor. Resposta interrompida, incompleta ou com falha **nunca** é tratada como concluída.
- **Modelos:** a lista vem da conta do usuário. Nenhum modelo é fixado no código. Se o modelo escolhido deixar de estar disponível, o aplicativo avisa e pede nova escolha.
- **Falhas:** sem conexão, conta não conectada, bloqueio por dados de criança (48.6), permissão do plano não concedida, sessão expirada ou revogada, conta não elegível, limite de uso do plano atingido, modelo indisponível, tempo esgotado ou recurso não suportado geram mensagem clara em português e **parada recuperável**, com a tarefa preservada (seção 47.6).
- **Limite de uso:** atingido o limite, o aplicativo **não repete** a requisição e indica onde o usuário acompanha o uso na própria conta ChatGPT.
- **Dados:** valem as ressalvas das seções 1.1 (item 9), 11 e 47.3. O aplicativo **não promete apagar** registros do provedor. Fica registrado que a política de retenção e de uso de dados para treinamento no modo *Sign in with ChatGPT* não estava documentada em 09/11/2026, e que o usuário deve conferir os termos do provedor.

## ESTADO FINAL — V4 + EMENDAS 1 E 2 (09/11/2026)

**INFERÊNCIA PELA GEMINI API (CHAVE DO USUÁRIO) OU PELO CHATGPT (PLANO DO USUÁRIO), À ESCOLHA EXPLÍCITA DO USUÁRIO E SEM TROCA AUTOMÁTICA. ENTRADA POR PEÇAS CADASTRADAS (UPLOAD OU SEI, SÓ OS SELECIONADOS). ENQUADRAMENTO DO PADS INFORMADO PELO USUÁRIO A PARTIR DE CATÁLOGO CURADO. PSEUDONIMIZAÇÃO COM BARREIRA OBRIGATÓRIA ANTES DO ENVIO E RESTAURAÇÃO LOCAL. TODAS AS DEMAIS REGRAS DA V4 PERMANECEM EM VIGOR.**
