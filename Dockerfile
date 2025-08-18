# syntax=docker/dockerfile:1

FROM alpine:latest AS builder

WORKDIR /app

RUN apk add --update go npm make git
RUN npm install -g typescript@5.5.4

COPY go.mod go.sum ./
RUN go mod download

COPY .git/ ./.git
COPY backend/ ./backend
COPY utils/ ./utils
COPY web/ ./web
COPY shared/ ./shared
COPY tsconfig.json .

COPY Makefile .

RUN git submodule update --init --recursive
RUN make backend

# Server image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/yeetfile-server /app
RUN chmod +x /app/yeetfile-server
EXPOSE 8090

CMD ["/app/yeetfile-server"]
