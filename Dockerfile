FROM node:25@sha256:78839ac448c23517f8eab2e8f7943d9b4f73979eb7f8bed2c73dbf72ff869e7b AS frontend-build
WORKDIR /app/frontend
COPY frontend .

RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.27.0@sha256:4013ae0f9e7994f8535c58c811f8f863fbed38b72e0d51e6592156f758d66146 AS backend-build

WORKDIR /app/backend
COPY backend .
# Static binary, so the runtime image doesn't need a libc.
RUN CGO_ENABLED=0 make build

# -----------------------------------------------------

# distroless instead of the full golang image: no shell, package manager or
# compiler in production, and far fewer packages for scanners to flag.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime

WORKDIR /app/backend

COPY --from=backend-build /app/backend/bin/server /app/backend/bin/server
# The mail templates are read at runtime (mail_template_dir in config.json).
COPY --from=backend-build /app/backend/internal/mail/templates /app/backend/internal/mail/templates
COPY --from=frontend-build /app/frontend/build/ /app/frontend/build

# Declared explicitly for scanners; the :nonroot tag already runs as UID 65532.
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT [ "/app/backend/bin/server" ]
