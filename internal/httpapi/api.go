package httpapi

import (
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/example/agnos-test/internal/auth"
	"github.com/example/agnos-test/internal/domain"
	"github.com/example/agnos-test/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type API struct {
	Staff               repository.StaffRepository
	Patients            repository.PatientRepository
	Hospitals           repository.HospitalRepository
	Tokens              repository.TokenRepository
	Audit               repository.AuditRepository
	JWTConfig           auth.Config
	ProvisioningAPIKey  string
	MaxRequestBodyBytes int64
}

func NewRouter(a API) *gin.Engine {
	r := gin.New()
	r.Use(requestBodyLimit(a.MaxRequestBodyBytes), safeLogger(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/staff/create", a.requireProvisioner, a.createStaff)
	r.POST("/staff/login", a.login)
	r.GET("/patient/search", a.requireAuth, a.searchPatients)
	r.POST("/staff/logout", a.requireAuth, a.logout)
	return r
}

func requestBodyLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

func safeLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		log.Printf("http method=%s route=%q status=%d duration_ms=%d", c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(startedAt).Milliseconds())
	}
}

func (a API) requireProvisioner(c *gin.Context) {
	provided := c.GetHeader("X-Provisioning-Key")
	if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(a.ProvisioningAPIKey)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "system provisioning authorization required"})
		return
	}
	c.Next()
}

type staffInput struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Password string `json:"password" binding:"required,min=12,max=128"`
	Hospital string `json:"hospital" binding:"required,max=64"`
}

func (a API) hospital(c *gin.Context, code string) (domain.Hospital, bool) {
	h, err := a.Hospitals.FindByCode(c, code)
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(404, gin.H{"error": "hospital not found"})
		return h, false
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return h, false
	}
	return h, true
}

func (a API) createStaff(c *gin.Context) {
	var in staffInput
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	h, ok := a.hospital(c, in.Hospital)
	if !ok {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	err = a.Staff.Create(c, domain.Staff{ID: uuid.NewString(), Username: in.Username, PasswordHash: string(hash), HospitalID: h.ID})
	if err != nil {
		c.JSON(409, gin.H{"error": "username already exists for this hospital"})
		return
	}
	c.JSON(201, gin.H{"id": "created", "hospital": h.Code})
}

func (a API) login(c *gin.Context) {
	var in staffInput
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	h, ok := a.hospital(c, in.Hospital)
	if !ok {
		return
	}
	s, err := a.Staff.FindByUsernameAndHospital(c, in.Username, h.ID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(s.PasswordHash), []byte(in.Password)) != nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	token, err := auth.Issue(a.JWTConfig, s.ID, h.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": 28800})
}

func (a API) requireAuth(c *gin.Context) {
	v := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if v == "" {
		c.JSON(401, gin.H{"error": "authentication required"})
		c.Abort()
		return
	}
	claims, err := auth.Parse(a.JWTConfig, v)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid token"})
		c.Abort()
		return
	}
	revoked, err := a.Tokens.IsRevoked(c, claims.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		c.Abort()
		return
	}
	if revoked {
		c.JSON(401, gin.H{"error": "invalid token"})
		c.Abort()
		return
	}
	c.Set("hospitalID", claims.HospitalID)
	c.Set("staffID", claims.Subject)
	c.Set("claims", claims)
	c.Next()
}

type searchInput struct {
	NationalID  string `form:"national_id"`
	PassportID  string `form:"passport_id"`
	FirstName   string `form:"first_name"`
	MiddleName  string `form:"middle_name"`
	LastName    string `form:"last_name"`
	DateOfBirth string `form:"date_of_birth"`
	PhoneNumber string `form:"phone_number"`
	Email       string `form:"email"`
}

var (
	nationalIDPattern = regexp.MustCompile(`^\d{13}$`)
	passportIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{6,20}$`)
	phonePattern      = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	emailPattern      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

func validSearchInput(in searchInput) bool {
	if in.NationalID != "" && !nationalIDPattern.MatchString(in.NationalID) {
		return false
	}
	if in.PassportID != "" && !passportIDPattern.MatchString(in.PassportID) {
		return false
	}
	if in.PhoneNumber != "" && !phonePattern.MatchString(in.PhoneNumber) {
		return false
	}
	if in.Email != "" && !emailPattern.MatchString(in.Email) {
		return false
	}
	return len(in.FirstName) <= 100 && len(in.MiddleName) <= 100 && len(in.LastName) <= 100
}

func searchFields(in searchInput) []string {
	fields := make([]string, 0, 8)
	values := []struct {
		name  string
		value string
	}{
		{"national_id", in.NationalID},
		{"passport_id", in.PassportID},
		{"first_name", in.FirstName},
		{"middle_name", in.MiddleName},
		{"last_name", in.LastName},
		{"date_of_birth", in.DateOfBirth},
		{"phone_number", in.PhoneNumber},
		{"email", in.Email},
	}
	for _, field := range values {
		if field.value != "" {
			fields = append(fields, field.name)
		}
	}
	return fields
}

func (a API) searchPatients(c *gin.Context) {
	var in searchInput
	if c.ShouldBindQuery(&in) != nil || !validSearchInput(in) {
		c.JSON(400, gin.H{"error": "invalid query"})
		return
	}
	q := domain.PatientSearch{NationalID: in.NationalID, PassportID: in.PassportID, FirstName: in.FirstName, MiddleName: in.MiddleName, LastName: in.LastName, PhoneNumber: in.PhoneNumber, Email: in.Email}
	if in.DateOfBirth != "" {
		d, e := time.Parse("2006-01-02", in.DateOfBirth)
		if e != nil {
			c.JSON(400, gin.H{"error": "date_of_birth must be YYYY-MM-DD"})
			return
		}
		q.DateOfBirth = &d
	}
	items, err := a.Patients.Search(c, c.MustGet("hospitalID").(string), q)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	if err := a.Audit.LogPatientSearch(c, domain.PatientSearchAudit{StaffID: c.MustGet("staffID").(string), HospitalID: c.MustGet("hospitalID").(string), QueryFields: searchFields(in), ResultCount: len(items)}); err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, gin.H{"data": items, "count": len(items)})
}

func (a API) logout(c *gin.Context) {
	claims := c.MustGet("claims").(auth.Claims)
	if claims.ExpiresAt == nil {
		c.JSON(401, gin.H{"error": "invalid token"})
		return
	}
	if err := a.Tokens.Revoke(c, domain.TokenRevocation{TokenID: claims.ID, StaffID: claims.Subject, HospitalID: claims.HospitalID, ExpiresAt: claims.ExpiresAt.Time}); err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
