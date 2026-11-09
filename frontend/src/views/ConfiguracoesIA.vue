<script setup lang="ts">
import { computed, ref } from "vue";
import {
  abrirPaginaChaves,
  estado,
  fecharConfigIA,
  removerChave,
  salvarChave,
  selecionarModelo,
  testarModelo,
  verificarIA,
} from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const m = computed(() => estado.ia);
// A chave digitada vive só neste campo até ser enviada ao Go; depois é
// apagada e nunca mais exibida (V4 §47.4).
const chave = ref("");
const removendo = ref(false);

const situacoes: Record<string, string> = {
  sem_chave: "Nenhuma chave cadastrada",
  nao_verificada: "Chave cadastrada (ainda não verificada nesta sessão)",
  valida: "Chave cadastrada e aceita pelo Google",
  invalida: "Chave cadastrada, mas recusada pelo Google",
  sem_conexao: "Chave cadastrada; sem conexão para verificar",
};
const estagios: Record<string, string> = {
  estavel: "Estável",
  previa: "Prévia",
  experimental: "Experimental",
  apelido: "Apelido (muda de versão sem aviso)",
};
const milhares = (n: number) => (n >= 1000 ? `${Math.round(n / 1000).toLocaleString("pt-BR")} mil` : n.toLocaleString("pt-BR"));
const num = (n: number) => n.toLocaleString("pt-BR");

async function salvar() {
  const c = chave.value;
  chave.value = "";
  await salvarChave(c);
}
async function remover() {
  removendo.value = false;
  await removerChave();
}
</script>

<template>
  <main class="config" data-teste="tela-config-ia">
    <div class="topo">
      <h1>Configurações de IA</h1>
      <button class="botao" data-teste="fechar-config-ia" @click="fecharConfigIA">Voltar</button>
    </div>
    <p class="intro">
      A leitura e a análise dos autos usam a Gemini API, do Google, com a sua chave. A análise por IA exige internet.
      Em cada tarefa você confirma o envio antes; nomes e outros dados pessoais reconhecidos são trocados por marcadores,
      e o arquivo PDF não é enviado.
    </p>

    <p v-if="!m" class="nota">Carregando…</p>
    <template v-else>
      <AvisoErro v-if="m.erro" :erro="m.erro" />
      <p v-if="m.ocupado" class="nota" role="status" aria-live="polite" data-teste="ia-ocupado">Consultando o Google…</p>

      <section aria-label="Chave da Gemini API">
        <h2>Chave da Gemini API</h2>
        <p class="situacao" :class="m.status" data-teste="situacao-chave">{{ situacoes[m.status] ?? m.status }}</p>
        <form class="chave" @submit.prevent="salvar">
          <label for="campo-chave">{{ m.chaveCadastrada ? "Substituir por outra chave" : "Chave de API" }}</label>
          <div class="linha">
            <input
              id="campo-chave"
              v-model="chave"
              type="password"
              autocomplete="off"
              spellcheck="false"
              autocapitalize="off"
              data-teste="campo-chave"
              placeholder="Cole aqui a chave criada no Google AI Studio"
            />
            <button class="botao primario" type="submit" :disabled="!chave.trim() || m.ocupado" data-teste="salvar-chave">
              Salvar e validar
            </button>
          </div>
        </form>
        <p class="nota">
          A chave é conferida no Google antes de ser guardada. Fica neste computador, protegida pelo Windows para a sua
          conta, e não é exibida de novo.
        </p>
        <div class="acoes">
          <button class="botao discreto" data-teste="obter-chave" @click="abrirPaginaChaves">
            Obter uma chave no Google AI Studio
          </button>
          <template v-if="m.chaveCadastrada">
            <button class="botao" :disabled="m.ocupado" data-teste="testar-conexao" @click="verificarIA">Testar conexão</button>
            <button v-if="!removendo" class="botao discreto" data-teste="remover-chave" @click="removendo = true">Remover chave</button>
            <template v-else>
              <button class="botao perigo" data-teste="confirmar-remover-chave" @click="remover">Remover a chave deste computador</button>
              <button class="botao" @click="removendo = false">Manter</button>
            </template>
          </template>
        </div>
        <p class="nota endereco">{{ m.urlChaves }}</p>
      </section>

      <section v-if="m.chaveCadastrada" aria-label="Modelo">
        <h2>Modelo</h2>
        <div v-if="m.modeloForaDaLista" class="aviso alerta" role="alert" data-teste="modelo-fora">
          O modelo escolhido ({{ m.modeloSelecionado }}) não aparece mais para a sua chave. Escolha outro.
        </div>
        <p v-if="!m.modelos?.length" class="nota" data-teste="sem-lista">
          Use “Testar conexão” para listar os modelos disponíveis para a sua chave.
          <template v-if="m.modeloSelecionado"> Modelo escolhido: {{ m.modeloSelecionado }}.</template>
        </p>
        <article
          v-for="x in m.modelos"
          :key="x.id"
          class="modelo"
          :class="{ uso: x.id === m.modeloSelecionado }"
          :data-teste="'modelo-' + x.id"
        >
          <div class="cabeca">
            <h3>{{ x.nome }}</h3>
            <span class="selo" :class="{ alerta: x.estagio !== 'estavel' }">{{ estagios[x.estagio] ?? x.estagio }}</span>
            <span v-if="x.id === m.modeloSelecionado" class="selo ok">Em uso</span>
            <span v-if="x.teste === 'ok'" class="selo ok" :data-teste="'teste-ok-' + x.id">Testado: respondeu</span>
            <span v-if="x.teste === 'falhou'" class="selo alerta" :data-teste="'teste-falhou-' + x.id">Teste falhou</span>
          </div>
          <p class="meta">
            {{ x.id }} · entrada até {{ milhares(x.entradaMax) }} tokens · saída até {{ milhares(x.saidaMax) }} tokens{{
              x.pensamento ? " · raciocina antes de responder" : ""
            }}
          </p>
          <p v-if="x.testeErro" class="nota">{{ x.testeErro.titulo }}: {{ x.testeErro.mensagem }}</p>
          <div class="acoes">
            <button
              v-if="x.id !== m.modeloSelecionado"
              class="botao"
              :disabled="m.ocupado"
              :data-teste="'usar-' + x.id"
              @click="selecionarModelo(x.id)"
            >
              Usar este modelo
            </button>
            <button class="botao discreto" :disabled="m.ocupado" :data-teste="'testar-' + x.id" @click="testarModelo(x.id)">
              Testar
            </button>
          </div>
        </article>
        <p class="nota">
          Nem todo modelo está liberado em todos os níveis de conta, e os limites de uso variam. O teste envia só uma
          frase curta, sem dados dos autos.
        </p>
      </section>

      <section v-if="m.uso" aria-label="Uso da última tarefa" data-teste="uso-ia">
        <h2>Uso da última tarefa</h2>
        <dl>
          <div><dt>Modelo</dt><dd>{{ m.uso.modelo }}</dd></div>
          <div><dt>Chamadas</dt><dd>{{ num(m.uso.chamadas) }}<template v-if="m.uso.falhas"> ({{ num(m.uso.falhas) }} com falha)</template></dd></div>
          <div><dt>Tokens enviados</dt><dd>{{ num(m.uso.tokensEntrada) }}</dd></div>
          <div><dt>Tokens recebidos</dt><dd>{{ num(m.uso.tokensSaida) }} (raciocínio: {{ num(m.uso.tokensPensamento) }})</dd></div>
          <div><dt>Tempo de resposta</dt><dd>{{ num(m.uso.segundos) }} s</dd></div>
        </dl>
        <p class="nota">Valores informados pela própria API. O aplicativo não calcula custo; consulte os preços no Google AI Studio.</p>
      </section>
    </template>
  </main>
</template>

<style scoped>
.config { max-width: 920px; margin: 0 auto; padding: 2.6rem 2rem; display: grid; gap: 1.4rem; }
.topo { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
.topo h1 { font-size: var(--t-xl); }
.intro { max-width: 68ch; color: var(--tinta-suave); }
section { background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); padding: 1.1rem 1.3rem; display: grid; gap: 0.7rem; }
h2 { font-size: var(--t-lg); }
.situacao { font-weight: 600; }
.situacao.valida { color: var(--ok); }
.situacao.invalida, .situacao.sem_conexao { color: var(--alerta); }
.chave { display: grid; gap: 0.35rem; }
.chave .linha { display: flex; flex-wrap: wrap; gap: 0.6rem; }
.chave input { flex: 1 1 22rem; min-width: 0; font: inherit; padding: 0.5rem 0.7rem; border: 1px solid var(--linha-forte); border-radius: var(--raio); background: #fff; }
dl { margin: 0; display: grid; gap: 0.3rem; }
dl div { display: grid; grid-template-columns: 11rem 1fr; gap: 0.8rem; }
dt { color: var(--tinta-suave); }
dd { margin: 0; }
.nota, .meta { font-size: var(--t-sm); color: var(--tinta-suave); }
.endereco { font-family: ui-monospace, Consolas, monospace; }
.modelo { border-top: 1px solid var(--linha); padding-top: 0.8rem; display: grid; gap: 0.35rem; }
.modelo.uso h3::before { content: ""; display: inline-block; width: 0.5rem; height: 0.5rem; border-radius: 50%; background: var(--ok); margin-right: 0.5rem; vertical-align: middle; }
.cabeca { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; }
.cabeca h3 { font-size: var(--t-md); font-family: var(--sans); }
.selo { font-size: var(--t-xs); border: 1px solid var(--linha-forte); border-radius: 999px; padding: 0.05rem 0.55rem; }
.selo.ok { border-color: var(--ok); color: var(--ok); }
.selo.alerta { border-color: var(--alerta); color: var(--alerta); }
.acoes { display: flex; flex-wrap: wrap; gap: 0.6rem; align-items: center; }
.aviso.alerta { border: 1px solid var(--alerta); background: var(--alerta-claro); color: var(--alerta); border-radius: var(--raio); padding: 0.7rem 0.9rem; }
</style>
