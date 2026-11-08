<script setup lang="ts">
// Obtenção dos autos pelo SEI/RO. Usuário, senha e código são digitados aqui
// (V4 §33), enviados uma vez e apagados do formulário em seguida.
import { computed, ref, watch } from "vue";
import { estado, informarCredenciais, informarSegredo, voltarParaEntrada } from "../estado";
import AvisoErro from "./AvisoErro.vue";

const v = computed(() => estado.visao!);
const pedido = computed(() => v.value.pedido ?? null);

const usuario = ref("");
const orgao = ref("PM");
const processo = ref("");
const senha = ref("");
const codigo = ref("");

watch(
  () => pedido.value?.id,
  () => {
    senha.value = "";
    codigo.value = "";
  },
);

async function entrar() {
  if (!pedido.value) return;
  const s = senha.value;
  senha.value = "";
  await informarCredenciais(pedido.value.id, usuario.value, s, orgao.value, processo.value);
}

async function enviarSegredo() {
  if (!pedido.value) return;
  const valor = pedido.value.tipo === "OTP" ? codigo.value : senha.value;
  codigo.value = "";
  senha.value = "";
  await informarSegredo(pedido.value.id, valor);
}
</script>

<template>
  <section class="sei" data-teste="painel-sei">
    <h2>Acessar o SEI/RO</h2>

    <AvisoErro v-if="pedido?.alerta" :erro="pedido.alerta" data-teste="alerta-sei" />

    <form v-if="pedido?.tipo === 'CREDENCIAIS'" class="form" autocomplete="off" data-teste="form-credenciais" @submit.prevent="entrar">
      <label>Usuário do SEI <input v-model="usuario" type="text" required autocomplete="off" spellcheck="false" /></label>
      <label>Senha <input v-model="senha" type="password" required autocomplete="off" data-teste="campo-senha" /></label>
      <label>Órgão <input v-model="orgao" type="text" required autocomplete="off" /></label>
      <label>Número do processo <input v-model="processo" type="text" required autocomplete="off" placeholder="0000.000000/0000-00" spellcheck="false" /></label>
      <p class="nota">A senha é usada só para este acesso e não é guardada pelo aplicativo.</p>
      <div class="acoes">
        <button class="botao primario" type="submit" :disabled="estado.ocupado">Entrar e obter os autos</button>
      </div>
    </form>

    <form v-else-if="pedido?.tipo === 'OTP'" class="form" autocomplete="off" data-teste="form-otp" @submit.prevent="enviarSegredo">
      <label>
        Código de 6 números do aplicativo autenticador
        <input v-model="codigo" type="text" inputmode="numeric" maxlength="8" required autocomplete="off" data-teste="campo-codigo" />
      </label>
      <div class="acoes">
        <button class="botao primario" type="submit" :disabled="estado.ocupado">Validar</button>
      </div>
    </form>

    <form v-else-if="pedido?.tipo === 'REAUTENTICACAO'" class="form" autocomplete="off" data-teste="form-reautenticacao" @submit.prevent="enviarSegredo">
      <p>O processo é sigiloso: o SEI/RO pede sua senha novamente para abri-lo.</p>
      <label>Senha <input v-model="senha" type="password" required autocomplete="off" /></label>
      <div class="acoes">
        <button class="botao primario" type="submit" :disabled="estado.ocupado">Confirmar</button>
      </div>
    </form>

    <div v-else class="andamento" role="status" aria-live="polite" data-teste="sei-andamento">
      <span class="ponto"></span><span class="ponto"></span><span class="ponto"></span>
      <p>{{ v.descricao }}</p>
    </div>

    <div>
      <button class="botao discreto" :disabled="estado.ocupado" data-teste="sei-voltar" @click="voltarParaEntrada">
        Parar e enviar o PDF em vez disso
      </button>
    </div>
  </section>
</template>

<style scoped>
.sei {
  display: grid;
  gap: 1rem;
  padding: 1.6rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  max-width: 640px;
}
.sei h2 { font-size: var(--t-lg); }
.form { display: grid; gap: 0.9rem; }
label { display: grid; gap: 0.3rem; font-weight: 500; }
input {
  font: inherit;
  font-weight: 400;
  padding: 0.55rem 0.7rem;
  border: 1px solid var(--linha-forte);
  border-radius: var(--raio);
  background: #fff;
}
.nota { color: var(--tinta-suave); font-size: var(--t-sm); }
.acoes { display: flex; gap: 0.7rem; }
.andamento { display: flex; align-items: center; gap: 0.4rem; }
.andamento p { margin-left: 0.6rem; }
.ponto { width: 8px; height: 8px; border-radius: 50%; background: var(--tinta-suave); animation: pulsa 1.2s infinite ease-in-out; }
.ponto:nth-child(2) { animation-delay: 0.2s; }
.ponto:nth-child(3) { animation-delay: 0.4s; }
@keyframes pulsa { 0%, 80%, 100% { opacity: 0.25; } 40% { opacity: 1; } }
</style>
