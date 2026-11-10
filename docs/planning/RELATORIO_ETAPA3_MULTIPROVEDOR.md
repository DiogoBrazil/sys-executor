# RELATÓRIO — Etapa 3 do plano D-11: generalização para mais de um provedor

- **Data:** 09/11/2026 · **Branch:** `add-mode-apikey-oauth`
- **Autorização:** usuário, 09/11/2026 ("Sim, seguir assim").
- **Regras:** V4 §48.1; catálogo TRV-SEG-010 v2 e nova TRV-IA-014; erros IA-015 a IA-020.

## 1. O que mudou

| Item | Antes | Agora |
|---|---|---|
| Porta `pipeline.FonteMotor` | `Modelo() string`; `Abrir(ctx, …)` usa o modelo das preferências no momento | `Alvo() inference.Alvo` (provedor, rótulo, modelo); `Abrir(ctx, alvo, …)` abre **exatamente o alvo autorizado** |
| Autorização de envio | `envioAutorizado bool`: sobrevivia à troca de modelo (o consentimento dado ao Google valeria para outro alvo) | `autorizado *inference.Alvo`. Delimitação e análise conferem o alvo atual; se o provedor ou o modelo mudou, o motor anterior é fechado e a pergunta é refeita antes de qualquer envio |
| Confirmação de envio | "Autorizar o envio à Gemini API" (texto fixo) | "Autorizar o envio à IA", com o serviço e o modelo do alvo; o resumo registra o serviço e o modelo |
| Escolha do provedor | só existia o Gemini, ligado direto em `main.go` | `app.FonteIA` despacha pelo `provedor_ia` das preferências. **Sem troca automática:** ChatGPT escolhido e não conectado → IA-015; nenhum serviço → IA-004 |
| Preferências | v2 (`modelo_gemini`) | **v3**: `provedor_ia` (`""`/`GEMINI`/`CHATGPT`) e `modelo_chatgpt` (slug validado); `migrarV2` explícita, em que modelo Gemini escolhido vira provedor `GEMINI`; v1 → v3. Escolher um modelo Gemini grava também o provedor |
| Credencial | arquivo e entropia fixos (`gemini.dpapi`) | `credencial.NovaLojaNomeada(dir, nome, entropia)`. `NovaLojaArquivo` continua igual para o Gemini. Lojas diferentes não se decifram: o teste roda no CI `go-windows` |
| Erros | IA-001 a IA-013 com texto do Gemini | IA-003 a IA-011 neutros. IA-001, 002, 012 e 013 continuam do Gemini, porque são da chave. Novos, para a etapa 4: **IA-015** conta ChatGPT não conectada; **IA-016** entrada não concluída (inclui conta sem plano elegível, conforme a PoC §2-A); **IA-017** uso do plano não autorizado; **IA-018** conexão encerrada; **IA-019** limite do plano; **IA-020** conta não elegível |

**Não mudou:** regras jurídicas, prompts (golden sem diff), barreira de privacidade, `MotorGemini`, tela de configurações (abas na etapa 4).

## 2. Verificação

| Verificação | Resultado |
|---|---|
| `go test -race` (internal, bench, rulescov, licencas) | Verde |
| `GOOS=windows go vet ./...` e build | Verde |
| `rulescov -estrito` | 195 regras; 124/124 IDs da V4 |
| Golden | Sem diff |
| Vitest (23), `vue-tsc` | Verde (bindings regenerados: modelo das preferências) |
| Testes novos | `TestAutorizacaoVinculadaAoAlvo` (SEG-ENV-02): troca de modelo depois da autorização → nada vai ao novo alvo, motor anterior fechado, nova confirmação com o novo modelo, motor novo aberto para o alvo autorizado; `TestFonteIAEscolhaExplicitaSemTrocaAutomatica` (IA-PROV-01); `TestMigraVersao2`; `TestProvedorInvalidoVoltaAoPadrao`; `TestLojasNomeadasIsoladas`; seleção do modelo grava o provedor |

## 3. Observação
- **Versão do mapa de pseudônimos fora do vínculo.** O plano previa prender a autorização também à "versão do mapa"; isso não foi feito, de propósito. O mapa só cresce (mais dados ocultados), e cada inclusão já passou pela decisão do usuário na revisão. Prender a autorização a ele faria a confirmação de envio voltar a cada item revisado, sem ganho de proteção.

## 4. Próxima etapa
**Etapa 4: provedor ChatGPT.** Inclui:
- `internal/chatgpt` (login, tokens, ID token, DPAPI nomeada);
- `inference/openai*.go` (Responses + SSE);
- `FonteChatGPT`, abas nas Configurações de IA e aviso de privacidade antes do 1º login;
- bloqueio de dados de criança;
- classificação dos erros vista na PoC.

Depende de autorização expressa.
