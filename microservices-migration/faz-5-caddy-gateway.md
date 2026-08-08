# Faz 5 — Caddy Gateway Routing

**Ön koşul:** Faz 4 tamamlandı
**Risk:** Düşük
**Süre:** Kısa — bu fazın tamamı tek dosyada ~15 satır

---

## Amaç

Caddy'nin `/api/*` → `monolith:8080` tek kuralını, servis başına path-prefix
kurallarına çevirmek.

Frontend ve mobil **hiç değişmiyor** — route prefix'leri servis sınırlarıyla
1:1 örtüşüyor. Bu, migrasyonun baştan beri en büyük şansıydı.

---

## Değişecek Tek Dosya

`frontend/Caddyfile`

## Mevcut Hali

```caddyfile
{$SITE_ADDRESS:localhost} {
	encode gzip zstd

	handle /api/* {
		reverse_proxy monolith:8080
	}
	handle /health {
		reverse_proxy monolith:8080
	}

	handle {
		root * /srv
		try_files {path} /index.html
		file_server
	}
}
```

## Yeni Hali

```caddyfile
{$SITE_ADDRESS:localhost} {
	encode gzip zstd

	# Servis başına path-prefix routing. Prefix'ler modüllerin Name()
	# değerlerinden geliyor, frontend'in api-client.ts'indeki prefixUrl'lerle
	# birebir aynı — SPA tarafında değişiklik gerekmiyor.
	#
	# Her prefix İKİ matcher ile yazılıyor: ky "/api/students" (path yok,
	# sadece query) isteği de atıyor ve o istek "/api/students/*" ile
	# EŞLEŞMEZ. Tek matcher yazarsan liste endpointleri 404 olur.
	handle /api/auth /api/auth/*               { reverse_proxy auth-service:8081 }
	handle /api/staff /api/staff/*             { reverse_proxy staff-service:8082 }
	handle /api/admin-staff /api/admin-staff/* { reverse_proxy staff-service:8082 }
	handle /api/students /api/students/*       { reverse_proxy student-service:8083 }
	handle /api/catalog /api/catalog/*         { reverse_proxy catalog-service:8084 }
	handle /api/semesters /api/semesters/*     { reverse_proxy catalog-service:8084 }
	handle /api/enrollment /api/enrollment/*   { reverse_proxy enrollment-service:8085 }
	handle /api/attendance /api/attendance/*   { reverse_proxy attendance-service:8086 }
	handle /api/grades /api/grades/*           { reverse_proxy grades-service:8087 }
	handle /api/meals /api/meals/*             { reverse_proxy meal-service:8088 }
	handle /api/payments /api/payments/*       { reverse_proxy payment-service:8089 }

	# Eşleşmeyen /api/* isteği SPA'ya düşmemeli — yanlış prefix 404 almalı,
	# index.html değil.
	handle /api/* {
		respond "not found" 404
	}

	# Backend canlılık kontrolü. DEPLOY.md ve autodeploy script'i bu yolu
	# kullanıyor; auth-service temsilci olarak seçildi (Redis + DB + RabbitMQ
	# bağımlılığı en yüksek servis, o ayaktaysa altyapı ayakta).
	handle /health {
		reverse_proxy auth-service:8081
	}

	handle {
		root * /srv
		try_files {path} /index.html
		file_server
	}
}
```

### İstek ID'si — kenarda üret

`03-IZLENEBILIRLIK.md` [2]. Site bloğunun başına, `handle` kurallarından
**önce**:

```caddyfile
	# Zincirin ilk halkası. Servise bırakılırsa kenardaki hop (Caddy access
	# log'u) farklı bir ID taşır ve istek gerçek başlangıcından izlenemez.
	# Client kendi ID'sini gönderdiyse (mobil debug) ezme.
	@no_request_id not header X-Request-ID *
	request_header @no_request_id X-Request-ID {http.request.uuid}
```

---

## Dikkat Edilecekler

- **`/internal/*` burada YOK.** Faz 4'te servislerin internal route'ları kök
  altına taşındı ve Caddy onları proxy'lemiyor. Yani dışarıdan erişilemezler.
  Buraya bir `/internal` kuralı **ekleme**.
- `handle` blokları **sırayla ve karşılıklı dışlayıcı** çalışır. `/api/*`
  fallback'i servis kurallarından **sonra** gelmeli.
- `handle_path` **kullanma** — prefix'i soyar, servisler `/api/<name>` altında
  mount edilmiş durumda ve tam yolu bekliyor.
- Servis adları compose'daki `container_name` değil, **service key**'i.
  Faz 6'da compose'da bu adları kullan (`auth-service`, `staff-service`, ...).

---

## Bitiş Kriteri

Faz 6 bitmeden gerçek doğrulama yapılamaz (servisler compose'da yok). Şimdilik
sözdizimi kontrolü:

```bash
# Caddy sözdizimi geçerli mi (kullanıcı çalıştırır)
sudo docker run --rm -v "$PWD/frontend/Caddyfile:/etc/caddy/Caddyfile:ro" \
  caddy:latest caddy validate --config /etc/caddy/Caddyfile
```

Prefix listesinin frontend ile uyumunu kontrol et:

```bash
grep -o "/api/[a-z-]*" frontend/src/lib/api-client.ts | sort -u
```

Çıkan her prefix Caddyfile'da olmalı. Fazla veya eksik varsa düzelt.

---

## Commit

```
chore(infra): route each api prefix to its own service in caddy
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-5-caddy-gateway-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 5 satırını `[x]` yap
3. "Sıradaki faz" satırını **6** yap
