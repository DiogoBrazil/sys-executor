<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from "vue";
import { abrirModelos, cancelar, estado, retomar, revisar, voltarAoInicio } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";
import CartaoPergunta from "../components/CartaoPergunta.vue";
import PainelAnalise from "../components/PainelAnalise.vue";
import PainelAutos from "../components/PainelAutos.vue";
// O PDF.js só é carregado quando o usuário abre os autos.
const VisualizadorFontes = defineAsyncComponent(() => import("../components/VisualizadorFontes.vue"));

const v = computed(() => estado.visao!);
const confirmando = ref(false);
const fonte = ref<{ pagina: number; inicio: number; fim: number } | null>(null);
function abrir(pagina: number, inicio: number, fim: number) {
  fonte.value = { pagina, inicio, fim };
}

const erroDeModelo = computed(() => (v.value.erro?.codigo ?? "").startsWith("MOD-"));

const percentual = computed(() => {
  const p = v.value.progresso;
  return p && p.total > 0 ? Math.min(100, Math.round((p.atual / p.total) * 100)) : null;
});

const tituloFinal = computed(() => {
  switch (v.value.estado) {
    case "FINALIZADA":
      return "Tarefa finalizada";
    case "CANCELADA":
      return "Tarefa cancelada";
    default:
      return "A tarefa não pôde continuar";
  }
});

async function confirmarCancelamento() {
  confirmando.value = false;
  await cancelar();
}
</script>

<template>
  <main class="proc">
    <section class="painel">
      <h1 v-if="!v.terminal">Processando os autos</h1>
      <h1 v-else>{{ tituloFinal }}</h1>

      <CartaoPergunta v-if="v.pergunta" :pergunta="v.pergunta" @abrir="abrir" />

      <div v-if="v.parada && v.retomavel" class="retomar" data-teste="parada-recuperavel">
        <AvisoErro v-if="v.erro" :erro="v.erro" tipo="aviso" />
        <div class="acoes">
          <button class="botao primario" :disabled="estado.ocupado" data-teste="retomar" @click="retomar">Tentar novamente</button>
          <button v-if="erroDeModelo" class="botao" data-teste="abrir-modelos" @click="abrirModelos">Abrir modelos de IA</button>
        </div>
      </div>
      <div v-else-if="v.parada" class="aviso" role="status" data-teste="parada">
        <h3>As próximas etapas ainda não estão disponíveis</h3>
        <p>
          A consulta à biblioteca jurídica, o cálculo e a redação da minuta chegam nas próximas versões do aplicativo.
          Você pode conferir a análise abaixo, alterar respostas, consultar os autos e cancelar a tarefa quando terminar.
        </p>
      </div>

      <div v-else-if="!v.terminal && !v.pergunta" class="agora" aria-live="polite" data-teste="estado-atual">
        <p class="descricao">{{ v.descricao }}</p>
        <p v-if="v.atividade" class="sub" data-teste="atividade-texto">{{ v.atividade }}</p>
        <div v-if="percentual !== null" class="medida" data-teste="progresso">
          <div
            class="barra"
            role="progressbar"
            :aria-valuenow="percentual"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-label="v.descricao"
          >
            <span :style="{ width: percentual + '%' }"></span>
          </div>
          <p class="conta">{{ v.progresso!.atual }} de {{ v.progresso!.total }} {{ v.progresso!.unidade }}</p>
        </div>
        <div v-else class="atividade" role="status" data-teste="atividade">
          <span class="ponto"></span><span class="ponto"></span><span class="ponto"></span>
          <span class="visualmente-oculto">Em andamento</span>
        </div>
      </div>

      <AvisoErro v-if="!v.terminal && v.erro && !v.parada" :erro="v.erro" tipo="aviso" />
      <AvisoErro v-if="v.terminal && v.erro && v.estado === 'FALHA'" :erro="v.erro" />
      <p v-else-if="v.terminal && v.erro" class="nota">{{ v.erro.mensagem }}</p>
      <AvisoErro v-if="v.aviso" :erro="v.aviso" tipo="aviso" />
      <AvisoErro v-if="estado.erro" :erro="estado.erro" />

      <div class="acoes">
        <template v-if="!v.terminal">
          <button v-if="!confirmando" class="botao perigo" data-teste="cancelar" @click="confirmando = true">
            Cancelar tarefa
          </button>
          <template v-else>
            <p class="confirma">Os arquivos temporários desta tarefa serão descartados. Não há como retomar depois.</p>
            <button class="botao perigo confirmar" data-teste="confirmar-cancelar" @click="confirmarCancelamento">
              Cancelar e descartar
            </button>
            <button class="botao" @click="confirmando = false">Continuar a tarefa</button>
          </template>
        </template>
        <button v-else class="botao primario" data-teste="voltar-inicio" @click="voltarAoInicio">
          Voltar ao início
        </button>
      </div>
      <button v-if="v.autos && !v.terminal" class="botao" data-teste="consultar-autos" @click="abrir(1, 0, 0)">
        Consultar os autos
      </button>
      <PainelAnalise v-if="v.painel && !v.terminal" :painel="v.painel" @abrir="abrir" />
      <section v-if="v.respostas?.length && !v.terminal" class="respostas" aria-label="Respostas dadas" data-teste="respostas">
        <h2>Suas respostas</h2>
        <ul role="list">
          <li v-for="r in v.respostas" :key="r.perguntaId">
            <span class="t">{{ r.titulo }}</span>
            <span class="r">{{ r.resumo }}</span>
            <button
              v-if="r.revisavel"
              class="botao discreto"
              :disabled="estado.ocupado"
              :data-teste="'revisar-' + r.perguntaId"
              @click="revisar(r.perguntaId)"
            >
              Alterar
            </button>
          </li>
        </ul>
      </section>
      <PainelAutos v-if="v.autos && !v.terminal" :autos="v.autos" @abrir="abrir" />
    </section>

    <VisualizadorFontes v-if="fonte" :pagina="fonte.pagina" :inicio="fonte.inicio" :fim="fonte.fim" @fechar="fonte = null" />

    <ol class="linha" aria-label="Etapas da tarefa" data-teste="linha-do-tempo">
      <li v-for="(m, i) in v.etapas" :key="m.chave" :class="m.situacao" :aria-current="m.situacao === 'ativa' ? 'step' : undefined">
        <span class="marca" aria-hidden="true">{{ m.situacao === "concluida" ? "✓" : m.situacao === "falhou" ? "!" : i + 1 }}</span>
        <span class="rotulo">{{ m.rotulo }}</span>
        <span class="visualmente-oculto">({{ m.situacao }})</span>
      </li>
    </ol>
  </main>
</template>

<style scoped>
.proc {
  max-width: 1080px;
  margin: 0 auto;
  padding: 2.6rem 2rem;
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(260px, 1fr);
  gap: 2.4rem;
  align-items: start;
}
@media (max-width: 860px) {
  .proc { grid-template-columns: 1fr; }
}
.painel { display: grid; gap: 1.4rem; }
.painel h1 { font-size: var(--t-xl); }
.agora {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  padding: 1.4rem 1.5rem;
  display: grid;
  gap: 1rem;
}
.descricao { font-size: var(--t-lg); }
.sub { color: var(--tinta-suave); }
.retomar { display: grid; gap: 0.8rem; }
.respostas { background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); padding: 0.9rem 1.2rem; display: grid; gap: 0.5rem; }
.respostas h2 { font-size: var(--t-md); }
.respostas ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 0.35rem; }
.respostas li { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) auto; gap: 0.8rem; align-items: center; font-size: var(--t-sm); }
.respostas .r { color: var(--tinta-suave); }
.barra { height: 8px; background: var(--linha); border-radius: 4px; overflow: hidden; }
.barra span { display: block; height: 100%; background: var(--tinta); transition: width 300ms ease; }
.conta { margin-top: 0.4rem; font-size: var(--t-sm); color: var(--tinta-suave); }
.atividade { display: flex; gap: 0.4rem; height: 8px; align-items: center; }
.ponto { width: 8px; height: 8px; border-radius: 50%; background: var(--tinta-suave); animation: pulsa 1.2s infinite ease-in-out; }
.ponto:nth-child(2) { animation-delay: 0.2s; }
.ponto:nth-child(3) { animation-delay: 0.4s; }
@keyframes pulsa { 0%, 80%, 100% { opacity: 0.25; } 40% { opacity: 1; } }

.nota { color: var(--tinta-suave); }
.acoes { display: flex; flex-wrap: wrap; gap: 0.8rem; align-items: center; }
.confirma { flex-basis: 100%; color: var(--erro); }

.linha {
  list-style: none;
  margin: 0;
  padding: 1.2rem 1.4rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  display: grid;
  gap: 0;
}
.linha li {
  display: grid;
  grid-template-columns: 1.9rem 1fr;
  gap: 0.8rem;
  align-items: center;
  padding: 0.45rem 0;
  color: var(--tinta-suave);
  position: relative;
}
.linha li + li::before {
  content: "";
  position: absolute;
  left: calc(0.95rem - 1px);
  top: -0.45rem;
  height: 0.9rem;
  border-left: 2px solid var(--linha);
}
.marca {
  width: 1.9rem;
  height: 1.9rem;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: var(--t-xs);
  font-weight: 600;
  border: 2px solid var(--linha);
  background: var(--papel);
}
.linha li.concluida { color: var(--tinta); }
.linha li.concluida .marca { background: var(--tinta); border-color: var(--tinta); color: #fff; }
.linha li.ativa { color: var(--tinta); font-weight: 600; }
.linha li.ativa .marca { border-color: var(--tinta); background: var(--superficie); }
.linha li.falhou { color: var(--erro); font-weight: 600; }
.linha li.falhou .marca { border-color: var(--erro); background: var(--erro); color: #fff; }
</style>
