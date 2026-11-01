import { afterEach, describe, expect, it, vi } from "vitest"
import {
  listarNotificacionesAdmin,
  listarNotificacionesPostulante,
  obtenerHistorialNotificacionAdmin,
  obtenerHistorialNotificacionPostulante,
  reintentarNotificacionAdmin,
  type EntradaReintentoNotificacionDto,
  type NotificacionEtapaDto,
} from "./notificaciones"

describe("Cliente de Notificaciones de Etapa (notificaciones.ts - FT22)", () => {
  const fetchOriginal = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = fetchOriginal
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  const mockNotificacion: NotificacionEtapaDto = {
    id: "notif-1",
    accion: "resultado.publicado",
    plantilla: "resultado-etapa-v2",
    estado: "smtp_accepted",
    intentosCiclo: 1,
    intentosTotales: 1,
    reenvios: 0,
    creadaEn: "2026-10-31T20:00:00Z",
    actualizadaEn: "2026-10-31T20:01:00Z",
  }

  describe("listarNotificacionesPostulante y obtenerHistorialNotificacionPostulante", () => {
    it("consume endpoints del postulante para consultar estado de avisos", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockNotificacion],
          hasMore: false,
          limit: 10,
        }),
      })

      const respuesta = await listarNotificacionesPostulante("sub-1")
      expect(respuesta.items).toHaveLength(1)
      expect(respuesta.items[0].estado).toBe("smtp_accepted")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/notifications/submissions/sub-1"),
        expect.anything(),
      )

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [
            {
              id: "ev-1",
              estado: "smtp_accepted",
              intento: 1,
              ciclo: 1,
              fecha: "2026-10-31T20:01:00Z",
            },
          ],
          hasMore: false,
          limit: 10,
        }),
      })

      const historial = await obtenerHistorialNotificacionPostulante(
        "sub-1",
        "notif-1",
      )
      expect(historial.items).toHaveLength(1)
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/notifications/submissions/sub-1/notif-1/history",
        ),
        expect.anything(),
      )
    })
  })

  describe("listarNotificacionesAdmin y obtenerHistorialNotificacionAdmin", () => {
    it("consume endpoints bajo /admin/notifications/", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          items: [mockNotificacion],
          hasMore: false,
          limit: 10,
        }),
      })

      await listarNotificacionesAdmin("sub-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining("/admin/notifications/submissions/sub-1"),
        expect.anything(),
      )

      await obtenerHistorialNotificacionAdmin("sub-1", "notif-1")
      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/notifications/submissions/sub-1/notif-1/history",
        ),
        expect.anything(),
      )
    })
  })

  describe("reintentarNotificacionAdmin", () => {
    it("envía petición POST para reintento con reenviosAnteriores y motivo", async () => {
      const entrada: EntradaReintentoNotificacionDto = {
        reenviosAnteriores: 0,
        motivo: "Se restableció el servidor SMTP institucional",
      }

      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "content-type": "application/json" }),
        json: async () => ({
          ...mockNotificacion,
          estado: "pending",
          reenvios: 1,
        }),
      })

      const res = await reintentarNotificacionAdmin("sub-1", "notif-1", entrada)
      expect(res.reenvios).toBe(1)
      expect(res.estado).toBe("pending")

      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining(
          "/admin/notifications/submissions/sub-1/notif-1/retry",
        ),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify(entrada),
        }),
      )
    })
  })
})
