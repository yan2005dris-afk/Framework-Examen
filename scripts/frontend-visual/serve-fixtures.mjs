/** Vista previa visual aislada: sólo GET, datos sintéticos y escucha loopback.
 * No conecta a PostgreSQL ni crea sesiones/roles reales. No usar en producción.
 * VITE_DEV_API_TARGET=http://127.0.0.1:5180 pnpm --dir apps/frontend dev
 */
import http from "node:http"
import { readFileSync } from "node:fs"

const raiz = new URL("../../", import.meta.url)
const leer = (ruta) => JSON.parse(readFileSync(new URL(ruta, raiz), "utf8"))
const definicion = leer("apps/backend/internal/submission/definition.json")
const formulario = leer(
  "apps/backend/internal/submission/testdata/form_cases.json",
).base
const rol =
  process.argv.find((a) => a.startsWith("--rol="))?.split("=")[1] ??
  "postulante"
if (!["postulante", "analista", "administrador", "anonimo"].includes(rol))
  throw Error("Rol de fixture inválido")
const periodo = "019a4000-0000-7000-8000-000000000001"
const id = "019a4000-0000-7000-8000-000000000002"
const usuario = {
  id: "019a4000-0000-7000-8000-000000000003",
  cedula: formulario.seccion1.cedula,
  nombres: "Persona",
  apellidos: "Prueba",
  nombre_completo: "Persona Prueba",
  correo: "visual@example.test",
  metodo_validacion: "otp_correo",
  fecha_registro: new Date().toISOString(),
  rol,
}
const apertura = new Date(Date.now() - 86400000).toISOString()
const cierre = new Date(Date.now() + 7 * 86400000).toISOString()
const sectores = ["afiliados", "jubilados", "empleadores"]
const fases = [
  {
    codigo: "postulacion",
    titulo: "Recepción de postulaciones",
    apertura,
    cierre,
    publicada: true,
    habilitada: true,
  },
]
const convocatoria = {
  id: periodo,
  codigo: "QA-VISUAL",
  titulo: "Concurso Público de Méritos y Oposición",
  apertura,
  cierre,
  sectores,
  fases,
  ef12Disponible: true,
  zonaHoraria: "America/Guayaquil",
  limites: "DATOS SINTÉTICOS · VISTA PREVIA VISUAL",
  version: 1,
  inicioAprobado: true,
  publicada: true,
  ef12Habilitado: true,
  cerrada: false,
}
const pagina = (items) => ({
  items,
  hasMore: false,
  nextCursor: "",
  total: items.length,
  page: 1,
  pageSize: 20,
  totalPages: 1,
})
const postulacion = {
  id,
  applicantId: usuario.id,
  convocatoriaId: periodo,
  code: "QA-VISUAL-0001",
  status: "draft",
  content: formulario,
  createdAt: apertura,
  updatedAt: apertura,
  progress: {
    total: 9,
    completed: 4,
    percentage: 44,
    sections: Array.from({ length: 9 }, (_, n) => ({
      number: n + 1,
      complete: n < 4,
    })),
  },
}
const politica = {
  id,
  convocatoriaId: periodo,
  version: 1,
  versionControl: 1,
  restringido: false,
  hash: "a".repeat(64),
  aprobadaPor: usuario.id,
  publicadaEn: apertura,
  definicion: {
    texto:
      "Política sintética para verificar la presentación. No tiene validez institucional.",
    informacionTratamiento: "Sólo datos de prueba.",
    referenciaAprobacion: "QA VISUAL · NO INSTITUCIONAL",
    diasAcceso: 365,
    diasDocumentos: 365,
    diasReportes: 365,
    diasPublicacionNombres: 365,
    diasCorreo: 180,
    diasRevisionConservacion: 365,
    disposicionHistoricos: "revision_humana_sin_borrado_automatico",
    permiteExportacion: true,
    permiteCorreo: true,
    permiteNombresPublicos: true,
  },
}

http
  .createServer((req, res) => {
    res.setHeader("Content-Type", "application/json; charset=utf-8")
    res.setHeader("Cache-Control", "no-store")
    res.setHeader("X-Data-Origin", "synthetic-visual-only")
    const enviar = (body, status = 200) => {
      res.writeHead(status)
      res.end(JSON.stringify(body))
    }
    // Un iframe acotado permite verificar los breakpoints sin cambiar la ventana del usuario.
    const direccion = new URL(req.url, "http://127.0.0.1")
    if (req.method === "GET" && direccion.pathname === "/__qa/mobile") {
      const seccion = Number(direccion.searchParams.get("seccion") || 1)
      if (!Number.isInteger(seccion) || seccion < 1 || seccion > 9)
        return enviar({}, 400)
      res.setHeader("Content-Type", "text/html; charset=utf-8")
      res.writeHead(200)
      return res.end(
        `<!doctype html><html lang="es"><meta charset="utf-8"><title>QA visual móvil · 390 px</title><h1>Vista sintética de 390 px</h1><nav>${Array.from({ length: 9 }, (_, n) => `<a href="/__qa/mobile?seccion=${n + 1}">Sección ${n + 1}</a>`).join(" · ")}</nav><iframe title="Formulario móvil" width="390" height="844" src="http://127.0.0.1:5175/postulacion/${seccion}?convocatoria=${periodo}"></iframe></html>`,
      )
    }
    if (req.method !== "GET")
      return enviar(
        {
          error: {
            codigo: "qa_solo_lectura",
            mensaje:
              "La vista previa no permite guardar, enviar ni autenticar cuentas reales.",
          },
        },
        405,
      )
    const path = new URL(req.url, "http://127.0.0.1").pathname
      .replace(/^\/api\/v1/, "")
      .replace(/\/$/, "")
    if (path === "/auth/session")
      return rol === "anonimo"
        ? enviar(
            {
              error: {
                codigo: "sesion_expirada",
                mensaje: "Sin sesión de prueba.",
              },
            },
            401,
          )
        : enviar({ postulante: usuario })
    if (path === "/competitions" || path === "/admin/competitions")
      return enviar(pagina([convocatoria]))
    if (
      path === `/competitions/${periodo}` ||
      path === `/admin/competitions/${periodo}`
    )
      return enviar(convocatoria)
    if (path.startsWith("/public/privacy/")) return enviar(politica)
    if (path === "/submissions/form-definition") return enviar(definicion)
    if (path === "/submissions") return enviar(pagina([postulacion]))
    if (path === `/submissions/${id}`) return enviar(postulacion)
    if (path.endsWith("/documents")) return enviar([])
    if (path === "/catalogs/countries")
      return enviar({ paises: ["Ecuador", "Colombia", "Perú"] })
    if (path.startsWith("/catalogs/provinces"))
      return enviar({ provincias: ["Pichincha", "Guayas"] })
    if (path.startsWith("/catalogs/cities"))
      return enviar({ cantones: ["Quito", "Guayaquil"] })
    if (path === "/admin/submissions/stats")
      return enviar({
        consultadoEn: new Date().toISOString(),
        actualizarEnSegundos: 30,
        conteos: sectores.map((sector, n) => ({
          sector,
          status: "submitted",
          postulantes: [612, 358, 177][n],
          postulaciones: [612, 358, 177][n],
        })),
      })
    if (path === "/admin/submissions")
      return enviar(
        pagina([
          {
            id,
            codigo: "QA-VISUAL-0001",
            convocatoria: convocatoria.titulo,
            nombres: "Persona",
            apellidos: "Prueba",
            sector: "afiliados",
            status: "submitted",
            submittedAt: apertura,
            hash: "a".repeat(64),
            titulos: [{ nivel: "Tercer nivel", titulo: "Economía" }],
            experiencias: [],
            publicaciones: 0,
            horasCapacitacion: 40,
          },
        ]),
      )
    if (path === `/admin/submissions/${id}`)
      return enviar({
        ...postulacion,
        status: "submitted",
        hash: "a".repeat(64),
        submittedAt: apertura,
      })
    if (path.endsWith("/events") || path.endsWith("/history"))
      return enviar(pagina([]))
    console.error(`Fixture visual no definido: ${path}`)
    enviar(
      {
        error: {
          codigo: "qa_no_definido",
          mensaje: "Este recurso no tiene fixture visual.",
        },
      },
      404,
    )
  })
  .listen(5180, "127.0.0.1", () =>
    console.log(
      `QA VISUAL · SÓLO DATOS SINTÉTICOS · rol=${rol} · http://127.0.0.1:5180`,
    ),
  )
