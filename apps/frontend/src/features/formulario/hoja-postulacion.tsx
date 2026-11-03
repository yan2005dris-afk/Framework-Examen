import type { ReactNode } from "react"
import type { PostulacionDto } from "@/lib/clientes/postulaciones"
import { formatearFechaEcuador } from "@/utils/fechas"
import {
  PREGUNTAS_INHABILIDADES,
  PUNTOS_CONSENTIMIENTO_EF12,
  type ConsentimientoEF12Data,
} from "./constantes"
import { duracionExperienciaImprimible } from "./modelo"
import type { Seccion1Data } from "./seccion-1-personal"
import type { Seccion2Data } from "./seccion-2-contacto"
import type { Seccion3Data } from "./seccion-3-adicional"
import type { Seccion4Data } from "./seccion-4-cargo"
import type { Seccion5Data } from "./seccion-5-academica"
import type { Seccion6Data } from "./seccion-6-experiencia"
import type { Seccion7Data } from "./seccion-7-publicaciones"
import type { Seccion8Data } from "./seccion-8-capacitacion"
import type { Seccion9Data } from "./seccion-9-inhabilidades"
import "@/impresion.css"

const texto = (valor: unknown) =>
  valor == null || valor === ""
    ? "No consta"
    : typeof valor === "boolean"
      ? valor
        ? "Sí"
        : "No"
      : String(valor)
function Filas({ valores }: { valores: [string, unknown][] }) {
  return (
    <dl className="print-rows">
      {valores.map(([nombre, valor]) => (
        <div key={nombre} className="print-row">
          <dt>{nombre}</dt>
          <dd>{texto(valor)}</dd>
        </div>
      ))}
    </dl>
  )
}
function Tabla({
  columnas,
  filas,
}: {
  columnas: string[]
  filas: ReactNode[][]
}) {
  return (
    <div className="print-table-wrap">
      <table className="print-table">
        <thead>
          <tr>
            {columnas.map((nombre) => (
              <th scope="col" key={nombre}>
                {nombre}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {filas.length ? (
            filas.map((fila, n) => (
              <tr key={n}>
                {fila.map((valor, c) => (
                  <td key={c}>{valor ?? "No consta"}</td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={columnas.length}>
                Sin registros declarados en el formulario.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
function Bloque({ titulo, children }: { titulo: string; children: ReactNode }) {
  return (
    <section className="print-block">
      <h2>{titulo}</h2>
      {children}
    </section>
  )
}

/** Hoja A4 basada en el mockup 09; representa sólo el contenido recibido, sin completar datos ilustrativos. */
export function HojaPostulacion({
  postulacion,
}: {
  postulacion: PostulacionDto
}) {
  const contenido = postulacion.content
  const s1 = (contenido.seccion1 ?? {}) as Partial<Seccion1Data>
  const s2 = (contenido.seccion2 ?? {}) as Partial<Seccion2Data>
  const s3 = (contenido.seccion3 ?? {}) as Partial<Seccion3Data>
  const s4 = (contenido.seccion4 ?? {}) as Partial<Seccion4Data>
  const s5 = (contenido.seccion5 ?? {}) as Partial<Seccion5Data>
  const s6 = (contenido.seccion6 ?? {}) as Partial<Seccion6Data>
  const s7 = (contenido.seccion7 ?? {}) as Partial<Seccion7Data>
  const s8 = (contenido.seccion8 ?? {}) as Partial<Seccion8Data>
  const s9 = (contenido.seccion9 ?? {}) as Partial<Seccion9Data>
  const consentimientos = (contenido.consentimiento ??
    {}) as Partial<ConsentimientoEF12Data>
  const sellada =
    postulacion.status !== "draft" &&
    Boolean(postulacion.hash && postulacion.submittedAt)
  return (
    <article
      className="hoja-postulacion"
      aria-label="Formulario completo para impresión y firma"
    >
      <header className="print-head">
        <div>
          <p className="print-institucion">
            INSTITUTO ECUATORIANO DE SEGURIDAD SOCIAL
          </p>
          <h1>Formulario de postulación</h1>
          <p>
            Concurso Público de Méritos y Oposición para la selección de los
            Miembros Principales y Alternos del Directorio del Banco del
            Instituto Ecuatoriano de Seguridad Social (BIESS), en representación
            de los Afiliados, Jubilados y Empleadores.
          </p>
        </div>
        <div className="print-seal">
          <p>
            <strong>Código único</strong>
          </p>
          <p>{postulacion.code}</p>
          <p>
            <strong>Fecha de envío</strong>
          </p>
          <p>
            {postulacion.submittedAt
              ? formatearFechaEcuador(postulacion.submittedAt, {
                  incluirHora: true,
                })
              : "Pendiente de envío"}
          </p>
          <p>
            {sellada
              ? "Enviada y sellada"
              : postulacion.status === "draft"
                ? "BORRADOR · SIN SELLO CONFIRMADO"
                : "FORMULARIO SIN SELLO DE ENVÍO"}
          </p>
        </div>
      </header>
      <Bloque titulo="1. Información personal">
        <Filas
          valores={[
            ["Nombres", s1.nombres],
            ["Apellidos", s1.apellidos],
            ["Cédula", s1.cedula],
            ["Género", s1.genero],
            ["Fecha de nacimiento", s1.fechaNacimiento],
            ["Nacionalidad", s1.nacionalidad],
            ["Estado civil", s1.estadoCivil],
            [
              "Pareja",
              [s1.nombreConyuge, s1.cedulaConyuge].filter(Boolean).join(" · "),
            ],
          ]}
        />
      </Bloque>
      <Bloque titulo="2. Información domiciliaria y de contacto">
        <Filas
          valores={[
            ["País", s2.pais],
            ["Provincia", s2.provincia],
            ["Cantón", s2.canton],
            [
              "Calle principal / número",
              [s2.callePrincipal, s2.numero].filter(Boolean).join(" · "),
            ],
            ["Calle secundaria", s2.calleSecundaria],
            ["Sector", s2.sector],
            ["Referencia", s2.referencia],
            ["Teléfono celular", s2.telefonoCelular],
            ["Teléfono domicilio", s2.telefonoDomiciliario],
            ["Correo electrónico 1", s2.correo1],
            ["Correo electrónico 2", s2.correo2],
          ]}
        />
      </Bloque>
      <Bloque titulo="3. Información adicional">
        <p>
          ¿Tiene familiares trabajando en el IESS o BIESS?{" "}
          <strong>{texto(s3.tieneFamiliaresIessBiess)}</strong>
        </p>
        <Tabla
          columnas={[
            "Nombres y apellidos",
            "Parentesco",
            "Institución",
            "Área",
            "Cargo",
          ]}
          filas={(s3.familiares ?? []).map((r) => [
            r.nombres,
            r.parentesco,
            r.institucion,
            r.area,
            r.cargo,
          ])}
        />
      </Bloque>
      <Bloque titulo="4. Selección del cargo">
        <Filas
          valores={[
            ["Sector", s4.sector],
            ["Respaldo de organización", s4.cuentaConRespaldoOrganizacion],
            ["Organización", s4.nombreOrganizacion],
            ["Documento", s4.documentoRespaldoPdf?.nombre],
          ]}
        />
      </Bloque>
      <Bloque titulo="5. Formación académica">
        <Tabla
          columnas={[
            "Nivel",
            "Título / área",
            "Institución",
            "SENESCYT / fecha",
            "Respaldo",
          ]}
          filas={(s5.titulos ?? []).map((r) => [
            r.nivel,
            <>
              {r.titulo}
              <br />
              {r.area}
            </>,
            <>
              {r.institucion}
              <br />
              {r.tipo}
            </>,
            <>
              {r.registroSenescyt}
              <br />
              {r.fechaRegistroSenescyt}
            </>,
            r.documentoPdf?.nombre,
          ])}
        />
      </Bloque>
      <Bloque titulo="6. Experiencia profesional">
        <Tabla
          columnas={[
            "Tipo / cargo",
            "Institución",
            "Período / duración al envío",
            "Actividades / funciones",
            "Respaldo",
          ]}
          filas={(s6.experiencias ?? []).map((r) => [
            <>
              {r.tipo}
              <br />
              {r.cargo}
            </>,
            r.institucion,
            <>
              {r.fechaInicio} –{" "}
              {r.actualmenteLaborando
                ? postulacion.submittedAt
                  ? "Vigente al envío"
                  : "Laborando actualmente"
                : r.fechaFin}
              <br />
              {duracionExperienciaImprimible(r, postulacion.submittedAt)}
            </>,
            <>
              {r.actividades}
              <br />
              {r.descripcionFunciones}
            </>,
            r.documentoPdf?.nombre,
          ])}
        />
      </Bloque>
      <Bloque titulo="7. Publicaciones e investigaciones">
        <p>
          ¿Cuenta con publicaciones?{" "}
          <strong>{texto(s7.cuentaConPublicaciones)}</strong>
        </p>
        <Tabla
          columnas={[
            "Tipo / título",
            "Medio",
            "Fecha",
            "Relación con el campo",
            "Respaldo",
          ]}
          filas={(s7.publicaciones ?? []).map((r) => [
            <>
              {r.tipo}
              <br />
              {r.titulo}
            </>,
            r.medio,
            r.fechaPublicacion,
            r.relacionCampo,
            r.documentoPdf?.nombre,
          ])}
        />
      </Bloque>
      <Bloque titulo="8. Capacitación">
        <Tabla
          columnas={[
            "Evento / tipo",
            "Institución",
            "Fechas",
            "Horas",
            "Respaldo",
          ]}
          filas={(s8.capacitaciones ?? []).map((r) => [
            <>
              {r.nombre}
              <br />
              {r.tipo}
            </>,
            r.institucion,
            <>
              {r.fechaInicio} – {r.fechaFin}
            </>,
            r.horas,
            r.documentoPdf?.nombre,
          ])}
        />
      </Bloque>
      <Bloque titulo="9. Inhabilidades y prohibiciones">
        <ol className="print-declaration print-preguntas">
          {PREGUNTAS_INHABILIDADES.map((pregunta) => (
            <li key={pregunta.id}>
              {pregunta.texto}{" "}
              <strong>{texto(s9[pregunta.id as keyof Seccion9Data])}</strong>
            </li>
          ))}
        </ol>
        {s9.q7Detalles && (
          <Filas
            valores={[
              ["Institución de remoción", s9.q7Detalles.institucion],
              ["Cargo", s9.q7Detalles.cargo],
              ["Fecha de fin", s9.q7Detalles.fechaFin],
              ["Motivo", s9.q7Detalles.motivo],
            ]}
          />
        )}
      </Bloque>
      <Bloque titulo="10. Declaraciones y consentimiento">
        <ol className="print-declaration">
          {PUNTOS_CONSENTIMIENTO_EF12.map((punto) => (
            <li key={punto.id}>
              <strong>{consentimientos[punto.id] ? "[X]" : "[ ]"}</strong> ·{" "}
              {punto.texto}
            </li>
          ))}
        </ol>
        {consentimientos.politicaPrivacidadId && (
          <p className="print-hash">
            Política aceptada: {consentimientos.politicaPrivacidadId}
            <br />
            Hash de política:{" "}
            {consentimientos.hashPoliticaPrivacidad ?? "No consta"}
          </p>
        )}
      </Bloque>
      <div className="print-signoff">
        <div className="signature-box">
          <div className="signature-line">
            <strong>Firma manuscrita del postulante</strong>
            <p>{[s1.nombres, s1.apellidos].filter(Boolean).join(" ")}</p>
            <p>C.I. {texto(s1.cedula)}</p>
          </div>
          <p>Lugar y fecha: __________________________________________</p>
        </div>
        <footer className="print-hash">
          <strong>Hash SHA-256 del envío</strong>
          <p>{postulacion.hash ?? "Sin hash de envío"}</p>
          <p>
            Documento para impresión íntegra y firma manuscrita. Los datos
            reproducen el formulario consultado; esta hoja no certifica
            identidad civil.
          </p>
        </footer>
      </div>
    </article>
  )
}
