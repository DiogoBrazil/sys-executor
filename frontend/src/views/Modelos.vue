<script setup lang="ts">
import { computed, ref } from "vue";
import {
  baixarModelo,
  cancelarAndamento,
  estado,
  fecharModelos,
  importarModelo,
  removerModelo,
  selecionarModelo,
  testarGPU,
  usarSoCPU,
} from "../estado";
import AvisoErro from "../components/AvisoErro.vue";

const m = computed(() => estado.modelos);
const gb = (b: number) => (b / 1024 ** 3).toLocaleString("pt-BR", { maximumFractionDigits: 1 });
const confirmarGPU = ref(false);
const removendo = ref("");

const perfis: Record<string, string> = {
  padrao: "Padrão",
  basico: "Básico (computadores com 8 GB)",
  embeddings: "Biblioteca jurídica (busca)",
  importado: "Importado",
};
const estadosGPU: Record<string, string> = {
  desconhecida: "não testada — o aplicativo usa o processador",
  estavel: "testada e em uso",
  instavel: "instável — o aplicativo usa o processador",
  desativada: "desativada — o aplicativo usa o processador",
};

const percentual = computed(() => {
  const a = m.value?.andamento;
  return a && a.total > 0 ? Math.round((a.feito / a.total) * 100) : null;
});
const geradores = computed(() => (m.value?.modelos ?? []).filter((x) => x.papel === "gerador"));
const auxiliares = computed(() => (m.value?.modelos ?? []).filter((x) => x.papel !== "gerador"));

async function testar() {
  confirmarGPU.value = false;
  await testarGPU();
}
async function remover(id: string) {
  removendo.value = "";
  await removerModelo(id);
}
</script>

<template>
  <main class="modelos" data-teste="tela-modelos">
    <div class="topo">
      <h1>Modelos de IA local</h1>
      <button class="botao" data-teste="fechar-modelos" @click="fecharModelos">Voltar</button>
    </div>
    <p class="intro">
      A análise dos autos roda neste computador, sem enviar nada à internet. Escolha um modelo da lista recomendada
      ou importe um arquivo GGUF que você já tenha.
    </p>

    <p v-if="!m" class="nota">Carregando…</p>
    <template v-else>
      <AvisoErro v-if="m.erro" :erro="m.erro" />
      <div v-if="!m.servidor" class="aviso erro" role="alert" data-teste="sem-servidor">
        <h3>Componente de IA não encontrado</h3>
        <p>O mecanismo de IA local não está na pasta do aplicativo. Reinstale o aplicativo.</p>
      </div>

      <section class="maquina" aria-label="Este computador">
        <h2>Este computador</h2>
        <dl>
          <div><dt>Memória</dt><dd>{{ gb(m.hardware.ramTotal) }} GB ({{ gb(m.hardware.ramLivre) }} GB livres agora)</dd></div>
          <div><dt>Processador</dt><dd>{{ m.hardware.nucleosFisicos }} núcleos físicos{{ m.hardware.avx2 ? ", AVX2" : "" }}</dd></div>
          <div><dt>Placa de vídeo</dt><dd>{{ estadosGPU[m.gpu] ?? m.gpu }}</dd></div>
        </dl>
        <ul v-if="m.avisos?.length" class="avisos" role="list">
          <li v-for="(a, i) in m.avisos" :key="i">{{ a }}</li>
        </ul>
      </section>

      <section v-if="m.andamento" class="andamento" aria-live="polite" data-teste="andamento">
        <p>{{ m.andamento.acao === "baixando" ? "Baixando" : "Importando" }} o modelo…</p>
        <div class="barra" role="progressbar" :aria-valuenow="percentual ?? undefined" aria-valuemin="0" aria-valuemax="100">
          <span :style="{ width: (percentual ?? 0) + '%' }"></span>
        </div>
        <p class="conta">{{ gb(m.andamento.feito) }} de {{ gb(m.andamento.total) }} GB</p>
        <button class="botao" data-teste="cancelar-andamento" @click="cancelarAndamento">Interromper</button>
        <p class="nota">Um download interrompido continua de onde parou.</p>
      </section>

      <section class="lista" aria-label="Modelos para analisar os autos">
        <h2>Modelos para analisar os autos</h2>
        <article v-for="x in geradores" :key="x.id" class="modelo" :class="{ rec: x.recomendado }" :data-teste="'modelo-' + x.id">
          <div class="cabeca">
            <h3>{{ x.nome }}</h3>
            <span v-if="x.recomendado" class="selo">Recomendado para este computador</span>
            <span v-if="x.instalado" class="selo ok">Instalado</span>
            <span v-if="!x.cabe" class="selo alerta">Não cabe na memória livre agora</span>
          </div>
          <p class="meta">{{ perfis[x.perfil] ?? x.perfil }} · {{ gb(x.tamanho) }} GB · licença {{ x.licenca || "não informada" }}</p>
          <p v-if="x.nota" class="nota">{{ x.nota }}</p>
          <div class="acoes">
            <button
              v-if="!x.instalado && x.testado"
              class="botao"
              :class="{ primario: x.recomendado }"
              :disabled="!!m.andamento"
              :data-teste="'baixar-' + x.id"
              @click="baixarModelo(x.id)"
            >
              Baixar ({{ gb(x.tamanho) }} GB)
            </button>
            <template v-if="x.instalado">
              <button v-if="m.manual !== x.id" class="botao discreto" :data-teste="'usar-' + x.id" @click="selecionarModelo(x.id)">
                Usar sempre este
              </button>
              <button v-else class="botao discreto" data-teste="automatico" @click="selecionarModelo('')">
                Em uso por escolha sua — voltar ao automático
              </button>
              <button v-if="removendo !== x.id" class="botao discreto" :data-teste="'remover-' + x.id" @click="removendo = x.id">Remover</button>
              <template v-else>
                <button class="botao perigo" :data-teste="'confirmar-remover-' + x.id" @click="remover(x.id)">Remover o arquivo</button>
                <button class="botao" @click="removendo = ''">Manter</button>
              </template>
            </template>
          </div>
        </article>
        <div class="acoes">
          <button class="botao" :disabled="!!m.andamento" data-teste="importar" @click="importarModelo">Importar arquivo GGUF…</button>
        </div>
        <p class="nota">Modelos importados ficam disponíveis só por escolha manual, porque não passaram pelos testes de qualidade do aplicativo.</p>
      </section>

      <section v-if="auxiliares.length" class="lista" aria-label="Componentes auxiliares">
        <h2>Componentes auxiliares</h2>
        <article v-for="x in auxiliares" :key="x.id" class="modelo" :data-teste="'modelo-' + x.id">
          <div class="cabeca">
            <h3>{{ x.nome }}</h3>
            <span v-if="x.instalado" class="selo ok">Instalado</span>
          </div>
          <p class="meta">{{ perfis[x.perfil] ?? x.perfil }} · {{ gb(x.tamanho) }} GB</p>
          <p v-if="x.nota" class="nota">{{ x.nota }}</p>
          <div class="acoes">
            <button v-if="!x.instalado" class="botao" :disabled="!!m.andamento" @click="baixarModelo(x.id)">Baixar</button>
            <button v-else class="botao discreto" @click="removerModelo(x.id)">Remover</button>
          </div>
        </article>
      </section>

      <details class="avancado">
        <summary>Avançado: placa de vídeo</summary>
        <p>
          Por segurança, o aplicativo usa o processador. A placa de vídeo só é usada depois de um teste curto. Em placas
          incompatíveis, o teste pode travar o computador: salve seus trabalhos antes.
        </p>
        <div class="acoes">
          <button v-if="!confirmarGPU" class="botao" data-teste="testar-gpu" @click="confirmarGPU = true">Testar a placa de vídeo</button>
          <template v-else>
            <button class="botao perigo" data-teste="confirmar-gpu" @click="testar">Salvei meus trabalhos — testar agora</button>
            <button class="botao" @click="confirmarGPU = false">Cancelar</button>
          </template>
          <button v-if="m.gpu === 'estavel'" class="botao discreto" @click="usarSoCPU">Usar só o processador</button>
        </div>
      </details>
    </template>
  </main>
</template>

<style scoped>
.modelos { max-width: 920px; margin: 0 auto; padding: 2.6rem 2rem; display: grid; gap: 1.4rem; }
.topo { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
.topo h1 { font-size: var(--t-xl); }
.intro { max-width: 68ch; color: var(--tinta-suave); }
section, .avancado { background: var(--superficie); border: 1px solid var(--linha); border-radius: var(--raio); padding: 1.1rem 1.3rem; display: grid; gap: 0.7rem; }
h2 { font-size: var(--t-lg); }
dl { margin: 0; display: grid; gap: 0.3rem; }
dl div { display: grid; grid-template-columns: 9rem 1fr; gap: 0.8rem; }
dt { color: var(--tinta-suave); }
dd { margin: 0; }
.avisos { margin: 0; padding: 0.6rem 0.8rem 0.6rem 1.8rem; background: var(--alerta-claro); color: var(--alerta); border-radius: var(--raio); }
.barra { height: 8px; background: var(--linha); border-radius: 4px; overflow: hidden; }
.barra span { display: block; height: 100%; background: var(--tinta); transition: width 300ms ease; }
.conta, .nota, .meta { font-size: var(--t-sm); color: var(--tinta-suave); }
.modelo { border-top: 1px solid var(--linha); padding-top: 0.8rem; display: grid; gap: 0.35rem; }
.modelo.rec h3::before { content: ""; display: inline-block; width: 0.5rem; height: 0.5rem; border-radius: 50%; background: var(--ok); margin-right: 0.5rem; vertical-align: middle; }
.cabeca { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; }
.cabeca h3 { font-size: var(--t-md); font-family: var(--sans); }
.selo { font-size: var(--t-xs); border: 1px solid var(--linha-forte); border-radius: 999px; padding: 0.05rem 0.55rem; }
.selo.ok { border-color: var(--ok); color: var(--ok); }
.selo.alerta { border-color: var(--alerta); color: var(--alerta); }
.acoes { display: flex; flex-wrap: wrap; gap: 0.6rem; align-items: center; }
summary { cursor: pointer; font-weight: 600; }
.aviso.erro { border: 1px solid var(--erro); background: var(--erro-claro); border-radius: var(--raio); padding: 0.9rem 1.1rem; }
</style>
