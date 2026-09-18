package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/example/agnos-test/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func (p Postgres) FindByCode(ctx context.Context, code string) (domain.Hospital, error) {
	var h domain.Hospital
	err := p.Pool.QueryRow(ctx, `SELECT id::text, code, name FROM hospitals WHERE code=$1`, code).Scan(&h.ID, &h.Code, &h.Name)
	if err == pgx.ErrNoRows {
		return h, ErrNotFound
	}
	return h, err
}

func (p Postgres) Create(ctx context.Context, s domain.Staff) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO staff(id, username, password_hash, hospital_id) VALUES($1,$2,$3,$4)`, s.ID, s.Username, s.PasswordHash, s.HospitalID)
	return err
}

func (p Postgres) FindByUsernameAndHospital(ctx context.Context, username, hospitalID string) (domain.Staff, error) {
	var s domain.Staff
	err := p.Pool.QueryRow(ctx, `SELECT id::text, username, password_hash, hospital_id::text FROM staff WHERE username=$1 AND hospital_id=$2`, username, hospitalID).Scan(&s.ID, &s.Username, &s.PasswordHash, &s.HospitalID)
	if err == pgx.ErrNoRows {
		return s, ErrNotFound
	}
	return s, err
}

func (p Postgres) Search(ctx context.Context, hospitalID string, q domain.PatientSearch) ([]domain.Patient, error) {
	where, args := []string{"hospital_id = $1"}, []any{hospitalID}
	add := func(column, value string) {
		if value != "" {
			args = append(args, value)
			where = append(where, fmt.Sprintf("%s = $%d", column, len(args)))
		}
	}
	add("national_id", q.NationalID)
	add("passport_id", q.PassportID)
	add("first_name_th", q.FirstName)
	add("middle_name_th", q.MiddleName)
	add("last_name_th", q.LastName)
	add("phone_number", q.PhoneNumber)
	add("email", q.Email)
	if q.DateOfBirth != nil {
		args = append(args, *q.DateOfBirth)
		where = append(where, fmt.Sprintf("date_of_birth = $%d", len(args)))
	}
	rows, err := p.Pool.Query(ctx, `SELECT id::text,hospital_id::text,COALESCE(national_id,''),COALESCE(passport_id,''),COALESCE(first_name_th,''),COALESCE(middle_name_th,''),COALESCE(last_name_th,''),COALESCE(first_name_en,''),COALESCE(middle_name_en,''),COALESCE(last_name_en,''),date_of_birth,COALESCE(phone_number,''),COALESCE(email,''),gender FROM patients WHERE `+strings.Join(where, " AND ")+` ORDER BY last_name_th, first_name_th LIMIT 100`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patients := []domain.Patient{}
	for rows.Next() {
		var x domain.Patient
		if err := rows.Scan(&x.ID, &x.HospitalID, &x.NationalID, &x.PassportID, &x.FirstNameTH, &x.MiddleNameTH, &x.LastNameTH, &x.FirstNameEN, &x.MiddleNameEN, &x.LastNameEN, &x.DateOfBirth, &x.PhoneNumber, &x.Email, &x.Gender); err != nil {
			return nil, err
		}
		patients = append(patients, x)
	}
	return patients, rows.Err()
}

func (p Postgres) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	var revoked bool
	err := p.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE token_id = $1)`, tokenID).Scan(&revoked)
	return revoked, err
}

func (p Postgres) Revoke(ctx context.Context, token domain.TokenRevocation) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO revoked_tokens(token_id, staff_id, hospital_id, expires_at) VALUES($1,$2,$3,$4) ON CONFLICT (token_id) DO NOTHING`, token.TokenID, token.StaffID, token.HospitalID, token.ExpiresAt)
	return err
}

func (p Postgres) LogPatientSearch(ctx context.Context, audit domain.PatientSearchAudit) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO audit_events(staff_id, hospital_id, event_type, query_fields, result_count) VALUES($1,$2,'patient.search',$3,$4)`, audit.StaffID, audit.HospitalID, audit.QueryFields, audit.ResultCount)
	return err
}
