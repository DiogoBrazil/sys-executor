<script setup lang="ts">
import { computed } from "vue";
import { TIPOS } from "../tipos";

const props = defineProps<{ tipo: string }>();
const info = computed(() => TIPOS[props.tipo as "PADS" | "SR"]);
</script>

<template>
  <header v-if="info" class="faixa" :style="{ '--cor': info.cor }" data-teste="faixa-tipo">
    <span class="aba" aria-hidden="true"></span>
    <h2>{{ info.titulo }}</h2>
    <p>Peça central: {{ info.pecaCentral }}</p>
  </header>
</template>

<style scoped>
.faixa {
  display: flex;
  align-items: baseline;
  gap: 1rem;
  padding: 0.9rem 2rem 0.9rem 1.4rem;
  background: var(--cor);
  color: #fff;
  position: relative;
}
.aba {
  align-self: stretch;
  width: 6px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.55);
}
h2 { font-size: var(--t-lg); }
p { font-size: var(--t-sm); opacity: 0.85; }
</style>
