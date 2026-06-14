> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 11. Migration Planı (Adım Adım)

### Faz 0: Hazırlık
1. Bu plan onaylandıktan sonra `backend/monolith/` iskeletini oluştur (`cmd`, `internal/platform`, `internal/eventbus`, `internal/http`).
2. Tek `Makefile` (sqlc, migrate, build, test).
3. `infrastructure/docker-compose.yml` güncelle: tek `mydreamcampus_postgres`, tek `notification_postgres`, RabbitMQ, Redis. **Traefik kaldırılır.** Frontend dev'de Vite proxy ile çalışır, prod'da monolith Gin static serve eder.
4. Outbox tablosu + relay goroutine iskeleti.
5. Shared event paketi (`backend/shared/events/`).

### Faz 1: İlk modül — `auth`
- En küçük + en bağımsız modülden başla.
- `auth schema` migration'ları, sqlc setup, service/repo/handler taşı.
- JWT middleware'i `internal/platform/jwt/` altına shared olarak çıkar.
- Test yeşil → commit.

### Faz 2: Sırasıyla diğer modüller
Bağımlılık yönüne göre sıra (ihtiyaç duyulan önce):
1. `auth` (Faz 1'de)
2. `staff`
3. `student`
4. `course_catalog`
5. `enrollment` (student + course_catalog'a bağımlı)
6. `attendance` (enrollment'a bağımlı + Strateji 3 read model `attendance.students_view` baştan kurulur)
7. `grades` (enrollment'a bağımlı)
8. `meal`
9. `payment` (meal'in event'ini consume eder)

Her modülde: schema migration → sqlc → repo → service → handler → routes → test → commit.

### Faz 3: Notification servisi
- `services/notification/` ayağa kalk.
- Event consumer (RabbitMQ).
- SMTP/FCM adapter (önce SMTP, MailHog ile dev).
- delivery_log tablosu.
- Monolith outbox relay → RabbitMQ → notification consumer end-to-end test.

### Faz 4: Eski mikroservis dosyalarının temizliği
- `backend/services/auth-service/`, `backend/services/staff-service/` vb. **sil** — modül kodu monolith'e taşındıktan ve testleri yeşil olduktan sonra.
- `backend/infrastructure/`'daki eski docker-compose entry'lerini ve **Traefik** config'lerini temizle.
- `backend/shared/{database,logger,errors,middleware,rabbitmq,redis,utils}` paketleri `backend/monolith/internal/platform/` altına taşındı; kalan `backend/shared/events/` korunur (monolith + notification ortak kontrat).
- README/CLAUDE.md güncelle: Traefik yok, port 8080 doğrudan, modül listesi ve naming.

### Faz 5: Frontend / Mobile düzeltme
- Frontend `vite.config.ts` proxy ekle: `/api` → `http://localhost:8080`. Frontend kodunda çağrı path'leri (`/api/<modul>/*`) **değişmez**.
- Mobile `.env` güncelle: `EXPO_PUBLIC_API_URL=http://<host>:8080` (eski Traefik :80 yerine).
- Prod build pipeline: `frontend/dist/` → monolith Docker image'a kopyala; Gin `r.Static` ile servis eder.

---

## 14. Başarı Kriterleri

Migration tamamlandı sayılır eğer:

- [ ] `backend/monolith/` tek binary build oluyor: `make build` yeşil
- [ ] Tüm modül testleri yeşil: `make test`
- [ ] `services/notification/` ayrı çalışıyor, RabbitMQ üzerinden event tüketiyor
- [ ] E2E test: kullanıcı kayıt → user.registered event → notification welcome email gönderdi (MailHog'da görünüyor)
- [ ] Frontend tüm sayfalar çalışıyor (manuel golden path test)
- [ ] Docker compose tek komutla ayağa kalkıyor: monolith + notification + Postgres + RabbitMQ + Redis (Traefik **yok**; frontend dev'de Vite proxy ile, prod'da monolith static serve)
- [ ] `go build ./...` ve `bun tsc --noEmit` ikisi de hatasız
- [ ] CI süresi belirgin şekilde kısaldı

---

## 15. Geriye Dönüş Stratejisi

Bu büyük bir refactor. Eğer migration ortasında çıkmaz sokak çıkarsa:

- Her modül **atomic commit** ile taşınıyor → istediğin commit'e geri dönülebilir.
- `services/auth-service/` vb. eski dosyalar Faz 4'e kadar **silinmiyor** (git history'de duruyor zaten).
- Yeni modüler kod ile eski servis kodu yan yana yaşayabilir bir süre — Traefik ile traffic shift mümkün.
