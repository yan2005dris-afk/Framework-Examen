package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

/**
 * Mailer define el puerto para el envío de notificaciones por correo.
 */
type Mailer interface {
	Send(ctx context.Context, msg mail.Message) error
}

/**
 * ServiceDependencies agrupa las dependencias necesarias para instanciar el servicio de identidad.
 */
type ServiceDependencies struct {
	Applicants ApplicantRepository
	Sessions   SessionRepository
	Recovery   PasswordRecoveryRepository
	Resets     PasswordResetRepository
	Attempts   *cache.AttemptsStore
	OTPStore   *cache.OTPStore
	Mailer     Mailer
	SessionTTL time.Duration
}

/**
 * Service implementa los casos de uso del contexto de identidad (EF-11).
 */
type Service struct {
	applicants ApplicantRepository
	sessions   SessionRepository
	recovery   PasswordRecoveryRepository
	resets     PasswordResetRepository
	attempts   *cache.AttemptsStore
	otpStore   *cache.OTPStore
	mailer     Mailer
	sessionTTL time.Duration
}

/**
 * NewService construye una instancia del servicio de identidad.
 *
 * @param {ServiceDependencies} deps - Dependencias inyectadas.
 * @returns {*Service} Servicio configurado.
 */
func NewService(deps ServiceDependencies) *Service {
	ttl := deps.SessionTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{
		applicants: deps.Applicants,
		sessions:   deps.Sessions,
		recovery:   deps.Recovery,
		resets:     deps.Resets,
		attempts:   deps.Attempts,
		otpStore:   deps.OTPStore,
		mailer:     deps.Mailer,
		sessionTTL: ttl,
	}
}

/**
 * RequestRegistration inicia el flujo de registro: valida cédula (módulo 10), correo,
 * genera el código OTP y lo envía por correo electrónico (Mailpit / SMTP).
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {RegisterRequestDTO} req - Datos iniciales de registro.
 * @returns {error} Error de validación o del envío.
 */
func (s *Service) RequestRegistration(ctx context.Context, req RegisterRequestDTO) error {
	_, err := s.BeginRegistration(ctx, req)
	return err
}

/** BeginRegistration vincula código y desafío a la finalidad, cédula y correo de destino. */
func (s *Service) BeginRegistration(ctx context.Context, req RegisterRequestDTO) (cache.Challenge, error) {
	idCard, err := NewIDCard(req.Cedula)
	if err != nil {
		return cache.Challenge{}, ErrInvalidInformation
	}
	req.Correo = strings.ToLower(strings.TrimSpace(req.Correo))
	if ValidateEmail(req.Correo) != nil {
		return cache.Challenge{}, ErrInvalidInformation
	}
	if err := s.checkSecurity(ctx, idCard.String()); err != nil {
		return cache.Challenge{}, err
	}
	if s.mailer == nil {
		return cache.Challenge{}, ErrDeliveryUnavailable
	}
	if _, err := s.applicants.GetByIDCard(ctx, idCard); err == nil {
		return cache.Challenge{}, ErrApplicantAlreadyExists
	} else if !errors.Is(err, ErrApplicantNotFound) {
		return cache.Challenge{}, err
	}
	if _, err := s.applicants.GetByEmail(ctx, req.Correo); err == nil {
		return cache.Challenge{}, ErrApplicantAlreadyExists
	} else if !errors.Is(err, ErrApplicantNotFound) {
		return cache.Challenge{}, err
	}
	code, err := cache.GenerateOTP()
	if err != nil {
		return cache.Challenge{}, err
	}
	challenge, err := s.otpStore.Issue(ctx, cache.PurposeRegistration, idCard.String(), req.Correo, code)
	if err != nil {
		return cache.Challenge{}, err
	}
	message := mail.Message{To: req.Correo, Kind: "otp.registro", ChallengeKey: challenge.Key, ChallengeID: challenge.ID, ExpiresAt: challenge.ExpiresAt,
		Subject: "Código de verificación de registro - Concurso BIESS", Body: fmt.Sprintf("Concurso BIESS\nSu código de registro es: %s\nExpira en 10 minutos.", code)}
	if err := s.mailer.Send(ctx, message); err != nil {
		if cancelErr := s.otpStore.Cancel(ctx, challenge); cancelErr != nil {
			return cache.Challenge{}, cache.ErrUnavailable
		}
		return cache.Challenge{}, ErrDeliveryUnavailable
	}
	return challenge, nil
}

func (s *Service) checkSecurity(ctx context.Context, identifier string) error {
	if s.otpStore == nil || s.attempts == nil {
		return cache.ErrUnavailable
	}
	locked, err := s.attempts.IsLocked(ctx, identifier)
	if err != nil {
		return cache.ErrUnavailable
	}
	if locked {
		return ErrAccountLocked
	}
	return nil
}

/**
 * VerifyAndRegister completa el registro comprobando el código OTP, validando la contraseña
 * con Argon2id, insertando el postulante y creando la primera sesión de usuario.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {RegisterVerifyDTO} req - Código OTP y datos personales del postulante.
 * @param {string} ip - Dirección IP del cliente.
 * @param {string} userAgent - Agente de usuario del navegador.
 * @returns {*Applicant, string, error} Postulante registrado, token de sesión en claro y error.
 */
func (s *Service) VerifyAndRegister(ctx context.Context, req RegisterVerifyDTO, ip, userAgent string) (*Applicant, string, error) {
	idCard, err := NewIDCard(req.Cedula)
	if err != nil {
		return nil, "", ErrInvalidInformation
	}

	// Verificar bloqueo
	locked, err := s.attempts.IsLocked(ctx, idCard.String())
	if err != nil {
		return nil, "", err
	}
	if locked {
		return nil, "", ErrAccountLocked
	}

	// Validar fortaleza de contraseña antes de consumir el código OTP
	if err := password.ValidateStrength(req.Password); err != nil {
		return nil, "", err
	}

	// Verificar código OTP
	validOTP, err := s.otpStore.Verify(ctx, cache.PurposeRegistration, idCard.String(), req.Correo, req.ChallengeID, req.CodigoOTP)
	if err != nil {
		return nil, "", err
	}
	if !validOTP {
		// Intento fallido
		count, recordErr := s.attempts.RecordFailure(ctx, idCard.String())
		if recordErr != nil {
			return nil, "", cache.ErrUnavailable
		}
		if count >= cache.MaxLoginAttempts {
			return nil, "", ErrAccountLocked
		}
		return nil, "", ErrInvalidOTP
	}

	// Hash seguro con Argon2id (RN-04)
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, "", err
	}

	applicantID, err := uuid.NewV7()
	if err != nil {
		return nil, "", err
	}

	applicant, err := NewApplicant(
		applicantID,
		idCard,
		req.Nombres,
		req.Apellidos,
		req.Correo,
		hash,
		"otp_correo",
		time.Now(),
	)
	if err != nil {
		return nil, "", err
	}

	if err := s.applicants.Create(ctx, applicant); err != nil {
		return nil, "", err
	}

	// Resetear cualquier contador de intentos tras éxito
	if err := s.attempts.ResetAttempts(ctx, idCard.String()); err != nil {
		return nil, "", cache.ErrUnavailable
	}

	// Crear sesión inicial
	token, sessionErr := s.createSession(ctx, applicant.ID, ip, userAgent, applicant.PasswordHash)
	if sessionErr != nil {
		return applicant, "", sessionErr
	}

	return applicant, token, nil
}

/**
 * Login autentica al postulante mediante cédula y contraseña, aplicando el límite
 * de 3 intentos con bloqueo de 10 minutos (EF-11 / RN-03) y emitiendo una sesión HttpOnly.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Cédula ingresada.
 * @param {string} plainPassword - Contraseña en texto claro.
 * @param {string} ip - Dirección IP del cliente.
 * @param {string} userAgent - Agente de usuario.
 * @returns {*Applicant, string, error} Postulante autenticado, token de sesión y error.
 */
func (s *Service) Login(ctx context.Context, cedula, plainPassword, ip, userAgent string) (*Applicant, string, error) {
	idCard, err := NewIDCard(cedula)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Verificar si está bloqueado en Redis
	locked, err := s.attempts.IsLocked(ctx, idCard.String())
	if err != nil {
		return nil, "", err
	}
	if locked {
		return nil, "", ErrAccountLocked
	}

	// Buscar postulante
	applicant, err := s.applicants.GetByIDCard(ctx, idCard)
	if err != nil {
		if errors.Is(err, ErrApplicantNotFound) {
			count, recordErr := s.attempts.RecordFailure(ctx, idCard.String())
			if recordErr != nil {
				return nil, "", cache.ErrUnavailable
			}
			if count >= cache.MaxLoginAttempts {
				return nil, "", ErrAccountLocked
			}
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	// Verificar hash de contraseña con Argon2id
	match, err := password.Verify(plainPassword, applicant.PasswordHash)
	if err != nil || !match {
		count, recordErr := s.attempts.RecordFailure(ctx, idCard.String())
		if recordErr != nil {
			return nil, "", cache.ErrUnavailable
		}
		if count >= cache.MaxLoginAttempts {
			return nil, "", ErrAccountLocked
		}
		return nil, "", ErrInvalidCredentials
	}

	// Éxito: limpiar intentos fallidos
	if err := s.attempts.ResetAttempts(ctx, idCard.String()); err != nil {
		return nil, "", cache.ErrUnavailable
	}

	// Crear sesión
	token, err := s.createSession(ctx, applicant.ID, ip, userAgent, applicant.PasswordHash)
	if err != nil {
		return nil, "", err
	}

	return applicant, token, nil
}

/**
 * Logout revoca la sesión activa eliminando su registro de PostgreSQL.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} rawToken - Token de sesión recibido de la cookie.
 * @returns {error} Error al eliminar la sesión si ocurre.
 */
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	tokenHash := HashSessionToken(rawToken)
	return s.sessions.DeleteByTokenHash(ctx, tokenHash)
}

/**
 * ValidateSession comprueba la validez del token de sesión contra PostgreSQL.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} rawToken - Token de sesión en claro.
 * @returns {*Applicant, error} Postulante dueño de la sesión o ErrSessionExpired.
 */
func (s *Service) ValidateSession(ctx context.Context, rawToken string) (*Applicant, error) {
	if rawToken == "" {
		return nil, ErrSessionExpired
	}

	tokenHash := HashSessionToken(rawToken)
	session, applicant, err := s.sessions.GetWithApplicant(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	if session.IsExpired() {
		if err := s.sessions.DeleteByTokenHash(ctx, tokenHash); err != nil {
			return nil, err
		}
		return nil, ErrSessionExpired
	}

	return applicant, nil
}

/**
 * ForgotPassword solicita un código OTP para restablecimiento de contraseña.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {string} cedula - Cédula del postulante.
 * @returns {error} Error si no se encuentra el postulante o falla el envío.
 */
func (s *Service) ForgotPassword(ctx context.Context, cedula string) error {
	_, err := s.BeginRecovery(ctx, cedula)
	return err
}

/** BeginRecovery usa el correo de la cuenta; una cuenta ausente recibe un desafío indistinguible sin correo. */
func (s *Service) BeginRecovery(ctx context.Context, cedula string) (cache.Challenge, error) {
	idCard, err := NewIDCard(cedula)
	if err != nil {
		return cache.Challenge{}, ErrInvalidInformation
	}
	if err := s.checkSecurity(ctx, idCard.String()); err != nil {
		return cache.Challenge{}, err
	}
	if s.mailer == nil {
		return cache.Challenge{}, ErrDeliveryUnavailable
	}
	if ready, ok := s.mailer.(interface{ Ready(context.Context) error }); ok {
		if ready.Ready(ctx) != nil {
			return cache.Challenge{}, ErrDeliveryUnavailable
		}
	}
	applicant, err := s.applicants.GetByIDCard(ctx, idCard)
	if err != nil && !errors.Is(err, ErrApplicantNotFound) {
		return cache.Challenge{}, err
	}
	email := ""
	if applicant != nil {
		email = applicant.Email
	}
	code, err := cache.GenerateOTP()
	if err != nil {
		return cache.Challenge{}, err
	}
	challenge, err := s.otpStore.Issue(ctx, cache.PurposeRecovery, idCard.String(), email, code)
	if err != nil {
		return cache.Challenge{}, err
	}
	if applicant == nil {
		// Misma aceptación durable sin enviar un correo a una dirección inexistente ni revelar la cuenta.
		if err := s.mailer.Send(ctx, mail.Message{Suppressed: true, Kind: "otp.recuperacion", ChallengeKey: challenge.Key, ChallengeID: challenge.ID, ExpiresAt: challenge.ExpiresAt}); err != nil {
			if cancelErr := s.otpStore.Cancel(ctx, challenge); cancelErr != nil {
				return cache.Challenge{}, cache.ErrUnavailable
			}
			return cache.Challenge{}, ErrDeliveryUnavailable
		}
		return challenge, nil
	}
	message := mail.Message{To: applicant.Email, Kind: "otp.recuperacion", ChallengeKey: challenge.Key, ChallengeID: challenge.ID, ExpiresAt: challenge.ExpiresAt,
		Subject: "Recuperación de contraseña - Concurso BIESS", Body: fmt.Sprintf("Recuperación de contraseña\nEstimado/a %s, su código es: %s\nExpira en 10 minutos.", applicant.FirstName, code)}
	if err := s.mailer.Send(ctx, message); err != nil {
		if cancelErr := s.otpStore.Cancel(ctx, challenge); cancelErr != nil {
			return cache.Challenge{}, cache.ErrUnavailable
		}
		return cache.Challenge{}, ErrDeliveryUnavailable
	}
	return challenge, nil
}

/** ResetPassword consume una sola vez y confirma contraseña/revocación de sesiones en una transacción. */
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordDTO) error {
	idCard, err := NewIDCard(req.Cedula)
	if err != nil {
		return ErrInvalidOTP
	}
	if err := s.checkSecurity(ctx, idCard.String()); err != nil {
		return err
	}
	if err := password.ValidateStrength(req.NewPassword); err != nil {
		return err
	}
	if s.recovery == nil {
		return cache.ErrUnavailable
	}
	applicant, err := s.applicants.GetByIDCard(ctx, idCard)
	if errors.Is(err, ErrApplicantNotFound) {
		return ErrInvalidOTP
	}
	if err != nil {
		return err
	}
	valid, err := s.otpStore.Verify(ctx, cache.PurposeRecovery, idCard.String(), applicant.Email, req.ChallengeID, req.CodigoOTP)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidOTP
	}
	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	return s.recovery.ResetPasswordAndSessions(ctx, applicant.ID, hash)
}

// createSession helper interno para emitir y almacenar un token de sesión
func (s *Service) createSession(ctx context.Context, applicantID uuid.UUID, ip, userAgent, expectedHash string) (string, error) {
	rawToken, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	tokenHash := HashSessionToken(rawToken)
	session := &Session{
		ID:          sessionID,
		ApplicantID: applicantID,
		TokenHash:   tokenHash,
		ExpiresAt:   time.Now().Add(s.sessionTTL),
		IP:          ip,
		UserAgent:   userAgent,
		CreatedAt:   time.Now(),
	}

	issuer, ok := s.sessions.(AuthenticatedSessionRepository)
	if !ok {
		return "", cache.ErrUnavailable
	}
	if err := issuer.CreateAuthenticatedSession(ctx, session, expectedHash); err != nil {
		return "", err
	}

	return rawToken, nil
}
