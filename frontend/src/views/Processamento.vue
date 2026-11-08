<script setup lang="ts">
import { computed, ref } from "vue";
import { cancelar, estado, voltarAoInicio } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const v = computed(() => estado.visao!);
const confirmando = ref(false);

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

      <div v-if="!v.terminal" class="agora" aria-live="polite" data-teste="estado-atual">
        <p class="descricao">{{ v.descricao }}</p>
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
    </section>

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
