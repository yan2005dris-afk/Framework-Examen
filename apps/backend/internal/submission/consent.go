package submission

import "time"

const consentVersion = "v1.4"

/** normalizeDeclarations publica los textos al guardar y elimina certificaciones del cliente.
 * @param {Form} form - Borrador; la aceptación efectiva se fecha únicamente al sellar.
 */
func normalizeDeclarations(form *Form) {
	if form.Personal != nil {
		form.Personal.CivilRegistryVerified = false
	}
	if form.Consent == nil {
		form.Consent = &Consent{}
	}
	form.Consent.Version = consentVersion
	form.Consent.Statements = append([]string(nil), consentStatements...)
	form.Consent.AcceptedAt = ""
}

/** Literales del RF v1.4, EF-12 1.10.1; se incorporan al contenido sellado. */
var consentStatements = []string{
	"1. Autorizo de manera libre y voluntaria, emitiendo consentimiento explícito al Instituto Ecuatoriano de Seguridad Social, para el uso de mis datos personales dentro del concurso de méritos y oposición para la selección de los Miembros Principales y Alternos del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social (BIESS), en Representación de los Afiliados, Jubilados y Empleadores, de conformidad con lo prescrito en el artículo 7 de la Ley Orgánica de Protección de Datos Personales.",
	"2. Declaro que la información del presente formulario y documentación adjunta corresponde a la verdad y autorizo al Instituto Ecuatoriano de Seguridad Social para que se compruebe ante cualquier organismo público o privado su veracidad.",
	"3. Acepto expresamente cumplir con las normas aplicables al concurso de méritos y oposición para la selección de los Miembros Principales y Alternos del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social (BIESS), en Representación de los Afiliados, Jubilados y Empleadores; así como las resoluciones y disposiciones impartidas por el Instituto Ecuatoriano de Seguridad Social. En caso de incumplimiento de requisitos o encontrarme inmerso/a en las prohibiciones o inhabilidades acepto mi descalificación del proceso.",
	"4. Acepto que mi nombre como postulante al concurso de méritos y oposición para la selección de los Miembros Principales y Alternos del Directorio del Banco del Instituto Ecuatoriano de Seguridad Social (BIESS), en Representación de los Afiliados, Jubilados y Empleadores; sea publicado en la página web del IESS - BIESS.",
	"5. El presente formulario, debe ser impreso y con firma manuscrita, entregado junto con los requisitos solicitados, en el lugar, fecha y horario establecido por el Instituto Ecuatoriano de Seguridad Social.",
}

/** AllowsNamePublication reconoce el literal de nombre efectivamente aceptado en el sello, sin extenderlo a contactos, notas o documentos. */
func AllowsNamePublication(c *Consent) bool {
	if c == nil || !c.NamePublication || c.Version != consentVersion || len(c.Statements) != len(consentStatements) || c.Statements[3] != consentStatements[3] {
		return false
	}
	accepted, err := time.Parse(time.RFC3339Nano, c.AcceptedAt)
	return err == nil && !accepted.IsZero()
}
