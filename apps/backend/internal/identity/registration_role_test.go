package identity_test

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/identity"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicOTPRegistrationCannotAssignAnAdministrativeRole(t *testing.T) {
	service, applicants, mailer, _ := setupService(t)
	ctx := context.Background()
	if err := service.RequestRegistration(ctx, identity.RegisterRequestDTO{Cedula: validCedula, Correo: validEmail}); err != nil {
		t.Fatal(err)
	}
	otp, err := readOTP(mailer)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{
		"cedula": validCedula, "correo": validEmail, "codigo_otp": otp,
		"nombres": "Persona", "apellidos": "Prueba", "password": validPass,
		"rol": identity.RoleAdministrator, "role": identity.RoleAdministrator,
	})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	identity.NewHandler(service, false).RegisterRoutes(router)
	r := httptest.NewRequest("POST", "/api/v1/auth/register/verify", strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("registro OTP rechazado: %d %s", w.Code, w.Body.String())
	}
	applicant, err := applicants.GetByIDCard(ctx, identity.IDCard(validCedula))
	if err != nil || applicant.Role != identity.RoleApplicant {
		t.Fatal("el registro público elevó permisos")
	}
	var response struct {
		Applicant identity.ApplicantResponseDTO `json:"postulante"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Applicant.Rol != identity.RoleApplicant {
		t.Fatal("perfil con rol inyectado")
	}
}
