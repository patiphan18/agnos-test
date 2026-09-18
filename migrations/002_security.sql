CREATE TABLE revoked_tokens (
  token_id UUID PRIMARY KEY,
  staff_id UUID NOT NULL REFERENCES staff(id),
  hospital_id UUID NOT NULL REFERENCES hospitals(id),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX revoked_tokens_expiry ON revoked_tokens(expires_at);

CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  staff_id UUID NOT NULL REFERENCES staff(id),
  hospital_id UUID NOT NULL REFERENCES hospitals(id),
  event_type VARCHAR(64) NOT NULL,
  query_fields TEXT[] NOT NULL DEFAULT '{}',
  result_count INTEGER NOT NULL CHECK (result_count >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_events_staff_created_at ON audit_events(staff_id, created_at DESC);
