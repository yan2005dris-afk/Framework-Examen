// Command demo opera exclusivamente la base y servicios synthetic-demo de desarrollo.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/competition"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/evaluation"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/exam"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/merit"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/mail"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/sqlcgen"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/platform/storage"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/privacy"
	"github.com/yan2005dris-afk/Framework-Examen/apps/backend/internal/results"
	"golang.org/x/image/font/gofont/goregular"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const mark = "DATOS SINTÉTICOS - NO VÁLIDO"
const base = "http://localhost:18080"

type account struct {
	ID       uuid.UUID `json:"id"`
	Cedula   string    `json:"cedula"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Role     string    `json:"role"`
	client   *http.Client
}
type state struct {
	Origin      string              `json:"origin"`
	Accounts    map[string]*account `json:"accounts"`
	Period      uuid.UUID           `json:"period"`
	PolicyID    string              `json:"policyId"`
	PolicyHash  string              `json:"policyHash"`
	Submissions map[string]string   `json:"submissions"`
	Resources   map[string]string   `json:"resources"`
	Checks      []string            `json:"checks"`
}

var root, out string
var pool *pgxpool.Pool
var st state

func fail(e error) {
	if e != nil {
		panic(e)
	}
}
func main() {
	if os.Getenv("DEMO_CONFIRM") != "synthetic-demo" {
		panic("Se requiere DEMO_CONFIRM=synthetic-demo")
	}
	root = os.Getenv("DEMO_ROOT")
	if root == "" {
		panic("DEMO_ROOT requerido")
	}
	out = filepath.Join(root, "apps", "backend", "data", "synthetic-demo")
	if dataset := os.Getenv("DEMO_DATASET"); dataset != "" {
		if !regexp.MustCompile(`^[a-z0-9-]{1,40}$`).MatchString(dataset) {
			panic("Dataset inválido")
		}
		out = filepath.Join(out, dataset)
	}
	u, e := url.Parse(os.Getenv("DATABASE_URL"))
	fail(e)
	if u.Path != "/framework_synthetic_demo" || u.Host != "localhost:15432" {
		panic("La demostración sólo acepta su base dedicada local")
	}
	ctx := context.Background()
	pool, e = pgxpool.New(ctx, u.String())
	fail(e)
	defer pool.Close()
	fail(pool.Ping(ctx))
	raw, e := os.ReadFile(filepath.Join(out, "state.json"))
	if e == nil {
		fail(json.Unmarshal(raw, &st))
		if st.Origin != "synthetic-demo" {
			panic("Estado ajeno")
		}
	} else {
		st = state{Origin: "synthetic-demo", Accounts: map[string]*account{}, Submissions: map[string]string{}}
	}
	if st.Resources == nil {
		st.Resources = map[string]string{}
	}
	defer func() {
		save()
		if p := recover(); p != nil {
			fmt.Fprintln(os.Stderr, "Demo incompleta:", p)
			os.Exit(1)
		}
	}()
	if len(os.Args) > 1 && os.Args[1] == "faults" {
		for _, name := range []string{"administrador", "minimo"} {
			a := st.Accounts[name]
			a.client = newClient()
			request(a, "POST", "/api/v1/auth/login", map[string]string{"cedula": a.Cedula, "password": a.Password}, 200)
		}
		faults(ctx)
	} else {
		run(ctx)
	}
}
func save() {
	raw, e := json.MarshalIndent(st, "", "  ")
	fail(e)
	fail(os.WriteFile(filepath.Join(out, "state.json"), raw, 0600))
}
func check(name string) {
	for _, v := range st.Checks {
		if v == name {
			return
		}
	}
	st.Checks = append(st.Checks, name)
	save()
	fmt.Println("OK:", name)
}
func newClient() *http.Client {
	jar, e := cookiejar.New(nil)
	fail(e)
	return &http.Client{Jar: jar, Timeout: 35 * time.Second}
}
func request(a *account, method, path string, body interface{}, want int) []byte {
	var reader io.Reader
	if body != nil {
		raw, e := json.Marshal(body)
		fail(e)
		reader = bytes.NewReader(raw)
	}
	r, e := http.NewRequest(method, base+path, reader)
	fail(e)
	r.Header.Set("Origin", "http://localhost:18081")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Idempotency-Key", uuid.NewString())
	client := &http.Client{Timeout: 35 * time.Second}
	if a != nil {
		client = a.client
	}
	resp, e := client.Do(r)
	fail(e)
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	fail(e)
	if resp.StatusCode == 429 && want == 200 && path == "/api/v1/auth/register/request" {
		seconds, e := strconv.Atoi(resp.Header.Get("Retry-After"))
		if e == nil && seconds > 0 && seconds <= 600 {
			resp.Body.Close()
			fmt.Println("OTP: se respeta Retry-After antes de reanudar")
			time.Sleep(time.Duration(seconds+1) * time.Second)
			return request(a, method, path, body, want)
		}
	}
	if resp.StatusCode != want {
		panic(fmt.Sprintf("%s %s: esperado %d, recibido %d: %s", method, path, want, resp.StatusCode, string(raw)))
	}
	return raw
}
func obj(raw []byte) map[string]interface{} {
	var v map[string]interface{}
	fail(json.Unmarshal(raw, &v))
	return v
}
func id(raw []byte) string {
	v, ok := obj(raw)["id"].(string)
	if !ok {
		panic("Respuesta sin id")
	}
	return v
}
func cedula(n int) string {
	v := fmt.Sprintf("240%06d", n)
	sum := 0
	for i, c := range v {
		d := int(c - '0')
		if i%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return v + fmt.Sprint((10-sum%10)%10)
}
func mailFor(email string, notBefore time.Time) string {
	deadline := time.Now().Add(50 * time.Second)
	for time.Now().Before(deadline) {
		resp, e := http.Get("http://localhost:18025/api/v1/messages")
		if e == nil {
			var listing struct {
				Messages []struct {
					ID      string
					Created time.Time
					To      []struct{ Address string }
				}
			}
			_ = json.NewDecoder(resp.Body).Decode(&listing)
			resp.Body.Close()
			for _, m := range listing.Messages {
				if m.Created.Before(notBefore) {
					continue
				}
				for _, to := range m.To {
					if to.Address == email {
						r, e := http.Get("http://localhost:18025/api/v1/message/" + m.ID)
						if e != nil {
							continue
						}
						var message struct{ Text string }
						_ = json.NewDecoder(r.Body).Decode(&message)
						r.Body.Close()
						re := regexp.MustCompile(`\b[0-9]{6}\b`)
						if code := re.FindString(message.Text); code != "" {
							return code
						}
					}
				}
			}
		}
		time.Sleep(time.Second)
	}
	panic("OTP no apareció en Mailpit sintético")
}
func register(name string, n int) *account {
	a := st.Accounts[name]
	if a == nil {
		var random [18]byte
		_, e := rand.Read(random[:])
		fail(e)
		seed, e := rand.Int(rand.Reader, big.NewInt(700000))
		fail(e)
		n = 200000 + int(seed.Int64())
		a = &account{Cedula: cedula(n), Email: fmt.Sprintf("%s-%s@synthetic.invalid", name, uuid.NewString()[:8]), Password: "TestPassword.2026*", Role: "postulante"}
		st.Accounts[name] = a
		save()
	}
	a.client = newClient()
	if a.ID == uuid.Nil {
		issuedAfter := time.Now().UTC().Add(-100 * time.Millisecond)
		challenge := obj(request(a, "POST", "/api/v1/auth/register/request", map[string]interface{}{"cedula": a.Cedula, "correo": a.Email}, 200))
		code := mailFor(a.Email, issuedAfter)
		request(a, "POST", "/api/v1/auth/register/verify", map[string]interface{}{"cedula": a.Cedula, "correo": a.Email, "codigo_otp": code, "desafio_id": challenge["desafio_id"], "nombres": "SINTETICO " + name, "apellidos": mark, "password": a.Password}, 201)
		fail(pool.QueryRow(context.Background(), "SELECT id FROM applicants WHERE correo=$1", a.Email).Scan(&a.ID))
		save()
	}
	request(a, "POST", "/api/v1/auth/login", map[string]string{"cedula": a.Cedula, "password": a.Password}, 200)
	return a
}
func role(ctx context.Context, a *account, role string) {
	tx, e := pool.Begin(ctx)
	fail(e)
	defer tx.Rollback(ctx)
	q := sqlcgen.New(tx)
	fail(q.SetRoleChangeContext(ctx, sqlcgen.SetRoleChangeContextParams{Operator: "synthetic-demo bootstrap", Reason: mark}))
	_, e = q.UpdateApplicantRoleByCedula(ctx, sqlcgen.UpdateApplicantRoleByCedulaParams{Cedula: a.Cedula, Role: role})
	fail(e)
	fail(tx.Commit(ctx))
	a.Role = role
	save()
}
func restart() {
	cmd := exec.Command("docker", "restart", "framework-backend-demo-api-1")
	if data, e := cmd.CombinedOutput(); e != nil {
		panic(string(data))
	}
	for i := 0; i < 40; i++ {
		r, e := http.Get(base + "/readyz")
		if e == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				return
			}
		}
		time.Sleep(time.Second)
	}
	panic("API demo no recuperó readiness")
}
func run(ctx context.Context) {
	request(nil, "GET", "/healthz", nil, 200)
	request(nil, "GET", "/readyz", nil, 200)
	check("salud y readiness")
	admin := register("administrador", 100101)
	analyst := register("analista", 100102)
	role(ctx, admin, "administrador")
	role(ctx, analyst, "analista")
	for i, name := range []string{"aprobado", "excluido", "reconsideracion", "vencido", "minimo", "cerrado", "abierta-jubilados"} {
		register(name, 100110+i)
	}
	check("registro OTP real en Mailpit y login")
	eligibility := map[string]map[string]bool{}
	for name, a := range st.Accounts {
		eligibility[a.ID.String()] = map[string]bool{"afiliados": name != "cerrado", "jubilados": true, "empleadores": true}
	}
	data, e := json.Marshal(eligibility)
	fail(e)
	fail(os.WriteFile(filepath.Join(out, "eligibility.json"), data, 0600))
	restart()
	if st.Period == uuid.Nil {
		raw := request(admin, "POST", "/api/v1/admin/competitions", map[string]interface{}{"codigo": "demo-2026-01", "titulo": "Convocatoria sintética BIESS 2026 - " + mark, "apertura": "2026-01-05T00:00:00-05:00", "cierre": "2026-12-31T23:59:59.999999-05:00", "sectores": []string{"afiliados", "jubilados", "empleadores"}}, 201)
		st.Period = uuid.MustParse(id(raw))
		save()
	}
	configure(admin, false)
	setupPolicies(admin)
	for _, name := range []string{"aprobado", "excluido", "reconsideracion", "vencido", "minimo", "abierta-jubilados"} {
		createSubmission(name)
	}
	// Cierre de elegibilidad no puede evadirse con un segundo borrador.
	a := st.Accounts["cerrado"]
	request(a, "POST", "/api/v1/submissions/", map[string]interface{}{"convocatoriaId": st.Period, "content": map[string]interface{}{"seccion1": map[string]string{"nacionalidad": "Ecuatoriana"}, "seccion4": map[string]string{"sector": "afiliados"}}}, 409)
	check("EF12 secciones, documentos y sello")
	documentReplacement(st.Accounts["minimo"], st.Submissions["minimo"])
	workflow(admin, analyst)
	negativeCases(admin)
	opposition(admin, analyst)
	auxiliaryFixtures(ctx)
	verifyAudit(ctx, admin)
	verifyMailpitAttachment()
}
func configure(admin *account, final bool) {
	var c competition.Competition
	fail(json.Unmarshal(request(admin, "GET", "/api/v1/admin/competitions/"+st.Period.String(), nil, 200), &c))
	start, _ := time.Parse(time.RFC3339Nano, "2026-01-05T00:00:00-05:00")
	end, _ := time.Parse(time.RFC3339Nano, "2026-12-31T23:59:59.999999-05:00")
	phases := []competition.Phase{}
	for _, code := range []string{"postulacion", "meritos", "oposicion_psicotecnica", "oposicion_teorica", "reconsideracion", "resultados"} {
		close := end
		enabled := true
		if final && (code == "postulacion" || code == "reconsideracion") {
			close = time.Now().UTC().Add(-time.Second).Truncate(time.Microsecond)
			enabled = false
		}
		phases = append(phases, competition.Phase{Code: code, Title: mark + " " + code, OpensAt: start, ClosesAt: close, Enabled: enabled, Published: true})
	}
	request(admin, "PUT", "/api/v1/admin/competitions/"+st.Period.String()+"/configuration", competition.Configuration{Version: c.Version, Approved: true, Published: true, FormEnabled: !final, Sectors: []string{"afiliados", "jubilados", "empleadores"}, Phases: phases}, 200)
}
func setupPolicies(admin *account) {
	if st.PolicyID != "" {
		return
	}
	rules := privacy.Rules{Text: "Los datos de este expediente son sintéticos y se usan exclusivamente para probar el sistema. La publicación de nombres requiere consentimiento explícito. Los documentos, resultados y notificaciones se conservan durante los períodos definidos en esta política de demostración.", Treatment: mark + " Origen synthetic-demo; eliminación sólo simulada; auditoría al menos 730 días; sesiones expiradas 30 días.", Reference: "politica-demo-2026-v1 - " + mark, AccessDays: 365, DocumentDays: 365, ReportDays: 365, PublicDays: 365, EmailDays: 180, ReviewDays: 365, Historical: "revision_humana_sin_borrado_automatico", AllowExports: true, AllowEmail: true, AllowNames: true}
	raw := obj(request(admin, "POST", "/api/v1/admin/privacy/competitions/"+st.Period.String()+"/policies", privacy.PublishInput{Rules: rules}, 200))
	st.PolicyID = raw["id"].(string)
	st.PolicyHash = raw["hash"].(string)
	save()
}
func createSubmission(name string) {
	a := st.Accounts[name]
	sub := st.Submissions[name]
	if sub != "" {
		existing := obj(request(a, "GET", "/api/v1/submissions/"+sub, nil, 200))
		if existing["status"] == "submitted" {
			return
		}
	}
	raw, e := os.ReadFile(filepath.Join(root, "apps/backend/internal/submission/testdata/form_cases.json"))
	fail(e)
	var fixture struct {
		Base map[string]interface{} `json:"base"`
	}
	fail(json.Unmarshal(raw, &fixture))
	form := fixture.Base
	if strings.HasSuffix(name, "-jubilados") {
		form["seccion4"].(map[string]interface{})["sector"] = "jubilados"
	}
	personal := form["seccion1"].(map[string]interface{})
	personal["cedula"] = a.Cedula
	personal["nombres"] = "SINTETICO " + name
	personal["apellidos"] = mark
	form["seccion2"].(map[string]interface{})["correo1"] = a.Email
	form["seccion6"].(map[string]interface{})["experiencias"].([]interface{})[0].(map[string]interface{})["fechaFin"] = "2024-01-01"
	form["seccion7"] = map[string]interface{}{"cuentaConPublicaciones": "Sí", "publicaciones": []interface{}{map[string]interface{}{"id": "pub-1", "titulo": mark, "tipo": "Publicación", "medio": "synthetic-demo", "fechaPublicacion": "2020-01-01", "relacionCampo": "Sí"}}}
	form["seccion8"] = map[string]interface{}{"capacitaciones": []interface{}{map[string]interface{}{"id": "curso-1", "tipo": "Curso", "nombre": mark, "institucion": "synthetic-demo", "fechaInicio": "2020-01-01", "fechaFin": "2020-02-01", "horas": 100}}}
	if sub == "" {
		sub = id(request(a, "POST", "/api/v1/submissions/", map[string]interface{}{"convocatoriaId": st.Period}, 201))
		st.Submissions[name] = sub
		save()
	}
	for n := 1; n <= 9; n++ {
		request(a, "POST", "/api/v1/submissions/"+sub+fmt.Sprintf("/sections/%d", n), form[fmt.Sprintf("seccion%d", n)], 200)
	}
	if name == "minimo" {
		check("guardado parcial y reanudación")
		return
	}
	for section, record := range map[int]string{5: "titulo-1", 6: "experiencia-1", 7: "pub-1", 8: "curso-1"} {
		upload(a, sub, section, record)
	}
	consent := form["consentimiento"].(map[string]interface{})
	consent["politicaPrivacidadId"] = st.PolicyID
	consent["hashPoliticaPrivacidad"] = st.PolicyHash
	request(a, "POST", "/api/v1/submissions/"+sub+"/submit", map[string]interface{}{"consentimiento": consent}, 200)
	pdf := request(a, "GET", "/api/v1/submissions/"+sub+"/print.pdf", nil, 200)
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		panic("sin PDF sellado")
	}
	fail(os.WriteFile(filepath.Join(out, name+"-ef12.pdf"), pdf, 0600))
}
func resource(key string, a *account, path string, body interface{}, status int) string {
	if v := st.Resources[key]; v != "" {
		return v
	}
	v := id(request(a, "POST", path, body, status))
	st.Resources[key] = v
	save()
	return v
}
func workflow(admin, analyst *account) {
	workflowSector(admin, analyst, "afiliados", []string{"aprobado", "excluido", "reconsideracion", "vencido"})
	workflowSector(admin, analyst, "jubilados", []string{"abierta-jubilados"})
}
func workflowSector(admin, analyst *account, sector string, names []string) {
	ruleKey := "merit-rules"
	approvedKey := "merit-approved"
	if sector != "afiliados" {
		ruleKey += "-" + sector
		approvedKey += "-" + sector
	}
	maximum, threshold := int64(10000), int64(7000)
	rules := merit.RuleDefinition{MaximumTotal: &maximum, Threshold: &threshold, Rounding: "mas_cercano", ExperienceAggregation: "unir_intervalos", DayCounting: "dias_transcurridos", Requirements: []merit.Requirement{{Code: "checklist", Label: mark}}, TieBreakers: []string{"experiencia", "formacion", "resolucion_manual"}}
	for _, v := range []struct {
		code, metric         string
		weight, unit, points int64
	}{{"formacion", "education_count", 3000, 1, 10000}, {"experiencia", "experience_days", 3000, 365, 2500}, {"capacitacion", "training_hours", 2000, 100, 10000}, {"publicaciones", "publication_count", 1000, 1, 10000}, {"checklist", "requirements_approved", 1000, 1, 10000}} {
		rules.Criteria = append(rules.Criteria, merit.Criterion{Code: v.code, Label: mark + " " + v.code, Metric: v.metric, UnitSize: v.unit, PointsPerUnit: v.points, WholeUnits: v.metric == "experience_days", MaximumPoints: 10000, Weight: v.weight, Required: v.metric != "requirements_approved"})
	}
	for n := 1; n <= 13; n++ {
		answer := "No"
		if n == 1 {
			answer = "Sí"
		}
		rules.Questions = append(rules.Questions, merit.QuestionRule{ID: fmt.Sprintf("q%d", n), AcceptedAnswer: answer})
	}
	rid := resource(ruleKey, admin, "/api/v1/admin/merits/rules", map[string]interface{}{"convocatoriaId": st.Period, "sector": sector, "definicion": rules}, 201)
	if st.Resources[approvedKey] == "" {
		request(admin, "POST", "/api/v1/admin/merits/rules/"+rid+"/approve", map[string]string{"referenciaAprobacion": "DEMO-SINTETICA-2026 - " + mark}, 200)
		st.Resources[approvedKey] = rid
		save()
	}
	for _, name := range names {
		sub := st.Submissions[name]
		if st.Resources[name+"-merit"] != "" {
			continue
		}
		var persisted merit.Case
		fail(json.Unmarshal(request(analyst, "GET", "/api/v1/admin/merits/submissions/"+sub, nil, 200), &persisted))
		if persisted.Publication != nil {
			if persisted.Publication.RuleSetID != uuid.MustParse(rid) {
				panic("Publicación de otra versión: requiere revisión")
			}
			st.Resources[name+"-merit"] = persisted.Publication.ID.String()
			save()
			continue
		}
		docs := request(analyst, "GET", "/api/v1/admin/submissions/"+sub+"/documents", nil, 200)
		var ds []struct {
			ID      string `json:"id"`
			Section int    `json:"section"`
			Record  string `json:"recordId"`
		}
		fail(json.Unmarshal(docs, &ds))
		review := merit.ReviewInput{RuleSetID: uuid.MustParse(rid), Motive: mark, Requirements: []merit.RequirementReview{{Code: "checklist", Satisfied: name != "excluido", Observation: mark}}}
		for _, d := range ds {
			review.Documents = append(review.Documents, merit.RecordReview{Section: d.Section, RecordID: d.Record, DocumentID: d.ID, Verdict: "valido", Observation: mark})
		}
		for n := 1; n <= 13; n++ {
			review.Questions = append(review.Questions, merit.QuestionReview{ID: fmt.Sprintf("q%d", n), Confirmed: true, Observation: mark})
		}
		var revision string
		if persisted.Review != nil {
			revision = persisted.Review.ID.String()
		} else {
			response := obj(request(analyst, "POST", "/api/v1/admin/merits/submissions/"+sub+"/review", review, 201))
			revision = response["revision"].(map[string]interface{})["id"].(string)
		}
		var calc string
		if persisted.Calculation != nil && persisted.Calculation.ReviewID == uuid.MustParse(revision) {
			calc = persisted.Calculation.ID.String()
		} else {
			version := int64(0)
			if persisted.Calculation != nil {
				version = persisted.Calculation.Version
			}
			response := obj(request(analyst, "POST", "/api/v1/admin/merits/submissions/"+sub+"/calculate", merit.CalculateInput{ReviewID: uuid.MustParse(revision), Version: version, Motive: mark}, 201))
			calc = response["calculo"].(map[string]interface{})["id"].(string)
		}
		decision := "avanza"
		if name == "excluido" {
			decision = "excluido"
		}
		raw := obj(request(admin, "POST", "/api/v1/admin/merits/submissions/"+sub+"/publish", merit.PublishInput{CalculationID: uuid.MustParse(calc), Decision: decision, Motive: mark}, 201))
		st.Resources[name+"-merit"] = raw["publicacion"].(map[string]interface{})["id"].(string)
		save()
	}
	check("revisión, cálculo y publicación méritos por API")
}

func upload(a *account, sub string, section int, record string) map[string]interface{} {
	return uploadStatus(a, sub, section, record, 201)
}
func uploadStatus(a *account, sub string, section int, record string, want int) map[string]interface{} {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fail(mw.WriteField("section", fmt.Sprint(section)))
	fail(mw.WriteField("recordId", record))
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="synthetic-demo.pdf"`)
	header.Set("Content-Type", "application/pdf")
	p, e := mw.CreatePart(header)
	fail(e)
	_, e = p.Write(syntheticPDF("Documento ficticio de respaldo EF-12. Origen: synthetic-demo."))
	fail(e)
	fail(mw.Close())
	r, e := http.NewRequest("POST", base+"/api/v1/submissions/"+sub+"/documents", &b)
	fail(e)
	r.Header.Set("Origin", "http://localhost:18081")
	r.Header.Set("Content-Type", mw.FormDataContentType())
	resp, e := a.client.Do(r)
	fail(e)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		panic(fmt.Sprintf("documento: %d %s", resp.StatusCode, raw))
	}
	return obj(raw)
}

func opposition(admin, analyst *account) {
	oppositionSector(admin, analyst, "jubilados", []string{"abierta-jubilados"}, false)
	openAppealFixtures(admin, analyst)
	oppositionSector(admin, analyst, "afiliados", []string{"aprobado", "reconsideracion", "vencido"}, true)
}
func oppositionSector(admin, analyst *account, sector string, names []string, finalize bool) {
	suffix := ""
	if sector != "afiliados" {
		suffix = "-" + sector
	}
	maximum := int64(100)
	minimum := int64(60)
	def := evaluation.Definition{Opposition: &evaluation.OppositionRules{Instrument: "PSICO-DEMO-1", InstrumentVersion: "1.0", PsychometricMode: "puntaje", PsychometricMaximum: &maximum, PsychometricMinimum: &minimum, Conclusions: []string{"admitido-demo", "no-admitido-demo"}, AcceptedConclusions: []string{"admitido-demo"}, TheoryMaximum: 100, MeritWeight: 4000, TheoryWeight: 3000, PsychometricWeight: 3000, TotalMaximum: 100, Threshold: 70, Rounding: "mas_cercano"}}
	pol := resource("opposition-policy-v2"+suffix, admin, "/api/v1/admin/grades/policies", map[string]interface{}{"convocatoriaId": st.Period, "sector": sector, "tipo": "oposicion", "definicion": def}, 201)
	approvePolicy(admin, pol)
	terms := evaluation.AppealRules{Applicant: evaluation.Term{Quantity: 5, Unit: "dias_calendario", Cutoff: "23:59"}, Talent: evaluation.Term{Quantity: 3, Unit: "dias_calendario", Cutoff: "23:59"}, Commission: evaluation.Term{Quantity: 3, Unit: "dias_calendario", Cutoff: "23:59"}, Calendar: evaluation.Calendar{EndPolicy: "limitar_a_fase", TimeZone: "America/Guayaquil", Weekdays: []int{0, 1, 2, 3, 4, 5, 6}, Holidays: []string{}}}
	ap := resource("appeal-policy"+suffix, admin, "/api/v1/admin/grades/policies", map[string]interface{}{"convocatoriaId": st.Period, "sector": sector, "tipo": "reconsideracion", "definicion": evaluation.Definition{Reconsideration: &terms}}, 201)
	approvePolicy(admin, ap)
	bank := exam.BankInput{Reference: "DEMO-BANCO-2026 banco-demo-2026-v1 - " + mark}
	for n := 1; n <= 20; n++ {
		bank.Questions = append(bank.Questions, exam.Question{ID: fmt.Sprintf("demo-%02d", n), Text: fmt.Sprintf("%s. Ejercicio ficticio %d: ¿qué opción contiene la letra A?", mark, n), Options: []exam.Option{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}, {ID: "d", Text: "D"}}, Correct: "a"})
	}
	raw, _ := json.MarshalIndent(bank, "", "  ")
	fail(os.WriteFile(filepath.Join(out, "banco-demo-2026-v1.json"), raw, 0600))
	bid := resource("exam-bank"+suffix, admin, "/api/v1/admin/question-bank/competitions/"+st.Period.String()+"/sectors/"+sector+"/versions", bank, 200)
	start, _ := time.Parse(time.RFC3339, "2026-01-05T00:00:00-05:00")
	end, _ := time.Parse(time.RFC3339, "2026-12-31T23:59:59-05:00")
	config := resource("exam-config"+suffix, admin, "/api/v1/admin/question-bank/versions/"+bid+"/configurations", exam.ConfigRules{Count: 10, DurationSeconds: 3600, Maximum: 100, Threshold: 70, Rounding: "mas_cercano", AttemptPolicy: "un_intento", UniquePolicy: "orden_preguntas_unico_rechazar_agotamiento", Reference: "DEMO-BANCO-2026 - " + mark, OpensAt: start, ClosesAt: end}, 200)

	fastConfig := ""
	if finalize && os.Getenv("DEMO_FAST_EXPIRY") == "1" {
		bank.PreviousVersion = 1
		bank.Reference = "DEMO-QA-VENCIMIENTO-5S - " + mark
		fastBank := resource("exam-bank-fast", admin, "/api/v1/admin/question-bank/competitions/"+st.Period.String()+"/sectors/"+sector+"/versions", bank, 200)
		fastConfig = resource("exam-config-fast", admin, "/api/v1/admin/question-bank/versions/"+fastBank+"/configurations", exam.ConfigRules{Count: 10, DurationSeconds: 5, Maximum: 100, Threshold: 70, Rounding: "mas_cercano", AttemptPolicy: "un_intento", UniquePolicy: "orden_preguntas_unico_rechazar_agotamiento", Reference: "DEMO-QA-VENCIMIENTO-5S - " + mark, OpensAt: start, ClosesAt: end}, 200)
	}
	for _, name := range names {
		a := st.Accounts[name]
		sub := st.Submissions[name]
		score := int64(80)
		var current evaluation.Case
		fail(json.Unmarshal(request(admin, "GET", "/api/v1/admin/grades/submissions/"+sub, nil, 200), &current))
		previousPsy := int64(0)
		if current.State.Psychometric != nil {
			previousPsy = current.State.Psychometric.Version
		}
		psy := resource(name+"-psy-v2", analyst, "/api/v1/admin/grades/submissions/"+sub+"/psychometric", evaluation.PsychometricInput{PreviousVersion: previousPsy, PolicyID: uuid.MustParse(pol), EvaluatedAt: time.Now().UTC(), Score: &score, Conclusion: "admitido-demo", Motive: mark, Evidence: evaluation.EvidenceInput{Name: "informe-psicotecnico-sintetico.pdf", PDF: syntheticPDF("Informe psicotécnico ficticio. No representa un diagnóstico.")}}, 201)
		previousPublished := current.State.Results["oposicion_psicotecnica"]
		var supersedes *uuid.UUID
		if previousPublished.ID != uuid.Nil {
			id := previousPublished.ID
			supersedes = &id
		}
		published := resource(name+"-psy-published-v2", admin, "/api/v1/admin/grades/submissions/"+sub+"/psychometric/publish", evaluation.PublishInput{PreviousVersion: previousPublished.Version, SupersedesID: supersedes, SourceID: uuid.MustParse(psy), PolicyID: uuid.MustParse(pol), Motive: mark}, 201)
		st.Resources[name+"-psy-published"] = published
		save()
		assignedConfig := config
		if name == "vencido" && fastConfig != "" {
			assignedConfig = fastConfig
		}
		if st.Resources[name+"-enrolled"] == "" {
			request(admin, "POST", "/api/v1/admin/question-bank/submissions/"+sub+"/enable", exam.EnrollmentInput{ConfigID: uuid.MustParse(assignedConfig), MeritID: uuid.MustParse(st.Resources[name+"-merit"])}, 200)
			st.Resources[name+"-enrolled"] = assignedConfig
			save()
		}
		var view exam.View
		fail(json.Unmarshal(request(a, "POST", "/api/v1/exams/submissions/"+sub+"/start", map[string]string{}, 200), &view))
		if name == "vencido" && view.Status != "finished" {
			var q exam.PublicQuestion
			fail(json.Unmarshal(request(a, "GET", "/api/v1/exams/submissions/"+sub+"/questions/1", nil, 200), &q))
			if q.Selected == "" {
				request(a, "PUT", "/api/v1/exams/submissions/"+sub+"/answers", exam.AnswerInput{PreviousVersion: q.Version, QuestionID: q.ID, OptionID: "a"}, 200)
			}
			if fastConfig == "" {
				check("intento de 60 minutos iniciado; vencimiento pendiente de prueba temporal")
				continue
			}
			time.Sleep(6 * time.Second)
			fail(json.Unmarshal(request(a, "GET", "/api/v1/exams/submissions/"+sub, nil, 200), &view))
			if view.Status != "finished" {
				panic("El servidor no cerró el intento QA vencido")
			}
			check("vencimiento real QA 5s con respuesta parcial y nota 10; examen principal conserva 3600s")
		}
		if view.Status != "finished" {
			for index := 1; index <= 10; index++ {
				var q exam.PublicQuestion
				fail(json.Unmarshal(request(a, "GET", fmt.Sprintf("/api/v1/exams/submissions/%s/questions/%d", sub, index), nil, 200), &q))
				if q.Selected == "a" {
					continue
				}
				fail(json.Unmarshal(request(a, "PUT", "/api/v1/exams/submissions/"+sub+"/answers", exam.AnswerInput{PreviousVersion: q.Version, QuestionID: q.ID, OptionID: "a"}, 200), &view))
			}
			fail(json.Unmarshal(request(a, "POST", "/api/v1/exams/submissions/"+sub+"/finish", map[string]string{}, 200), &view))
		}
		wantScore := int64(100)
		if name == "vencido" {
			wantScore = 10
		}
		if view.Result == nil || *view.Result.Score != wantScore {
			panic("nota inesperada en examen sintético")
		}
		request(a, "POST", "/api/v1/exams/submissions/"+sub+"/finish", map[string]string{}, 200)
		pdf := request(a, "GET", "/api/v1/exams/submissions/"+sub+"/report.pdf", nil, 200)
		fail(os.WriteFile(filepath.Join(out, name+"-examen.pdf"), pdf, 0600))
		st.Resources[name+"-theory"] = view.Result.ID.String()
		save()
		expected := map[string]uuid.UUID{"meritos": uuid.MustParse(st.Resources[name+"-merit"]), "oposicion_teorica": view.Result.ID, "oposicion_psicotecnica": uuid.MustParse(st.Resources[name+"-psy-published"])}
		oldConsolidated := current.State.Results["resultados"]
		var supersedesConsolidated *uuid.UUID
		if oldConsolidated.ID != uuid.Nil {
			id := oldConsolidated.ID
			supersedesConsolidated = &id
		}
		resource(name+"-consolidated-v2", admin, "/api/v1/admin/grades/submissions/"+sub+"/consolidate", evaluation.PublishInput{PreviousVersion: oldConsolidated.Version, SupersedesID: supersedesConsolidated, PolicyID: uuid.MustParse(pol), ExpectedSources: expected, Motive: mark}, 201)
	}
	check("psicotécnica publicada, banco 20/10, examen finalizado idempotente y consolidación 40/30/30")
	if !finalize {
		return
	}
	// Primero se observa cada estado; resolver todos antes de generar la lista.
	for i, name := range []string{"excluido", "reconsideracion", "aprobado"} {
		sub := st.Submissions[name]
		aid := resource(name+"-appeal", st.Accounts[name], "/api/v1/reconsiderations/submissions/"+sub, evaluation.FileAppealInput{Phase: "meritos", ResultID: uuid.MustParse(st.Resources[name+"-merit"]), Motive: mark}, 201)
		if st.Resources[name+"-attended"] == "" {
			request(analyst, "POST", "/api/v1/admin/reconsiderations/submissions/"+sub+"/"+aid+"/attend", evaluation.AttendInput{PreviousVersion: 1, Observation: mark, Recommendation: "confirmar"}, 201)
			st.Resources[name+"-attended"] = aid
			save()
		}
		if i == 0 {
			check("reconsideraciones: recibida y atendida observadas por API")
		}
		if st.Resources[name+"-resolved"] == "" {
			request(admin, "POST", "/api/v1/admin/reconsiderations/submissions/"+sub+"/"+aid+"/resolve", evaluation.ResolveInput{PreviousVersion: 2, Decision: "confirmar", Motive: mark}, 201)
			st.Resources[name+"-resolved"] = aid
			save()
		}
	}
	check("reconsideración resuelta por Comisión con separación de roles")
	// Si un intento de una hora sigue abierto, la lista falla de forma cerrada, nunca se inventa una nota.
	if st.Resources["vencido-theory"] == "" {
		check("lista definitiva bloqueada hasta cierre del examen pendiente")
		return
	}
	configure(admin, true)
	lid := resource("results-list", admin, "/api/v1/admin/results/competitions/"+st.Period.String()+"/sectors/afiliados/lists", results.GenerateInput{Rules: results.Rules{Order: []string{"resultados", "meritos"}, Reference: mark}}, 200)
	resource("results-publication", admin, "/api/v1/admin/results/lists/"+lid+"/publish", results.PublishInput{Reference: mark, Public: true, PrivacyReference: "politica-demo-2026-v1 - " + mark}, 200)
	request(nil, "GET", "/api/v1/public/results/competitions/"+st.Period.String()+"/sectors/afiliados", nil, 200)
	csv := request(admin, "GET", "/api/v1/admin/results/lists/"+lid+"/export.csv", nil, 200)
	fail(os.WriteFile(filepath.Join(out, "lista-interna.csv"), csv, 0600))
	check("lista interna y proyección pública con consentimiento")
}
func approvePolicy(admin *account, pol string) {
	if st.Resources[pol+"-approved"] != "" {
		return
	}
	request(admin, "POST", "/api/v1/admin/grades/policies/"+pol+"/approve", map[string]string{"referenciaAprobacion": "DEMO-SINTETICA-2026 - " + mark}, 200)
	st.Resources[pol+"-approved"] = pol
	save()
}

func negativeCases(admin *account) {
	a := st.Accounts["minimo"]
	sub := st.Submissions["minimo"]
	request(nil, "GET", "/api/v1/submissions/"+sub, nil, 401)
	request(a, "GET", "/api/v1/admin/competitions/"+st.Period.String(), nil, 403)
	request(admin, "POST", "/api/v1/admin/question-bank/versions/"+uuid.NewString()+"/configurations", map[string]string{}, 422)
	request(st.Accounts["aprobado"], "POST", "/api/v1/submissions/"+st.Submissions["aprobado"]+"/sections/1", map[string]string{}, 409)
	request(a, "POST", "/api/v1/submissions/"+sub+"/sections/1", map[string]string{"nombres": strings.Repeat("x", 11<<20)}, 413)
	request(st.Accounts["cerrado"], "POST", "/api/v1/submissions/", map[string]interface{}{"convocatoriaId": st.Period}, 409)
	check("negativos HTTP 401/403/409/422/413 y cierre no evadible")
}

func container(action, service string) {
	allowed := map[string]bool{"api": true, "postgres": true, "redis": true, "rustfs": true, "mailpit": true}
	if !allowed[service] || (action != "stop" && action != "start") {
		panic("Contenedor ajeno")
	}
	data, e := exec.Command("docker", action, "framework-backend-demo-"+service+"-1").CombinedOutput()
	if e != nil {
		panic(string(data))
	}
}
func ready() {
	for i := 0; i < 60; i++ {
		r, e := (&http.Client{Timeout: 3 * time.Second}).Get(base + "/readyz")
		if e == nil {
			r.Body.Close()
			if r.StatusCode == 200 {
				return
			}
		}
		time.Sleep(time.Second)
	}
	panic("Readiness no recuperó")
}
func faults(ctx context.Context) {
	configure(st.Accounts["administrador"], false)
	defer configure(st.Accounts["administrador"], true)
	randomSeed, e := rand.Int(rand.Reader, big.NewInt(600000))
	fail(e)
	faultSeed := 200000 + int(randomSeed.Int64())
	// Restaurar cada dependencia incluso si falla una aserción.
	func() {
		container("stop", "redis")
		defer container("start", "redis")
		request(nil, "POST", "/api/v1/auth/register/request", map[string]string{"cedula": cedula(faultSeed), "correo": "redis-outage@synthetic.invalid"}, 503)
	}()
	check("Redis caído: OTP rechaza 503 sin eludir seguridad; servicio recuperado")
	time.Sleep(2 * time.Second)
	func() {
		container("stop", "postgres")
		defer func() { container("start", "postgres"); ready() }()
		request(nil, "GET", "/healthz", nil, 200)
		request(nil, "GET", "/readyz", nil, 503)
	}()
	check("PostgreSQL caído: liveness 200 y readiness 503; recuperación confirmada")
	func() {
		container("stop", "rustfs")
		defer container("start", "rustfs")
		uploadStatus(st.Accounts["minimo"], st.Submissions["minimo"], 5, "titulo-1", 503)
	}()
	time.Sleep(4 * time.Second)
	s, e := storage.NewS3(storage.S3Config{Endpoint: "http://localhost:19000/synthetic-demo", Region: "us-east-1", AccessKey: os.Getenv("DEMO_S3_ACCESS_KEY"), SecretKey: os.Getenv("DEMO_S3_SECRET_KEY")})
	fail(e)
	var key string
	fail(pool.QueryRow(ctx, `SELECT storage_key FROM documents LIMIT 1`).Scan(&key))
	signed, e := s.PresignGet(ctx, key, "synthetic-demo.pdf", 5*time.Minute)
	fail(e)
	r, e := http.Get(signed)
	fail(e)
	r.Body.Close()
	if r.StatusCode != 200 {
		panic("S3 no recuperó")
	}
	check("RustFS caído: carga 503 sin documento confirmado; recuperación y descarga real")
	func() {
		container("stop", "mailpit")
		defer container("start", "mailpit")
		request(nil, "POST", "/api/v1/auth/register/request", map[string]string{"cedula": cedula(faultSeed + 1), "correo": "smtp-outage-" + uuid.NewString() + "@synthetic.invalid"}, 200)
		var job string
		fail(pool.QueryRow(ctx, `SELECT id::text FROM mail_outbox ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&job))
		waitMail(ctx, job, "pending", true)
		container("stop", "api")
		container("start", "api")
		ready()
		var status string
		fail(pool.QueryRow(ctx, `SELECT status FROM mail_outbox WHERE id=$1`, job).Scan(&status))
		if status != "pending" && status != "processing" {
			panic("Se perdió el trabajo SMTP pendiente")
		}
		st.Resources["smtp-retry-job"] = job
		save()
	}()
	waitMail(ctx, st.Resources["smtp-retry-job"], "smtp_accepted", false)
	check("SMTP caído y reinicio de API: trabajo durable reintentado y aceptado por Mailpit")
	var wg sync.WaitGroup
	codes := make(chan int, 100)
	for n := 0; n < 100; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := (&http.Client{Timeout: 10 * time.Second}).Get("http://localhost:18081/api/v1/submissions/")
			if e != nil {
				codes <- 0
				return
			}
			r.Body.Close()
			codes <- r.StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	limited := 0
	for c := range codes {
		if c == 429 {
			limited++
		}
	}
	if limited == 0 {
		panic("nginx no limitó la ráfaga")
	}
	check("nginx: límite general por IP devuelve 429 bajo ráfaga real")
	raw, e := json.MarshalIndent(map[string]interface{}{"origin": "synthetic-demo", "warning": mark, "nginx429": limited, "checks": st.Checks}, "", "  ")
	fail(e)
	fail(os.WriteFile(filepath.Join(out, "fault-evidence.json"), raw, 0600))
}
func waitMail(ctx context.Context, job, want string, errorCode bool) {
	for n := 0; n < 75; n++ {
		var status, code string
		fail(pool.QueryRow(ctx, `SELECT status,last_error_code FROM mail_outbox WHERE id=$1`, job).Scan(&status, &code))
		if status == want && (!errorCode || code == "smtp_error") {
			return
		}
		time.Sleep(time.Second)
	}
	panic("El outbox no llegó al estado esperado")
}

func auxiliaryFixtures(ctx context.Context) {
	// El contrato vigente prohíbe crear legados nuevos. Se prueban en esquemas históricos aislados.
	queue, e := mail.NewOutbox(sqlcgen.New(pool), os.Getenv("MAIL_OUTBOX_KEY"))
	fail(e)
	for _, kind := range []string{"pending", "failure"} {
		key := "mail-fixture-" + kind
		if st.Resources[key] != "" {
			continue
		}
		resource := uuid.New()
		to := "pending@synthetic.invalid"
		if kind == "failure" {
			to = "invalid-address"
		}
		fail(queue.Send(ctx, mail.Message{Kind: "otp.synthetic-demo." + kind, ResourceID: resource.String(), To: to, Subject: mark, Body: mark + " Fixture operativo, no corresponde a una persona ni a un resultado institucional."}))
		if kind == "pending" {
			_, e = pool.Exec(ctx, `UPDATE mail_outbox SET next_attempt_at=clock_timestamp()+interval '24 hours' WHERE resource_id=$1 AND kind='otp.synthetic-demo.pending'`, resource)
			fail(e)
		}
		st.Resources[key] = resource.String()
		save()
	}
	sender := &mail.MarkedSender{Next: mail.NewSMTPSender(mail.Config{Host: "localhost", Port: 11025, From: "synthetic-demo@example.test", Timeout: 5 * time.Second}), Mark: mark}
	for n := 0; n < 12; n++ {
		var state string
		fail(pool.QueryRow(ctx, `SELECT status FROM mail_outbox WHERE resource_id=$1`, st.Resources["mail-fixture-failure"]).Scan(&state))
		if state == "failed" {
			break
		}
		time.Sleep(200 * time.Millisecond)
		_, e = pool.Exec(ctx, `UPDATE mail_outbox SET next_attempt_at=clock_timestamp() WHERE resource_id=$1 AND kind='otp.synthetic-demo.failure' AND status='pending'`, st.Resources["mail-fixture-failure"])
		fail(e)
		fail(queue.Dispatch(ctx, sender, nil))
	}
	var failed, pending string
	fail(pool.QueryRow(ctx, `SELECT status FROM mail_outbox WHERE resource_id=$1`, st.Resources["mail-fixture-failure"]).Scan(&failed))
	fail(pool.QueryRow(ctx, `SELECT status FROM mail_outbox WHERE resource_id=$1`, st.Resources["mail-fixture-pending"]).Scan(&pending))
	if failed != "failed" || pending != "pending" {
		panic("Fixtures de notificaciones no preservados")
	}
	check("notificaciones pendientes/fallidas con historial de reintentos")
}

func syntheticPDF(text string) []byte {
	p := fpdf.New("P", "mm", "A4", "")
	p.AddUTF8FontFromBytes("go", "", goregular.TTF)
	p.AddPage()
	p.SetFont("go", "", 14)
	p.MultiCell(175, 9, mark+"\n\n"+text, "", "L", false)
	var b bytes.Buffer
	fail(p.Output(&b))
	return b.Bytes()
}

func openAppealFixtures(admin, analyst *account) {
	name := "abierta-jubilados"
	sub := st.Submissions[name]
	a := st.Accounts[name]
	for _, p := range []struct{ phase, resource, target string }{{"meritos", name + "-merit", "resuelta"}, {"oposicion_teorica", name + "-theory", "atendida"}, {"oposicion_psicotecnica", name + "-psy-published", "recibida"}} {
		key := name + "-" + p.phase
		aid := resource(key+"-appeal", a, "/api/v1/reconsiderations/submissions/"+sub, evaluation.FileAppealInput{Phase: p.phase, ResultID: uuid.MustParse(st.Resources[p.resource]), Motive: mark}, 201)
		if p.target != "recibida" && st.Resources[key+"-attended"] == "" {
			request(analyst, "POST", "/api/v1/admin/reconsiderations/submissions/"+sub+"/"+aid+"/attend", evaluation.AttendInput{PreviousVersion: 1, Observation: mark, Recommendation: "confirmar"}, 201)
			st.Resources[key+"-attended"] = aid
			save()
		}
		if p.target == "resuelta" && st.Resources[key+"-resolved"] == "" {
			request(admin, "POST", "/api/v1/admin/reconsiderations/submissions/"+sub+"/"+aid+"/resolve", evaluation.ResolveInput{PreviousVersion: 2, Decision: "confirmar", Motive: mark}, 201)
			st.Resources[key+"-resolved"] = aid
			save()
		}
	}
	check("fixtures simultáneos recibida/atendida/resuelta aislados en sector jubilados")
}

func verifyAudit(ctx context.Context, admin *account) {
	var legacy string
	fail(pool.QueryRow(ctx, `SELECT id::text FROM submissions WHERE code='SYNTHETIC-DEMO-SIN-CONVOCATORIA' AND competition_id IS NULL AND status='submitted'`).Scan(&legacy))
	st.Submissions["sinConvocatoria"] = legacy
	save()
	request(admin, "GET", "/api/v1/admin/submissions/"+legacy, nil, 409)
	var privateStates int
	fail(pool.QueryRow(ctx, `SELECT count(*) FROM audit_state_events WHERE (COALESCE(previous_state::text,'')||COALESCE(next_state::text,'')) LIKE '%nombres%' OR (COALESCE(previous_state::text,'')||COALESCE(next_state::text,'')) LIKE '%correo%'`).Scan(&privateStates))
	if privateStates != 0 {
		panic("Proyección de auditoría no permitida")
	}
	var correlated int
	fail(pool.QueryRow(ctx, `SELECT count(*) FROM audit_state_events WHERE request_id IS NOT NULL AND actor_id IS NOT NULL AND reason<>''`).Scan(&correlated))
	if correlated == 0 {
		panic("Sin correlación de actor y solicitud")
	}
	rows, e := pool.Query(ctx, `SELECT source_table,count(*) FROM audit_state_events GROUP BY source_table ORDER BY source_table`)
	fail(e)
	counts := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		fail(rows.Scan(&name, &n))
		counts[name] = n
	}
	fail(rows.Err())
	rows.Close()
	for _, name := range []string{"submissions", "documents", "competition_versions", "merit_reviews", "merit_calculations", "merit_publications", "evaluation_records", "reconsiderations", "mail_outbox", "mail_delivery_events", "result_lists", "result_list_publications", "exam_banks", "exam_configs", "exam_events", "exam_reports", "privacy_bindings", "administrative_access_events"} {
		if counts[name] == 0 {
			panic("Falta auditoría: " + name)
		}
	}
	raw, e := json.MarshalIndent(map[string]interface{}{"origin": "synthetic-demo", "warning": mark, "eventsByTable": counts, "correlatedEvents": correlated, "privateStateFields": privateStates}, "", "  ")
	fail(e)
	fail(os.WriteFile(filepath.Join(out, "audit-evidence.json"), raw, 0600))
	check("auditoría uniforme correlacionada y legado sellado sin consentimiento bloqueado")
}

func documentReplacement(a *account, sub string) {
	if st.Resources["document-replacement-tested"] != "" {
		return
	}
	// Reanudar desde metadatos persistidos si una respuesta previa se cortó.
	var existing []map[string]interface{}
	fail(json.Unmarshal(request(a, "GET", "/api/v1/submissions/"+sub+"/documents", nil, 200), &existing))
	for _, d := range existing {
		if d["section"] == float64(5) {
			request(a, "DELETE", "/api/v1/documents/"+d["id"].(string), nil, 204)
		}
	}
	first := upload(a, sub, 5, "titulo-1")
	verifyDownload(a, first)
	request(a, "DELETE", "/api/v1/documents/"+first["id"].(string), nil, 204)
	second := upload(a, sub, 5, "titulo-1")
	if first["id"] == second["id"] {
		panic("El reemplazo no creó otra identidad")
	}
	verifyDownload(a, second)
	request(a, "DELETE", "/api/v1/documents/"+second["id"].(string), nil, 204)
	st.Resources["document-replacement-tested"] = second["id"].(string)
	save()
	check("carga, reemplazo y descarga temporal HTTP con SHA-256 y TTL 300s")
}
func verifyDownload(a *account, d map[string]interface{}) {
	info := obj(request(a, "GET", "/api/v1/documents/"+d["id"].(string)+"/download", nil, 200))
	u, e := url.Parse(info["url"].(string))
	fail(e)
	if u.Host != "localhost:19000" || !strings.HasPrefix(u.Path, "/synthetic-demo/") || u.Query().Get("X-Amz-Expires") != "300" {
		panic("Capacidad de descarga fuera de QA o de plazo")
	}
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, e := client.Get(u.String())
	fail(e)
	defer r.Body.Close()
	if r.StatusCode != 200 {
		panic("Descarga temporal denegada")
	}
	data, e := io.ReadAll(io.LimitReader(r.Body, 3<<20))
	fail(e)
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != d["sha256"] {
		panic("La descarga no coincide con el metadato")
	}
}

func verifyMailpitAttachment() {
	hashes := map[string]bool{}
	files, e := filepath.Glob(filepath.Join(out, "*.pdf"))
	fail(e)
	for _, file := range files {
		data, e := os.ReadFile(file)
		fail(e)
		sum := sha256.Sum256(data)
		hashes[hex.EncodeToString(sum[:])] = true
	}
	allowed := map[string]bool{}
	for _, a := range st.Accounts {
		allowed[a.Email] = true
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for n := 0; n < 35; n++ {
		r, e := client.Get("http://localhost:18025/api/v1/messages?limit=200")
		fail(e)
		var listing struct {
			Messages []struct {
				ID          string
				To          []struct{ Address string }
				Attachments int
			}
		}
		fail(json.NewDecoder(r.Body).Decode(&listing))
		r.Body.Close()
		for _, m := range listing.Messages {
			recipient := false
			for _, to := range m.To {
				recipient = recipient || allowed[to.Address]
			}
			if !recipient || m.Attachments == 0 {
				continue
			}
			r, e := client.Get("http://localhost:18025/api/v1/message/" + m.ID)
			fail(e)
			var message struct {
				Text        string
				Attachments []struct{ PartID, ContentType string }
			}
			fail(json.NewDecoder(r.Body).Decode(&message))
			r.Body.Close()
			if !strings.Contains(message.Text, mark) {
				panic("Correo de demostración sin marca visible")
			}
			for _, a := range message.Attachments {
				if a.ContentType != "application/pdf" {
					continue
				}
				r, e := client.Get("http://localhost:18025/api/v1/message/" + m.ID + "/part/" + a.PartID)
				fail(e)
				data, e := io.ReadAll(io.LimitReader(r.Body, 4<<20))
				r.Body.Close()
				fail(e)
				if r.StatusCode != 200 {
					continue
				}
				sum := sha256.Sum256(data)
				if hashes[hex.EncodeToString(sum[:])] {
					check("Mailpit recibió PDF idéntico al emitido por HTTP y correo marcado como sintético")
					return
				}
			}
		}
		time.Sleep(time.Second)
	}
	panic("No se confirmó un adjunto PDF idéntico en Mailpit")
}
