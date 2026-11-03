package config

import (
	"testing"
	"time"
)

func TestOTPConfigurationBounds(t *testing.T) {
	for _, tt := range []struct{ name, key, value string }{{"demasiado_rapido", "OTP_RESEND_DELAY", "0s"}, {"mayor_al_ttl", "OTP_RESEND_DELAY", "10m"}, {"cuota_cero", "OTP_MAX_SENDS", "0"}, {"cuota_excesiva", "OTP_MAX_SENDS", "11"}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTP_RESEND_DELAY", "1m")
			t.Setenv("OTP_MAX_SENDS", "3")
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Fatal("configuración insegura aceptada")
			}
		})
	}
	t.Setenv("OTP_RESEND_DELAY", "2m")
	t.Setenv("OTP_MAX_SENDS", "2")
	c, err := Load()
	if err != nil || c.OTPResendDelay != 2*time.Minute || c.OTPMaxSends != 2 {
		t.Fatal(c, err)
	}
}
