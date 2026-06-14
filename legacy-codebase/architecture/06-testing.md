> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 10. Test Stratejisi

### 10.1 Mevcut testlerin akıbeti

| Test türü | Akıbet |
|---|---|
| Unit (service logic) | Modül klasörüne taşınır, sadece import path değişir. ~%90 efor düşük. |
| Repository / DB integration | Tek shared test postgres + schema-per-module fixture. Her test transaction içinde, sonunda rollback. |
| Handler (HTTP) | Modüle taşınır, `httptest` aynen çalışır. Tek Gin router fixture. |
| Cross-module event | **İyileşir** — RabbitMQ'suz, in-process outbox + dispatcher test edilir. Çok hızlı, az flake. |
| Notification testi | `services/notification/` altında kalır, testcontainers ile gerçek RabbitMQ + MailHog. |
| E2E | `monolith/test/e2e/` — monolith + notification + Postgres + RabbitMQ ayakta. Az ama değerli. |

### 10.2 Test setup birleşmesi

- 9 ayrı test postgres bootstrap → 1 shared fixture
- CI süresi tahmini: 5-10 dk → 1-2 dk

### 10.3 Migration sırasında

**Modül başına döngü, CI yeşilken bir sonrakine geç:**
1. Modül kodunu taşı + import düzelt
2. Modül testlerini taşı + `make test ./internal/modules/<modul>/...` yeşil
3. Atomic commit (`refactor(<modul>): migrate to monolith module`)
4. Sonraki modüle geç

---

