package repository

import (
	"context"
	"errors"

	"github.com/example/agnos-test/internal/domain"
)

var ErrNotFound = errors.New("not found")

type StaffRepository interface {
	Create(context.Context, domain.Staff) error
	FindByUsernameAndHospital(context.Context, string, string) (domain.Staff, error)
}

type PatientRepository interface {
	Search(context.Context, string, domain.PatientSearch) ([]domain.Patient, error)
}

type HospitalRepository interface {
	FindByCode(context.Context, string) (domain.Hospital, error)
}

type TokenRepository interface {
	IsRevoked(context.Context, string) (bool, error)
	Revoke(context.Context, domain.TokenRevocation) error
}

type AuditRepository interface {
	LogPatientSearch(context.Context, domain.PatientSearchAudit) error
}
