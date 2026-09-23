# Faz 0 — Hazırlık

## Başlamadan
- README §1–§4 okunmuş olmalı.

## Görevler

- [x] **0.1 Plan onayı** — `cfbf0de`
  README'deki "Plan durumu" `ONAYLANDI` değilse kullanıcıya planı onaylayıp
  onaylamadığını sor. Onaylayınca README'de durumu tarihle güncelle.

- [x] **0.2 Açık kararlar** — `cfbf0de`
  README §4'teki K1–K5'i kullanıcıya sor (hepsini birden sormak tercih edilir;
  kullanıcı isterse ilgili faza bırakılabilir). Cevapları tablonun "Cevap"
  sütununa yaz.

- [x] **0.3 Çalışma dalı** — `cfbf0de`
  - Çalışma ağacı temiz olmalı (`landingpage/` hariç izlenmeyen dosya
    olmamalı; varsa kullanıcıya sor).
  - `git switch -c demo-readiness` (main'den).
  - `docs/demo-launch/` dosyalarını tek tek stage et ve commit'le:
    `docs(infra): add public demo launch plan`.
  > Not (23.09): Ağaçta commit'lenmemiş Go 1.27 yükseltmesi vardı. Kullanıcı
  > "önce her şeyi güncelle, commit'le, sonra işe başla" dedi; değişiklikler
  > dala taşındı ve 0.4'te commit'lendi.

- [x] **0.4 Bağımlılık ve imaj güncellemesi** (kullanıcı isteği, 23.09) —
  `1fcc2b8`…`e2bebd7` (18 commit)
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
  > Not (24.09): Bilerek geride kalanlar — postgres 18 (19 beta),
  > frontend TypeScript 6.0 (typescript-eslint 7'yi desteklemiyor), mobilde
  > Expo SDK 57'nin pin'lediği native paketler, tailwindcss 3 (NativeWind 4),
  > TypeScript 6.0 (Expo araçları TS'nin JS API'sini kullanıyor), Babel 7
  > preset'leri (Expo zinciri Babel 7) ve `@testing-library/react-native` 13
  > (14 yeni `test-renderer` paketini istiyor; kütüphane kodda kullanılmıyor).
  > Mobilde jest 30 kalsın diye `expo.install.exclude`'a jest eklendi.
  > Kullanılmayan `openapi-typescript` (mobil) ve `period-tabs.tsx` silindi.

- [x] **0.5 Başlangıç doğrulaması** — kod değişikliği yok; sonuç §7 günlükte
  README §5'teki backend, frontend ve mobil komutlarını çalıştır. Sonucu
  (geçen/kalan) Oturum günlüğüne yaz. Kırmızı bir şey varsa önce kullanıcıya
  bildir; plan yeşil bir başlangıç varsayıyor.

## Faz sonu
README §1 madde 6'yı uygula.
