# ============================================================
# Stage 1 - build binary Go (tanpa CGO, ukuran sekecil mungkin)
# ============================================================
FROM golang:1.26-alpine AS build

WORKDIR /src

# Unduh dependensi dulu agar lapisan Docker cache-nya stabil.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# ============================================================
# Stage 2 - image runtime ringan (alpine) dengan user non-root
# ============================================================
FROM alpine:3.24

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/app ./app

# Folder logs ditulis worker (mail.log) — dibuat lebih dulu lalu dimiliki user app.
RUN mkdir logs && chown app:app logs

USER app

EXPOSE 3000

ENTRYPOINT ["./app"]
CMD ["-role=api"]
