<script setup lang="ts">
import type { ErroTela } from "../estado";

defineProps<{ erro: ErroTela; tipo?: "erro" | "aviso" }>();
</script>

<template>
  <div class="aviso" :class="{ erro: tipo !== 'aviso' }" role="alert" data-teste="aviso-erro">
    <h3>{{ erro.titulo }}</h3>
    <p>{{ erro.mensagem }}</p>
    <p class="acao">{{ erro.acao }}</p>
  </div>
</template>
