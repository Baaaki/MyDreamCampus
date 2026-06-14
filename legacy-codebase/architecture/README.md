# Modüler Monolith Migration Plan

> Mevcut 9 mikroservislik mimariyi modüler monolith + 1 ayrı servis (notification) yapısına taşıma planı. Bu plan, kararların kaydıdır; uygulama detayları her adımda ayrıca konuşulur.

> **AI için:** Yeni oturumda **önce** [00-ai-rules.md](00-ai-rules.md) oku. Diğer dosyaları ihtiyaç anında aç (full plan ~1400 satır — toplu okuma context maliyeti yüksek).

---

## Dosya Haritası

| Dosya | İçerik | Orjinal Bölümler | Satır |
|---|---|---|---|
| [00-ai-rules.md](00-ai-rules.md) | AI çalışma kuralları, glossary, naming | 0 | ~55 |
| [01-overview.md](01-overview.md) | Hedef, mimari kararlar, klasör yapısı | 1, 2, 3 | ~120 |
| [02-data.md](02-data.md) | DB schema-per-module, cross-module stratejileri, modül haritası | 4, 8, 9 | ~170 |
| [03-events.md](03-events.md) | RabbitMQ + Outbox + Event kataloğu | 5 | ~625 |
| [04-notification.md](04-notification.md) | Notification servisi (kurulum, DB, template, test) | 6 | ~230 |
| [05-http.md](05-http.md) | HTTP routing, frontend serving (dev + prod) | 7 | ~70 |
| [06-testing.md](06-testing.md) | Test stratejisi (unit, integration, E2E) | 10 | ~30 |
| [07-migration.md](07-migration.md) | Faz planı, başarı kriterleri, geri dönüş | 11, 14, 15 | ~70 |
| [08-rules.md](08-rules.md) | Yasaklar, kırmızı çizgiler, açık sorular | 12, 13 | ~55 |

---

## Bölüm → Dosya Aramaları

Plan içindeki "Bölüm X.Y" referansları için hızlı arama:

| Bölüm | Konu | Dosya |
|---|---|---|
| 0.1 | AI çalışma kuralları | [00-ai-rules.md](00-ai-rules.md) |
| 0.2 | Glossary / terimler | [00-ai-rules.md](00-ai-rules.md) |
| 1 | Hedef, motivasyon, yük profili | [01-overview.md](01-overview.md) |
| 2 | Mimari kararlar (özet) | [01-overview.md](01-overview.md) |
| 3 | Klasör yapısı | [01-overview.md](01-overview.md) |
| 4.x | Database stratejisi (schema-per-module) | [02-data.md](02-data.md) |
| 5.x | Event bus, outbox, RabbitMQ | [03-events.md](03-events.md) |
| 5.9 | **Event kataloğu (eksiksiz)** | [03-events.md](03-events.md) |
| 6.x | Notification servisi | [04-notification.md](04-notification.md) |
| 7.x | HTTP / frontend serving | [05-http.md](05-http.md) |
| 8 | Cross-module veri stratejileri | [02-data.md](02-data.md) |
| 9 | **Modül-modül strateji haritası (eksiksiz)** | [02-data.md](02-data.md) |
| 10 | Test stratejisi | [06-testing.md](06-testing.md) |
| 11 | Migration faz planı | [07-migration.md](07-migration.md) |
| 12 | Yasaklar ve kırmızı çizgiler | [08-rules.md](08-rules.md) |
| 13 | Açık sorular | [08-rules.md](08-rules.md) |
| 14 | Başarı kriterleri | [07-migration.md](07-migration.md) |
| 15 | Geriye dönüş stratejisi | [07-migration.md](07-migration.md) |

---

## Kritik Kurallar (Hızlı Hatırlatma)

- **Cross-schema FK/JOIN yasak** — [02-data.md](02-data.md) Bölüm 4.2.
- **Outbox'sız `publisher.Publish` yasak** — [03-events.md](03-events.md) Bölüm 5.5.
- **Yeni event/route/exchange için önce sor** — [00-ai-rules.md](00-ai-rules.md) Bölüm 0.1.
- **Modül adlarında tire (`-`) yasak**, alt çizgi (`_`) zorunlu — [00-ai-rules.md](00-ai-rules.md) Bölüm 0.2.
- **Konuşma dili Türkçe, kod İngilizce** — proje [CLAUDE.md](../CLAUDE.md) Bölüm 3.
