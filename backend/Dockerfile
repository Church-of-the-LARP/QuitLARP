FROM golang:1.26-alpine
WORKDIR /app
# The local git server (handlers/git.go) shells out to git for upload-pack,
# receive-pack and repo creation.
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8888
CMD ["go", "run", "."]
