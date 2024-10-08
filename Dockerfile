FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY . .

RUN go mod tidy && \
    GOOS=linux GOARCH=amd64 go build -o /reservation


FROM debian:bookworm

WORKDIR /

ENV POSTGRES_USER=youruser
ENV POSTGRES_PASSWORD=yourpass
ENV POSTGRES_URL=localhost:5432
ENV POSTGRES_DB=yourdb

COPY --from=builder /reservation /reservation

ENTRYPOINT [ "/reservation" ]
