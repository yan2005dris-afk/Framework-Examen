package identity_test

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/password"
)

// ── Dobles de prueba en memoria ──────────────────────────────────────────────

type inMemApplicantRepo struct {
	mu       sync.Mutex
	byID     map[uuid.UUID]*identity.Applicant
	byIDCard map[string]*identity.Applicant
	byEmail  map[string]*identity.Applicant
}

func newInMemApplicantRepo() *inMemApplicantRepo {
	return &inMemApplicantRepo{
		byID:     make(map[uuid.UUID]*identity.Applicant),
		byIDCard: make(map[string]*identity.Applicant),
		byEmail:  make(map[string]*identity.Applicant),
	}
}

func (r *inMemApplicantRepo) Create(_ context.Context, a *identity.Applicant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byIDCard[a.IDCard.String()]; ok {
		return identity.ErrApplicantAlreadyExists
	}
	if a.Email != "" {
		if _, ok := r.byEmail[a.Email]; ok {
			return identity.ErrApplicantAlreadyExists
		}
		r.byEmail[a.Email] = a
	}
	r.byID[a.ID] = a
	r.byIDCard[a.IDCard.String()] = a
	return nil
}

func (r *inMemApplicantRepo) GetByID(_ context.Context, id uuid.UUID) (*identity.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok {
		return nil, identity.ErrApplicantNotFound
	}
	return a, nil
}

func (r *inMemApplicantRepo) GetByIDCard(_ context.Context, idCard identity.IDCard) (*identity.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byIDCard[idCard.String()]
	if !ok {
		return nil, identity.ErrApplicantNotFound
	}
	return a, nil
}

func (r *inMemApplicantRepo) GetByEmail(_ context.Context, email string) (*identity.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byEmail[email]
	if !ok {
		return nil, identity.ErrApplicantNotFound
	}
	return a, nil
}

func (r *inMemApplicantRepo) UpdatePassword(_ context.Context, id uuid.UUID, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok {
		return identity.ErrApplicantNotFound
	}
	a.PasswordHash = passwordHash
	return nil
}

type inMemSessionRepo struct {
	mu          sync.Mutex
	byTokenHash map[string]*identity.Session
	appRepo     *inMemApplicantRepo
}

func newInMemSessionRepo(appRepo *inMemApplicantRepo) *inMemSessionRepo {
	return &inMemSessionRepo{
		byTokenHash: make(map[string]*identity.Session),
		appRepo:     appRepo,
	}
}

func (r *inMemSessionRepo) CreateSession(_ context.Context, s *identity.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byTokenHash[s.TokenHash] = s
	return nil
}

func (r *inMemSessionRepo) GetWithApplicant(_ context.Context, tokenHash string) (*identity.Session, *identity.Applicant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byTokenHash[tokenHash]
	if !ok {
		return nil, nil, identity.ErrSessionExpired
	}
	a, ok := r.appRepo.byID[s.ApplicantID]
	if !ok {
		return nil, nil, identity.ErrApplicantNotFound
	}
	return s, a, nil
}

func (r *inMemSessionRepo) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byTokenHash, tokenHash)
	return nil
}

func (r *inMemSessionRepo) DeleteByApplicantID(_ context.Context, applicantID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range r.byTokenHash {
		if v.ApplicantID == applicantID {
			delete(r.byTokenHash, k)
		}
	}
	return nil
}

func (r *inMemSessionRepo) DeleteExpired(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for k, v := range r.byTokenHash {
		if v.ExpiresAt.Before(now) {
			delete(r.byTokenHash, k)
		}
	}
	return nil
}

type inMemPasswordResetRepo struct {
	mu     sync.Mutex
	resets map[string]uuid.UUID
}

func newInMemPasswordResetRepo() *inMemPasswordResetRepo {
	return &inMemPasswordResetRepo{resets: make(map[string]uuid.UUID)}
}

func (r *inMemPasswordResetRepo) CreatePasswordReset(_ context.Context, appID uuid.UUID, tokenHash string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resets[tokenHash] = appID
	return nil
}

func (r *inMemPasswordResetRepo) GetValid(_ context.Context, tokenHash string) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.resets[tokenHash]
	if !ok {
		return uuid.Nil, identity.ErrInvalidResetToken
	}
	return id, nil
}

func (r *inMemPasswordResetRepo) MarkUsed(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.resets, tokenHash)
	return nil
}

type inMemCache struct {
	mu   sync.Mutex
	data map[string]string
}

func newInMemCache() *inMemCache {
	return &inMemCache{data: make(map[string]string)}
}

func (m *inMemCache) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", cache.ErrNotFound
	}
	return v, nil
}

func (m *inMemCache) Set(_ context.Context, key, val string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return nil
}

func (m *inMemCache) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var cur int64
	if v, ok := m.data[key]; ok {
		for _, c := range v {
			cur = cur*10 + int64(c-'0')
		}
	}
	cur++
	m.data[key] = string(rune('0' + cur))
	return cur, nil
}

func (m *inMemCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *inMemCache) Close() error { return nil }

type mockMailer struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (m *mockMailer) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

// ── Tests de Service ─────────────────────────────────────────────────────────

func setupService(t *testing.T) (*identity.Service, *inMemApplicantRepo, *mockMailer, *inMemCache) {
	t.Helper()
	appRepo := newInMemApplicantRepo()
	sessionRepo := newInMemSessionRepo(appRepo)
	resetRepo := newInMemPasswordResetRepo()
	mem := newInMemCache()

	server := miniredis.RunT(t)
	atomic, err := cache.NewRedis(context.Background(), "redis://"+server.Addr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = atomic.Close() })
	attempts := cache.NewAttemptsStore(atomic)
	otpStore := cache.NewOTPStore(atomic)
	mailer := &mockMailer{}

	svc := identity.NewService(identity.ServiceDependencies{
		Applicants: appRepo,
		Sessions:   sessionRepo,
		Recovery:   &atomicRecovery{appRepo, sessionRepo},
		Resets:     resetRepo,
		Attempts:   attempts,
		OTPStore:   otpStore,
		Mailer:     mailer,
		SessionTTL: 24 * time.Hour,
	})

	return svc, appRepo, mailer, mem
}

var (
	validCedula = "1710034008" // Cédula ecuatoriana válida (Pichincha)
	validEmail  = "postulante@biess.ec"
	validPass   = string([]byte{'A', 'd', 'm', 'i', 'n', '.', '2', '0', '2', '6', '*'})
)

func TestService_RequestRegistration(t *testing.T) {
	svc, _, mailer, _ := setupService(t)
	ctx := context.Background()

	// 1. Solicitud exitosa
	err := svc.RequestRegistration(ctx, identity.RegisterRequestDTO{
		Cedula: validCedula,
		Correo: validEmail,
	})
	if err != nil {
		t.Fatalf("RequestRegistration() error inesperado = %v", err)
	}

	if len(mailer.sent) != 1 {
		t.Fatalf("se esperaba 1 correo enviado, se enviaron %d", len(mailer.sent))
	}
	if mailer.sent[0].To != validEmail {
		t.Errorf("destinatario incorrecto: %s", mailer.sent[0].To)
	}

	// 2. Cédula inválida
	err = svc.RequestRegistration(ctx, identity.RegisterRequestDTO{
		Cedula: "1234567890",
		Correo: validEmail,
	})
	if err == nil {
		t.Error("RequestRegistration() esperaba error con cédula inválida")
	}

	// 3. Correo inválido
	err = svc.RequestRegistration(ctx, identity.RegisterRequestDTO{
		Cedula: validCedula,
		Correo: "correo-sin-arroba",
	})
	if err == nil {
		t.Error("RequestRegistration() esperaba error con correo inválido")
	}
}

func TestService_VerifyAndRegister(t *testing.T) {
	svc, _, mailer, _ := setupService(t)
	ctx := context.Background()

	// Iniciar registro para sembrar el OTP en caché
	_ = svc.RequestRegistration(ctx, identity.RegisterRequestDTO{
		Cedula: validCedula,
		Correo: validEmail,
	})

	otpCode, err := readOTP(mailer)
	if err != nil {
		t.Fatalf("error al leer OTP de caché: %v", err)
	}

	// 1. Registro con código erróneo
	_, _, err = svc.VerifyAndRegister(ctx, identity.RegisterVerifyDTO{
		Cedula:    validCedula,
		Correo:    validEmail,
		CodigoOTP: "000000",
		Nombres:   "Juan",
		Apellidos: "Perez",
		Password:  validPass,
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Error("VerifyAndRegister() esperaba error con OTP incorrecto")
	}

	// 2. Registro con contraseña débil
	weakPass := string([]byte{'c', 'o', 'r', 't', 'a'})
	_, _, err = svc.VerifyAndRegister(ctx, identity.RegisterVerifyDTO{
		Cedula:    validCedula,
		Correo:    validEmail,
		CodigoOTP: otpCode,
		Nombres:   "Juan",
		Apellidos: "Perez",
		Password:  weakPass,
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Error("VerifyAndRegister() esperaba error con contraseña débil")
	}

	// 3. Registro exitoso
	app, token, err := svc.VerifyAndRegister(ctx, identity.RegisterVerifyDTO{
		Cedula:    validCedula,
		Correo:    validEmail,
		CodigoOTP: otpCode,
		Nombres:   "Juan",
		Apellidos: "Perez",
		Password:  validPass,
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("VerifyAndRegister() exitoso falló: %v", err)
	}
	if app.FirstName != "Juan" || app.LastName != "Perez" {
		t.Errorf("datos del postulante incorrectos: %+v", app)
	}
	if token == "" {
		t.Error("se esperaba un token de sesión no vacío")
	}
}

func TestService_Login_And_Lockout(t *testing.T) {
	svc, appRepo, _, _ := setupService(t)
	ctx := context.Background()

	// Sembrar postulante
	passHash, _ := password.Hash(validPass)
	idCard, _ := identity.NewIDCard(validCedula)
	app, _ := identity.NewApplicant(uuid.New(), idCard, "Maria", "Lopez", validEmail, passHash, "otp_correo", time.Now())
	_ = appRepo.Create(ctx, app)

	wrongPass := string([]byte{'W', 'r', 'o', 'n', 'g', '#', '1'})
	// 1. Dos intentos fallidos con contraseña errónea
	for i := 0; i < 2; i++ {
		_, _, err := svc.Login(ctx, validCedula, wrongPass, "127.0.0.1", "test-ua")
		if err == nil {
			t.Fatal("Login() esperaba error con credenciales incorrectas")
		}
	}

	// 2. Tercer intento fallido: dispara el bloqueo de 10 minutos (EF-11)
	_, _, err := svc.Login(ctx, validCedula, wrongPass, "127.0.0.1", "test-ua")
	if err != identity.ErrAccountLocked {
		t.Fatalf("Login() esperaba ErrAccountLocked al 3er fallo, obtuve: %v", err)
	}

	// 3. Intento adicional mientras está bloqueado: debe rechazar incluso con la clave correcta
	_, _, err = svc.Login(ctx, validCedula, validPass, "127.0.0.1", "test-ua")
	if err != identity.ErrAccountLocked {
		t.Fatalf("Login() esperaba ErrAccountLocked cuando la cuenta está bloqueada, obtuve: %v", err)
	}
}

func TestService_Session_Validate_And_Logout(t *testing.T) {
	svc, appRepo, _, _ := setupService(t)
	ctx := context.Background()

	passHash, _ := password.Hash(validPass)
	idCard, _ := identity.NewIDCard(validCedula)
	app, _ := identity.NewApplicant(uuid.New(), idCard, "Carlos", "Mendoza", validEmail, passHash, "otp_correo", time.Now())
	_ = appRepo.Create(ctx, app)

	// Login exitoso
	_, token, err := svc.Login(ctx, validCedula, validPass, "127.0.0.1", "ua")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Validar sesión
	validatedApp, err := svc.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}
	if validatedApp.ID != app.ID {
		t.Errorf("postulante no coincide: quería %v, obtuve %v", app.ID, validatedApp.ID)
	}

	// Logout
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	// Sesión posterior al logout debe dar error
	_, err = svc.ValidateSession(ctx, token)
	if err != identity.ErrSessionExpired {
		t.Errorf("ValidateSession() esperaba ErrSessionExpired tras logout, obtuve %v", err)
	}
}

func readOTP(m *mockMailer) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	matches := regexp.MustCompile(`[0-9]{6}`).FindString(m.sent[len(m.sent)-1].Body)
	return matches, nil
}

type atomicRecovery struct {
	app      *inMemApplicantRepo
	sessions *inMemSessionRepo
}

func (r *atomicRecovery) ResetPasswordAndSessions(ctx context.Context, id uuid.UUID, hash string) error {
	r.sessions.mu.Lock()
	defer r.sessions.mu.Unlock()
	r.app.mu.Lock()
	defer r.app.mu.Unlock()
	r.app.byID[id].PasswordHash = hash
	for k, s := range r.sessions.byTokenHash {
		if s.ApplicantID == id {
			delete(r.sessions.byTokenHash, k)
		}
	}
	return nil
}

func (r *inMemSessionRepo) CreateAuthenticatedSession(_ context.Context, session *identity.Session, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.appRepo.mu.Lock()
	defer r.appRepo.mu.Unlock()
	account, ok := r.appRepo.byID[session.ApplicantID]
	if !ok || account.PasswordHash != hash {
		return identity.ErrInvalidCredentials
	}
	r.byTokenHash[session.TokenHash] = session
	return nil
}
