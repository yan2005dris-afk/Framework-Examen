/*
 * mock.js — Único JavaScript del set de mockups.
 *
 * NO contiene reglas de negocio: sin validación, sin hash, sin estado
 * persistente, sin llamadas de red. Solo dos mejoras progresivas que existen
 * para poder mirar estados que en React serían estado de componente:
 *
 *   1. Mostrar/ocultar contraseña        -> [data-password-toggle]
 *   2. Abrir/cerrar un diálogo de demo   -> [data-dialog-open] / [data-dialog-close]
 *
 * Todo lo demás (condicionales Sí/No, pasos del wizard, navegación) funciona
 * con CSS o con enlaces, sin JS.
 */
(() => {
  "use strict";

  document.addEventListener("click", (event) => {
    const toggle = event.target.closest("[data-password-toggle]");
    if (toggle) {
      const input = document.getElementById(toggle.getAttribute("aria-controls"));
      if (!input) return;
      const show = input.type === "password";
      input.type = show ? "text" : "password";
      // El estado se comunica con el texto del boton ("Mostrar" / "Ocultar").
      // Un boton con nombre cambiante no debe usar aria-pressed: el lector de
      // pantalla anunciaria el estado dos veces.
      toggle.textContent = show ? "Ocultar" : "Mostrar";
      return;
    }

    const open = event.target.closest("[data-dialog-open]");
    if (open) {
      const dialog = document.getElementById(open.getAttribute("data-dialog-open"));
      if (dialog) dialog.hidden = false;
      return;
    }

    const close = event.target.closest("[data-dialog-close]");
    if (close) {
      const dialog = close.closest("[data-dialog]");
      if (dialog) dialog.hidden = true;
    }
  });
})();
