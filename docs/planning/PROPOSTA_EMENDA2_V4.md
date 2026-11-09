# PROPOSTA — Emenda 2 à V4 (seção 48)

> **Situação:** **APROVADA pelo usuário em 09/11/2026** ("Tudo certo") e inserida na V4 como seção 48. Antes da inserção, foram feitos três acréscimos decorrentes dos Termos do *Sign in with ChatGPT* fornecidos pelo usuário: aviso de privacidade antes do primeiro login (Termos §2.3), uso da marca (Termos §5) e bloqueio de dados de crianças menores de 13 anos (Termos §4). O usuário escolheu pergunta em cada tarefa + bloqueio.
>
> **Passos depois do aval (feitos em 09/11/2026, exceto o item 2):**
> 1. inserir a seção 48 e as marcas **[Emenda §48]** nas seções alcançadas, conservando o texto anterior, como foi feito na §47;
> 2. atualizar o catálogo de regras, a matriz e o plano de testes;
> 3. registrar no `policies/CHANGELOG.md`;
> 4. executar `go run ./cmd/rulescov -atualizar-lock` e `-estrito`.
>
> **Origem:** D-11 (`REGISTRO_DUVIDAS_DECISOES.md`) e `PLANO_CHATGPT_PECAS_PRIVACIDADE.md`.

---

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

---

## Impacto previsto no catálogo de regras (para conferência; aplicado só após o aval)

- **Reescritas (`versao+1`):**
  - TRV-SEG-012 (destinos permitidos por provedor);
  - TRV-IA-011 (um motor por provedor);
  - TRV-ING-001/002 (conferência da função declarada);
  - TRV-UI-001…004 (cadastro de peças, conferência da peça central);
  - TRV-PRV-005 (origem registrada);
  - PADS-ESC-001 (dispositivos pelo usuário);
  - TRV-SEG-011 (barreira e revisão).
- **Novas:** regras para credencial do ChatGPT, verificação da permissão do plano, conclusão por evento, autorização vinculada a provedor e modelo, barreira obrigatória, cadastro de peças, catálogo de dispositivos e divergência com o Memorando.
- **Revogadas:** as que só fazem sentido com o PDF monolítico (definidas na aplicação, sempre com `status: revogada` e sem reaproveitar IDs).
- **Golden dos prompts:** sem alteração prevista.
