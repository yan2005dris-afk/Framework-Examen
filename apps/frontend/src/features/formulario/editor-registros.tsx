import { useId, useState, type ReactNode } from "react"
import { XIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { AdjuntoUploader, type AdjuntoInfo } from "./adjunto-uploader"
import { confirmarEdicionRegistro } from "./modelo"

interface Registro {
  id: string
  documentoPdf?: AdjuntoInfo | null
}
export interface CampoRegistro<T> {
  nombre: Extract<keyof T, string>
  etiqueta: string
  tipo?: "texto" | "fecha" | "numero" | "area" | "opcion" | "casilla"
  opciones?: readonly string[]
  deshabilitado?: (registro: T) => boolean
  alCambiar?: (registro: T, valor: string | number | boolean) => T
}
interface EditorRegistrosProps<T extends Registro> {
  titulo: string
  singular: string
  registros: T[]
  crear: () => T
  campos: CampoRegistro<T>[]
  columnas: { titulo: string; contenido: (registro: T) => ReactNode }[]
  onChange: (registros: T[]) => void
  validar: (registro: T) => string | null
  disabled?: boolean
  onSubirRespaldo?: (id: string, archivo: File) => Promise<void>
  onEliminarRespaldo?: (id: string) => Promise<void>
  onDescargarRespaldo?: (id: string) => void
  cargandoRespaldoId?: string | null
}

/** Tablas y diálogos del formulario: Cancelar no escribe datos ni ejecuta operaciones de documentos. */
export function EditorRegistros<T extends Registro>({
  titulo,
  singular,
  registros,
  crear,
  campos,
  columnas,
  onChange,
  validar,
  disabled = false,
  onSubirRespaldo,
  onEliminarRespaldo,
  onDescargarRespaldo,
  cargandoRespaldoId,
}: EditorRegistrosProps<T>) {
  const prefijo = useId()
  const [edicion, setEdicion] = useState<{
    original: T | null
    borrador: T
  } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [anexoId, setAnexoId] = useState<string | null>(null)
  const [baja, setBaja] = useState<T | null>(null)
  const anexo = registros.find((item) => item.id === anexoId)
  const subiendo = Boolean(anexoId && cargandoRespaldoId === anexoId)
  function abrir(registro?: T) {
    setError(null)
    setEdicion({
      original: registro ? structuredClone(registro) : null,
      borrador: registro ? structuredClone(registro) : crear(),
    })
  }
  function cambiar(campo: CampoRegistro<T>, valor: string | number | boolean) {
    if (!edicion || disabled) return
    const siguiente = campo.alCambiar
      ? campo.alCambiar(edicion.borrador, valor)
      : { ...edicion.borrador, [campo.nombre]: valor }
    setEdicion({ ...edicion, borrador: siguiente })
    setError(null)
  }
  function guardar() {
    if (!edicion || disabled) return
    const mensaje = validar(edicion.borrador)
    if (mensaje) {
      setError(mensaje)
      return
    }
    try {
      onChange(
        confirmarEdicionRegistro(registros, edicion.borrador, edicion.original),
      )
      setEdicion(null)
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "No se pudo actualizar el registro.",
      )
    }
  }
  return (
    <section className="min-w-0 space-y-4" aria-label={titulo}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">
          {titulo} ({registros.length})
        </h3>
        <Button type="button" onClick={() => abrir()} disabled={disabled}>
          Agregar {singular}
        </Button>
      </div>
      <div className="relative overflow-x-auto rounded-lg border">
        <table className="w-full min-w-[640px] text-left text-[13px]">
          <caption className="sr-only">{titulo}</caption>
          <thead>
            <tr>
              {columnas.map((c) => (
                <th key={c.titulo} scope="col">
                  {c.titulo}
                </th>
              ))}
              <th scope="col">Respaldo PDF</th>
              <th scope="col">Acciones</th>
            </tr>
          </thead>
          <tbody>
            {registros.length ? (
              registros.map((registro) => (
                <tr key={registro.id} className="border-t align-top">
                  {columnas.map((c) => (
                    <td key={c.titulo}>{c.contenido(registro)}</td>
                  ))}
                  <td>
                    <Button
                      type="button"
                      variant="link"
                      className="h-auto max-w-44 justify-start whitespace-normal px-0 text-left"
                      onClick={() => setAnexoId(registro.id)}
                    >
                      {registro.documentoPdf?.nombre ??
                        (disabled ? "Sin respaldo" : "Adjuntar PDF")}
                    </Button>
                  </td>
                  <td>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => abrir(registro)}
                        aria-label={`${disabled ? "Ver" : "Editar"} ${singular}`}
                      >
                        {disabled ? "Ver" : "Editar"}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        disabled={disabled}
                        className="text-destructive"
                        onClick={() => setBaja(structuredClone(registro))}
                        aria-label={`Eliminar ${singular}`}
                      >
                        Eliminar
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={columnas.length + 2}
                  className="py-8 text-center text-muted-foreground"
                >
                  No hay registros.{" "}
                  {disabled
                    ? ""
                    : `Seleccione «Agregar ${singular}» para comenzar.`}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-muted-foreground">
        Los registros se incorporan al formulario al confirmar. Use «Guardar
        borrador» para conservar la sección en el servidor.
      </p>
      <Dialog
        open={Boolean(edicion)}
        onOpenChange={(abierto) => {
          if (!abierto) setEdicion(null)
        }}
      >
        <DialogContent
          showCloseButton={false}
          className="max-h-[90svh] overflow-y-auto sm:max-w-3xl"
        >
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="absolute top-2 right-2"
            aria-label="Cerrar edición"
            onClick={() => setEdicion(null)}
          >
            <XIcon />
          </Button>
          <DialogHeader>
            <DialogTitle>
              {disabled ? "Ver" : edicion?.original ? "Editar" : "Agregar"}{" "}
              {singular}
            </DialogTitle>
            <DialogDescription>
              Complete los datos del registro. Cancelar descarta los cambios de
              este diálogo; no modifica el registro anterior.
            </DialogDescription>
          </DialogHeader>
          <form
            noValidate
            onSubmit={(e) => {
              e.preventDefault()
              guardar()
            }}
            className="space-y-5"
          >
            {error && (
              <p
                role="alert"
                className="rounded-md border border-destructive/30 bg-destructive-soft p-3 text-sm text-destructive"
              >
                {error}
              </p>
            )}
            <div className="grid gap-5 sm:grid-cols-2">
              {edicion &&
                campos.map((campo) => {
                  const id = `${prefijo}-${campo.nombre}`
                  const valor = edicion.borrador[campo.nombre]
                  const bloqueado =
                    disabled || Boolean(campo.deshabilitado?.(edicion.borrador))
                  return (
                    <div
                      key={campo.nombre}
                      className={
                        campo.tipo === "area"
                          ? "space-y-2 sm:col-span-2"
                          : "space-y-2"
                      }
                    >
                      <Label htmlFor={id}>{campo.etiqueta}</Label>
                      {campo.tipo === "opcion" ? (
                        <Select
                          value={String(valor ?? "")}
                          disabled={bloqueado}
                          onValueChange={(v) => cambiar(campo, v)}
                        >
                          <SelectTrigger
                            id={id}
                            aria-required={campo.etiqueta.includes("*")}
                          >
                            <SelectValue placeholder="Seleccione una opción" />
                          </SelectTrigger>
                          <SelectContent>
                            {campo.opciones?.map((v) => (
                              <SelectItem key={v} value={v}>
                                {v}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      ) : campo.tipo === "casilla" ? (
                        <Checkbox
                          id={id}
                          checked={Boolean(valor)}
                          disabled={bloqueado}
                          onCheckedChange={(v) => cambiar(campo, v === true)}
                        />
                      ) : campo.tipo === "area" ? (
                        <Textarea
                          id={id}
                          value={String(valor ?? "")}
                          maxLength={1000}
                          rows={5}
                          aria-required={campo.etiqueta.includes("*")}
                          disabled={bloqueado}
                          onChange={(e) => cambiar(campo, e.target.value)}
                        />
                      ) : (
                        <Input
                          id={id}
                          aria-required={campo.etiqueta.includes("*")}
                          type={
                            campo.tipo === "fecha"
                              ? "date"
                              : campo.tipo === "numero"
                                ? "number"
                                : "text"
                          }
                          min={campo.tipo === "numero" ? 1 : undefined}
                          step={campo.tipo === "numero" ? 1 : undefined}
                          value={
                            campo.tipo === "numero" && !valor
                              ? ""
                              : String(valor ?? "")
                          }
                          disabled={bloqueado}
                          onChange={(e) =>
                            cambiar(
                              campo,
                              campo.tipo === "numero"
                                ? Number(e.target.value)
                                : e.target.value,
                            )
                          }
                        />
                      )}
                    </div>
                  )
                })}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setEdicion(null)}
              >
                {disabled ? "Cerrar" : "Cancelar"}
              </Button>
              {!disabled && <Button type="submit">Guardar registro</Button>}
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(anexo)}
        onOpenChange={(abierto) => {
          if (!abierto && !subiendo) setAnexoId(null)
        }}
      >
        <DialogContent
          showCloseButton={false}
          className="max-h-[90svh] overflow-y-auto sm:max-w-xl"
          onEscapeKeyDown={(e) => {
            if (subiendo) e.preventDefault()
          }}
          onInteractOutside={(e) => {
            if (subiendo) e.preventDefault()
          }}
        >
          <DialogHeader>
            <DialogTitle>Respaldo de {singular}</DialogTitle>
            <DialogDescription>
              PDF de hasta 2 MB. La carga y la eliminación confirman cambios en
              el servidor; cerrar no deshace una operación ya confirmada.
            </DialogDescription>
          </DialogHeader>
          {anexo && (
            <AdjuntoUploader
              maxMB={2}
              valor={anexo.documentoPdf}
              disabled={disabled || !onSubirRespaldo}
              cargando={subiendo}
              onArchivoSeleccionado={(_, archivo) => {
                if (archivo && onSubirRespaldo)
                  void onSubirRespaldo(anexo.id, archivo)
              }}
              onEliminar={() => {
                if (onEliminarRespaldo) void onEliminarRespaldo(anexo.id)
              }}
              onDescargar={
                anexo.documentoPdf?.id && onDescargarRespaldo
                  ? () => onDescargarRespaldo(anexo.documentoPdf!.id!)
                  : undefined
              }
            />
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={subiendo}
              onClick={() => setAnexoId(null)}
            >
              Cerrar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(baja)}
        onOpenChange={(abierto) => {
          if (!abierto) setBaja(null)
        }}
      >
        <DialogContent
          showCloseButton={false}
          className="max-h-[90svh] overflow-y-auto"
        >
          <DialogHeader>
            <DialogTitle>Eliminar {singular}</DialogTitle>
            <DialogDescription>
              Se quitará este registro del formulario local. Guardar borrador
              confirma el cambio. Esta acción no elimina directamente el PDF del
              servidor.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setBaja(null)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={disabled}
              onClick={() => {
                if (baja && !disabled) {
                  onChange(registros.filter((r) => r.id !== baja.id))
                  setBaja(null)
                }
              }}
            >
              Eliminar registro
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  )
}
