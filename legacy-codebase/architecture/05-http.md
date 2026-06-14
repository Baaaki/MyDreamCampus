> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 7. HTTP (Reverse Proxy YOK)

### 7.1 Monolith routing

Tek Gin app, modül başına route group:

```go
// monolith/internal/http/server.go
r := gin.New()
api := r.Group("/api")

auth.RegisterRoutes(api.Group("/auth"))
staff.RegisterRoutes(api.Group("/staff"))
student.RegisterRoutes(api.Group("/student"))
// ...
```

Port: **8080**.

### 7.2 Frontend serving — dev vs prod

**Dev (Vite dev server proxy):**
- Vite :3000 üzerinde çalışır, `vite.config.ts`'de proxy ayarı:
  ```ts
  server: {
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  }
  ```
- Browser → :3000 (frontend HMR), `/api/*` çağrıları proxy ile :8080'e gider.
- CORS yok, single-origin gibi davranır. HMR korunur.

**Prod (monolith static serve):**
- Frontend build çıktısı (`frontend/dist/`) Docker image'a `./frontend_dist/` olarak kopyalanır (multi-stage Dockerfile).
- Monolith Gin static serve + SPA fallback yapar:
  ```go
  // monolith/internal/http/server.go
  r.Static("/assets", "./frontend_dist/assets")
  r.StaticFile("/favicon.ico", "./frontend_dist/favicon.ico")
  r.NoRoute(func(c *gin.Context) {
      // /api/* matchlemeyen her sey -> SPA index.html (React Router client-side handle eder)
      c.File("./frontend_dist/index.html")
  })
  ```
- Tek binary, tek port (8080). Cloud deploy = tek container.
- Path notu: `./frontend_dist/` monolith binary'sinin çalışma dizinine relative. Dockerfile içinde `WORKDIR /app` + `COPY --from=frontend-builder /frontend/dist ./frontend_dist`.

### 7.3 Notification servisi

HTTP **expose etmez**. Sadece RabbitMQ consumer.
Health check için Docker'a iç port (ör. :9090/health) verilir, dışarı route edilmez. Reverse proxy olmadığı için zaten harici erişim ihtiyacı yok.

### 7.4 Mobile

- Dev: `EXPO_PUBLIC_API_URL=http://<dev-host>:8080`
- Prod: domain üzerinden, monolith doğrudan internet'e bakar (cloud LB/TLS arkasında).

### 7.5 TLS / HTTPS

Prod deploy hedefi (Render, Fly.io, Railway, AWS App Runner vb.) kendi TLS'ini sağlar — uygulama düz HTTP serve eder, platform TLS termine eder.
Self-host senaryosu gerekirse o gün Caddy eklenir (10 satır config, otomatik Let's Encrypt). Şu an spekülatif olarak taşımıyoruz.

### 7.6 İleride servis ayırma

Bir modül servise ayrılırsa **o gün** Caddy/Traefik eklenir (5 dakikalık iş). Şimdiden esneklik için reverse proxy taşımak premature; modüler monolith'in ruhuna aykırı.

---

