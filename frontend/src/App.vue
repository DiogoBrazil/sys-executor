<script setup lang="ts">
import { computed, onMounted } from "vue";
import { carregar, estado, telaAtual } from "./estado";
import FaixaTipo from "./components/FaixaTipo.vue";
import Inicio from "./views/Inicio.vue";
import Entrada from "./views/Entrada.vue";
import Processamento from "./views/Processamento.vue";
import ConfiguracoesIA from "./views/ConfiguracoesIA.vue";

const tela = computed(() => telaAtual(estado.visao));
const tipo = computed(() => estado.visao?.tipo ?? "");
// A cor do procedimento vale para o botão principal, a etapa ativa e a aba
// do cartão da pergunta. Fora de uma tarefa (início, configurações), a tinta.
const acento = computed(() => {
  if (tela.value === "inicio" || estado.verConfigIA) return {};
  if (tipo.value === "PADS") return { "--acento": "var(--pads)", "--acento-claro": "var(--pads-claro)" };
  if (tipo.value === "SR") return { "--acento": "var(--sr)", "--acento-claro": "var(--sr-claro)" };
  return {};
});

onMounted(carregar);
</script>

<template>
  <div class="app" :style="acento">
  <FaixaTipo v-if="tela !== 'inicio' && tipo && !estado.verConfigIA" :tipo="tipo" />
  <ConfiguracoesIA v-if="estado.verConfigIA" />
  <Inicio v-else-if="tela === 'inicio'" />
  <Entrada v-else-if="tela === 'entrada'" />
  <Processamento v-else />
  </div>
</template>

<style scoped>
.app { min-height: 100vh; }
</style>
