-- +goose Up
-- Un postulante tiene como máximo una postulación por convocatoria (RF 1.4: se
-- elige un solo cargo). Puede postular a convocatorias distintas y a la de un
-- período nuevo. El legado sin convocatoria queda fuera del índice.
-- Si ya existen duplicados la migración se detiene sin borrar nada: deben
-- resolverse antes de aplicarla.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM submissions WHERE competition_id IS NOT NULL
           GROUP BY applicant_id,competition_id HAVING count(*)>1) THEN
  RAISE EXCEPTION 'Hay postulantes con más de una postulación en la misma convocatoria; resuélvalos antes de aplicar la migración 00020';
 END IF;
END $$;
-- +goose StatementEnd
CREATE UNIQUE INDEX submissions_one_per_competition ON submissions (applicant_id, competition_id)
    WHERE competition_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS submissions_one_per_competition;
