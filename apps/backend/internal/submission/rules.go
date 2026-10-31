package submission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var phonePattern = regexp.MustCompile(`^\+?[0-9 ()-]{7,20}$`)

var studyAreas = formDefinition.Options["areas"]

/**
 * DecodeForm rechaza tipos y campos desconocidos, sin incluir datos personales
 * en los errores. Solo admite un objeto JSON y una única raíz.
 * @param {json.RawMessage} content - Contenido recibido.
 * @returns {Form, error} Formulario tipado o error de estructura.
 */
func DecodeForm(content json.RawMessage) (*Form, error) {
	if _, err := CanonicalContent(content); err != nil {
		return nil, err
	}
	var form Form
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&form); err != nil {
		return nil, ErrInvalidContent
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return nil, ErrInvalidContent
	}
	return &form, nil
}

/**
 * ValidateForm distingue un borrador incompleto de valores inválidos. En el
 * envío exige las nueve secciones completas y las cinco aceptaciones.
 * @param {Form} form - Formulario a validar.
 * @param {bool} complete - Exige todos los campos al enviar.
 * @param {time.Time} now - Fecha de referencia de las reglas.
 * @returns {error} Primera regla incumplida.
 */
func ValidateForm(form *Form, complete bool, now time.Time) error {
	if definitionError != nil {
		return definitionError
	}
	if form == nil {
		return ErrInvalidContent
	}
	for section := 1; section <= 9; section++ {
		if err := validateSection(form, section, complete, now); err != nil {
			return err
		}
	}
	if complete && (form.Consent == nil || !form.Consent.DataProcessing || !form.Consent.Truthfulness || !form.Consent.CompetitionRules || !form.Consent.NamePublication || !form.Consent.PhysicalDelivery) {
		return rule("consentimiento", "debe aceptar los cinco puntos del consentimiento")
	}
	return nil
}

/**
 * validateSection aplica las reglas de una sección, sin confiar en banderas
 * de completitud enviadas por el cliente.
 * @returns {error} Regla incumplida con ruta estable del campo.
 */
func validateSection(f *Form, section int, full bool, now time.Time) error {
	path := fmt.Sprintf("seccion%d", section)
	if err := validateRecordIDs(sectionRecordIDs(f, section), full, path); err != nil {
		return err
	}
	missing := func() error {
		if full {
			return rule(path, "la sección es obligatoria")
		}
		return nil
	}
	check := func(err error) error {
		if err != nil {
			return rule(path, err.Error())
		}
		return nil
	}
	switch section {
	case 1:
		s := f.Personal
		if s == nil {
			return missing()
		}
		if s.Nationality != "" && s.Nationality != "Ecuatoriana" {
			return rule(path+".nacionalidad", "la nacionalidad debe ser ecuatoriana; no puede continuar con la postulación")
		}
		if err := required(full, s.FirstName, s.LastName, s.IDCard, s.Gender, s.BirthDate, s.Nationality, s.MaritalStatus); err != nil {
			return check(err)
		}
		if err := choice(s.Gender, formDefinition.Options["generos"]...); err != nil {
			return check(err)
		}
		if err := choice(s.MaritalStatus, formDefinition.Options["estadosCiviles"]...); err != nil {
			return check(err)
		}
		if err := validID(s.IDCard); err != nil {
			return check(err)
		}
		if err := validDate(s.BirthDate, now); err != nil {
			return check(err)
		}
		if s.MaritalStatus == "Casado/a" || s.MaritalStatus == "Unión de hecho" {
			if err := required(full, s.SpouseName, s.SpouseIDCard); err != nil {
				return check(err)
			}
			if err := validID(s.SpouseIDCard); err != nil {
				return check(err)
			}
		}
	case 2:
		s := f.Contact
		if s == nil {
			return missing()
		}
		if err := required(full, s.Country, s.Province, s.City, s.Mobile, s.Email, s.MainStreet, s.Number, s.SecondaryStreet, s.Area, s.Reference, s.Phone); err != nil {
			return check(err)
		}
		if err := choice(s.Country, formDefinition.Options["paises"]...); err != nil {
			return check(err)
		}
		for _, value := range []string{s.Country, s.Province, s.City} {
			if err := placeName(value); err != nil {
				return check(err)
			}
		}
		if s.Province != "" && s.Country == "" || s.City != "" && s.Province == "" {
			return rule(path, "seleccione primero el país y la provincia")
		}
		if s.Country == "Ecuador" {
			cities, exists := ecuadorProvinces[s.Province]
			if s.Province != "" && !exists {
				return rule(path+".provincia", "la provincia no existe en el catálogo de Ecuador")
			}
			if s.City != "" && !slices.Contains(cities, s.City) {
				return rule(path+".canton", "el cantón no pertenece a la provincia seleccionada")
			}
		}
		for _, email := range []string{s.Email, s.SecondaryEmail} {
			if email != "" && !emailPattern.MatchString(email) {
				return rule(path, "el correo no tiene un formato válido")
			}
		}
		for _, phone := range []string{s.Mobile, s.Phone} {
			if phone != "" && !phonePattern.MatchString(phone) {
				return rule(path, "el teléfono no tiene un formato válido")
			}
		}
	case 3:
		s := f.Additional
		if s == nil {
			return missing()
		}
		if err := required(full, s.HasRelatives); err != nil {
			return check(err)
		}
		if err := yesNo(s.HasRelatives); err != nil {
			return check(err)
		}
		if s.HasRelatives == "No" && len(s.Relatives) > 0 {
			return rule(path, "la respuesta No no admite familiares")
		}
		if full && s.HasRelatives == "Sí" && len(s.Relatives) == 0 {
			return rule(path, "debe agregar al menos un familiar")
		}
		for _, r := range s.Relatives {
			if err := required(full, r.Name, r.Relationship, r.Institution, r.Area, r.Position); err != nil {
				return check(err)
			}
			if err := choice(r.Relationship, formDefinition.Options["parentescos"]...); err != nil {
				return check(err)
			}
			if err := choice(r.Institution, formDefinition.Options["instituciones"]...); err != nil {
				return check(err)
			}
		}
	case 4:
		s := f.Position
		if s == nil {
			return missing()
		}
		if err := choice(s.Sector, formDefinition.Options["sectores"]...); err != nil {
			return check(err)
		}
		if err := required(full, s.Sector, s.HasOrganization); err != nil {
			return check(err)
		}
		if err := yesNo(s.HasOrganization); err != nil {
			return check(err)
		}
		if s.Eligible != nil && !*s.Eligible && s.Sector != "" {
			return rule(path, ineligibleMessage(s.Sector))
		}
		if full && (s.Eligible == nil || !*s.Eligible) {
			return rule(path, "debe completar la verificación de elegibilidad del sector")
		}
		if s.HasOrganization == "Sí" {
			if err := required(full, s.Organization); err != nil {
				return check(err)
			}
		}
	case 5:
		s := f.Education
		if s == nil {
			return missing()
		}
		if full && len(s.Degrees) == 0 {
			return rule(path, "debe agregar al menos un título académico")
		}
		for _, d := range s.Degrees {
			if err := required(full, d.Level, d.Institution, d.Type, d.Title, d.Area, d.Registry, d.RegistryDate); err != nil {
				return check(err)
			}
			if err := choice(d.Level, formDefinition.Options["niveles"]...); err != nil {
				return check(err)
			}
			if err := choice(d.Type, formDefinition.Options["tiposInstitucion"]...); err != nil {
				return check(err)
			}
			if err := choice(d.Area, studyAreas...); err != nil {
				return check(err)
			}
			if err := validDate(d.RegistryDate, now); err != nil {
				return check(err)
			}
		}
	case 6:
		s := f.Experience
		if s == nil {
			return missing()
		}
		if full && len(s.Experiences) == 0 {
			return rule(path, "debe agregar al menos una experiencia profesional")
		}
		for _, e := range s.Experiences {
			if err := required(full, e.Type, e.Position, e.Institution, e.StartDate, e.Activities, e.Functions); err != nil {
				return check(err)
			}
			if !e.Current {
				if err := required(full, e.EndDate); err != nil {
					return check(err)
				}
			}
			if err := choice(e.Type, formDefinition.Options["tiposExperiencia"]...); err != nil {
				return check(err)
			}
			if err := choice(e.Activities, studyAreas...); err != nil {
				return check(err)
			}
			if utf8.RuneCountInString(e.Functions) > 1000 {
				return rule(path+".descripcionFunciones", "las funciones admiten máximo 1000 caracteres")
			}
			if err := dateRange(e.StartDate, e.EndDate, e.Current, now); err != nil {
				return check(err)
			}
		}
	case 7:
		s := f.Publications
		if s == nil {
			return missing()
		}
		if err := required(full, s.HasPublications); err != nil {
			return check(err)
		}
		if err := yesNo(s.HasPublications); err != nil {
			return check(err)
		}
		if s.HasPublications == "No" && len(s.Publications) > 0 {
			return rule(path, "la respuesta No no admite publicaciones")
		}
		if full && s.HasPublications == "Sí" && len(s.Publications) == 0 {
			return rule(path, "debe agregar al menos una publicación")
		}
		for _, p := range s.Publications {
			if err := required(full, p.Title, p.Type, p.Medium, p.Date, p.Related); err != nil {
				return check(err)
			}
			if err := choice(p.Type, formDefinition.Options["tiposPublicacion"]...); err != nil {
				return check(err)
			}
			if err := yesNo(p.Related); err != nil {
				return check(err)
			}
			if err := validDate(p.Date, now); err != nil {
				return check(err)
			}
		}
	case 8:
		s := f.Training
		if s == nil {
			return missing()
		}
		if full && s.Events == nil {
			return rule(path, "declare la lista de capacitaciones, aunque esté vacía")
		}
		for _, e := range s.Events {
			if err := required(full, e.Type, e.Name, e.Institution, e.StartDate, e.EndDate); err != nil {
				return check(err)
			}
			if err := choice(e.Type, formDefinition.Options["tiposEvento"]...); err != nil {
				return check(err)
			}
			if e.Hours < 0 || full && e.Hours == 0 {
				return rule(path, "las horas deben ser mayores que cero")
			}
			if err := dateRange(e.StartDate, e.EndDate, false, now); err != nil {
				return check(err)
			}
		}
	case 9:
		s := f.Disqualifications
		if s == nil {
			return missing()
		}
		for i, answer := range []string{s.Q1, s.Q2, s.Q3, s.Q4, s.Q5, s.Q6, s.Q7, s.Q8, s.Q9, s.Q10, s.Q11, s.Q12, s.Q13} {
			if full && answer == "" {
				return rule(fmt.Sprintf("%s.q%d", path, i+1), "las trece respuestas son obligatorias")
			}
			if err := yesNo(answer); err != nil {
				return check(err)
			}
		}
		if s.Q7 == "Sí" {
			if s.Q7Details == nil {
				if full {
					return rule(path+".q7Detalles", "debe indicar institución, cargo, fecha de fin y motivo")
				}
				return nil
			}
			d := s.Q7Details
			if err := required(full, d.Institution, d.Position, d.EndDate, d.Reason); err != nil {
				return check(err)
			}
			if err := validDate(d.EndDate, now); err != nil {
				return check(err)
			}
		} else if s.Q7Details != nil {
			return rule(path+".q7Detalles", "los detalles solo corresponden a una respuesta Sí")
		}
	default:
		return ErrInvalidSection
	}
	return nil
}

/** rule construye un error de negocio sin incluir valores del postulante. */
func rule(path, message string) error {
	return fmt.Errorf("%w: %s: %s", ErrBusinessRule, path, message)
}

/** required exige textos no vacíos únicamente al completar la sección. */
func required(full bool, values ...string) error {
	for _, value := range values {
		if full && strings.TrimSpace(value) == "" {
			return fmt.Errorf("complete los campos obligatorios")
		}
		if strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("el texto contiene caracteres no permitidos")
		}
	}
	return nil
}

/** choice valida un valor presente contra las opciones literales del RF. */
func choice(value string, choices ...string) error {
	if value != "" && !slices.Contains(choices, value) {
		return fmt.Errorf("seleccione una opción válida")
	}
	return nil
}

/**
 * yesNo valida las dos opciones de una respuesta presente.
 * @param {string} value - Respuesta del formulario.
 * @returns {error} Error si difiere de Sí o No.
 */
func yesNo(value string) error { return choice(value, formDefinition.Options["respuestas"]...) }

/**
 * validID reutiliza módulo 10 sin admitir caracteres ajenos a la cédula.
 * @param {string} value - Cédula, vacía para un campo pendiente.
 * @returns {error} Error de validación sin el valor ingresado.
 */
func validID(value string) error {
	if value == "" {
		return nil
	}
	card, err := identity.NewIDCard(value)
	if err != nil || card.String() != value {
		return fmt.Errorf("la cédula no es válida")
	}
	return nil
}

/**
 * placeName limita formato y longitud de una ubicación escrita manualmente.
 * @param {string} value - Ubicación a validar.
 * @returns {error} Error de formato o longitud.
 */
func placeName(value string) error {
	if utf8.RuneCountInString(value) > 120 {
		return fmt.Errorf("la ubicación admite máximo 120 caracteres")
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !strings.ContainsRune(" .,'’()-", r) {
			return fmt.Errorf("el formato de ubicación no es válido")
		}
	}
	return nil
}

/**
 * validDate rechaza fechas imposibles y futuras sin normalizarlas.
 * @param {string} value - Fecha en formato AAAA-MM-DD.
 * @param {time.Time} now - Momento de referencia.
 * @returns {error} Error de fecha cuando el campo está informado.
 */
func validDate(value string, now time.Time) error {
	if value == "" {
		return nil
	}
	date, err := time.Parse(time.DateOnly, value)
	if err != nil || date.After(calendarDate(now)) {
		return fmt.Errorf("la fecha debe ser válida y no futura (AAAA-MM-DD)")
	}
	return nil
}

/**
 * calendarDate expresa el día de Ecuador sobre una fecha sin hora en UTC.
 * @param {time.Time} now - Instante a convertir.
 * @returns {time.Time} Fecha civil usada en los cálculos.
 */
func calendarDate(now time.Time) time.Time {
	local := now.In(time.FixedZone("America/Guayaquil", -5*60*60))
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

/**
 * dateRange valida fechas y su orden sin exigir campos pendientes.
 * @param {string} start - Fecha de inicio.
 * @param {string} end - Fecha de fin.
 * @param {bool} current - Empleo que continúa hasta hoy.
 * @param {time.Time} now - Momento de referencia.
 * @returns {error} Error de calendario u orden.
 */
func dateRange(start, end string, current bool, now time.Time) error {
	if err := validDate(start, now); err != nil {
		return err
	}
	if !current {
		if err := validDate(end, now); err != nil {
			return err
		}
	}
	if current && end != "" {
		return fmt.Errorf("un empleo actual no debe declarar fecha de finalización")
	}
	if start != "" && !current && end != "" && end < start {
		return fmt.Errorf("la fecha de fin no puede ser anterior a la de inicio")
	}
	return nil
}

/**
 * ineligibleMessage conserva el mensaje del RF para cada sector.
 * @param {string} sector - Sector ya validado.
 * @returns {string} Mensaje de cierre para el cliente.
 */
func ineligibleMessage(sector string) string {
	label, reason := "de Afiliados", "no mantiene la calidad de afiliado al IESS"
	if sector == "jubilados" {
		label, reason = "de los Jubilados", "no acredita jubilación"
	}
	if sector == "empleadores" {
		label, reason = "Empleador", "no es empleador"
	}
	return fmt.Sprintf("Su postulación para el cargo de Miembro Principal y Alterno en Representación del Sector %s ante el Directorio del BIESS, NO puede continuar, por cuanto %s", label, reason)
}

/**
 * CalculateDuration calcula tiempo de calendario con préstamo de días del mes
 * anterior; coincide con calcularDuracionExperiencia del frontend.
 * @param {time.Time} start - Inicio del período.
 * @param {time.Time} end - Fin del período.
 * @returns {Duration} Años, meses, días y texto; cero si el intervalo es inverso.
 */
func CalculateDuration(start, end time.Time) Duration {
	if end.Before(start) {
		return Duration{Text: "Fecha no válida"}
	}
	years, months, days := end.Year()-start.Year(), int(end.Month()-start.Month()), end.Day()-start.Day()
	for borrowed := 0; days < 0; borrowed++ {
		months--
		days += time.Date(end.Year(), end.Month()-time.Month(borrowed), 0, 0, 0, 0, 0, time.UTC).Day()
	}
	if months < 0 {
		years--
		months += 12
	}
	parts := []string{}
	for i, value := range []int{years, months, days} {
		if value > 0 || i == 2 && len(parts) == 0 {
			names := [3][2]string{{"año", "años"}, {"mes", "meses"}, {"día", "días"}}
			word := names[i][1]
			if value == 1 {
				word = names[i][0]
			}
			parts = append(parts, fmt.Sprintf("%d %s", value, word))
		}
	}
	return Duration{Years: years, Months: months, Days: days, Text: strings.Join(parts, ", ")}
}

/** normalizeDurations reemplaza duraciones suministradas por cálculos propios. */
func normalizeDurations(form *Form, now time.Time) {
	if form.Experience == nil {
		return
	}
	for i := range form.Experience.Experiences {
		e := &form.Experience.Experiences[i]
		e.Duration = nil
		start, err := time.Parse(time.DateOnly, e.StartDate)
		if err != nil {
			continue
		}
		end, err := time.Parse(time.DateOnly, e.EndDate)
		if e.Current {
			end, err = calendarDate(now), nil
		}
		if err != nil {
			continue
		}
		duration := CalculateDuration(start, end)
		e.Duration = &duration
	}
}
