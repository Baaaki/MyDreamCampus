# System Design Mülakat Provası: MyDreamCampus

Bu doküman, MyDreamCampus projesini bir mülakatta anlatman için hazırlanmış bir konuşma rehberidir. Her bölüm, mülakatçının sorabileceği sorularla birlikte verilmiştir.

---

## 1. Elevator Pitch (30 saniye)

> "MyDreamCampus, bir üniversite yönetim sistemi. 9 Go mikroservis, React web uygulaması ve React Native mobil uygulamadan oluşuyor. Her servisin kendi PostgreSQL veritabanı var, servisler arası iletişim RabbitMQ üzerinden event-driven. Traefik API gateway ile tek bir giriş noktası sağlıyoruz. Öğrenci kaydından not girişine, yoklama takibinden yemekhane rezervasyonuna kadar tüm kampüs süreçlerini kapsıyor."

---

## 2. High-Level Architecture (Whiteboard Çizimi)

Whiteboard'a şunu çiz:

```
                    ┌─────────────┐
                    │   Clients   │
                    │ Web / Mobile│
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Traefik   │
                    │ API Gateway │
                    │   (:80)     │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼───┐  ┌────▼───┐  ┌────▼────┐
         │  Auth  │  │ Staff  │  │ Student │  ... (9 servis)
         │ :8001  │  │ :8002  │  │  :8003  │
         └───┬────┘  └───┬────┘  └────┬────┘
             │           │            │
         ┌───▼────┐  ┌───▼────┐  ┌───▼─────┐
         │ PG Auth│  │PG Staff│  │PG Student│  ... (9 DB)
         │ :5432  │  │ :5433  │  │  :5434   │
         └────────┘  └────────┘  └─────────-┘
              │            │            │
              └────────────┼────────────┘
                           │
                    ┌──────▼──────┐
                    │  RabbitMQ   │    ┌───────┐
                    │  (Events)   │◄───│ Redis │
                    └─────────────┘    └───────┘
```

**Anlatım:**
> "Üstte iki client var: React web uygulaması ve React Native mobil uygulama. Tüm istekler Traefik API Gateway üzerinden geçiyor. Traefik, path-based routing yapıyor — `/api/auth/*` auth servisine, `/api/students/*` student servisine gidiyor. Her servisin kendi PostgreSQL veritabanı var — database per service pattern. Servisler arası asenkron iletişim RabbitMQ ile event-driven. Redis ise session cache ve rate limiting için kullanılıyor."

---

## 3. Deep Dive: Bir Request'in Yolculuğu

**Mülakatçı sorusu:** *"Bir öğrenci ders kaydı yaptığında ne oluyor? Uçtan uca anlat."*

### Senaryo: Öğrenci enrollment yapar

```
Adım 1: Frontend
─────────────────
Student, web'den "Kayıt Ol" butonuna tıklar.
POST /api/enrollment/enroll { semester_course_id: "...", student_id: "..." }
Authorization: Bearer <JWT>

Adım 2: Traefik — Forward Auth
───────────────────────────────
Traefik isteği alır, önce auth-service'e forward eder:
  → GET /api/auth/verify (aynı JWT header ile)

Auth-service JWT'yi doğrular:
  - Token süresi dolmuş mu? → 401
  - Token blacklist'te mi? (Redis kontrolü) → 401
  - Geçerli → 200 + Response Headers:
    X-User-ID: "uuid-123"
    X-User-Role: "student"
    X-User-Department: "CS"

Traefik bu header'ları orijinal isteğe ekler ve enrollment-service'e yönlendirir.

Adım 3: Enrollment Service — Handler
─────────────────────────────────────
Handler, X-User-ID ve X-User-Role header'larını okur.
RBAC middleware kontrol eder: student rolü enrollment yapabilir mi? → Evet.

Adım 4: Enrollment Service — Business Logic
────────────────────────────────────────────
Service katmanı şunları kontrol eder:
  a. Öğrenci zaten bu derse kayıtlı mı? → Duplicate check
  b. Dersin kontenjanı dolu mu? → Quota check
  c. Ön koşul dersleri geçmiş mi? → Prerequisite validation
  d. Akademik takvim kayıt döneminde mi? → Period check

Tüm kontroller geçerse:

Adım 5: Database Transaction (Outbox Pattern)
──────────────────────────────────────────────
Tek bir transaction içinde:
  1. INSERT INTO enrollments (...) → Kayıt oluştur
  2. UPDATE semester_courses SET current_count = current_count + 1 → Kontenjan güncelle
  3. INSERT INTO outbox_events (event_type: "enrollment.enrolled", payload: {...})

COMMIT → Üçü birden başarılı veya üçü birden geri alınır.

Adım 6: Response
─────────────────
201 Created → Frontend'e başarılı yanıt döner.

Adım 7: Asenkron Event Propagation
───────────────────────────────────
Outbox worker (background goroutine, her 2 saniyede çalışır):
  1. SELECT FROM outbox_events WHERE status = 'pending'
  2. RabbitMQ'ya publish: "enrollment.enrolled"
  3. UPDATE outbox_events SET status = 'sent'

Adım 8: Consumer Services
──────────────────────────
  - Attendance Service: Kayıt alır → Bu öğrenci için yoklama kaydı oluşturur
  - Grades Service: Kayıt alır → Bu öğrenci için not kaydı oluşturur

Her consumer:
  1. processed_events tablosunu kontrol eder (idempotency)
  2. İşlemi yapar
  3. processed_events'e yazar
  4. RabbitMQ'ya ACK gönderir
```

---

## 4. Sıkça Sorulan Sorular ve Cevaplar

### Q: "Neden microservices? Monolith yetmez miydi?"

> "Haklısınız, bir startup için monolith-first yaklaşımı genelde daha mantıklı. Ama burada bilinçli bir tercih yaptım. Domain boundary'ler çok net — enrollment, grades, attendance tamamen farklı data model'lere ve change frequency'lere sahip. Ve asıl amaçlarımdan biri distributed systems pattern'lerini gerçek bir projede öğrenmekti: eventual consistency, outbox pattern, API gateway, event-driven architecture. Monolith'le bunları öğrenemezdim. Ama trade-off'un farkındayım — operasyonel karmaşıklık çok daha yüksek."

### Q: "RabbitMQ çökerse ne olur?"

> "Outbox pattern tam da bu senaryoyu çözüyor. Event'ler önce PostgreSQL'deki outbox tablosuna yazılıyor — aynı transaction içinde business data ile birlikte. RabbitMQ çökse bile event'ler outbox'ta pending olarak kalır. RabbitMQ geri geldiğinde outbox worker bir sonraki polling cycle'da event'leri publish eder. Yani hiçbir event kaybolmaz. At-least-once delivery garantisi veriyoruz. Consumer tarafında da processed_events tablosu ile idempotency sağlıyoruz — aynı event iki kez gelse bile ikinci sefer skip edilir."

### Q: "Bir servis down olursa diğerleri etkilenir mi?"

> "Hayır, bu event-driven architecture'ın en büyük avantajı. Diyelim grades-service çöktü. Enrollment hala çalışır, event'ler RabbitMQ kuyruğunda birikir. Grades-service geri geldiğinde kuyruktan event'leri alıp işler. Tek istisna auth-service — çünkü Traefik'in forward-auth middleware'i her authenticated request'te auth-service'i çağırıyor. Auth-service çökerse hiçbir authenticated endpoint çalışmaz. Bu bir single point of failure, ve bunu çözmenin yolu auth-service'i horizontally scale etmek veya JWT validation'ı gateway seviyesinde yapmak olurdu."

### Q: "Cross-service join'e ihtiyacın olursa ne yapıyorsun?"

> "İki yaklaşımımız var. Birincisi, frontend'den paralel request atıp client-side join yapmak — mesela öğrenci dashboard'unda grades-service'ten notları, enrollment-service'ten kayıtları ayrı ayrı çekip frontend'de birleştiriyoruz. İkincisi, event'lerle data replication — mesela enrollment-service'te dersin adı ve hocasının bilgisi var, bunlar course.semester.created event'iyle geldi. Böylece enrollment-service, kendi verisini gösterirken catalog-service'i çağırmak zorunda kalmıyor. Trade-off: data duplication var, ama servis bağımsızlığı kazanıyoruz."

### Q: "Authentication nasıl çalışıyor?"

> "JWT-based, ama pure stateless değil — hybrid bir yaklaşım. Login'de access token (15 dk) ve refresh token (24 saat) veriyoruz. Access token her request'te Traefik üzerinden forward-auth ile doğrulanıyor. Auth-service JWT'yi validate ediyor ve X-User-ID, X-User-Role gibi header'ları downstream service'lere inject ediyor. Redis'te token blacklist tutuyoruz — logout olunca veya şüpheli aktivite olunca token'ı blacklist'e ekliyoruz. Bu sayede token'ı anında invalidate edebiliyoruz, pure stateless JWT'nin en büyük zayıflığını kapatıyoruz."

### Q: "Event ordering garanti ediliyor mu?"

> "Bir servis içinde evet — outbox tablosuna sequential ID ile yazıyoruz. Ama servisler arası global ordering yok, çünkü RabbitMQ FIFO garanti vermiyor (özellikle retry durumlarında). Pratikte bu bizim için sorun değil çünkü event'lerimiz bağımsız — bir enrollment event'i ile bir grades event'i arasında sıralama ilişkisi yok. Ama aynı entity üzerinde ordering önemliyse (mesela bir öğrencinin enrollment → drop → re-enroll sırası), event'in içinde timestamp var ve consumer idempotency check'i yapıyor."

### Q: "Rate limiting nasıl çalışıyor?"

> "Redis-backed sliding window counter. Üç seviyemiz var: IP-based (tüm endpoint'ler), user-based (authenticated endpoint'ler), ve endpoint-specific (login gibi hassas endpoint'ler). Redis'in INCR + EXPIRE komutlarıyla atomic counter tutuyoruz. Window süresi dolduğunda counter sıfırlanıyor. Neden Redis? In-memory olduğu için hızlı, ve tüm servislerden erişilebilir — distributed rate limiting."

### Q: "Tekrar yapsan neyi farklı yapardın?"

> "Üç şey:
> 1. **Monolith-first başlayıp sonra parçalardım.** Şimdi mimari pattern'leri öğrendim, ama başlangıçta setup overhead'i çok zaman aldı.
> 2. **Test'leri en baştan yazardım.** Şu an test coverage düşük. Integration test'lerle outbox pattern'ini ve event consumer'ları test etmek çok değerli olurdu.
> 3. **Event schema registry eklerdim.** Şu an event payload'ları implicitly defined — JSON içinde ne olduğu sadece producer ve consumer koduna bakarak anlaşılıyor. Bir schema registry (veya en azından shared event types) daha güvenli olurdu."

---

## 5. Teknik Derinlik Soruları

### Q: "Outbox pattern'de ordering ve exactly-once nasıl sağlanıyor?"

> "Exactly-once delivery mümkün değil distributed systems'da — bunu biliyorum. Biz at-least-once delivery sağlıyoruz outbox + ACK mekanizmasıyla. Consumer tarafında exactly-once processing sağlıyoruz processed_events tablosuyla — bu idempotent consumer pattern. Outbox worker event'i publish ettiğinde status'ü 'sent' yapıyor. Eğer publish başarılı ama status update başarısız olursa, aynı event tekrar publish edilir — ama consumer zaten processed_events'te gördüğü için skip eder."

### Q: "pgtype nedir, neden kullanıyorsunuz?"

> "sqlc, pgx v5 driver'ı ile çalıştığında PostgreSQL'in native type'larını (TEXT, UUID, NUMERIC, TIMESTAMP) Go'nun basit type'larına değil, pgtype struct'larına map ediyor. Mesela bir nullable string pgtype.Text oluyor — içinde String ve Valid alanları var. Bu, NULL değerleri güvenli bir şekilde temsil etmemizi sağlıyor — Go'da *string ile pointer kullanmak yerine. Ama pgtype ile çalışmak verbose, bu yüzden shared/utils/pgtype_helpers.go'da conversion helper'larımız var: StringToPgText, PgUUIDToString gibi."

### Q: "Neden sqlc, neden GORM değil?"

> "GORM runtime reflection kullanıyor — query ne üretecek önceden bilemiyorsun. sqlc ile yazdığım SQL, çalışan SQL. Compile-time type safety var: kolonun adını yanlış yazarsam sqlc generate hata verir, GORM runtime'da panic atar. Ve PostgreSQL'in tüm gücünü kullanabiliyorum — CTE, window function, JSONB — GORM bunları ya desteklemiyor ya da raw SQL'e düşmeni gerektiriyor. Trade-off olarak code generation step var — her SQL değişikliğinde make sqlc çalıştırmam gerekiyor."

---

## 6. Behavioral / Soft Skill Soruları

### Q: "AI kullandın mı?"

> "Evet, aktif olarak Claude kullandım. Ama burada önemli bir ayrım var: AI benim code completion aracım, mimar değil. Her mimari kararı ben verdim — microservices mı monolith mi, outbox pattern mı direct publish mı, sqlc mı GORM mü. AI ile implementasyon hızımı artırdım, ama neden outbox pattern kullanıyorum, trade-off'ları neler, alternatifler neydi — bunları biliyorum ve anlatabilirim. AI, benim 'neden' anlamamı değiştirmedi, 'nasıl' hızımı artırdı."

### Q: "En zor problem ne oldu?"

> "Eventual consistency. İlk başta dual-write yapıyordum — önce DB'ye yaz, sonra RabbitMQ'ya publish et. Arada crash olunca event kayboluyordu. Bunu anladığımda outbox pattern'e geçtim. Ama sonra consumer tarafında da sorun çıktı — aynı event birden fazla geliyordu. Idempotent consumer pattern'i ekledim. Bu süreçte distributed systems'ın temel prensiplerini — CAP theorem, at-least-once delivery, idempotency — sadece teoride değil, pratikte de öğrendim."

### Q: "Bu projede en çok ne öğrendin?"

> "İki şey. Teknik olarak: distributed systems gerçekten zor, ve zorluk kodlamada değil — consistency, failure handling, ve observability'de. Mimari bir karar 5 dakikada alınıyor ama sonuçları tüm projeye yayılıyor. Process olarak: AI aracı kullanmak, neyin neden yapıldığını anlamayı daha da önemli kılıyor. Çünkü AI hızlı kod üretebilir, ama trade-off'ları, failure mode'ları, ve mimari uyumu senin bilmen gerekiyor."

---

## 7. Bonus: Ölçeklenebilirlik Senaryoları

**Mülakatçı:** *"Bu sistemi 10,000 öğrenciye scale etmen gerekse ne yaparsın?"*

> "Şu anki mimaride birkaç bottleneck var:
>
> 1. **Auth-service**: Her request'te çağrılıyor (forward-auth). Horizontally scale edilmeli — birden fazla instance, load balancer arkasında. Veya JWT validation'ı Traefik plugin'i olarak yapılabilir, böylece auth-service'e network hop kalmaz.
>
> 2. **Enrollment döneminde spike**: Dönem başında tüm öğrenciler aynı anda kayıt yapar. Quota kontrolleri pessimistic lock ile yapılmalı (`SELECT ... FOR UPDATE`). Eğer yetmezse, Redis distributed lock ile queue-based enrollment'a geçilebilir.
>
> 3. **Database**: Her servisin kendi DB'si olduğu için bağımsız scale edilebilir. Enrollment DB'si read-heavy'yse read replica eklenebilir. Connection pooling zaten pgxpool ile yapılıyor.
>
> 4. **RabbitMQ**: Tek instance yeterli olmayabilir. Mirrored queue veya quorum queue'larla HA sağlanabilir. Consumer'lar horizontally scale edilebilir (aynı queue'yu dinleyen birden fazla consumer instance).
>
> 5. **Frontend**: Static dosyalar CDN'den serve edilir. API istekleri zaten Traefik üzerinden geçiyor, Traefik'e birden fazla backend instance tanımlanabilir."

---

## 8. Prova Checklist

Mülakattan önce bu soruları sesli olarak cevapla:

- [ ] Projeyi 30 saniyede anlat (elevator pitch)
- [ ] Whiteboard'a mimariyi çiz ve 2 dakikada açıkla
- [ ] "Bir öğrenci ders kaydı yaptığında ne olur?" — uçtan uca anlat
- [ ] "Neden microservices?" — trade-off'larıyla birlikte
- [ ] "RabbitMQ çökerse ne olur?" — outbox pattern açıkla
- [ ] "Tekrar yapsan neyi değiştirirdin?" — özfarkındalık göster
- [ ] "AI kullandın mı?" — dürüst ve güçlü bir şekilde yanıtla
- [ ] "En zor problem ne oldu?" — eventual consistency hikayesi
- [ ] Scale senaryosu — bottleneck'leri ve çözümleri biliyorsun
