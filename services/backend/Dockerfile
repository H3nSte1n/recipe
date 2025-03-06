FROM golang:1.23-alpine


WORKDIR /app

# Install required tools including migrate and air
RUN apk add --no-cache postgresql-client make git gcc musl-dev \
    && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    && go install github.com/air-verse/air@latest  # Updated package path

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Expose port
EXPOSE 8080

# Use air for development (hot reload)
CMD ["air"]