CREATE TABLE hospitals (
  id UUID PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE staff (
  id UUID PRIMARY KEY,
  username VARCHAR(100) NOT NULL,
  password_hash TEXT NOT NULL,
  hospital_id UUID NOT NULL REFERENCES hospitals(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (hospital_id, username)
);

CREATE TABLE patients (
  id UUID PRIMARY KEY,
  hospital_id UUID NOT NULL REFERENCES hospitals(id),
  national_id VARCHAR(20),
  passport_id VARCHAR(30),
  first_name_th VARCHAR(255),
  middle_name_th VARCHAR(255),
  last_name_th VARCHAR(255),
  first_name_en VARCHAR(255),
  middle_name_en VARCHAR(255),
  last_name_en VARCHAR(255),
  date_of_birth DATE,
  phone_number VARCHAR(30),
  email VARCHAR(320),
  gender CHAR(1) NOT NULL CHECK (gender IN ('M', 'F')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (national_id IS NOT NULL OR passport_id IS NOT NULL)
);

CREATE UNIQUE INDEX patients_national_id_per_hospital
  ON patients(hospital_id, national_id) WHERE national_id IS NOT NULL;
CREATE UNIQUE INDEX patients_passport_id_per_hospital
  ON patients(hospital_id, passport_id) WHERE passport_id IS NOT NULL;
CREATE INDEX patients_search_scope ON patients(hospital_id, last_name_th, date_of_birth);

INSERT INTO hospitals (id, code, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'hospital-a', 'Hospital A');
