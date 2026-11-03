package review

import (
	"github.com/google/uuid"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/submission"
	"time"
)

/** Applicant es la proyección mínima del snapshot para la bandeja y la matriz interna. */
type Applicant struct {
	ID            uuid.UUID    `json:"id"`
	Code          string       `json:"codigo"`
	Competition   string       `json:"convocatoria"`
	FirstName     string       `json:"nombres"`
	LastName      string       `json:"apellidos"`
	Sector        string       `json:"sector"`
	Status        string       `json:"status"`
	SubmittedAt   time.Time    `json:"submittedAt"`
	Hash          string       `json:"hash"`
	Education     []Education  `json:"titulos"`
	Experiences   []Experience `json:"experiencias"`
	Publications  int          `json:"publicaciones"`
	TrainingHours int          `json:"horasCapacitacion"`
	CreatedAt     time.Time    `json:"-"`
}

/** Education conserva el título declarado sin interpretarlo como mérito calificado. */
type Education struct {
	Level string `json:"nivel"`
	Title string `json:"titulo"`
}

/** Experience conserva la duración del sello; nunca crece al consultar un año posterior. */
type Experience struct {
	ID         string               `json:"id"`
	Type       string               `json:"tipo"`
	Position   string               `json:"cargo"`
	Duration   *submission.Duration `json:"duracion"`
	DocumentID string               `json:"documentoId,omitempty"`
}

/** Stats ofrece una lectura fresca y una recomendación de polling, no una promesa de entrega push. */
type Stats struct {
	Filter              Filter    `json:"filtros"`
	AsOf                time.Time `json:"consultadoEn"`
	RefreshAfterSeconds int       `json:"actualizarEnSegundos"`
	Counts              []Count   `json:"conteos"`
}
