FROM golang:1.23-alpine

WORKDIR /app

RUN apk add --no-cache postgresql-client make git gcc musl-dev \
    && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1 \
    && go install github.com/air-verse/air@v1.52.3

COPY go.mod go.sum ./

RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air"]