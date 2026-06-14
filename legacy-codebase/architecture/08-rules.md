> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 12. Yasaklar ve Kırmızı Çizgiler

> Bölüm 0.1 AI Kuralları'nın **devamı**. Bölüm 0.1 AI çalışma davranışını, bu bölüm kod ve mimari kuralları kapsar.

**Mimari yasaklar:**

| Yasak | Sebep |
|---|---|
| Cross-schema FK (PostgreSQL `REFERENCES <other_schema>.<table>`) | Modül ayırma maliyetini patlatır |
| Cross-schema SELECT / JOIN (`JOIN auth.users ON staff.staff.user_id = auth.users.id`) | Gizli bağımlılık; PR review'da reject |
| Bir modülün başka modülün `internal/...` paketini import etmesi | Sadece public `module.go` veya `Service` interface açıktır |
| Outbox'sız direkt `publisher.Publish` çağırmak | Atomicity garantisi kaybolur; event domain insert'la sync olmaz |
| Notification servisinden monolith DB'sine SQL/HTTP erişimi | Servis sınırı net olmalı; payload self-contained (Bölüm 5.3) |
| Notification servisinden başka servisi event publish etmesi | Notification leaf consumer; kendi event'i yok |
| meal modülünün payment dışında bir modül ile iletişim kurması | Bölüm 9'daki meal-payment izolasyonu kuralı |
| Bir modülü monolith'ten servise ayırmak (notification dışında) | Plan otoritesi; geri ayırma için yeni karar gerekli |

**Kod yazım yasakları:**

| Yasak | Sebep |
|---|---|
| Yeni helper/interface uydurmak (`db.InTx`, `OutboxWriter` gibi) | Mevcut servisler referansdır; `pool.Begin/Commit` direkt kullanılır (`staff_service` kalıbı) |
| Typed event payload struct'ı yazmak (örn. `UserRegisteredEvent struct`) | Mevcut kalıp `map[string]any`; tutarlılık |
| Modül adında tire (`-`) kullanmak | Underscore (`_`) standardı (Bölüm 0.2 Glossary) |
| Test'i `t.Skip()` ile atlamak | CLAUDE.md failure mode kuralı |
| `--no-verify` ile commit | Hook bypass, CLAUDE.md kuralı |
| `as any`, `@ts-ignore` (TS), `interface{}` cast'ları (Go) | Type safety kaybı; alternatif önerilmeden eklenmez |

**Süreç yasakları:**

| Yasak | Sebep |
|---|---|
| Yeni modül, event, route, exchange, queue eklemeden ÖNCE kullanıcıya sormamak | Bölüm 0.1 AI Kuralları |
| Boyut tahmini vermeden iş başlatmak | Bölüm 0.1 — sürpriz iş yapma |
| Plan'da bulamadığını uydurmak (eşik, helper, payload alanı) | Bölüm 0.1 — ya plan'a ya mevcut servise bak, bulamazsan sor |
| Migration commit'lerini birleştirmek (10 modülü tek commit'e koymak) | Atomic commit kuralı; her modül ayrı commit, CI yeşil olduktan sonra geç |

---

## 13. Açık Sorular (Migration başlamadan önce karar)

Aşağıdakiler plana eklenmedi — uygulamaya geçmeden önce konuşulacak:

- [ ] **Outbox worker stratejisi:** Modül başına ayrı goroutine (mevcut kalıp, Faz 0 default) mi, yoksa tek "Multi-Module Outbox Worker" mı? Faz 0'da A ile başlanır, ölçüm sonrası karar.
- [ ] **Outbox tetikleme:** 5sn polling (mevcut staff_service kalıbı) mı, `LISTEN/NOTIFY` push-based mi? 5sn welcome email/payment receipt için kabul edilebilir; ödeme onayı UX'i için gerekirse push'a geçilir.
- [ ] **sqlc config:** Her modül ayrı `sqlc.yaml` mi (mevcut servis kalıbı), tek root `sqlc.yaml` ile multi-package mı?
- [ ] **PostgreSQL role-based enforcement** (Bölüm 4.3): Faz 0'da kurulsun mu, yoksa migration tamamlandıktan sonra mı? Erken kurulması cross-schema kaçaklarını DB seviyesinde engeller.
- [ ] **Notification retry stratejisi:** Sadece RabbitMQ DLQ mı, application-level retry da mı (consumer içinde requeue/DLQ kararı), ikisi birden mi? Şu an plan A+B önerir; netleşmeli.
- [ ] **Mevcut servisler clean slate mi taşınıyor?** Şu an `backend/services/{auth,staff,student,...}_service` çalışır durumda görünüyor. Migration "kod taşıma + DB consolidation" mı, "sıfırdan yazım" mı? Plan **kod taşıma** varsayıyor (mevcut servisler referansdır — Bölüm 0.1).
- [ ] **attendance Strateji 3 read model timing:** `attendance.students_view` ilk implementasyonda mı yapılır, yoksa attendance modülü taşındıktan sonra ayrı PR mi? Plan ilk implementasyonu öneriyor (Bölüm 11 Faz 2 madde 6).

---

