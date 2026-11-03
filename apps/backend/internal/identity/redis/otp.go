package redis

import (
	"context"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

/**
 * OTPStore implementa el puerto de desafíos OTP del contexto de identidad
 * sobre la caché de Redis.
 */
type OTPStore struct {
	store *cache.OTPStore
}

// El adaptador satisface el puerto declarado por el dominio.
var _ identity.OTPStore = (*OTPStore)(nil)

/**
 * NewOTPStore envuelve el almacén de desafíos de la infraestructura.
 *
 * @param {*cache.OTPStore} store - Almacén de desafíos ya configurado.
 * @returns {*OTPStore} Adaptador listo para el dominio.
 */
func NewOTPStore(store *cache.OTPStore) *OTPStore {
	return &OTPStore{store: store}
}

/**
 * Issue emite un desafío OTP para la finalidad y la cuenta indicadas.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {identity.OTPPurpose} purpose - Finalidad del código.
 * @param {string} identifier - Identificador de la cuenta.
 * @param {string} email - Correo al que se envía el código.
 * @param {string} code - Código de un solo uso.
 * @returns {identity.OTPChallenge} Desafío emitido.
 * @returns {error} Error del dominio si la emisión no es posible.
 */
func (s *OTPStore) Issue(ctx context.Context, purpose identity.OTPPurpose, identifier, email, code string) (identity.OTPChallenge, error) {
	if s == nil || s.store == nil {
		return identity.OTPChallenge{}, identity.ErrUnavailable
	}
	infraPurpose, err := infraPurpose(purpose)
	if err != nil {
		return identity.OTPChallenge{}, err
	}
	challenge, err := s.store.Issue(ctx, infraPurpose, identifier, email, code)
	if err != nil {
		return identity.OTPChallenge{}, translate(err)
	}
	return identity.OTPChallenge{ID: challenge.ID, Key: challenge.Key, ExpiresAt: challenge.ExpiresAt}, nil
}

/**
 * Cancel invalida un desafío que no pudo entregarse, sin borrar un reenvío
 * posterior.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {identity.OTPChallenge} challenge - Desafío a invalidar.
 * @returns {error} Error del dominio si el desafío no puede invalidarse.
 */
func (s *OTPStore) Cancel(ctx context.Context, challenge identity.OTPChallenge) error {
	if s == nil || s.store == nil {
		return identity.ErrUnavailable
	}
	return translate(s.store.Cancel(ctx, cache.Challenge{ID: challenge.ID, Key: challenge.Key, ExpiresAt: challenge.ExpiresAt}))
}

/**
 * Verify consume el desafío si la finalidad, la cuenta, el correo y el código
 * coinciden. Un código incorrecto no consume el desafío: agota sus intentos.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {identity.OTPPurpose} purpose - Finalidad del código.
 * @param {string} identifier - Identificador de la cuenta.
 * @param {string} email - Correo de destino del desafío.
 * @param {string} id - Identificador del desafío.
 * @param {string} code - Código de un solo uso.
 * @returns {bool} True si el código era válido y se consumió el desafío.
 * @returns {error} Error del dominio si no fue posible verificar.
 */
func (s *OTPStore) Verify(ctx context.Context, purpose identity.OTPPurpose, identifier, email, id, code string) (bool, error) {
	if s == nil || s.store == nil {
		return false, identity.ErrUnavailable
	}
	infraPurpose, err := infraPurpose(purpose)
	if err != nil {
		return false, err
	}
	valid, err := s.store.Verify(ctx, infraPurpose, identifier, email, id, code)
	if err != nil {
		return false, translate(err)
	}
	return valid, nil
}

/**
 * Active indica si el desafío sigue vigente: no fue reemplazado, consumido ni
 * agotó sus intentos. La cola de correo la consulta antes de entregar un
 * código, para no enviar el de un desafío superado.
 *
 * @param {context.Context} ctx - Contexto de la petición.
 * @param {string} key - Clave interna del desafío.
 * @param {string} id - Identificador del desafío.
 * @returns {bool} True si el desafío sigue vigente.
 * @returns {error} Error del dominio si no fue posible comprobarlo.
 */
func (s *OTPStore) Active(ctx context.Context, key, id string) (bool, error) {
	if s == nil || s.store == nil {
		return false, identity.ErrUnavailable
	}
	active, err := s.store.Active(ctx, key, id)
	if err != nil {
		return false, translate(err)
	}
	return active, nil
}

/**
 * infraPurpose traduce la finalidad del dominio al vocabulario de la caché.
 * Una finalidad desconocida no se emite: el control de seguridad falla cerrado.
 *
 * @param {identity.OTPPurpose} purpose - Finalidad del dominio.
 * @returns {string} Finalidad de la infraestructura.
 * @returns {error} ErrUnavailable si la finalidad no está contemplada.
 */
func infraPurpose(purpose identity.OTPPurpose) (string, error) {
	switch purpose {
	case identity.PurposeRegistration:
		return cache.PurposeRegistration, nil
	case identity.PurposeRecovery:
		return cache.PurposeRecovery, nil
	default:
		return "", identity.ErrUnavailable
	}
}
