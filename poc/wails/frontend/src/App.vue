<script setup lang="ts">
import {ref, onMounted} from 'vue'
import {Events} from "@wailsio/runtime";
import {GreetService} from "../bindings/sysexecutor/poc/wails";

const saudacao = ref('…');
const hora = ref('aguardando evento "time"…');

onMounted(async () => {
  saudacao.value = await GreetService.Greet("Fase 0 — ação, atenção, Rondônia");
  const off = Events.On("time", (ev: any) => {
    hora.value = ev.data;
    off();
    GreetService.Report(saudacao.value, String(ev.data), Math.round(performance.now()));
  });
});
</script>

<template>
  <main style="font-family: sans-serif; color: #eee; padding: 2rem">
    <h1>sys-executor — PoC F0-01</h1>
    <p>Binding Go: <strong>{{ saudacao }}</strong></p>
    <p>Evento Go → front: <strong>{{ hora }}</strong></p>
  </main>
</template>
