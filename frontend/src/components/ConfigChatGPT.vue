<script setup lang="ts">
import { computed, ref } from "vue";
import {
  abrirUsoChatGPT,
  atualizarModelosChatGPT,
  cancelarEntradaChatGPT,
  desconectarChatGPT,
  entrarChatGPT,
  estado,
  selecionarModeloChatGPT,
  testarModeloChatGPT,
} from "../estado";

const m = computed(() => estado.ia);
const c = computed(() => estado.ia?.chatgpt ?? null);
// Aviso de privacidade antes da conexão (Termos do Sign in with ChatGPT,
// V4 §48.3): sem a confirmação, o botão de entrar fica desabilitado.
const leuAviso = ref(false);
const desconectando = ref(false);
const milhares = (n: number) => (n >= 1000 ? `${Math.round(n / 1000).toLocaleString("pt-BR")} mil` : n.toLocaleString("pt-BR"));
const num = (n: number) => n.toLocaleString("pt-BR");

async function desconectar() {
  desconectando.value = false;
  await desconectarChatGPT();
}
</script>

<template>
  <div v-if="m && c" class="painel" data-teste="painel-chatgpt">
    <p v-if="!c.disponivel" class="nota">O ChatGPT não está disponível nesta versão do aplicativo.</p>
    <template v-else>
      <section aria-label="Conta ChatGPT">
        <h2>Conta ChatGPT</h2>

        <template v-if="!c.conectada">
          <div class="privacidade" data-teste="aviso-privacidade">
            <h3>Antes de conectar a sua conta</h3>
            <ul>
              <li>
                O sys-executor usa o seu plano ChatGPT (Plus ou Pro) para analisar os autos. Contas sem um desses planos
                não podem ser usadas.
              </li>
              <li>
                Em cada tarefa, você confirma o envio antes. Vão ao ChatGPT só trechos de texto dos autos, com os nomes e
                outros dados pessoais reconhecidos trocados por marcadores. A tabela que liga os marcadores aos dados reais,
                os arquivos PDF e as credenciais do SEI não saem deste computador.
              </li>
              <li>
                Não envie autos com dados pessoais de criança menor de 13 anos: os termos de uso do plano em outros
                aplicativos proíbem. O aplicativo pergunta isso em cada tarefa e bloqueia o envio ao ChatGPT se houver.
              </li>
              <li>
                A conexão fica guardada neste computador, protegida pelo Windows para a sua conta; nenhum token é exibido.
                Você pode desconectar aqui a qualquer momento e também nas configurações do ChatGPT.
              </li>
              <li>
                No plano Plus, o limite de uso a cada 5 horas é compartilhado com os outros aplicativos conectados.
                Pseudonimização não é anonimização, e a confirmação de envio não substitui eventual autorização
                institucional para tratar os dados em serviço externo.
              </li>
            </ul>
            <label class="confirmar">
              <input v-model="leuAviso" type="checkbox" data-teste="li-aviso" />
              Li o aviso e quero conectar a minha conta
            </label>
          </div>
          <p v-if="c.reconectar" class="aviso alerta" role="alert" data-teste="reconectar">
            A conexão anterior expirou ou foi desfeita. Entre de novo para continuar usando o ChatGPT.
          </p>
          <div class="acoes">
            <button
              v-if="!c.entrando"
              class="botao primario"
              :disabled="!leuAviso || m.ocupado"
              data-teste="entrar-chatgpt"
              @click="entrarChatGPT"
            >
              Entrar com ChatGPT
            </button>
            <template v-else>
              <p class="nota" role="status" aria-live="polite" data-teste="entrando">
                Aguardando a autorização no navegador… Conclua a entrada na janela que abriu.
              </p>
              <button class="botao" data-teste="cancelar-entrada" @click="cancelarEntradaChatGPT">Cancelar</button>
            </template>
          </div>
        </template>

        <template v-else>
          <p class="situacao valida" data-teste="conta-conectada">Conectada: {{ c.email || "conta ChatGPT" }}</p>
          <p v-if="c.planoAutorizado" class="nota" data-teste="plano-autorizado">
            Uso do plano autorizado para o sys-executor.
          </p>
          <div v-else class="aviso alerta" role="alert" data-teste="plano-nao-autorizado">
            A conta foi conectada, mas o uso do plano não foi autorizado; sem essa permissão, nada é enviado. Entre de novo
            e autorize o uso do plano.
            <div class="acoes">
              <button class="botao" :disabled="c.entrando" data-teste="entrar-de-novo" @click="entrarChatGPT">Entrar de novo</button>
            </div>
          </div>
          <div class="acoes">
            <button class="botao discreto" data-teste="gerenciar-uso" @click="abrirUsoChatGPT">Gerenciar uso no ChatGPT</button>
            <button v-if="!desconectando" class="botao discreto" data-teste="desconectar" @click="desconectando = true">
              Desconectar
            </button>
            <template v-else>
              <button class="botao perigo" data-teste="confirmar-desconectar" @click="desconectar">
                Desconectar a conta deste computador
              </button>
              <button class="botao" @click="desconectando = false">Manter</button>
            </template>
          </div>
        </template>
        <p v-if="m.aviso" class="aviso alerta" role="status" data-teste="aviso-ia">{{ m.aviso }}</p>
      </section>

      <section v-if="c.conectada && c.planoAutorizado" aria-label="Modelo do ChatGPT">
        <h2>Modelo</h2>
        <div v-if="c.modeloForaDaLista" class="aviso alerta" role="alert" data-teste="chatgpt-modelo-fora">
          O modelo escolhido ({{ c.modeloSelecionado }}) não aparece mais para a sua conta. Escolha outro.
        </div>
        <p v-if="!c.modelos?.length" class="nota" data-teste="chatgpt-sem-lista">
          Use “Atualizar lista” para ver os modelos disponíveis no seu plano.
          <template v-if="c.modeloSelecionado"> Modelo escolhido: {{ c.modeloSelecionado }}.</template>
        </p>
        <div class="acoes">
          <button class="botao" :disabled="m.ocupado" data-teste="atualizar-modelos-chatgpt" @click="atualizarModelosChatGPT">
            Atualizar lista
          </button>
        </div>
        <article
          v-for="x in c.modelos"
          :key="x.id"
          class="modelo"
          :class="{ uso: x.id === c.modeloSelecionado }"
          :data-teste="'chatgpt-modelo-' + x.id"
        >
          <div class="cabeca">
            <h3>{{ x.nome }}</h3>
            <span v-if="x.id === c.modeloSelecionado" class="selo ok">Em uso</span>
            <span v-if="x.teste === 'ok'" class="selo ok" :data-teste="'chatgpt-teste-ok-' + x.id">Testado: respondeu</span>
            <span v-if="x.teste === 'falhou'" class="selo alerta" :data-teste="'chatgpt-teste-falhou-' + x.id">Teste falhou</span>
          </div>
          <p class="meta">
            {{ x.id }}<template v-if="x.contexto"> · contexto de {{ milhares(x.contexto) }} tokens</template
            ><template v-if="x.raciocinio?.length"> · raciocínio: {{ x.raciocinio.join(", ") }}</template>
          </p>
          <p v-if="x.testeErro" class="nota">{{ x.testeErro.titulo }}: {{ x.testeErro.mensagem }}</p>
          <p v-if="x.testeErro?.detalhe" class="nota diagnostico">Diagnóstico ({{ x.testeErro.codigo }}): {{ x.testeErro.detalhe }}</p>
          <div class="acoes">
            <button
              v-if="x.id !== c.modeloSelecionado"
              class="botao"
              :disabled="m.ocupado"
              :data-teste="'chatgpt-usar-' + x.id"
              @click="selecionarModeloChatGPT(x.id)"
            >
              Usar este modelo
            </button>
            <button class="botao discreto" :disabled="m.ocupado" :data-teste="'chatgpt-testar-' + x.id" @click="testarModeloChatGPT(x.id)">
              Testar
            </button>
          </div>
        </article>
        <p class="nota">O teste envia só uma frase curta, sem dados dos autos, e conta no limite de uso do seu plano.</p>
      </section>

      <section v-if="c.uso" aria-label="Uso da última tarefa com o ChatGPT" data-teste="uso-chatgpt">
        <h2>Uso da última tarefa</h2>
        <dl>
          <div><dt>Modelo</dt><dd>{{ c.uso.modelo }}</dd></div>
          <div><dt>Chamadas</dt><dd>{{ num(c.uso.chamadas) }}<template v-if="c.uso.falhas"> ({{ num(c.uso.falhas) }} com falha)</template></dd></div>
          <div><dt>Tokens enviados</dt><dd>{{ num(c.uso.tokensEntrada) }}</dd></div>
          <div><dt>Tokens recebidos</dt><dd>{{ num(c.uso.tokensSaida) }} (raciocínio: {{ num(c.uso.tokensPensamento) }})</dd></div>
          <div><dt>Tempo de resposta</dt><dd>{{ num(c.uso.segundos) }} s</dd></div>
        </dl>
        <p class="nota">Valores informados pela própria API. O consumo do plano pode ser acompanhado em “Gerenciar uso no ChatGPT”.</p>
      </section>
    </template>
  </div>
</template>

<style scoped>
.painel { display: grid; gap: 1.4rem; }
section { background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); padding: 1.1rem 1.3rem; display: grid; gap: 0.7rem; }
h2 { font-size: var(--t-lg); }
h3 { font-size: var(--t-md); font-family: var(--sans); }
.privacidade { border: 1px solid var(--linha-forte); border-radius: var(--raio); padding: 0.9rem 1.1rem; display: grid; gap: 0.6rem; background: var(--papel); }
.privacidade ul { margin: 0; padding-left: 1.2rem; display: grid; gap: 0.4rem; max-width: 72ch; }
.confirmar { display: flex; gap: 0.5rem; align-items: center; font-weight: 600; }
.situacao { font-weight: 600; }
.situacao.valida { color: var(--ok); }
dl { margin: 0; display: grid; gap: 0.3rem; }
dl div { display: grid; grid-template-columns: 11rem 1fr; gap: 0.8rem; }
dt { color: var(--tinta-suave); }
dd { margin: 0; }
.nota, .meta { font-size: var(--t-sm); color: var(--tinta-suave); }
.diagnostico { word-break: break-word; max-height: 9rem; overflow: auto; }
.modelo { border-top: 1px solid var(--linha); padding-top: 0.8rem; display: grid; gap: 0.35rem; }
.modelo.uso h3::before { content: ""; display: inline-block; width: 0.5rem; height: 0.5rem; border-radius: 50%; background: var(--ok); margin-right: 0.5rem; vertical-align: middle; }
.cabeca { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; }
.selo { font-size: var(--t-xs); border: 1px solid var(--linha-forte); border-radius: 999px; padding: 0.05rem 0.55rem; }
.selo.ok { border-color: var(--ok); color: var(--ok); }
.selo.alerta { border-color: var(--alerta); color: var(--alerta); }
.acoes { display: flex; flex-wrap: wrap; gap: 0.6rem; align-items: center; }
.aviso.alerta { border: 1px solid var(--alerta); background: var(--alerta-claro); color: var(--alerta); border-radius: var(--raio); padding: 0.7rem 0.9rem; display: grid; gap: 0.5rem; }
</style>
