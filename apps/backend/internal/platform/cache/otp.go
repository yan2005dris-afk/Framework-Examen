package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"strings"
	"time"
)

const OTPTTL = 10 * time.Minute
const OTPResendDelay = time.Minute
const MaxOTPAttempts = 3
const MaxOTPSends = 3
const PurposeRegistration = "registro"
const PurposeRecovery = "recuperacion"

var ErrUnavailable = errors.New("el servicio de seguridad no está disponible")
var ErrOTPLocked = errors.New("el desafío agotó sus intentos")
var ErrResendLimited = errors.New("espere antes de solicitar otro código")

/** ScriptCache requiere atomicidad real; no se emula mediante Get/Delete ni mutex del proceso. */
type ScriptCache interface {
	RunScript(context.Context, string, []string, ...interface{}) (int64, error)
}

/** OTPStore mantiene un desafío por finalidad/cuenta, ligado al correo y con consumo único. */
type OTPStore struct {
	cache       Cache
	resendDelay time.Duration
	maxSends    int
}

/** Challenge identifica el desafío; no contiene el código ni datos personales. */
type Challenge struct {
	ID        string    `json:"desafio_id"`
	Key       string    `json:"-"`
	ExpiresAt time.Time `json:"expira_en"`
}
type otpState struct {
	ID       string `json:"id"`
	Binding  string `json:"binding"`
	Digest   string `json:"digest"`
	Attempts int    `json:"attempts"`
}

/** NewOTPStore construye el puerto; un adaptador sin scripts falla de forma cerrada. */
func NewOTPStore(c Cache) *OTPStore {
	return &OTPStore{cache: c, resendDelay: OTPResendDelay, maxSends: MaxOTPSends}
}

/** NewConfiguredOTPStore permite configurar reenvíos sin cambiar tres intentos ni TTL de diez minutos. */
func NewConfiguredOTPStore(c Cache, delay time.Duration, maxSends int) (*OTPStore, error) {
	if delay < time.Second || delay >= OTPTTL || maxSends < 1 || maxSends > 10 {
		return nil, ErrUnavailable
	}
	return &OTPStore{cache: c, resendDelay: delay, maxSends: maxSends}, nil
}

/** GenerateOTP usa una fuente criptográfica, nunca el reloj ni math/rand. */
func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", errors.New("no se pudo generar el código")
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}
func digest(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }
func challengeKey(purpose, identifier string) string {
	return "identity:otp:{" + digest(purpose+":"+identifier) + "}"
}
func binding(purpose, identifier, email string) string {
	return digest(purpose + "|" + identifier + "|" + strings.ToLower(strings.TrimSpace(email)))
}
func (s *OTPStore) atomic() (ScriptCache, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	a, ok := s.cache.(ScriptCache)
	if !ok {
		return nil, ErrUnavailable
	}
	return a, nil
}

const issueOTP = `
local old=redis.call('GET',KEYS[1])
if old and cjson.decode(old).attempts>=tonumber(ARGV[5]) then return -2 end
if redis.call('EXISTS',KEYS[2])==1 then return -1 end
local sends=tonumber(redis.call('GET',KEYS[3]) or '0')
if sends>=tonumber(ARGV[4]) then return -1 end
sends=redis.call('INCR',KEYS[3])
if sends==1 then redis.call('PEXPIRE',KEYS[3],ARGV[2]) end
redis.call('SET',KEYS[1],ARGV[1],'PX',ARGV[2])
redis.call('SET',KEYS[2],'1','PX',ARGV[3])
return 1`
const consumeOTP = `
local raw=redis.call('GET',KEYS[1]); if not raw then return 0 end
local c=cjson.decode(raw)
if c.attempts>=tonumber(ARGV[4]) then return -2 end
if c.id==ARGV[1] and c.binding==ARGV[2] and c.digest==ARGV[3] then redis.call('DEL',KEYS[1]);return 1 end
c.attempts=c.attempts+1
local ttl=redis.call('PTTL',KEYS[1]);if ttl<=0 then return 0 end
redis.call('SET',KEYS[1],cjson.encode(c),'PX',ttl)
if c.attempts>=tonumber(ARGV[4]) then return -2 end
return 0`
const cancelOTP = `local raw=redis.call('GET',KEYS[1]); if raw and cjson.decode(raw).id==ARGV[1] then return redis.call('DEL',KEYS[1]) end;return 0`

/** Issue reemplaza el código sólo tras cooldown y cuota; los tres cambios ocurren en Redis atómicamente. */
func (s *OTPStore) Issue(ctx context.Context, purpose, identifier, email, code string) (Challenge, error) {
	a, err := s.atomic()
	if err != nil {
		return Challenge{}, err
	}
	if purpose != PurposeRegistration && purpose != PurposeRecovery {
		return Challenge{}, ErrUnavailable
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return Challenge{}, ErrUnavailable
	}
	state := otpState{ID: id.String(), Binding: binding(purpose, identifier, email)}
	state.Digest = digest(state.ID + "|" + state.Binding + "|" + code)
	raw, err := json.Marshal(state)
	if err != nil {
		return Challenge{}, ErrUnavailable
	}
	key := challengeKey(purpose, identifier)
	result, err := a.RunScript(ctx, issueOTP, []string{key, key + ":resend", key + ":sends"}, string(raw), OTPTTL.Milliseconds(), s.resendDelay.Milliseconds(), s.maxSends, MaxOTPAttempts)
	if err != nil {
		return Challenge{}, ErrUnavailable
	}
	if result == -2 {
		return Challenge{}, s.retryError(ctx, ErrOTPLocked, key)
	}
	if result == -1 {
		// Antes de agotar la cuota rige sólo el cooldown; después rige la ventana de emisiones.
		ttl, ttlErr := a.RunScript(ctx, `local ttl=redis.call('PTTL',KEYS[1]);if tonumber(redis.call('GET',KEYS[2]) or '0')>=tonumber(ARGV[1]) then ttl=math.max(ttl,redis.call('PTTL',KEYS[2])) end;return math.max(ttl,0)`, []string{key + ":resend", key + ":sends"}, s.maxSends)
		if ttlErr != nil {
			return Challenge{}, ErrUnavailable
		}
		return Challenge{}, &RetryError{Cause: ErrResendLimited, RetryAfter: max(time.Duration(ttl)*time.Millisecond, time.Second)}
	}
	if result != 1 {
		return Challenge{}, ErrUnavailable
	}
	return Challenge{ID: state.ID, Key: key, ExpiresAt: time.Now().UTC().Add(OTPTTL)}, nil
}

func (s *OTPStore) retryError(ctx context.Context, cause error, keys ...string) error {
	ttl, err := remaining(ctx, s.cache, keys...)
	if err != nil {
		return cause
	}
	return &RetryError{Cause: cause, RetryAfter: max(ttl, time.Second)}
}

/** Verify compara finalidad, correo, desafío y hash antes de consumir una sola vez.
 * @param {string} id - Opcional para clientes anteriores; siempre se valida el desafío activo del servidor.
 * @returns {bool,error} Un único ganador aun entre distintas instancias de API.
 */
func (s *OTPStore) Verify(ctx context.Context, purpose, identifier, email, id, code string) (bool, error) {
	a, err := s.atomic()
	if err != nil {
		return false, err
	}
	key := challengeKey(purpose, identifier)
	raw, err := s.cache.Get(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	var state otpState
	if json.Unmarshal([]byte(raw), &state) != nil || state.ID == "" {
		return false, ErrUnavailable
	}
	if id == "" {
		id = state.ID
	}
	bound := binding(purpose, identifier, email)
	result, err := a.RunScript(ctx, consumeOTP, []string{key}, id, bound, digest(state.ID+"|"+bound+"|"+code), MaxOTPAttempts)
	if err != nil {
		return false, ErrUnavailable
	}
	if result == -2 {
		return false, ErrOTPLocked
	}
	return result == 1, nil
}

/** Cancel invalida sólo el desafío que no pudo encolarse, sin borrar un reenvío posterior. */
func (s *OTPStore) Cancel(ctx context.Context, c Challenge) error {
	a, err := s.atomic()
	if err != nil {
		return err
	}
	_, err = a.RunScript(ctx, cancelOTP, []string{c.Key}, c.ID)
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

/** Active evita entregar correos de desafíos reemplazados, consumidos o vencidos. */
func (s *OTPStore) Active(ctx context.Context, key, id string) (bool, error) {
	if _, err := s.atomic(); err != nil {
		return false, err
	}
	raw, err := s.cache.Get(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	var c otpState
	if json.Unmarshal([]byte(raw), &c) != nil {
		return false, ErrUnavailable
	}
	return c.ID == id && c.Attempts < MaxOTPAttempts, nil
}
