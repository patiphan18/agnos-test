# Development plan and project structure

## Design

The service is a hospital middleware boundary. Hospital-specific HIS connectors should normalize source records into `patients`; the query API reads only that canonical representation. The current migration creates `hospital-a` for the supplied Hospital A integration.

Authorization is enforced server-side from a signed JWT claim. A client never supplies `hospital_id`, preventing cross-hospital lookup. Passwords use bcrypt; tokens expire after 8 hours; Nginx rate-limits inbound traffic.

## Layout

```
cmd/server/              executable wiring
internal/auth/           JWT creation and validation
internal/domain/         business data types
internal/httpapi/        Gin handlers and auth middleware
internal/repository/     PostgreSQL repository interfaces/implementation
migrations/              PostgreSQL schema
deploy/                  Nginx config
docs/                    API and data design
```

## Operational notes

- Keep `JWT_SECRET` in a secret manager, rotate it, and use TLS at the ingress.
- Do not log tokens, passwords, national IDs, or patient payloads.
- Run database migrations through a controlled deployment pipeline in production; the Compose initialization mount is for local development.
- Add audit records and consent/retention policy before processing real patient data.
