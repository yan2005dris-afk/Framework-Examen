package print

import "github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"

var labels = map[string]string{
	"cedula": "Cédula", "genero": "Género", "nacionalidad": "Nacionalidad declarada", "verificadoRegistroCivil": "Indicador legado de Registro Civil (no certifica verificación)",
	"pais": "País", "provincia": "Provincia", "canton": "Cantón", "numero": "Número", "correo1": "Correo principal", "correo2": "Correo adicional", "institucion": "Institución", "area": "Área", "titulo": "Título", "titulos": "Títulos",
	"verificacionIessCumplida": "Indicador de elegibilidad conservado en el envío", "detalleVerificacionIess": "Origen y detalle de elegibilidad conservados", "registroSenescyt": "Registro SENESCYT", "fechaRegistroSenescyt": "Fecha de registro SENESCYT", "documentoPdf": "Respaldo PDF", "documentoRespaldoPdf": "Respaldo de organización", "id": "Identificador", "sha256": "SHA-256 del documento",
	"anios": "Años", "meses": "Meses", "dias": "Días", "duracionCalculada": "Duración almacenada al envío", "descripcionFunciones": "Descripción de funciones", "capacitaciones": "Capacitaciones",
	"version": "Versión de los literales", "puntos": "Literales consentidos y conservados al envío", "aceptadoEn": "Aceptación fechada al envío", "firmaManuscritaDataUrl": "Firma legada", "aceptacionTratamientoDatos": "Aceptación del tratamiento de datos", "declaracionVeracidad": "Declaración de veracidad", "aceptacionNormasConcurso": "Aceptación de normas del concurso", "autorizacionPublicacionNombre": "Autorización de publicación del nombre", "constanciaEntregaFisica": "Constancia de entrega física",
	"q1": "1.9.1 - Nacionalidad y derechos de participación (respuesta positiva)", "q2": "1.9.2 - Inhabilitación para ejercer comercio", "q3": "1.9.3 - Mora con el Estado o instituciones controladas", "q4": "1.9.4 - Vínculo con sistema financiero, seguros o bolsa", "q5": "1.9.5 - Funcionario o empleado IESS/BIESS", "q6": "1.9.6 - Interés en aseguradoras o seguridad social", "q7": "1.9.7 - Remoción por organismo de control", "q7Detalles": "Detalles declarados de la remoción", "q8": "1.9.8 - Obligaciones pendientes SRI", "q9": "1.9.9 - Castigo de obligaciones", "q10": "1.9.10 - Litigio contra IESS/BIESS", "q11": "1.9.11 - Procesamiento, enjuiciamiento o condena", "q12": "1.9.12 - Responsabilidad en firme CGE", "q13": "1.9.13 - Registro UAFE",
}

func init() {
	definition, err := submission.FormDefinition()
	if err == nil {
		for _, question := range definition.Questions {
			labels[question.ID] = "1.9." + question.ID[1:] + " - " + question.Text
		}
	}
}
