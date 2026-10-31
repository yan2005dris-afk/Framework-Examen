package cache_test

import (
	"context"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

func TestOTPStore_IssueAndVerify(t *testing.T) {
	mem := cache.NewMemoryCache()
	store := cache.NewOTPStore(mem)
	ctx := context.Background()

	code, err := cache.GenerateOTP()
	if err != nil {
		t.Fatalf("GenerateOTP() error = %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("longitud de OTP esperada 6, obtuve %d (%q)", len(code), code)
	}

	identifier := "0921234567"
	email := "test@biess.ec"

	// 1. Emitir desafío de registro
	challenge, err := store.Issue(ctx, cache.PurposeRegistration, identifier, email, code)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if challenge.ID == "" {
		t.Fatal("desafío sin ID")
	}

	// 2. Intento con código erróneo
	valid, err := store.Verify(ctx, cache.PurposeRegistration, identifier, email, challenge.ID, "000000")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() esperaba false con código incorrecto")
	}

	// 3. Intento con propósito cruzado (recuperación en vez de registro)
	valid, err = store.Verify(ctx, cache.PurposeRecovery, identifier, email, challenge.ID, code)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() no debe permitir consumir OTP de registro para recuperación")
	}

	// 4. Intento exitoso con propósito y código correctos
	valid, err = store.Verify(ctx, cache.PurposeRegistration, identifier, email, challenge.ID, code)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Error("Verify() esperaba true con código y propósito correctos")
	}

	// 5. Consumo único: segundo intento debe retornar false
	valid, err = store.Verify(ctx, cache.PurposeRegistration, identifier, email, challenge.ID, code)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if valid {
		t.Error("Verify() esperaba false tras haber sido consumido")
	}
}
