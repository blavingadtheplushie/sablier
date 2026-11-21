# the server with the website built in: docker compose -f compose.prod.yaml up -d --build

FROM node:24-alpine AS site
WORKDIR /src/website
COPY website/package.json website/package-lock.json ./
RUN npm ci
COPY website/ ./
RUN npm run build

FROM golang:1.27-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /sablier-server ./cmd/sablier-server

FROM alpine:3.20
RUN adduser -D -H sablier
COPY --from=server /sablier-server /usr/local/bin/sablier-server
COPY --from=site /src/website/dist /srv/site
ENV SABLIER_ADDR=:8080 SABLIER_SITE_DIR=/srv/site
USER sablier
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["sablier-server"]
