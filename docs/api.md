# API specification

All JSON errors have an `error` field. The service deliberately returns the same login failure for an unknown user and wrong password.

## `POST /staff/create`

Creates a staff account in a hospital. This is a system-provisioning endpoint, not a public registration endpoint: it requires `X-Provisioning-Key`, matching `PROVISIONING_API_KEY`. `username` is unique within the hospital; `password` must be 12-128 characters.

```json
{"username":"nurse.a","password":"a-long-unique-password","hospital":"hospital-a"}
```

Returns `201`, `400`, `401`, `404`, or `409`.

## `POST /staff/login`

Uses the same request body. Returns an 8-hour HS256 bearer token containing issuer, audience, token ID, staff ID, and hospital ID.

```json
{"access_token":"<jwt>","token_type":"Bearer","expires_in":28800}
```

Returns `200`, `400`, `401`, or `404`.

## `GET /patient/search`

Requires `Authorization: Bearer <jwt>`. All parameters are optional and combined with AND: `national_id`, `passport_id`, `first_name`, `middle_name`, `last_name`, `date_of_birth` (`YYYY-MM-DD`), `phone_number`, `email`.

The hospital is extracted exclusively from the JWT. The result contains only matching patients assigned to that hospital and is capped at 100 rows. `national_id` must be 13 digits; `passport_id` must be 6-20 alphanumeric characters; `phone_number` must be 7-15 digits with an optional leading `+`; email must use a valid basic email format; and names are limited to 100 characters. Each search writes an audit event containing the staff ID, hospital ID, the names of supplied filters, and result count - never the filter values.

Returns `200`, `400`, or `401`.

## `POST /staff/logout`

Requires `Authorization: Bearer <jwt>`. Stores the current token ID in the revocation list until its original expiry, so the same access token cannot be reused. Returns `204`, `401`, or `500`.

## `GET /healthz`

Unauthenticated liveness probe; returns `200`.
