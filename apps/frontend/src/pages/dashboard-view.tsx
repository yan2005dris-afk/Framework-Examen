import {
  CalendarIcon,
  CheckCircle2Icon,
  CircleDashedIcon,
  FileCheck2Icon,
  FileTextIcon,
  HashIcon,
  Loader2Icon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  FORMULARIO_DEMO_COMPLETO,
  POSTULANTE_DEMO,
} from "@/data/mockInitialData"

type Seccion = {
  id: number
  titulo: string
  detalle: string
  completa: boolean
}

function construirSecciones(): Seccion[] {
  const f = FORMULARIO_DEMO_COMPLETO

  return [
    {
      id: 1,
      titulo: "Información personal",
      detalle: "Nombres, cédula y datos de registro civil",
      completa: Boolean(f.seccion1.nombres && f.seccion1.cedula),
    },
    {
      id: 2,
      titulo: "Domicilio y contacto",
      detalle: "Provincia, cantón, dirección y teléfonos",
      completa: Boolean(f.seccion2.provincia && f.seccion2.callePrincipal),
    },
    {
      id: 3,
      titulo: "Familiares IESS/BIESS",
      detalle: "Parentesco y datos de familiares",
      completa: f.seccion3.tieneFamiliaresIessBiess !== "",
    },
    {
      id: 4,
      titulo: "Cargo y sector",
      detalle: "Sector de postulación y respaldo organizacional",
      completa: Boolean(f.seccion4.sector),
    },
    {
      id: 5,
      titulo: "Formación académica",
      detalle: "Títulos registrados en el SENESCYT",
      completa: f.seccion5.titulos.length > 0,
    },
    {
      id: 6,
      titulo: "Experiencia profesional",
      detalle: "Cargos y funciones ejercidas",
      completa: f.seccion6.experiencias.length > 0,
    },
    {
      id: 7,
      titulo: "Publicaciones",
      detalle: "Publicaciones e investigaciones",
      completa: f.seccion7.cuentaConPublicaciones !== "",
    },
    {
      id: 8,
      titulo: "Capacitación",
      detalle: "Cursos, seminarios y talleres",
      completa: f.seccion8.capacitaciones.length > 0,
    },
    {
      id: 9,
      titulo: "Inhabilidades",
      detalle: "Declaración de impedimentos legales",
      completa: Object.values(f.seccion9).every((v) => v !== ""),
    },
  ]
}

export function DashboardView() {
  const secciones = construirSecciones()
  const completadas = secciones.filter((s) => s.completa).length
  const porcentaje = Math.round((completadas / secciones.length) * 100)
  const documentos =
    FORMULARIO_DEMO_COMPLETO.seccion6.experiencias.filter((e) => e.documentoPdf)
      .length +
    FORMULARIO_DEMO_COMPLETO.seccion5.titulos.filter((t) => t.documentoPdf)
      .length

  return (
    <div className="flex flex-1 flex-col gap-4 p-4">
      <div className="grid auto-rows-min gap-4 md:grid-cols-4">
        <TarjetaIndicador
          titulo="Estado"
          valor="Borrador"
          descripcion="Aún no ha sido remitido a la Comisión"
          icono={<FileTextIcon />}
        />
        <TarjetaIndicador
          titulo="Código de postulación"
          valor={FORMULARIO_DEMO_COMPLETO.codigoPostulacion}
          descripcion="Identificador único del expediente"
          icono={<HashIcon />}
        />
        <TarjetaIndicador
          titulo="Secciones completas"
          valor={`${completadas} de ${secciones.length}`}
          descripcion={`${porcentaje}% del formulario EF-12`}
          icono={<FileCheck2Icon />}
        />
        <TarjetaIndicador
          titulo="Documentos adjuntos"
          valor={String(documentos)}
          descripcion="Títulos y certificaciones en PDF"
          icono={<CalendarIcon />}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Secciones del formulario</CardTitle>
            <CardDescription>
              Complete las nueve secciones antes de remitir su postulación.
            </CardDescription>
            <CardAction>
              <Badge variant={porcentaje === 100 ? "default" : "secondary"}>
                {porcentaje}%
              </Badge>
            </CardAction>
          </CardHeader>
          <CardContent className="space-y-4">
            <Progress value={porcentaje} />
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">#</TableHead>
                  <TableHead>Sección</TableHead>
                  <TableHead className="hidden md:table-cell">
                    Detalle
                  </TableHead>
                  <TableHead className="w-28 text-right">Estado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {secciones.map((seccion) => (
                  <TableRow key={seccion.id}>
                    <TableCell className="font-medium">{seccion.id}</TableCell>
                    <TableCell className="font-medium">
                      {seccion.titulo}
                    </TableCell>
                    <TableCell className="hidden text-muted-foreground md:table-cell">
                      {seccion.detalle}
                    </TableCell>
                    <TableCell className="text-right">
                      {seccion.completa ? (
                        <Badge variant="default" className="gap-1">
                          <CheckCircle2Icon />
                          Completa
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="gap-1">
                          <CircleDashedIcon />
                          Pendiente
                        </Badge>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Postulante</CardTitle>
            <CardDescription>Datos de la cuenta registrada</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm">
            <div className="space-y-1">
              <p className="text-muted-foreground">Nombres</p>
              <p className="font-medium">
                {POSTULANTE_DEMO.nombres} {POSTULANTE_DEMO.apellidos}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Cédula</p>
              <p className="font-mono text-xs">{POSTULANTE_DEMO.cedula}</p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Correo</p>
              <p className="break-all text-xs">{POSTULANTE_DEMO.correo}</p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground">Validación</p>
              <Badge variant="outline" className="gap-1">
                <Loader2Icon className="size-3" />
                Código dactilar
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

type TarjetaIndicadorProps = {
  titulo: string
  valor: string
  descripcion: string
  icono: React.ReactNode
}

function TarjetaIndicador({
  titulo,
  valor,
  descripcion,
  icono,
}: TarjetaIndicadorProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{titulo}</CardDescription>
        <CardTitle className="text-2xl">{valor}</CardTitle>
        <CardAction className="rounded-lg border p-2">
          <span className="text-muted-foreground">{icono}</span>
        </CardAction>
      </CardHeader>
      <CardContent>
        <p className="text-sm text-muted-foreground">{descripcion}</p>
      </CardContent>
    </Card>
  )
}
