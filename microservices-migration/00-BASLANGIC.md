# Mikroservis Migrasyonu — Başlangıç Dosyası

> **AI için:** Oturuma bu dosyayla başla. Sadece bu dosyayı ve sıradaki tek faz
> dosyasını oku. Diğer faz dosyalarını **açma** — bağımsız yazıldılar.

---

## Çalışma Protokolü

1. Aşağıdaki **Durum Tablosu**'ndan sıradaki fazı bul (ilk `[ ]` olan satır).
2. O fazın dosyasını oku, **sadece onu** uygula.
3. Faz bitince:
   - Fazın "Bitiş Kriteri" bölümündeki doğrulama komutlarını çalıştır.
   - Atomic commit at (commit mesajı faz dosyasının sonunda yazılı).
   - Dosyayı yeniden adlandır: `faz-N-xxx.md` → `faz-N-xxx-TAMAMLANDI.md`
   - Bu dosyadaki durum tablosunda o satırı `[x]` yap.
4. Sıradaki faza geç veya kullanıcıya "Faz N bitti" diye rapor et.

**Ortak referans:** `01-REFERANS-MIMARI.md` — port tablosu, DB adları, servis
isimleri, iletişim kontratları. Faz dosyaları buraya atıfta bulunur. Sadece
gerektiğinde aç, baştan sona okuma.

---

## Durum Tablosu

| Faz | Dosya | Konu | Durum |
|---|---|---|---|
| 0 | `faz-0-shared-platform.md` | `platform/` → `shared/platform/` taşıma | [ ] |
| 1 | `faz-1-veritabani-ayrimi.md` | Tek Postgres, 9 ayrı DB + 9 DB kullanıcısı | [ ] |
| 2 | `faz-2-period-projeksiyonu.md` | `academic_periods` cross-service okumasını kaldır | [ ] |
| 3 | `faz-3-http-client-katmani.md` | `shared/client/` internal REST client'ları | [ ] |
| 4 | `faz-4-servis-iskeletleri.md` | 9 × `cmd/main.go` + `go.mod` + `Dockerfile` + `go.work` | [ ] |
| 5 | `faz-5-caddy-gateway.md` | Caddyfile path-prefix routing | [ ] |
| 6 | `faz-6-docker-compose.md` | 16 konteynerli compose + kaynak limitleri | [ ] |
| 7 | `faz-7-e2e-dogrulama.md` | Uçtan uca golden path testi | [ ] |
| 8 | `faz-8-temizlik.md` | `monolith/` kaldırma + doküman güncelleme | [ ] |

**Sıradaki faz: 0**

---

## Kesinleşen Kararlar (yeniden tartışılmaz)

| Konu | Karar |
|---|---|
| Veritabanı | **Tek Postgres konteyneri**, servis başına ayrı DB + ayrı DB kullanıcısı |
| Schema adları | **Değişmiyor** — `auth` DB'sinin içinde `auth` schema'sı. Tüm `.sql` ve sqlc kodu olduğu gibi kalır. |
| Servis paketleme | Her servis kendi konteyneri (9 iş servisi + notification) |
| Sync iletişim | **Internal REST + `X-Internal-Secret`**. gRPC yok (ileride tek seam'de pilot yapılabilir). |
| Async iletişim | **RabbitMQ + outbox pattern** — mevcut yapı korunuyor, değişmiyor |
| Gateway | **Caddy**, path-prefix routing. Traefik/nginx değerlendirildi, elendi. |
| Frontend / Mobil | **Hiç değişmiyor** — route prefix'leri servis sınırlarıyla 1:1 örtüşüyor |

---

## Genel Kurallar (her fazda geçerli)

- Her faz sonunda `go build ./...` hatasız olmalı.
- Faz 0-3 boyunca **monolith çalışır durumda kalır**. Kırılma Faz 4'te başlar.
- Migration **yazılır**, çalıştırılması için kullanıcıya sorulur (CLAUDE.md §6).
- Docker komutları `sudo` gerektirir → **çalıştırma, kullanıcıya göster** (CLAUDE.md §5).
- Generated dosyalara elle dokunma (`db/*.go`) — `make sqlc-<modul>` çalıştır.
- Commit formatı: `<type>(<scope>): <description>` (CLAUDE.md §7).

---

## Geri Dönüş Noktaları

Her faz kendi commit'inde. Bir faz bozarsa:

```bash
git log --oneline -12          # faz commit'lerini gör
git revert <commit>            # tek fazı geri al
```

Faz 1 ve 6 veri kaybı riski taşır (DB volume). O fazların dosyalarında
yedekleme adımı var — atlama.
