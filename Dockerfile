FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY . .

ARG RELEASE=false
ARG VERSION=1.0.0
ARG GOOS=linux
ARG GOARCH=amd64
ARG CGO_ENABLED=0

RUN make tidy && \
    if [ "$RELEASE" = "true" ]; then \
        make build-release version=${VERSION}; \
    else \
        make build; \
    fi

FROM scratch

WORKDIR /

ENV POSTGRES_USER=youruser
ENV POSTGRES_PASSWORD=yourpass
ENV POSTGRES_URL=localhost:5432
ENV POSTGRES_DB=yourdb

COPY --from=builder /app/reservation /reservation

ENTRYPOINT [ "/reservation" ]
