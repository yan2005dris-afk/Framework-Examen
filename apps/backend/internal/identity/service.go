package identity

import (
	"context"
	"errors"
	"fmt"
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
	idCard, err := NewIDCard(req.Cedula)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInformation, err.Error())
	}

	if err := ValidateEmail(req.Correo); err != nil {
		return fmt.Errorf("%w: formato de correo no válido", ErrInvalidInformation)
	}

	// Comprobar bloqueo activo por intentos fallidos
	locked, err := s.attempts.IsLocked(ctx, idCard.String())
	if err != nil {
		return err
	}
	if locked {
		return ErrAccountLocked
	}

	// Comprobar si la cédula ya está registrada
	if _, err := s.applicants.GetByIDCard(ctx, idCard); err == nil {
		return ErrApplicantAlreadyExists
	}

	// Comprobar si el correo ya está registrado
	if _, err := s.applicants.GetByEmail(ctx, req.Correo); err == nil {
		return ErrApplicantAlreadyExists
	}

	// Generar código OTP de 6 dígitos
	otp, err := cache.GenerateOTP()
	if err != nil {
		return err
	}

	// Guardar en Redis con TTL de 10 minutos
	if err := s.otpStore.SaveOTP(ctx, idCard.String(), otp); err != nil {
		return err
	}

	// Enviar correo de verificación
	if s.mailer != nil {
		body := fmt.Sprintf(`
			<h2>Concurso de Méritos y Oposición - Directorio BIESS</h2>
			<p>Su código de verificación para completar su registro es:</p>
			<h1 style="letter-spacing: 5px; color: #1e3a8a;">%s</h1>
			<p>Este código tiene una vigencia de <strong>10 minutos</strong>.</p>
			<p>Si usted no solicitó este registro, puede ignorar este mensaje.</p>
		`, otp)

		_ = s.mailer.Send(ctx, mail.Message{
			To:      req.Correo,
			Subject: "Código de verificación de registro - Concurso BIESS",
			Body:    body,
		})
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
	validOTP, err := s.otpStore.VerifyOTP(ctx, idCard.String(), req.CodigoOTP)
	if err != nil {
		return nil, "", err
	}
	if !validOTP {
		// Intento fallido
		count, recordErr := s.attempts.RecordFailure(ctx, idCard.String())
		if recordErr == nil && count >= cache.MaxLoginAttempts {
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
		applicantID = uuid.New()
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
	_ = s.attempts.ResetAttempts(ctx, idCard.String())

	// Crear sesión inicial
	token, sessionErr := s.createSession(ctx, applicant.ID, ip, userAgent)
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
			count, _ := s.attempts.RecordFailure(ctx, idCard.String())
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
		count, _ := s.attempts.RecordFailure(ctx, idCard.String())
		if count >= cache.MaxLoginAttempts {
			return nil, "", ErrAccountLocked
		}
		return nil, "", ErrInvalidCredentials
	}

	// Éxito: limpiar intentos fallidos
	_ = s.attempts.ResetAttempts(ctx, idCard.String())

	// Crear sesión
	token, err := s.createSession(ctx, applicant.ID, ip, userAgent)
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
		_ = s.sessions.DeleteByTokenHash(ctx, tokenHash)
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
	idCard, err := NewIDCard(cedula)
	if err != nil {
		return ErrInvalidInformation
	}

	applicant, err := s.applicants.GetByIDCard(ctx, idCard)
	if err != nil {
		// Por seguridad, si el postulante no existe no revelamos el dato
		return nil
	}

	otp, err := cache.GenerateOTP()
	if err != nil {
		return err
	}

	if err := s.otpStore.SaveOTP(ctx, idCard.String(), otp); err != nil {
		return err
	}

	if s.mailer != nil {
		body := fmt.Sprintf(`
			<h2>Recuperación de Contraseña - Concurso BIESS</h2>
			<p>Estimado/a %s,</p>
			<p>Su código para restablecer su contraseña es:</p>
			<h1 style="letter-spacing: 5px; color: #1e3a8a;">%s</h1>
			<p>Este código expira en <strong>10 minutos</strong>.</p>
		`, applicant.FirstName, otp)

		_ = s.mailer.Send(ctx, mail.Message{
			To:      applicant.Email,
			Subject: "Recuperación de contraseña - Concurso BIESS",
			Body:    body,
		})
	}

	return nil
}

/**
 * ResetPassword verifica el código OTP y actualiza la contraseña con un nuevo hash Argon2id,
 * cerrando todas las sesiones previas del postulante por seguridad.
 *
 * @param {Context} ctx - Contexto de la petición.
 * @param {ResetPasswordDTO} req - Datos de restablecimiento.
 * @returns {error} Error si el código es inválido o la contraseña es débil.
 */
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordDTO) error {
	idCard, err := NewIDCard(req.Cedula)
	if err != nil {
		return ErrInvalidInformation
	}

	applicant, err := s.applicants.GetByIDCard(ctx, idCard)
	if err != nil {
		return ErrInvalidInformation
	}

	valid, err := s.otpStore.VerifyOTP(ctx, idCard.String(), req.CodigoOTP)
	if err != nil || !valid {
		return ErrInvalidOTP
	}

	if err := password.ValidateStrength(req.NewPassword); err != nil {
		return err
	}

	newHash, err := password.Hash(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.applicants.UpdatePassword(ctx, applicant.ID, newHash); err != nil {
		return err
	}

	// Revocar todas las sesiones existentes tras cambio de clave
	_ = s.sessions.DeleteByApplicantID(ctx, applicant.ID)

	return nil
}

// createSession helper interno para emitir y almacenar un token de sesión
func (s *Service) createSession(ctx context.Context, applicantID uuid.UUID, ip, userAgent string) (string, error) {
	rawToken, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		sessionID = uuid.New()
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

	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return "", err
	}

	return rawToken, nil
}
