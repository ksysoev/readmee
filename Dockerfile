FROM golang:1.26.1-alpine AS builder

ARG VERSION=${VERSION}

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 go build -o readmee -ldflags "-X main.version=$VERSION -X main.name=readmee" ./cmd/readmee/main.go

FROM scratch

COPY --from=builder /app/readmee .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/readmee"]
