// Galeria de telas (só desenvolvimento). Escolha a tela por #cenario.
import { createApp } from "vue";
import "@fontsource/source-serif-4/latin-400.css";
import "@fontsource/source-serif-4/latin-600.css";
import "@fontsource/ibm-plex-sans/latin-400.css";
import "@fontsource/ibm-plex-sans/latin-500.css";
import "@fontsource/ibm-plex-sans/latin-600.css";
import "../estilo.css";
import App from "../App.vue";
import { palco } from "./api-falsa";
import { cenarios, iaPadrao } from "./cenarios";
import { estado } from "../estado";

const nome = decodeURIComponent(location.hash.replace(/^#/, "")) || "inicio";
const c = cenarios[nome];
if (!c) {
  document.body.innerHTML = "<ul style='font:15px sans-serif;padding:2rem'>" +
    Object.entries(cenarios).map(([k, v]) => `<li><a href="#${k}" onclick="setTimeout(()=>location.reload())">${v.titulo}</a></li>`).join("") + "</ul>";
} else {
  palco.visao = c.visao;
  palco.ia = c.ia ?? iaPadrao;
  if (c.config) estado.verConfigIA = true;
  createApp(App).mount("#app");
  // Cenários de modal/gaveta: clica no botão que os abre. Sem animações, para a
  // captura sem GPU (o tempo virtual não avança as transições de CSS).
  if (c.clicar) {
    const st = document.createElement("style");
    st.textContent = "*,*::before,*::after{transition:none!important;animation:none!important}";
    document.head.appendChild(st);
  }
  if (c.clicar) setTimeout(() => document.querySelector<HTMLElement>(`[data-teste="${c.clicar}"]`)?.click(), 300);
  window.addEventListener("hashchange", () => location.reload());
}
