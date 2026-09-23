# Faz 0 — Hazırlık

## Başlamadan
- README §1–§4 okunmuş olmalı.

## Görevler

- [x] **0.1 Plan onayı**
  README'deki "Plan durumu" `ONAYLANDI` değilse kullanıcıya planı onaylayıp
  onaylamadığını sor. Onaylayınca README'de durumu tarihle güncelle.

- [x] **0.2 Açık kararlar**
  README §4'teki K1–K5'i kullanıcıya sor (hepsini birden sormak tercih edilir;
  kullanıcı isterse ilgili faza bırakılabilir). Cevapları tablonun "Cevap"
  sütununa yaz.

- [x] **0.3 Çalışma dalı**
  - Çalışma ağacı temiz olmalı (`landingpage/` hariç izlenmeyen dosya
    olmamalı; varsa kullanıcıya sor).
  - `git switch -c demo-readiness` (main'den).
  - `docs/demo-launch/` dosyalarını tek tek stage et ve commit'le:
    `docs(infra): add public demo launch plan`.
  > Not (23.09): Ağaçta commit'lenmemiş Go 1.27 yükseltmesi vardı. Kullanıcı
  > "önce her şeyi güncelle, commit'le, sonra işe başla" dedi; değişiklikler
  > dala taşındı ve 0.4'te commit'lendi.

- [ ] **0.4 Bağımlılık ve imaj güncellemesi** (kullanıcı isteği, 23.09)
  Plana başlamadan önce projedeki her şey en güncel sürüme çekilir; her alan
  ayrı `chore` commit'i:
  - Go: toolchain + tüm modüllerin bağımlılıkları (`go get -u`, `go mod tidy`),
    Dockerfile'lardaki `golang` imajı, CI `GO_VERSION`.
  - Docker: compose imajları (postgres, rabbitmq, redis, mailhog) ve
    Dockerfile taban imajları (alpine, distroless, bun, caddy).
  - CI: GitHub Actions sürümleri.
  - Frontend (`bun`) ve mobil (`npm` / `npx expo install`) paketleri.
  Her commit öncesi ilgili §5 doğrulaması yeşil olmalı. Kod değişikliği
  gerektiren büyük sürüm atlamalarında kullanıcıya sor. İmajları çekmek
  (`sudo docker compose pull` / `build`) kullanıcıya ait.

- [ ] **0.5 Başlangıç doğrulaması**
  README §5'teki backend, frontend ve mobil komutlarını çalıştır. Sonucu
  (geçen/kalan) Oturum günlüğüne yaz. Kırmızı bir şey varsa önce kullanıcıya
  bildir; plan yeşil bir başlangıç varsayıyor.

## Faz sonu
README §1 madde 6'yı uygula.
