FROM node:23 AS frontend-build
WORKDIR /app/frontend
COPY frontend .

RUN npm install
RUN npm run build

# -----------------------------------------------------

FROM golang:1.26.10 AS backend-build

WORKDIR /app/backend
COPY backend .
# Static binary, so the runtime image doesn't need a libc.
RUN CGO_ENABLED=0 make build

# -----------------------------------------------------

# distroless instead of the full golang image: no shell, package manager or
# compiler in production, and far fewer packages for scanners to flag.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app/backend

COPY --from=backend-build /app/backend/bin/server /app/backend/bin/server
# The mail templates are read at runtime (mail_template_dir in config.json).
COPY --from=backend-build /app/backend/internal/mail/templates /app/backend/internal/mail/templates
COPY --from=frontend-build /app/frontend/build/ /app/frontend/build

# Declared explicitly for scanners; the :nonroot tag already runs as UID 65532.
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT [ "/app/backend/bin/server" ]
