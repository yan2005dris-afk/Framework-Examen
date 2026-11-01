package submission

/**
 * Form conserva el contrato JSON del EF-12; los punteros distinguen secciones
 * ausentes de secciones guardadas, incluso cuando sus listas están vacías.
 */
type Form struct {
	Personal          *PersonalSection          `json:"seccion1,omitempty"`
	Contact           *ContactSection           `json:"seccion2,omitempty"`
	Additional        *AdditionalSection        `json:"seccion3,omitempty"`
	Position          *PositionSection          `json:"seccion4,omitempty"`
	Education         *EducationSection         `json:"seccion5,omitempty"`
	Experience        *ExperienceSection        `json:"seccion6,omitempty"`
	Publications      *PublicationsSection      `json:"seccion7,omitempty"`
	Training          *TrainingSection          `json:"seccion8,omitempty"`
	Disqualifications *DisqualificationsSection `json:"seccion9,omitempty"`
	Consent           *Consent                  `json:"consentimiento,omitempty"`
}

/** PersonalSection contiene la información personal y los datos de pareja. */
type PersonalSection struct {
	FirstName             string `json:"nombres"`
	LastName              string `json:"apellidos"`
	IDCard                string `json:"cedula"`
	Gender                string `json:"genero"`
	BirthDate             string `json:"fechaNacimiento"`
	Nationality           string `json:"nacionalidad"`
	MaritalStatus         string `json:"estadoCivil"`
	SpouseName            string `json:"nombreConyuge,omitempty"`
	SpouseIDCard          string `json:"cedulaConyuge,omitempty"`
	CivilRegistryVerified bool   `json:"verificadoRegistroCivil"`
}

/** ContactSection contiene residencia y medios de contacto. */
type ContactSection struct {
	Country         string `json:"pais"`
	Province        string `json:"provincia"`
	City            string `json:"canton"`
	Mobile          string `json:"telefonoCelular"`
	Email           string `json:"correo1"`
	SecondaryEmail  string `json:"correo2"`
	MainStreet      string `json:"callePrincipal"`
	Number          string `json:"numero"`
	SecondaryStreet string `json:"calleSecundaria"`
	Area            string `json:"sector"`
	Reference       string `json:"referencia"`
	Phone           string `json:"telefonoDomiciliario"`
}

/** AdditionalSection declara familiares que trabajan en IESS o BIESS. */
type AdditionalSection struct {
	HasRelatives string     `json:"tieneFamiliaresIessBiess"`
	Relatives    []Relative `json:"familiares"`
}

/** Relative es un familiar declarado por el postulante. */
type Relative struct {
	ID           string `json:"id"`
	Name         string `json:"nombres"`
	Relationship string `json:"parentesco"`
	Institution  string `json:"institucion"`
	Area         string `json:"area"`
	Position     string `json:"cargo"`
}

/** PositionSection contiene sector y respaldo; IESS aún no está integrado. */
type PositionSection struct {
	Sector             string             `json:"sector"`
	Eligible           *bool              `json:"verificacionIessCumplida,omitempty"`
	VerificationDetail string             `json:"detalleVerificacionIess"`
	HasOrganization    string             `json:"cuentaConRespaldoOrganizacion"`
	Organization       string             `json:"nombreOrganizacion"`
	Document           *DocumentReference `json:"documentoRespaldoPdf,omitempty"`
}

/** DocumentReference vincula el respaldo; el envío verifica y normaliza sus metadatos. */
type DocumentReference struct {
	ID         string `json:"id,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	Name       string `json:"nombre"`
	Size       string `json:"tamano"`
	UploadedAt string `json:"fechaSubida,omitempty"`
}

/** EducationSection agrupa los títulos académicos declarados. */
type EducationSection struct {
	Degrees []Degree `json:"titulos"`
}

/** Degree describe un título y su registro SENESCYT. */
type Degree struct {
	ID           string             `json:"id"`
	Level        string             `json:"nivel"`
	Institution  string             `json:"institucion"`
	Type         string             `json:"tipo"`
	Title        string             `json:"titulo"`
	Area         string             `json:"area"`
	Registry     string             `json:"registroSenescyt"`
	RegistryDate string             `json:"fechaRegistroSenescyt"`
	Document     *DocumentReference `json:"documentoPdf,omitempty"`
}

/** ExperienceSection agrupa la experiencia profesional. */
type ExperienceSection struct {
	Experiences []Experience `json:"experiencias"`
}

/** Experience conserva fechas y funciones; la duración la calcula el servidor. */
type Experience struct {
	ID          string             `json:"id"`
	Type        string             `json:"tipo"`
	Position    string             `json:"cargo"`
	Institution string             `json:"institucion"`
	StartDate   string             `json:"fechaInicio"`
	EndDate     string             `json:"fechaFin"`
	Current     bool               `json:"actualmenteLaborando,omitempty"`
	Activities  string             `json:"actividades"`
	Functions   string             `json:"descripcionFunciones"`
	Duration    *Duration          `json:"duracionCalculada,omitempty"`
	Document    *DocumentReference `json:"documentoPdf,omitempty"`
}

/** Duration expresa años, meses y días de calendario, con texto en español. */
type Duration struct {
	Years  int    `json:"anios"`
	Months int    `json:"meses"`
	Days   int    `json:"dias"`
	Text   string `json:"texto"`
}

/** PublicationsSection permite declarar expresamente que no hay publicaciones. */
type PublicationsSection struct {
	HasPublications string        `json:"cuentaConPublicaciones"`
	Publications    []Publication `json:"publicaciones"`
}

/** Publication es una publicación o un trabajo de investigación. */
type Publication struct {
	ID       string             `json:"id"`
	Title    string             `json:"titulo"`
	Type     string             `json:"tipo"`
	Medium   string             `json:"medio"`
	Date     string             `json:"fechaPublicacion"`
	Related  string             `json:"relacionCampo"`
	Document *DocumentReference `json:"documentoPdf,omitempty"`
}

/** TrainingSection agrupa los eventos de capacitación. */
type TrainingSection struct {
	Events []Training `json:"capacitaciones"`
}

/** Training representa un curso, seminario o taller. */
type Training struct {
	ID          string             `json:"id"`
	Type        string             `json:"tipo"`
	Name        string             `json:"nombre"`
	Institution string             `json:"institucion"`
	StartDate   string             `json:"fechaInicio"`
	EndDate     string             `json:"fechaFin"`
	Hours       int                `json:"horas"`
	Document    *DocumentReference `json:"documentoPdf,omitempty"`
}

/** DisqualificationsSection conserva las trece respuestas sin inferirlas. */
type DisqualificationsSection struct {
	Q1        string          `json:"q1"`
	Q2        string          `json:"q2"`
	Q3        string          `json:"q3"`
	Q4        string          `json:"q4"`
	Q5        string          `json:"q5"`
	Q6        string          `json:"q6"`
	Q7        string          `json:"q7"`
	Q7Details *RemovalDetails `json:"q7Detalles,omitempty"`
	Q8        string          `json:"q8"`
	Q9        string          `json:"q9"`
	Q10       string          `json:"q10"`
	Q11       string          `json:"q11"`
	Q12       string          `json:"q12"`
	Q13       string          `json:"q13"`
}

/** RemovalDetails son los datos condicionales de EF-12 1.9.7. */
type RemovalDetails struct {
	Institution string `json:"institucion"`
	Position    string `json:"cargo"`
	EndDate     string `json:"fechaFin"`
	Reason      string `json:"motivo"`
}

/** Consent conserva por separado las cinco aceptaciones del RF 1.10. */
type Consent struct {
	PrivacyPolicyID   string   `json:"politicaPrivacidadId,omitempty"`
	PrivacyPolicyHash string   `json:"hashPoliticaPrivacidad,omitempty"`
	Version           string   `json:"version,omitempty"`
	Statements        []string `json:"puntos,omitempty"`
	AcceptedAt        string   `json:"aceptadoEn,omitempty"`
	DataProcessing    bool     `json:"aceptacionTratamientoDatos"`
	Truthfulness      bool     `json:"declaracionVeracidad"`
	CompetitionRules  bool     `json:"aceptacionNormasConcurso"`
	NamePublication   bool     `json:"autorizacionPublicacionNombre"`
	PhysicalDelivery  bool     `json:"constanciaEntregaFisica"`
	Signature         string   `json:"firmaManuscritaDataUrl"`
	SignedAt          string   `json:"fechaFirma"`
}
