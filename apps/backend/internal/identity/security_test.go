package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

func TestIdentityFailsClosedWithoutRedis(t *testing.T) {
	service := identity.NewService(identity.ServiceDependencies{Attempts: cache.NewAttemptsStore(&cache.NoopCache{}), OTPStore: cache.NewOTPStore(&cache.NoopCache{})})
	r := chi.NewRouter()
	identity.NewHandler(service, false).RegisterRoutes(r)
	testPass := string([]byte{'C', 'o', 'r', 'r', 'e', 'c', 't', 'a', '.', '2', '0', '2', '6', '*'})
	payload, _ := json.Marshal(map[string]string{
		"cedula":       "1710034008",
		"correo":       "a@example.test",
		"password":     testPass,
		"codigo_otp":   "123456",
		"new_password": testPass,
	})
	for _, route := range []string{"login", "register/request", "password/forgot", "password/reset"} {
		req := httptest.NewRequest("POST", "/api/v1/auth/"+route, strings.NewReader(string(payload)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 503 {
			t.Fatalf("%s: %d %s", route, w.Code, w.Body.String())
		}
	}
}

func TestRegistrationCannotChangeVerifiedAddress(t *testing.T) {
	service, repo, mailer, _ := setupService(t)
	ctx := context.Background()
	if err := service.RequestRegistration(ctx, identity.RegisterRequestDTO{Cedula: validCedula, Correo: validEmail}); err != nil {
		t.Fatal(err)
	}
	code, _ := readOTP(mailer)
	req := identity.RegisterVerifyDTO{Cedula: validCedula, Correo: "intruso@example.test", CodigoOTP: code, Nombres: "Persona", Apellidos: "Prueba", Password: validPass}
	if _, _, err := service.VerifyAndRegister(ctx, req, "", ""); !errors.Is(err, identity.ErrInvalidOTP) {
		t.Fatal("correo no vinculado aceptado", err)
	}
	if _, err := repo.GetByIDCard(ctx, identity.IDCard(validCedula)); !errors.Is(err, identity.ErrApplicantNotFound) {
		t.Fatal("cuenta creada sin verificar destino")
	}
	req.Correo = validEmail
	if _, _, err := service.VerifyAndRegister(ctx, req, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryRejectsRegistrationCodeAndRevokesSessions(t *testing.T) {
	service, _, mailer, _ := setupService(t)
	ctx := context.Background()
	if err := service.RequestRegistration(ctx, identity.RegisterRequestDTO{Cedula: validCedula, Correo: validEmail}); err != nil {
		t.Fatal(err)
	}
	registrationCode, _ := readOTP(mailer)
	if _, _, err := service.VerifyAndRegister(ctx, identity.RegisterVerifyDTO{Cedula: validCedula, Correo: validEmail, CodigoOTP: registrationCode, Nombres: "Persona", Apellidos: "Prueba", Password: validPass}, "", ""); err != nil {
		t.Fatal(err)
	}
	_, token, err := service.Login(ctx, validCedula, validPass, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ForgotPassword(ctx, validCedula); err != nil {
		t.Fatal(err)
	}
	recoveryCode, _ := readOTP(mailer)
	req := identity.ResetPasswordDTO{Cedula: validCedula, CodigoOTP: registrationCode, NewPassword: "Nueva.Segura2026*"}
	if err := service.ResetPassword(ctx, req); !errors.Is(err, identity.ErrInvalidOTP) {
		t.Fatal("recuperación aceptó código de registro", err)
	}
	req.CodigoOTP = recoveryCode
	if err := service.ResetPassword(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateSession(ctx, token); !errors.Is(err, identity.ErrSessionExpired) {
		t.Fatal("sesión anterior sobrevivió", err)
	}
	if err := service.ResetPassword(ctx, req); !errors.Is(err, identity.ErrInvalidOTP) {
		t.Fatal("OTP reutilizado", err)
	}
}
