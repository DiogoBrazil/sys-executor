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

onMounted(carregar);
</script>

<template>
  <FaixaTipo v-if="tela !== 'inicio' && tipo && !estado.verConfigIA" :tipo="tipo" />
  <ConfiguracoesIA v-if="estado.verConfigIA" />
  <Inicio v-else-if="tela === 'inicio'" />
  <Entrada v-else-if="tela === 'entrada'" />
  <Processamento v-else />
</template>
