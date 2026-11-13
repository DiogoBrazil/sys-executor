<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import type { MinutaVisao } from "../../bindings/github.com/DiogoBrazil/sys-executor/internal/workflow/models";
import { aprovar, estado, salvarMinuta, solicitarAjuste } from "../estado";
import Icone from "./ui/Icone.vue";

// emRevisao: a minuta passou pela validação e aguarda o usuário (V4 §31.3).
// Fora disso (ajuste ou validação em curso) ela é só leitura.
const props = defineProps<{ minuta: MinutaVisao; emRevisao?: boolean }>();
const emit = defineEmits<{ abrir: [pagina: number, inicio: number, fim: number] }>();

const pedido = ref("");
const salvos = ref<string[]>([]); // formatos já salvos desta versão
const confirmando = ref(false);
// Versão nova: os arquivos salvos antes são da versão anterior.
watch(() => props.minuta.versao, () => { salvos.value = []; confirmando.value = false; });

async function salvar(formato: "docx" | "pdf") {
  if ((await salvarMinuta(formato)) && !salvos.value.includes(formato)) salvos.value.push(formato);
}
async function enviarAjuste() {
  const t = pedido.value.trim();
  if (t.length < 5) return;
  if (await solicitarAjuste(t)) pedido.value = ""; // o texto não fica guardado na tela
}
async function finalizar() {
  confirmando.value = false;
  await aprovar();
}
const ajustes = computed(() => [...(props.minuta.ajustes ?? [])].reverse());
const ehSR = computed(() => estado.visao?.tipo === "SR");

const copiado = ref(false);
const texto = computed(() =>
  props.minuta.blocos
    .map((b) => (b.tipo === "TITULO" || b.tipo === "SECAO" || b.tipo === "SUBTITULO" ? "\n" + b.texto + "\n" : b.texto))
    .join("\n")
    .trim(),
);

// Prévia em tela cheia (Esc volta). O mesmo elemento muda de lugar: nada é
// duplicado na página.
const cheia = ref(false);
function tecla(e: KeyboardEvent) {
  if (e.key === "Escape") cheia.value = false;
}
watch(cheia, (c) => {
  if (c) document.addEventListener("keydown", tecla);
  else document.removeEventListener("keydown", tecla);
  document.body.classList.toggle("com-janela", c);
});
onBeforeUnmount(() => {
  document.removeEventListener("keydown", tecla);
  if (cheia.value) document.body.classList.remove("com-janela");
});

async function copiar() {
  try {
    await navigator.clipboard.writeText(texto.value);
    copiado.value = true;
    setTimeout(() => (copiado.value = false), 2500);
  } catch {
    copiado.value = false;
  }
}

// Realça os campos a preencher dentro do texto.
function partes(t: string): { texto: string; pendente: boolean }[] {
  return t
    .split(/(\[A PREENCHER\])/)
    .filter((p) => p !== "")
    .map((p) => ({ texto: p, pendente: p === "[A PREENCHER]" }));
}
</script>

<template>
  <section class="minuta" aria-label="Minuta" data-teste="minuta">
    <header class="topo">
      <h2>Minuta<span v-if="minuta.versao" class="versao" data-teste="minuta-versao"> · versão {{ minuta.versao }}</span></h2>
      <span v-if="emRevisao && minuta.validada" class="selo ok" data-teste="minuta-validada">Passou pela validação automática</span>
      <span v-if="minuta.camposPendentes" class="selo" data-teste="minuta-pendentes">
        {{ minuta.camposPendentes }} campo(s) a preencher
      </span>
      <button type="button" class="botao" data-teste="minuta-copiar" @click="copiar">
        <Icone :nome="copiado ? 'check' : 'documento'" :tamanho="16" />{{ copiado ? "Texto copiado" : "Copiar o texto" }}
      </button>
    </header>

    <div class="colunas">
    <div class="leitor" :class="{ cheia }">
      <div class="leitor-barra">
        <span>Prévia da minuta</span>
        <button type="button" class="botao" data-teste="minuta-tela-cheia" @click="cheia = !cheia">
          <Icone :nome="cheia ? 'fechar' : 'expandir'" :tamanho="16" />{{ cheia ? "Sair da tela cheia" : "Ler em tela cheia" }}
        </button>
      </div>
      <article class="folha">
      <template v-for="(b, i) in minuta.blocos" :key="i">
        <h3 v-if="b.tipo === 'TITULO'" class="titulo">{{ b.texto }}</h3>
        <h4 v-else-if="b.tipo === 'SECAO'" class="secao">{{ b.texto }}</h4>
        <h5 v-else-if="b.tipo === 'SUBTITULO'" class="subtitulo">{{ b.texto }}</h5>
        <div v-else :class="['bloco', b.tipo.toLowerCase(), { ia: b.origem === 'IA' }]">
          <p>
            <template v-for="(p, k) in partes(b.texto)" :key="k">
              <mark v-if="p.pendente">{{ p.texto }}</mark>
              <template v-else>{{ p.texto }}</template>
            </template>
          </p>
          <div v-if="b.fontes?.length" class="fontes">
            <button
              v-for="(f, j) in b.fontes"
              :key="j"
              type="button"
              class="fonte"
              :title="f.trecho"
              :aria-label="'Ver no PDF: ' + f.rotulo"
              @click="emit('abrir', f.pagina, f.inicio, f.fim)"
            >
              {{ f.rotulo }}
            </button>
          </div>
        </div>
      </template>
    </article>
    </div>

    <aside class="lado">
    <div class="callout" role="note" data-teste="minuta-aviso">
      <Icone nome="alerta" class="callout-icone" />
      <p>
        <strong>Atenção — revisão obrigatória:</strong> esta minuta é um documento de apoio. Confira integralmente os
        fatos, as provas, a fundamentação, os enquadramentos jurídicos, os cálculos e as conclusões antes de utilizá-la.
        A revisão, a aprovação e a decisão final cabem à autoridade competente.
      </p>
    </div>
    <div v-if="minuta.achados?.length" class="pendencias falhas" role="alert" data-teste="minuta-achados">
      <h3>A validação automática encontrou falhas que não pôde corrigir</h3>
      <ul>
        <li v-for="(a, i) in minuta.achados" :key="i">{{ a }}</li>
      </ul>
    </div>
    <div v-if="minuta.pendencias?.length" class="pendencias" data-teste="minuta-pendencias">
      <h3>O que conferir ou completar</h3>
      <ul>
        <li v-for="(p, i) in minuta.pendencias" :key="i">{{ p }}</li>
      </ul>
    </div>
    <section v-if="emRevisao" class="revisao" aria-label="Revisão da minuta" data-teste="minuta-revisao">
      <div class="grupo">
        <h3>Arquivos para conferência</h3>
        <p class="dica">O DOCX é editável; o PDF serve para conferir a aparência. Nenhum dos dois leva assinatura.</p>
        <div class="linha-acoes">
          <button type="button" class="botao" :disabled="estado.ocupado" data-teste="minuta-docx" @click="salvar('docx')"><Icone nome="documento" :tamanho="16" />Salvar DOCX</button>
          <button type="button" class="botao" :disabled="estado.ocupado" data-teste="minuta-pdf" @click="salvar('pdf')"><Icone nome="pdf" :tamanho="16" />Salvar PDF</button>
          <span v-if="salvos.length" class="feito" role="status" data-teste="minuta-salvo">
            Arquivo salvo ({{ salvos.map((f) => f.toUpperCase()).join(" e ") }}).
          </span>
        </div>
      </div>

      <div class="grupo">
        <h3><label for="pedido-ajuste">Solicitar ajustes</label></h3>
        <p v-if="ehSR" class="dica">
          Diga em qual parte (síntese dos fatos ou análise das provas de qual fato) e o que deve mudar. Só os trechos
          atingidos são reescritos e conferidos de novo. Para mudar a conclusão ou o enquadramento, use “Alterar” em “Suas respostas”.
        </p>
        <p v-else class="dica">
          Diga em qual parte (relatório, fundamentação de qual imputação, exame do art. 36) e o que deve mudar. Só os trechos
          atingidos são reescritos e conferidos de novo. Para mudar resultado, punição, dias ou pontos, use “Alterar” em “Suas respostas”.
        </p>
        <textarea id="pedido-ajuste" v-model="pedido" rows="3" maxlength="2000" data-teste="minuta-pedido"
          :placeholder="ehSR ? 'Ex.: na análise das provas do fato 1, acrescente o que a testemunha disse sobre a chave da reserva.' : 'Ex.: no relatório, acrescente que a defesa juntou a escala de serviço.'"></textarea>
        <div class="linha-acoes">
          <button type="button" class="botao" :disabled="estado.ocupado || pedido.trim().length < 5" data-teste="minuta-ajustar" @click="enviarAjuste">
            Enviar pedido de ajuste
          </button>
        </div>
        <ul v-if="ajustes.length" class="ajustes" data-teste="minuta-ajustes">
          <li v-for="(a, i) in ajustes" :key="i" :class="{ aplicado: a.aplicado }">
            <span class="pedido">“{{ a.pedido }}”</span>
            <span class="resultado">{{ a.resultado }}</span>
          </li>
        </ul>
      </div>

      <div class="grupo">
        <h3>Concluir</h3>
        <div v-if="!confirmando" class="linha-acoes">
          <button type="button" class="botao primario" :disabled="estado.ocupado" data-teste="minuta-aprovar" @click="confirmando = true">
            Aprovar e finalizar
          </button>
        </div>
        <div v-else class="confirmacao" role="alertdialog" aria-label="Confirmar a finalização" data-teste="minuta-confirmar">
          <p>
            Ao finalizar, os autos, a análise e a minuta são apagados deste computador e a tarefa não pode ser retomada.
            Os arquivos que você salvou permanecem onde os gravou.
          </p>
          <p v-if="!salvos.length" class="alerta-salvar" data-teste="minuta-nao-salva">
            Você ainda não salvou o DOCX nem o PDF desta versão.
          </p>
          <div class="linha-acoes">
            <button type="button" class="botao primario" :disabled="estado.ocupado" data-teste="minuta-confirmar-aprovar" @click="finalizar">
              Pode finalizar
            </button>
            <button type="button" class="botao" @click="confirmando = false">Continuar revisando</button>
          </div>
        </div>
      </div>
    </section>

    <p class="nota">
      Os parágrafos com a barra lateral foram redigidos pelo assistente a partir dos trechos indicados; o restante é montado pelo
      aplicativo com as suas respostas e o cálculo oficial.
    </p>
    </aside>
    </div>
  </section>
</template>

<style scoped>
.minuta { display: grid; gap: 0.9rem; }
.colunas { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(320px, 1fr); gap: 1.2rem; align-items: start; }
.lado { display: grid; gap: 0.9rem; min-width: 0; }
.leitor {
  position: sticky;
  top: 1rem;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  max-height: calc(100vh - 2rem);
  border: 1px solid var(--linha);
  border-radius: var(--r-md);
  background: var(--superficie-2);
  overflow: hidden;
  min-width: 0;
}
.leitor-barra {
  display: flex; align-items: center; justify-content: space-between; gap: 0.6rem;
  padding: 0.5rem 0.6rem 0.5rem 1rem; border-bottom: 1px solid var(--linha); background: var(--superficie);
  font-size: var(--t-sm); font-weight: 600; color: var(--tinta-suave);
}
.leitor-barra .botao { min-height: 2rem; padding: 0.3rem 0.75rem; font-size: var(--t-sm); }
.leitor .folha { overflow: auto; margin: 0; border: 0; border-radius: 0; box-shadow: none; overscroll-behavior: contain; }
.leitor.cheia {
  position: fixed; inset: 0; z-index: 35; max-height: none; border-radius: 0; border: 0;
  background: rgba(27, 36, 48, 0.5); animation: entra var(--dur-3) var(--ease);
}
.leitor.cheia .leitor-barra { justify-content: center; gap: 1.2rem; }
.leitor.cheia .folha { width: min(900px, 100%); margin: 1.2rem auto; border-radius: var(--r-sm); box-shadow: var(--sombra-2); }
@keyframes entra { from { opacity: 0; } to { opacity: 1; } }
@media (max-width: 1200px) {
  .colunas { grid-template-columns: 1fr; }
  .leitor { position: relative; top: 0; max-height: 75vh; }
}
.topo { display: flex; flex-wrap: wrap; align-items: center; gap: 0.8rem; }
.topo h2 { margin: 0; margin-right: auto; }
.selo {
  font-size: var(--t-sm);
  color: var(--alerta);
  background: var(--alerta-claro);
  border: 1px solid var(--alerta-linha);
  border-radius: 999px;
  padding: 0.1rem 0.7rem;
}
.selo.ok { color: var(--tinta-suave); background: var(--superficie-2); border-color: var(--linha); }
.versao { font-weight: 400; color: var(--tinta-suave); font-size: var(--t-md); }
.pendencias.falhas { border-color: var(--erro-linha); background: var(--erro-claro); }
.revisao {
  display: grid;
  gap: 1.1rem;
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--r-md);
  padding: 1.1rem 1.3rem;
}
.grupo { display: grid; gap: 0.5rem; }
.grupo + .grupo { border-top: 1px solid var(--linha); padding-top: 1.1rem; }
.grupo h3 { margin: 0; font-size: var(--t-md); }
.dica { margin: 0; font-size: var(--t-sm); color: var(--tinta-suave); }
.linha-acoes { display: flex; flex-wrap: wrap; gap: 0.7rem; align-items: center; }
.feito { font-size: var(--t-sm); color: var(--tinta-suave); }
.revisao textarea {
  width: 100%;
  font: inherit;
  padding: 0.6rem 0.7rem;
  border: 1px solid var(--linha-forte);
  border-radius: var(--r-sm);
  background: var(--papel);
  color: var(--tinta);
  resize: vertical;
}
.ajustes { list-style: none; margin: 0.2rem 0 0; padding: 0; display: grid; gap: 0.5rem; font-size: var(--t-sm); }
.ajustes li { display: grid; gap: 0.15rem; border-left: 3px solid var(--linha-forte); padding-left: 0.7rem; }
.ajustes li.aplicado { border-left-color: var(--acento); }
.ajustes .pedido { color: var(--tinta-suave); }
.confirmacao { display: grid; gap: 0.6rem; }
.confirmacao p { margin: 0; }
.alerta-salvar { color: var(--erro); font-weight: 600; }
.pendencias {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--r-md);
  padding: 0.8rem 1.2rem;
}
.pendencias h3 { margin: 0 0 0.4rem; font-size: var(--t-md); }
.pendencias ul { margin: 0; padding-left: 1.1rem; display: grid; gap: 0.3rem; font-size: var(--t-sm); }
.folha {
  background: var(--superficie);
  border: 1px solid var(--linha);
  border-radius: var(--r-sm);
  box-shadow: var(--sombra-1);
  padding: 2.2rem 2.6rem;
  font-family: var(--serif);
  font-size: 1.02rem;
  line-height: 1.6;
  color: var(--tinta);
}
.titulo { text-align: center; font-size: var(--t-lg); margin: 0 0 1.2rem; letter-spacing: 0.02em; }
.secao { font-size: var(--t-md); font-weight: 700; margin: 1.6rem 0 0.7rem; }
.subtitulo { font-size: var(--t-md); font-weight: 600; font-style: italic; margin: 1.1rem 0 0.4rem; }
.bloco p { margin: 0; }
.bloco.paragrafo { margin: 0 0 0.75rem; }
.bloco.paragrafo p { text-align: justify; text-indent: 2.5rem; }
.bloco.cabecalho { margin: 0 0 0.15rem; }
.bloco.item { margin: 0 0 0.45rem 2.5rem; }
.bloco.item p { text-align: justify; }
.bloco.fecho { margin: 1rem 0 0; }
.bloco.assinatura { margin: 2.2rem 0 0; text-align: center; }
.bloco.ia { border-left: 3px solid var(--acento-claro); padding-left: 0.7rem; margin-left: -0.9rem; }
.bloco.item.ia { margin-left: 1.6rem; }
mark { background: var(--alerta-claro); color: var(--alerta); border-radius: 3px; padding: 0 0.2rem; font-family: var(--sans); font-size: 0.9em; }
.fontes { display: flex; flex-wrap: wrap; gap: 0.35rem; margin-top: 0.3rem; font-family: var(--sans); }
.fonte {
  border: 1px solid var(--linha-forte);
  background: var(--papel);
  border-radius: 999px;
  padding: 0.05rem 0.6rem;
  font-size: var(--t-xs);
  cursor: pointer;
}
.fonte:hover { border-color: var(--acento); }
.nota { font-size: var(--t-sm); color: var(--tinta-suave); margin: 0; }
@media (max-width: 700px) {
  .folha { padding: 1.2rem 1.1rem; }
  .bloco.paragrafo p { text-indent: 1.5rem; }
}
</style>
