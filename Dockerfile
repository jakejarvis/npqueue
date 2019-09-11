FROM golang:1.13 AS builder

WORKDIR /src
COPY . /src

RUN CGO_ENABLED=0 GOOS=linux go build -o npqueue .

# ---

FROM heroku/heroku:18

WORKDIR /app
COPY --from=builder /src /app

ENV GIN_MODE=release

CMD ["./npqueue"]
