<script setup lang="ts">
import { TIPOS } from "../tipos";
import { computed } from "vue";
import { abrirConfigIA, escolherTipo, estado, iaPronta } from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const opcoes = [TIPOS.PADS, TIPOS.SR];
// Primeiro uso: o serviço escolhido em Configurações do assistente ainda não está
// pronto (conta conectada ou chave cadastrada, e modelo escolhido). Fora das
// Configurações, o aplicativo não nomeia o serviço (V4 §50).
const faltaIA = computed(() => !!estado.ia && !iaPronta());
</script>

<template>
  <main class="inicio">
    <section class="marca" aria-label="JURISPM" data-teste="marca">
      <span class="marca-icone" aria-hidden="true">
        <svg viewBox="0 0 48 48" width="40" height="40" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">
          <path d="M24 9v29M13 14h22M13 14l-5 12M13 14l5 12M35 14l-5 12M35 14l5 12" />
          <path d="M6.5 26h13c-.6 4-12.4 4-13 0zM28.5 26h13c-.6 4-12.4 4-13 0zM17 38h14" />
          <circle cx="24" cy="9" r="2" fill="currentColor" stroke="none" />
        </svg>
      </span>
      <h1 class="nome">JURISPM</h1>
      <p class="descricao">Assistente Inteligente de Análise Jurídico-Disciplinar</p>
      <p class="unidade">7º Batalhão de Polícia Militar · PMRO</p>
    </section>

    <div class="cabeca">
      <h2>Qual minuta você vai elaborar?</h2>
      <p>Depois desta escolha, você indicará como obter os autos.</p>
    </div>

    <div class="pastas">
      <button
        v-for="o in opcoes"
        :key="o.tipo"
        class="pasta"
        :style="{ '--cor': o.cor, '--cor-clara': o.corClara }"
        :disabled="estado.ocupado"
        :data-teste="'tipo-' + o.tipo"
        @click="escolherTipo(o.tipo)"
      >
        <span class="aba">{{ o.tipo }}</span>
        <span class="capa">
          <span class="titulo">{{ o.titulo }}</span>
          <span class="produz">{{ o.produz }}</span>
          <span class="peca">Peça central: <strong>{{ o.pecaCentral }}</strong></span>
        </span>
      </button>
    </div>

    <AvisoErro v-if="estado.erro" :erro="estado.erro" />

    <div v-if="faltaIA" class="primeiro-uso" role="status" data-teste="primeiro-uso">
      <h2>Configure o assistente</h2>
      <p>
        A leitura e a análise dos autos são feitas pelo assistente, que usa um serviço externo escolhido por você. Em
        Configurações do assistente, escolha o serviço, conecte a conta ou cadastre a chave e escolha o modelo; isso é feito uma vez
        só. O assistente exige internet.
      </p>
      <button class="botao primario" data-teste="configurar-ia" @click="abrirConfigIA">Configurar o assistente</button>
    </div>

    <p class="rodape">
      O aplicativo prepara minutas de apoio. A decisão e a revisão integral do texto são sempre suas.
      <button class="botao discreto" data-teste="link-config-ia" @click="abrirConfigIA">Configurações do assistente</button>
    </p>
  </main>
</template>

<style scoped>
.primeiro-uso { border: 2px solid var(--tinta); border-radius: var(--raio); background: var(--superficie); padding: 1.2rem 1.4rem; display: grid; gap: 0.7rem; justify-items: start; }
.primeiro-uso h2 { font-size: var(--t-lg); }
.primeiro-uso p { max-width: 68ch; color: var(--tinta-suave); }
.inicio {
  max-width: 980px;
  margin: 0 auto;
  padding: 1.8rem 2rem 2rem;
  display: grid;
  gap: 2.2rem;
}
.cabeca h2 { font-size: var(--t-2xl); }

/* Identidade do aplicativo (decisão do usuário, 12/11/2026). */
.marca {
  /* Sem fundo próprio: fica sobre o fundo da página (pedido do usuário). */
  color: var(--tinta);
  padding: 0.4rem 1.6rem 0;
  display: grid;
  justify-items: center;
  text-align: center;
  gap: 0.35rem;
}
.marca-icone {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  /* Tom suave de dourado, na família do ícone do aplicativo. */
  background: #f3ead2;
  border: 1.5px solid #dccb9c;
  border-radius: 14px;
  color: #8a6a2c;
  margin-bottom: 0.6rem;
}
.nome { font-size: 1.6rem; font-weight: 700; letter-spacing: 0.02em; color: var(--tinta); font-family: inherit; }
.descricao { color: var(--tinta); margin-top: 0.3rem; }
.unidade { font-size: var(--t-sm); color: var(--tinta-suave); }
.cabeca p { margin-top: 0.5rem; color: var(--tinta-suave); }

.pastas {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.8rem;
}

.pasta {
  text-align: left;
  border: 0;
  padding: 0;
  background: none;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}
.aba {
  background: var(--cor);
  color: #fff;
  font-weight: 600;
  font-size: var(--t-sm);
  letter-spacing: 0.04em;
  padding: 0.35rem 1.2rem 0.5rem;
  border-radius: 8px 8px 0 0;
  margin-bottom: -4px;
}
.capa {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
  width: 100%;
  min-height: 230px;
  padding: 1.6rem 1.6rem 1.4rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-top: 6px solid var(--cor);
  border-radius: var(--raio-pasta);
  transition: border-color 120ms ease, background 120ms ease;
}
.pasta:hover:not(:disabled) .capa,
.pasta:focus-visible .capa {
  background: var(--cor-clara);
  border-color: var(--cor);
}
.pasta:focus-visible { outline: none; }
.pasta:focus-visible .capa { outline: 3px solid var(--foco); outline-offset: 3px; }
.titulo {
  font-family: var(--serif);
  font-size: var(--t-xl);
  font-weight: 600;
  line-height: 1.2;
  color: var(--cor);
}
.produz { color: var(--tinta); max-width: 44ch; }
.peca {
  margin-top: auto;
  padding-top: 0.8rem;
  border-top: 1px dashed var(--linha-forte);
  font-size: var(--t-sm);
  color: var(--tinta-suave);
}
.peca strong { color: var(--tinta); font-weight: 600; }

.rodape { font-size: var(--t-sm); color: var(--tinta-suave); max-width: 70ch; }
</style>
