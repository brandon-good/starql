FROM golang:latest AS build

WORKDIR /app

# Copy the Go module files
COPY go.mod .
COPY go.sum .

# Download the Go module dependencies
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o myapp .

COPY shuffled_mini_dev_postgresql.jsonl .
# Copy the application executable from the build image


CMD ["./myapp"]