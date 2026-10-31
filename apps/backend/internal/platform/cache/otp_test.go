package cache_test

import (
	"context"
	"testing"

	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/cache"
)

func TestOTPStore_GenerateAndVerify(t *testing.T) {
	mem := newMemCache()
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

	// Guardar
	if err := store.SaveOTP(ctx, identifier, code); err != nil {
		t.Fatalf("SaveOTP() error = %v", err)
	}

	// Código erróneo
	valid, err := store.VerifyOTP(ctx, identifier, "000000")
	if err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if valid {
		t.Error("VerifyOTP() esperaba false con código incorrecto")
	}

	// Código correcto
	valid, err = store.VerifyOTP(ctx, identifier, code)
	if err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if !valid {
		t.Error("VerifyOTP() esperaba true con código correcto")
	}

	// Ya fue consumido: segundo intento debe retornar false
	valid, err = store.VerifyOTP(ctx, identifier, code)
	if err != nil {
		t.Fatalf("VerifyOTP() error = %v", err)
	}
	if valid {
		t.Error("VerifyOTP() esperaba false tras haber sido consumido")
	}
}
