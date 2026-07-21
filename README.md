## Go email issuer

### Local Development

With `make run`

### Docker

With `docker-compose up`

**Prerequisites**: Add the key to the email issuer irma server in
`backend/issue/keys`. Use `config.sample.json` to set up your config for the go
app.

### Resetting the rate limit for a user

A user who retries too often locks out their own email address. An operator can
clear the per-email rate limit through `POST /api/admin/reset-rate-limit`.

The endpoint is disabled until you set `app.admin_token` in the config (at least
16 characters). Pass that token as a bearer token:

```sh
curl -X POST http://localhost:8080/api/admin/reset-rate-limit \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

A successful reset returns `{"message":"rate_limit_reset"}`. The endpoint returns
403 when no admin token is configured, 401 for a wrong token, and 400 for a
missing or invalid email. Only the per-email limit is cleared; the per-IP limit
is left in place.
