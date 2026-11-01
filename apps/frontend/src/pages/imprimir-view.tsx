import {
  ArrowLeftIcon,
  PrinterIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
} from "lucide-react"
import { Link } from "react-router"

import { EstadoCargando, EstadoError } from "@/components/estados"
import { Button } from "@/components/ui/button"
import {
  PREGUNTAS_INHABILIDADES,
  PUNTOS_CONSENTIMIENTO_EF12,
  type ConsentimientoEF12Data,
} from "@/features/formulario/constantes"
import { useListarPostulaciones } from "@/features/postulaciones/queries"
import { RUTAS } from "@/lib/rutas"
import { calcularDuracionExperiencia } from "@/helpers/experiencia"
import type {
  Seccion1Data,
  Seccion2Data,
  Seccion3Data,
  Seccion4Data,
  Seccion5Data,
  Seccion6Data,
  Seccion7Data,
  Seccion8Data,
  Seccion9Data,
} from "@/features/formulario"

/**
 * Vista de impresión oficial del formulario EF-12 (HU-11).
 * Incluye el formulario íntegro sellado, los 5 puntos de consentimiento,
 * el hash criptográfico inmutable y el espacio para la firma manuscrita.
 */
export function ImprimirView() {
  const {
    data: postulacionesData,
    isLoading,
    error,
    refetch,
  } = useListarPostulaciones({ limit: 1 })

  const postulacion = postulacionesData?.items?.[0]
  const content = postulacion?.content || {}

  const s1 = (content.seccion1 || {}) as Partial<Seccion1Data>
  const s2 = (content.seccion2 || {}) as Partial<Seccion2Data>
  const s3 = (content.seccion3 || {}) as Partial<Seccion3Data>
  const s4 = (content.seccion4 || {}) as Partial<Seccion4Data>
  const s5 = (content.seccion5 || {}) as Partial<Seccion5Data>
  const s6 = (content.seccion6 || {}) as Partial<Seccion6Data>
  const s7 = (content.seccion7 || {}) as Partial<Seccion7Data>
  const s8 = (content.seccion8 || {}) as Partial<Seccion8Data>
  const s9 = (content.seccion9 || {}) as Partial<Seccion9Data>
  const consentimiento = (content.consentimiento ||
    {}) as Partial<ConsentimientoEF12Data>

  const yaEnviada = postulacion?.status !== "draft"

  function handleImprimir() {
    window.print()
  }

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoCargando />
      </div>
    )
  }

  if (error || !postulacion) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-12">
        <EstadoError
          error={error}
          titulo="Error al cargar el formulario"
          mensaje="No se pudo obtener la información para la impresión."
          onReintentar={() => {
            void refetch()
          }}
        />
      </div>
    )
  }

  const nombreCompleto =
    `${s1.nombres || ""} ${s1.apellidos || ""}`.trim() || "Postulante"
  const cedulaIdentidad = s1.cedula || "No registrada"

  return (
    <div className="min-h-screen bg-muted/20 py-8 print:bg-white print:p-0">
      {/* Barra superior de acciones (oculta al imprimir) */}
      <div className="mx-auto mb-6 flex w-full max-w-4xl flex-wrap items-center justify-between gap-3 px-4 print:hidden">
        <div className="flex items-center gap-2">
          <Button
            asChild
            variant="outline"
            size="sm"
            className="flex items-center gap-1.5"
          >
            <Link to={RUTAS.postulacionRevision}>
              <ArrowLeftIcon className="size-4" aria-hidden="true" />
              <span>Volver a la revisión</span>
            </Link>
          </Button>
          <Button asChild variant="ghost" size="sm">
            <Link to={RUTAS.postulacion}>Ir al formulario</Link>
          </Button>
        </div>

        <div className="flex items-center gap-3">
          <Button
            type="button"
            onClick={handleImprimir}
            className="flex items-center gap-2 bg-primary text-primary-foreground shadow-sm"
          >
            <PrinterIcon className="size-4" aria-hidden="true" />
            <span>Imprimir Formulario (o Guardar PDF)</span>
          </Button>
        </div>
      </div>

      {/* Alerta de aviso sobre firma manuscrita (oculta al imprimir) */}
      <div className="mx-auto mb-6 w-full max-w-4xl px-4 print:hidden">
        {yaEnviada ? (
          <div
            role="status"
            className="rounded-lg border border-emerald-500/40 bg-emerald-50/70 p-4 text-xs text-emerald-950 dark:bg-emerald-950/20 dark:text-emerald-100"
          >
            <p className="flex items-center gap-2 font-semibold text-sm">
              <ShieldCheckIcon className="size-4 text-emerald-600" />
              Formulario sellado con Hash Criptográfico
            </p>
            <p className="mt-1">
              Imprima este documento completo. Conforme a las bases del concurso
              (HU-11 / EF-12 1.10.2), debe firmarlo a mano de forma manuscrita y
              entregarlo con los requisitos físicos solicitados.
            </p>
          </div>
        ) : (
          <div
            role="alert"
            className="rounded-lg border border-amber-500/40 bg-amber-50/70 p-4 text-xs text-amber-950 dark:bg-amber-950/20 dark:text-amber-100"
          >
            <p className="flex items-center gap-2 font-semibold text-sm">
              <ShieldAlertIcon className="size-4 text-amber-600" />
              Borrador Preliminar No Sellado
            </p>
            <p className="mt-1">
              Este formulario aún no ha sido enviado definitivamente. Puede
              imprimir una vista previa, pero recuerde que para formalizar su
              participación debe realizar el envío desde la pantalla de
              revisión.
            </p>
          </div>
        )}
      </div>

      {/* DOCUMENTO OFICIAL IMPRIMIBLE */}
      <main className="mx-auto w-full max-w-4xl rounded-lg border border-border bg-white p-8 text-neutral-900 shadow-sm print:max-w-none print:rounded-none print:border-none print:p-0 print:text-black print:shadow-none">
        {/* Encabezado Institucional */}
        <header className="border-b-2 border-neutral-900 pb-4 text-center">
          <p className="text-xs font-bold tracking-wider text-neutral-700 uppercase">
            República del Ecuador
          </p>
          <h1 className="mt-1 text-base font-extrabold tracking-tight text-neutral-900 uppercase">
            Instituto Ecuatoriano de Seguridad Social (IESS)
          </h1>
          <h2 className="text-sm font-semibold tracking-tight text-neutral-800 uppercase">
            Banco del Instituto Ecuatoriano de Seguridad Social (BIESS)
          </h2>
          <div className="mt-3 inline-block rounded border border-neutral-900 px-3 py-1 text-xs font-bold uppercase">
            Concurso de Méritos y Oposición · Formulario Oficial de Postulación
            EF-12
          </div>
        </header>

        {/* Metadatos de la Postulación */}
        <section className="my-4 grid grid-cols-2 gap-2 border border-neutral-400 bg-neutral-50 p-3 text-xs sm:grid-cols-4 print:bg-transparent">
          <div>
            <span className="block font-bold text-neutral-600">
              Código Único:
            </span>
            <span className="font-mono font-semibold">
              {postulacion.code || "BORRADOR"}
            </span>
          </div>
          <div>
            <span className="block font-bold text-neutral-600">Estado:</span>
            <span className="font-semibold uppercase">
              {yaEnviada ? "Sellada y Remitida" : "Borrador"}
            </span>
          </div>
          <div>
            <span className="block font-bold text-neutral-600">
              Fecha de Envío:
            </span>
            <span>
              {postulacion.submittedAt
                ? new Date(postulacion.submittedAt).toLocaleString("es-EC")
                : "Pendiente de envío"}
            </span>
          </div>
          <div>
            <span className="block font-bold text-neutral-600">
              Sector Postulado:
            </span>
            <span className="font-semibold capitalize">
              {s4.sector || "No especificado"}
            </span>
          </div>
          {postulacion.hash && (
            <div className="col-span-2 mt-1 border-t border-neutral-300 pt-2 sm:col-span-4">
              <span className="block font-bold text-neutral-600">
                Firma Criptográfica SHA-256 (Sello Digital Inmutable):
              </span>
              <code className="text-2xs font-mono font-semibold break-all">
                {postulacion.hash}
              </code>
            </div>
          )}
        </section>

        {/* 1. INFORMACIÓN PERSONAL */}
        <section className="mb-5 border border-neutral-300">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            1. Información Personal
          </div>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2 p-3 text-xs sm:grid-cols-3">
            <div>
              <span className="block font-semibold text-neutral-600">
                Nombres:
              </span>
              <span>{s1.nombres || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Apellidos:
              </span>
              <span>{s1.apellidos || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Cédula de Identidad:
              </span>
              <span className="font-mono font-semibold">
                {s1.cedula || "—"}
              </span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Género:
              </span>
              <span>{s1.genero || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Fecha de Nacimiento:
              </span>
              <span>{s1.fechaNacimiento || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Nacionalidad:
              </span>
              <span>{s1.nacionalidad || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Estado Civil:
              </span>
              <span>{s1.estadoCivil || "—"}</span>
            </div>
            {(s1.estadoCivil === "Casado/a" ||
              s1.estadoCivil === "Unión de hecho") && (
              <div className="col-span-2">
                <span className="block font-semibold text-neutral-600">
                  Cónyuge o Pareja:
                </span>
                <span>
                  {s1.nombreConyuge || "—"} (Cédula: {s1.cedulaConyuge || "—"})
                </span>
              </div>
            )}
          </div>
        </section>

        {/* 2. CONTACTO Y DOMICILIO */}
        <section className="mb-5 border border-neutral-300">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            2. Información Domiciliaria y de Contacto
          </div>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2 p-3 text-xs sm:grid-cols-3">
            <div>
              <span className="block font-semibold text-neutral-600">
                País de Residencia:
              </span>
              <span>{s2.pais || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Provincia:
              </span>
              <span>{s2.provincia || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Cantón:
              </span>
              <span>{s2.canton || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Teléfono Celular:
              </span>
              <span>{s2.telefonoCelular || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Teléfono Domiciliario:
              </span>
              <span>{s2.telefonoDomiciliario || "—"}</span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Correo Electrónico:
              </span>
              <span>{s2.correo1 || "—"}</span>
            </div>
            <div className="col-span-2 sm:col-span-3">
              <span className="block font-semibold text-neutral-600">
                Dirección de Domicilio:
              </span>
              <span>
                {s2.callePrincipal || ""} {s2.numero || ""}{" "}
                {s2.calleSecundaria ? `y ${s2.calleSecundaria}` : ""}{" "}
                {s2.sector ? `(Sector: ${s2.sector})` : ""}{" "}
                {s2.referencia ? `— Referencia: ${s2.referencia}` : ""}
              </span>
            </div>
          </div>
        </section>

        {/* 3. INFORMACIÓN ADICIONAL - PARENTESCO */}
        <section className="mb-5 border border-neutral-300">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            3. Información Adicional (Parentesco IESS / BIESS)
          </div>
          <div className="p-3 text-xs">
            <p>
              <span className="font-semibold text-neutral-700">
                ¿Mantiene familiares laborando en IESS o BIESS (hasta 4to
                consanguinidad / 2do afinidad)?:
              </span>{" "}
              <strong>{s3.tieneFamiliaresIessBiess || "No"}</strong>
            </p>
            {s3.familiares && s3.familiares.length > 0 && (
              <table className="text-2xs mt-2 w-full border border-neutral-300 text-left">
                <thead className="bg-neutral-100">
                  <tr className="border-b">
                    <th className="p-1.5">Nombres del Familiar</th>
                    <th className="p-1.5">Parentesco</th>
                    <th className="p-1.5">Institución</th>
                    <th className="p-1.5">Área</th>
                    <th className="p-1.5">Cargo</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {s3.familiares.map((f, i) => (
                    <tr key={i}>
                      <td className="p-1.5 font-semibold">{f.nombres}</td>
                      <td className="p-1.5">{f.parentesco}</td>
                      <td className="p-1.5">{f.institucion}</td>
                      <td className="p-1.5">{f.area}</td>
                      <td className="p-1.5">{f.cargo}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>

        {/* 4. CARGO POSTULADO */}
        <section className="mb-5 border border-neutral-300">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            4. Selección del Cargo al que Postula
          </div>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2 p-3 text-xs sm:grid-cols-3">
            <div>
              <span className="block font-semibold text-neutral-600">
                Sector:
              </span>
              <span className="font-semibold capitalize">
                {s4.sector || "—"}
              </span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Verificación IESS:
              </span>
              <span>
                {s4.verificacionIessCumplida ? "Cumple condición" : "No"}
              </span>
            </div>
            <div>
              <span className="block font-semibold text-neutral-600">
                Respaldo Organizacional:
              </span>
              <span>
                {s4.cuentaConRespaldoOrganizacion === "Sí"
                  ? `Sí (${s4.nombreOrganizacion || "Organización declarada"})`
                  : "No cuenta con respaldo"}
              </span>
            </div>
          </div>
        </section>

        {/* 5. FORMACIÓN ACADÉMICA */}
        <section className="mb-5 border border-neutral-300 break-inside-avoid">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            5. Formación Académica
          </div>
          <div className="p-3 text-xs">
            {s5.titulos && s5.titulos.length > 0 ? (
              <table className="text-2xs w-full border border-neutral-300 text-left">
                <thead className="bg-neutral-100">
                  <tr className="border-b">
                    <th className="p-1.5">Título</th>
                    <th className="p-1.5">Nivel</th>
                    <th className="p-1.5">Institución</th>
                    <th className="p-1.5">Tipo</th>
                    <th className="p-1.5">Área</th>
                    <th className="p-1.5">N° Registro SENESCYT</th>
                    <th className="p-1.5">Fecha Registro</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {s5.titulos.map((t, idx) => (
                    <tr key={idx}>
                      <td className="p-1.5 font-semibold">{t.titulo}</td>
                      <td className="p-1.5">{t.nivel}</td>
                      <td className="p-1.5">{t.institucion}</td>
                      <td className="p-1.5">{t.tipo}</td>
                      <td className="p-1.5">{t.area}</td>
                      <td className="p-1.5 font-mono">{t.registroSenescyt}</td>
                      <td className="p-1.5">{t.fechaRegistroSenescyt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="italic text-neutral-600">
                No ha registrado títulos de formación superior.
              </p>
            )}
          </div>
        </section>

        {/* 6. EXPERIENCIA PROFESIONAL */}
        <section className="mb-5 border border-neutral-300 break-inside-avoid">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            6. Experiencia Profesional y Laboral
          </div>
          <div className="p-3 text-xs">
            {s6.experiencias && s6.experiencias.length > 0 ? (
              <div className="flex flex-col gap-3">
                {s6.experiencias.map((exp, idx) => {
                  const fin = exp.actualmenteLaborando
                    ? new Date().toISOString().slice(0, 10)
                    : exp.fechaFin
                  const dur = calcularDuracionExperiencia(exp.fechaInicio, fin)
                  return (
                    <div key={idx} className="border-b pb-2 last:border-b-0">
                      <div className="flex items-baseline justify-between">
                        <span className="font-bold text-neutral-900">
                          {idx + 1}. {exp.cargo} — {exp.institucion}
                        </span>
                        <span className="text-2xs font-semibold uppercase">
                          {exp.tipo}
                        </span>
                      </div>
                      <div className="text-2xs mt-1 text-neutral-700">
                        Periodo: {exp.fechaInicio} al{" "}
                        {exp.actualmenteLaborando ? "Presente" : exp.fechaFin}
                        {dur.texto ? ` · Duración: ${dur.texto}` : ""}
                      </div>
                      <p className="text-2xs mt-1 text-neutral-800">
                        <strong className="text-neutral-900">Funciones:</strong>{" "}
                        {exp.descripcionFunciones || exp.actividades}
                      </p>
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="italic text-neutral-600">
                No ha registrado experiencia laboral.
              </p>
            )}
          </div>
        </section>

        {/* 7 & 8: PUBLICACIONES Y CAPACITACIÓN */}
        <div className="mb-5 grid grid-cols-1 gap-4 sm:grid-cols-2 break-inside-avoid">
          <section className="border border-neutral-300">
            <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
              7. Publicaciones
            </div>
            <div className="p-3 text-xs">
              <p className="font-semibold">
                ¿Cuenta con publicaciones?: {s7.cuentaConPublicaciones || "No"}
              </p>
              {s7.publicaciones && s7.publicaciones.length > 0 && (
                <ul className="text-2xs mt-2 list-disc pl-4">
                  {s7.publicaciones.map((pub, idx) => (
                    <li key={idx} className="mb-1">
                      <strong>{pub.titulo}</strong> ({pub.tipo}) — {pub.medio} (
                      {pub.fechaPublicacion})
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </section>

          <section className="border border-neutral-300">
            <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
              8. Capacitación
            </div>
            <div className="p-3 text-xs">
              {s8.capacitaciones && s8.capacitaciones.length > 0 ? (
                <ul className="text-2xs list-disc pl-4">
                  {s8.capacitaciones.map((cap, idx) => (
                    <li key={idx} className="mb-1">
                      <strong>{cap.nombre}</strong> ({cap.horas} h) —{" "}
                      {cap.institucion} ({cap.tipo})
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="italic text-neutral-600">
                  No registra eventos de capacitación.
                </p>
              )}
            </div>
          </section>
        </div>

        {/* 9. INHABILIDADES Y PROHIBICIONES */}
        <section className="mb-5 border border-neutral-300 break-inside-avoid">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            9. Declaración Juramentada de Inhabilidades (13 Causales Legales)
          </div>
          <div className="text-2xs p-3">
            <table className="w-full border border-neutral-300 text-left">
              <thead className="bg-neutral-100">
                <tr className="border-b">
                  <th className="w-16 p-1">Numeral</th>
                  <th className="p-1">Causal Legal de Inhabilidad</th>
                  <th className="w-20 p-1 text-center">Respuesta</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {PREGUNTAS_INHABILIDADES.map((p) => {
                  const val =
                    (s9 as Record<string, string | undefined>)[p.id] || "—"
                  return (
                    <tr key={p.id}>
                      <td className="p-1 font-bold text-neutral-700">
                        {p.numero}
                      </td>
                      <td className="p-1 text-neutral-800">
                        {p.texto}
                        {p.id === "q7" && val === "Sí" && s9.q7Detalles && (
                          <div className="mt-1 border-t pt-1 font-sans text-neutral-600">
                            <strong>Detalles de remoción:</strong> Inst:{" "}
                            {s9.q7Detalles.institucion} · Cargo:{" "}
                            {s9.q7Detalles.cargo} · Fecha:{" "}
                            {s9.q7Detalles.fechaFin} · Motivo:{" "}
                            {s9.q7Detalles.motivo}
                          </div>
                        )}
                      </td>
                      <td className="p-1 text-center font-bold">{val}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </section>

        {/* 10. CONSENTIMIENTOS Y DECLARACIONES JURADAS (EF-12 1.10.1) */}
        <section className="mb-6 border border-neutral-300 break-inside-avoid">
          <div className="bg-neutral-200 px-3 py-1 text-xs font-bold text-neutral-900 uppercase print:bg-neutral-200">
            10. Consentimientos y Declaración Juramentada (EF-12 1.10.1)
          </div>
          <div className="text-2xs flex flex-col gap-2 p-3 leading-tight text-neutral-800">
            {PUNTOS_CONSENTIMIENTO_EF12.map((punto) => {
              const aceptado = Boolean(consentimiento[punto.id])
              return (
                <div key={punto.id} className="flex items-start gap-2">
                  <span className="font-mono font-bold shrink-0">
                    {aceptado ? "[X]" : "[ ]"}
                  </span>
                  <p>
                    <strong>Punto {punto.numero}:</strong> {punto.texto}
                  </p>
                </div>
              )
            })}
            {Boolean(
              consentimiento.politicaPrivacidadId ||
              consentimiento.hashPoliticaPrivacidad,
            ) && (
              <div className="mt-2 border-t border-neutral-300 pt-2 font-mono text-[9px] text-neutral-600">
                <span>
                  Vinculación institucional: Política v
                  {consentimiento.versionPolitica ?? "1"} · Ref:{" "}
                  {consentimiento.referenciaAprobacion || "Resolución CD-BIESS"}{" "}
                  · Hash:{" "}
                  {consentimiento.hashPoliticaPrivacidad?.substring(0, 32)}…
                </span>
              </div>
            )}
          </div>
        </section>

        {/* 11. APARTADO DE FIRMA MANUSCRITA (HU-11 / EF-12 1.10.2) */}
        <section className="mt-8 border-2 border-neutral-800 p-6 break-inside-avoid print:mt-12">
          <div className="text-center">
            <div className="mx-auto mt-16 w-80 border-b-2 border-neutral-800" />
            <p className="mt-2 text-xs font-bold tracking-wider text-neutral-900 uppercase">
              Firma Manuscrita del Postulante
            </p>
            <p className="mt-1 text-xs font-semibold text-neutral-800">
              {nombreCompleto}
            </p>
            <p className="text-xs font-mono text-neutral-700">
              C.I. {cedulaIdentidad}
            </p>
            <p className="text-2xs mt-2 text-neutral-600">
              Lugar y Fecha: _____________________________________________
            </p>
          </div>

          <div className="text-3xs mt-6 rounded border border-neutral-400 bg-neutral-50 p-2.5 text-center text-neutral-700 print:bg-transparent">
            <strong>Nota Obligatoria (EF-12 1.10.2):</strong> El presente
            formulario debe ser impreso íntegramente y contar con la{" "}
            <strong>firma manuscrita original</strong> del postulante, para ser
            entregado físicamente junto con los requisitos solicitados en el
            lugar, fecha y horario establecido por el Instituto Ecuatoriano de
            Seguridad Social. No se admite firma dibujada en pantalla o firma
            electrónica como sustituto de la firma manuscrita original en papel.
          </div>
        </section>
      </main>
    </div>
  )
}
