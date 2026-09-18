package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/agnos-test/internal/auth"
	"github.com/example/agnos-test/internal/domain"
	"github.com/example/agnos-test/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type fakeStore struct {
	hospital         domain.Hospital
	staff            domain.Staff
	patients         []domain.Patient
	created          bool
	searchedHospital string
	revoked          map[string]bool
	auditEvents      []domain.PatientSearchAudit
}

func (f *fakeStore) FindByCode(_ context.Context, code string) (domain.Hospital, error) {
	if code != f.hospital.Code {
		return domain.Hospital{}, repository.ErrNotFound
	}
	return f.hospital, nil
}
func (f *fakeStore) Create(_ context.Context, s domain.Staff) error {
	if f.created {
		return errors.New("duplicate")
	}
	f.created = true
	f.staff = s
	return nil
}
func (f *fakeStore) FindByUsernameAndHospital(_ context.Context, u, h string) (domain.Staff, error) {
	if u != f.staff.Username || h != f.staff.HospitalID {
		return domain.Staff{}, repository.ErrNotFound
	}
	return f.staff, nil
}
func (f *fakeStore) Search(_ context.Context, h string, _ domain.PatientSearch) ([]domain.Patient, error) {
	f.searchedHospital = h
	return f.patients, nil
}

func (f *fakeStore) IsRevoked(_ context.Context, tokenID string) (bool, error) {
	return f.revoked[tokenID], nil
}

func (f *fakeStore) Revoke(_ context.Context, token domain.TokenRevocation) error {
	f.revoked[token.TokenID] = true
	return nil
}

func (f *fakeStore) LogPatientSearch(_ context.Context, audit domain.PatientSearchAudit) error {
	f.auditEvents = append(f.auditEvents, audit)
	return nil
}

func router() (*fakeStore, http.Handler) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("a-long-unique-password"), bcrypt.MinCost)
	f := &fakeStore{hospital: domain.Hospital{ID: "h-a", Code: "hospital-a"}, staff: domain.Staff{ID: "s-1", Username: "staff", PasswordHash: string(hash), HospitalID: "h-a"}, patients: []domain.Patient{{ID: "p-1", HospitalID: "h-a"}}, revoked: map[string]bool{}}
	config := auth.Config{Secret: "12345678901234567890123456789012", Issuer: "test-issuer", Audience: "test-audience"}
	return f, NewRouter(API{Staff: f, Patients: f, Hospitals: f, Tokens: f, Audit: f, JWTConfig: config, ProvisioningAPIKey: "12345678901234567890123456789012", MaxRequestBodyBytes: 1024})
}
func request(t *testing.T, h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func provision(t *testing.T, h http.Handler, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Provisioning-Key", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestStaffLoginPositiveAndNegative(t *testing.T) {
	_, h := router()
	ok := request(t, h, "POST", "/staff/login", `{"username":"staff","password":"a-long-unique-password","hospital":"hospital-a"}`, "")
	if ok.Code != http.StatusOK {
		t.Fatalf("got %d: %s", ok.Code, ok.Body.String())
	}
	bad := request(t, h, "POST", "/staff/login", `{"username":"staff","password":"wrong-password","hospital":"hospital-a"}`, "")
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", bad.Code)
	}
}
func TestCreateAndSearchAuthorization(t *testing.T) {
	f, h := router()
	denied := provision(t, h, `{"username":"new","password":"a-long-unique-password","hospital":"hospital-a"}`, "wrong")
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("provisioning authorization: %d", denied.Code)
	}
	created := provision(t, h, `{"username":"new","password":"a-long-unique-password","hospital":"hospital-a"}`, "12345678901234567890123456789012")
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	unauth := request(t, h, "GET", "/patient/search?last_name=Somchai", "", "")
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth: %d", unauth.Code)
	}
	token, _ := auth.Issue(auth.Config{Secret: "12345678901234567890123456789012", Issuer: "test-issuer", Audience: "test-audience"}, "s-1", "h-a")
	ok := request(t, h, "GET", "/patient/search?date_of_birth=bad", "", token)
	if ok.Code != http.StatusBadRequest {
		t.Fatalf("bad date: %d", ok.Code)
	}
	ok = request(t, h, "GET", "/patient/search?last_name=Somchai", "", token)
	if ok.Code != http.StatusOK || f.searchedHospital != "h-a" || len(f.auditEvents) != 1 {
		t.Fatalf("search scope/status: %s %d", f.searchedHospital, ok.Code)
	}
}

func TestSearchRejectsInvalidIdentifierAndLogoutRevokesToken(t *testing.T) {
	_, h := router()
	config := auth.Config{Secret: "12345678901234567890123456789012", Issuer: "test-issuer", Audience: "test-audience"}
	token, _ := auth.Issue(config, "s-1", "h-a")
	invalid := request(t, h, "GET", "/patient/search?national_id=invalid", "", token)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid identifier: %d", invalid.Code)
	}
	logout := request(t, h, "POST", "/staff/logout", "", token)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", logout.Code)
	}
	reused := request(t, h, "GET", "/patient/search", "", token)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token: %d", reused.Code)
	}
}

func TestRequestBodyLimitAndJWTIssuerValidation(t *testing.T) {
	_, h := router()
	overstated := provision(t, h, `{"username":"`+strings.Repeat("a", 1100)+`","password":"a-long-unique-password","hospital":"hospital-a"}`, "12345678901234567890123456789012")
	if overstated.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("body limit: %d", overstated.Code)
	}
	wrongIssuer, _ := auth.Issue(auth.Config{Secret: "12345678901234567890123456789012", Issuer: "other-issuer", Audience: "test-audience"}, "s-1", "h-a")
	rejected := request(t, h, "GET", "/patient/search", "", wrongIssuer)
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("issuer validation: %d", rejected.Code)
	}
}
