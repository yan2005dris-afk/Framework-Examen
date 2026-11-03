import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { toast } from "sonner"
import { useSesion } from "@/features/auth/sesion-context"
import { esErrorApi } from "@/lib/api-error"
import {
  configurarVersionBanco,
  crearVersionBancoPreguntas,
  finalizarExamen,
  habilitarPostulanteExamen,
  iniciarExamen,
  listarVersionesBancoPreguntas,
  obtenerEstadoExamen,
  obtenerPreguntaExamen,
  obtenerVersionBancoPreguntas,
  responderPreguntaExamen,
  type EntradaBancoPreguntasDto,
  type EntradaHabilitacionExamenDto,
  type EntradaResponderExamenDto,
  type ReglasConfiguracionExamenDto,
} from "@/lib/clientes/preguntas"

const PRIVADO = {
  gcTime: 0,
  staleTime: 0,
  refetchOnWindowFocus: "always" as const,
}

export function useCuentaPreguntas() {
  const { postulante } = useSesion()
  return `${postulante?.id ?? "sin-sesion"}:${postulante?.rol ?? "postulante"}`
}

export const clavesPreguntas = {
  cuenta: (cuenta: string) => ["preguntas-privada", cuenta] as const,
  bancos: (cuenta: string, period: string, sector: string) =>
    [...clavesPreguntas.cuenta(cuenta), "bancos", period, sector] as const,
  bancoDetalle: (cuenta: string, id: string) =>
    [...clavesPreguntas.cuenta(cuenta), "banco-detalle", id] as const,
  examenEstado: (cuenta: string, subId: string) =>
    [...clavesPreguntas.cuenta(cuenta), "examen-estado", subId] as const,
  examenPregunta: (cuenta: string, subId: string, index: number) =>
    [
      ...clavesPreguntas.cuenta(cuenta),
      "examen-pregunta",
      subId,
      index,
    ] as const,
}

/**
 * Consulta de versiones históricas del banco de preguntas para administrador (FT24).
 */
export function useVersionesBanco(period: string, sector: string) {
  const cuenta = useCuentaPreguntas()
  return useInfiniteQuery({
    ...PRIVADO,
    queryKey: clavesPreguntas.bancos(cuenta, period, sector),
    queryFn: ({ pageParam, signal }) =>
      listarVersionesBancoPreguntas(
        period,
        sector,
        { cursor: pageParam },
        signal,
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (pagina) =>
      pagina.hasMore ? pagina.nextCursor : undefined,
    enabled: Boolean(period && sector),
  })
}

/**
 * Consulta del banco completo con matriz de respuestas auditada para Administrador (FT24).
 */
export function useVersionBancoDetalle(id: string) {
  const cuenta = useCuentaPreguntas()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesPreguntas.bancoDetalle(cuenta, id),
    queryFn: ({ signal }) => obtenerVersionBancoPreguntas(id, signal),
    enabled: Boolean(id),
  })
}

/**
 * Mutación para crear una nueva versión inmutable del banco de preguntas (FT24).
 */
export function useCrearVersionBanco(period: string, sector: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaBancoPreguntasDto) =>
      crearVersionBancoPreguntas(period, sector, entrada),
    onSuccess: () => {
      toast.success("Nueva versión del banco de preguntas registrada.")
      void queryClient.invalidateQueries({
        queryKey: ["preguntas-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo registrar la versión del banco de preguntas."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para aprobar y congelar los parámetros de evaluación técnica (FT25).
 */
export function useConfigurarBanco(bankId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (reglas: ReglasConfiguracionExamenDto) =>
      configurarVersionBanco(bankId, reglas),
    onSuccess: () => {
      toast.success("Parámetros y cronograma de evaluación aprobados.")
      void queryClient.invalidateQueries({
        queryKey: ["preguntas-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No se pudo aprobar la configuración de la evaluación."
      toast.error(msg)
    },
  })
}

/**
 * Mutación para habilitar individualmente a un postulante que superó méritos (FT25).
 */
export function useHabilitarPostulanteExamen(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaHabilitacionExamenDto) =>
      habilitarPostulanteExamen(submissionId, entrada),
    onSuccess: () => {
      toast.success("Postulante habilitado para la fase de evaluación técnica.")
      void queryClient.invalidateQueries({
        queryKey: ["preguntas-privada"],
      })
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No fue posible habilitar al postulante para la evaluación."
      toast.error(msg)
    },
  })
}

/**
 * Consulta de estado, reloj y progreso del examen del postulante (FT25-FT26).
 */
export function useEstadoExamen(submissionId: string, habilitado = true) {
  const cuenta = useCuentaPreguntas()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesPreguntas.examenEstado(cuenta, submissionId),
    queryFn: ({ signal }) => obtenerEstadoExamen(submissionId, signal),
    enabled: Boolean(submissionId && habilitado),
  })
}

/**
 * Inicia o recupera el intento oficial persistido (FT26).
 */
export function useIniciarExamen(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => iniciarExamen(submissionId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["preguntas-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No fue posible iniciar el examen."
      toast.error(msg)
    },
  })
}

/**
 * Consulta de una pregunta por índice (FT26).
 */
export function usePreguntaExamen(
  submissionId: string,
  index: number,
  habilitado = true,
) {
  const cuenta = useCuentaPreguntas()
  return useQuery({
    ...PRIVADO,
    queryKey: clavesPreguntas.examenPregunta(cuenta, submissionId, index),
    queryFn: ({ signal }) => obtenerPreguntaExamen(submissionId, index, signal),
    enabled: Boolean(submissionId && index >= 1 && habilitado),
  })
}

/**
 * Guarda una respuesta individual (FT26).
 */
export function useResponderExamen(submissionId: string) {
  const cuenta = useCuentaPreguntas()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (entrada: EntradaResponderExamenDto) =>
      responderPreguntaExamen(submissionId, entrada),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: clavesPreguntas.examenEstado(cuenta, submissionId),
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "Error al guardar respuesta."
      toast.error(msg)
    },
  })
}

/**
 * Finaliza y califica el examen (FT26-FT27).
 */
export function useFinalizarExamen(submissionId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => finalizarExamen(submissionId),
    onSuccess: () => {
      toast.success("Evaluación finalizada y calificada con éxito.")
      void queryClient.invalidateQueries({
        queryKey: ["preguntas-privada"],
      })
      void queryClient.invalidateQueries({
        queryKey: ["evaluacion-privada"],
      })
    },
    onError: (error) => {
      const msg = esErrorApi(error)
        ? error.message
        : "No fue posible finalizar la evaluación."
      toast.error(msg)
    },
  })
}
