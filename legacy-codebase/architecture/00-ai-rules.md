# 00 — AI Çalışma Kuralları ve Terimler

> Modüler monolith planının AI'a yönelik kuralları ve glossary. Diğer bölümler için: [README.md](README.md).

---

## 0. AI Çalışma Kuralları ve Terimler

> **Bu bölümü her oturum başında oku.** Plan'ın geri kalanı bu kuralları varsayar.

### 0.1 AI Çalışma Kuralları (zorunlu)

- **Bu plan otoritedir.** Çelişen davranış varsa **buradaki** geçerlidir. CLAUDE.md, skills.md veya başka bir doküman ile çatışırsa sebebi söyle, plan'ı uygulamaya başla, kullanıcıya bildir.
- **Yeni modül, event, route, exchange, queue eklemeden ÖNCE kullanıcıya sor.** Bölüm 5.9 ([Event kataloğu](03-events.md)) ve Bölüm 9 ([Modül haritası](02-data.md)) **eksiksiz**dir; eksik gibi görünse bile uydurmadan sor.
- **Outbox kullanmadan `publisher.Publish` çağırma.** Domain insert ile event insert aynı transaction'da olmalı. Atomicity garantisi tek koruma.
- **Cross-schema FK/JOIN yazma.** Convention enforcement; AI test etmesin. Bölüm 4.2'deki ([Database](02-data.md)) kurallar yasaktır, "teknik mümkün" gerekçesi geçerli değil.
- **Boyut tahmini ver.** Yeni iş başlatmadan önce: "X dosya, ~Y satır, ~Z dakika" formatında tahmin yaz, kullanıcı onaylasın. Sürpriz iş yapma.
- **Plan'da bulamadığın şeyi UYDURMA — sor.** Eşik, fonksiyon imzası, helper, paket adı, payload alanı görünmüyorsa ya plan'a bak ya **mevcut servislerin koduna** bak (`backend/services/staff_service/` referans), bulamazsan sor.
- **Kod yazarken mevcut servisler referansdır.** Özellikle `backend/services/staff_service/` altın standart. Repository-with-event pattern, `map[string]any` payload, manual `pool.Begin/Commit`, zap logger, `sharedErrors.Wrap` — bunlar **gerçek kalıplar**. Yeni helper veya pattern uydurma.
- **Doğru naming.** Modül adı, schema adı, Go folder adı her yerde **alt çizgi (`_`)** kullanır: `course_catalog`, `auth`, `staff`. Tire (`-`) sadece dış teknolojilerde geçerli (Docker container adı vb.) — kod tarafında **asla**.

### 0.2 Terimler (Glossary)

> Bu tablo **referansdır**. Bu plan ve gelecek konuşmalarda **sadece sol sütundaki terimler** kullanılır. Sağdaki "yasak" terimler eş anlamlıdır ama tutarsızlık yaratır.

| Doğru terim | Tanım | Yasak / kullanma |
|---|---|---|
| **modül** | Monolith içindeki mantıksal birim. Aynı binary'de çalışır, kendi schema'sına sahip, public Service interface ile dış dünyaya bakar. | "servis" (notification haricinde), "bileşen", "alt-modül" |
| **servis** | Ayrı process, ayrı binary, ayrı DB, ayrı deploy. **Tek servis**: notification. | "modül" (notification için), "mikroservis" (eski terim) |
| **monolith** | 9 modülü (auth, staff, student, course_catalog, enrollment, attendance, grades, meal, payment) içeren tek binary. | "ana app", "core", "main service" |
| **in-process call** | Aynı binary içinde Go fonksiyon çağrısı. Modüller arası read için **tek yol**. | "internal call", "direct call", "function call" |
| **outbox** | `<modul>.outbox_events` tablosu — domain insert ile aynı transaction'da yazılan event kaydı. | "event store", "queue table", "buffer" |
| **outbox worker** | Outbox'tan publish edilmemiş event'leri çekip RabbitMQ'ya yollayan goroutine. | "relay", "dispatcher", "publisher worker" |
| **event** | Outbox üzerinden publish edilen domain olayı. Routing key formatı: `<modul>.<aksiyon>` (`staff.created`). | "message", "notification" (notification servisine ait), "signal" |
| **public Service interface** | Modülün dış dünyaya açtığı Go interface'i (örn. `student.Service.GetByIDs`). Diğer modüller **sadece** bu üzerinden okur. | "API", "facade", "manager" |
| **schema** | PostgreSQL schema (örn. `auth`, `staff`). Modül başına bir schema. | "namespace", "database" (yanlış — DB tek) |
| **cross-schema** | Bir modülün başka modülün schema'sındaki tabloya direkt erişimi. **Yasak** (FK, JOIN, SELECT). | "cross-module SQL", "external table query" |
| **routing key** | RabbitMQ topic exchange'de event'i queue'lara yönlendiren string. Format: `<modul>.<aksiyon>`. | "topic", "channel", "subject" |
| **module exchange** | Her modülün kendi RabbitMQ exchange'i: `staff.events`, `student.events` vb. | "shared exchange", "domain exchange" |
| **modül adı** | Tüm modül adları **küçük harf + alt çizgi**: `auth`, `staff`, `student`, `course_catalog`, `enrollment`, `attendance`, `grades`, `meal`, `payment`. | "course-catalog", "courseCatalog", "Course_Catalog" |

**Naming tutarlılığı (kritik):**
- Go folder: `internal/modules/course_catalog/`
- Go package: `coursecatalog` (Go convention — tek kelime, alt çizgisiz)
- Schema: `course_catalog`
- Exchange: `course_catalog.events`
- Routing key: `course_catalog.semester.created`
- DB tablo: `course_catalog.semester_courses`

Aynı entity için bu beş yer tutarlı olur — **sadece Go package adı** istisnai (Go convention'ı override edemiyoruz).

---

