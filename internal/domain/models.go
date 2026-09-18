package domain

import "time"

type Hospital struct {
	ID   string
	Code string
	Name string
}

type Staff struct {
	ID           string
	Username     string
	PasswordHash string
	HospitalID   string
}

type Patient struct {
	ID           string
	HospitalID   string
	NationalID   string
	PassportID   string
	FirstNameTH  string
	MiddleNameTH string
	LastNameTH   string
	FirstNameEN  string
	MiddleNameEN string
	LastNameEN   string
	DateOfBirth  *time.Time
	PhoneNumber  string
	Email        string
	Gender       string
}

type PatientSearch struct {
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	PhoneNumber string
	Email       string
	DateOfBirth *time.Time
}

type TokenRevocation struct {
	TokenID    string
	StaffID    string
	HospitalID string
	ExpiresAt  time.Time
}

type PatientSearchAudit struct {
	StaffID     string
	HospitalID  string
	QueryFields []string
	ResultCount int
}
