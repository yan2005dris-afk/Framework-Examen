import { useState } from "react"
import { FileSpreadsheetIcon, ListOrderedIcon } from "lucide-react"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { GestionListasAdmin } from "@/features/resultados/gestion-listas-admin"
import { PanelRevision } from "@/features/revision/panel-revision"

export function DashboardReportesView() {
  const [tab, setTab] = useState<string>("matriz")

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold tracking-tight text-foreground">
          Reportes y Listas de Elegibles
        </h1>
        <p className="text-sm text-muted-foreground">
          Gestión de la matriz técnica preliminar y de las listas oficiales de
          elegibles por convocatoria y sector.
        </p>
      </div>

      <Tabs value={tab} onValueChange={setTab} className="w-full space-y-6">
        <TabsList className="grid w-full max-w-md grid-cols-2">
          <TabsTrigger value="matriz" className="flex items-center gap-2">
            <FileSpreadsheetIcon className="size-4" />
            <span>Matriz técnica</span>
          </TabsTrigger>
          <TabsTrigger value="listas" className="flex items-center gap-2">
            <ListOrderedIcon className="size-4" />
            <span>Listas de elegibles</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="matriz" className="space-y-4">
          <PanelRevision vista="reportes" />
        </TabsContent>

        <TabsContent value="listas" className="space-y-4">
          <GestionListasAdmin />
        </TabsContent>
      </Tabs>
    </div>
  )
}
