# Infrastructure Setup

Bu klasör MyDreamCampus projesinin altyapı yapılandırmasını içerir.

## 📦 İçerik

```
infrastructure/
├── docker-compose.yml           # Ana Docker orchestration dosyası
├── traefik/                     # API Gateway / Reverse Proxy
│   ├── traefik.yml             # Traefik static config
│   └── dynamic.yml             # Traefik dynamic routing
├── postgres/                    # PostgreSQL
│   └── init-dbs.sql            # Database initialization script
├── rabbitmq/                    # RabbitMQ (Message Broker)
│   └── rabbitmq.conf           # RabbitMQ configuration
├── redis/                       # Redis (Cache)
│   └── redis.conf              # Redis configuration
├── loki/                        # Grafana Loki (Log Aggregation) - Faz 1 sonrası
│   └── loki-config.yml
├── promtail/                    # Promtail (Log Scraper) - Faz 1 sonrası
│   └── promtail-config.yml
└── grafana/                     # Grafana (Visualization) - Faz 1 sonrası
    └── provisioning/
        ├── datasources/
        └── dashboards/
```

## 🚀 Kullanım

### Tüm Servisleri Başlat
```bash
cd infrastructure
docker-compose up -d
```

### Belirli Servisleri Başlat
```bash
docker-compose up -d postgres rabbitmq redis traefik
```

### Servis Durumlarını Kontrol Et
```bash
docker-compose ps
```

### Logları İzle
```bash
docker-compose logs -f
```

### Servisleri Durdur
```bash
docker-compose down
```

### Volume'ları da Sil (Dikkat: Tüm data silinir!)
```bash
docker-compose down -v
```

## 🔗 Servis Erişim Noktaları

| Servis | Port | URL | Credentials |
|--------|------|-----|-------------|
| PostgreSQL | 5432 | `localhost:5432` | `postgres:postgres` |
| RabbitMQ | 5672 | `localhost:5672` | `rabbitmq:rabbitmq` |
| RabbitMQ Management | 15672 | `http://localhost:15672` | `rabbitmq:rabbitmq` |
| Redis | 6379 | `localhost:6379` | - |
| Traefik Dashboard | 8080 | `http://localhost:8080` | - |
| Grafana | 3000 | `http://localhost:3000` | `admin:admin` (Faz 1 sonrası) |
| Loki | 3100 | `http://localhost:3100` | - (Faz 1 sonrası) |

## 📊 Database'ler

PostgreSQL aşağıdaki database'leri otomatik oluşturur:

- `mydreamcampus_auth` - Authentication Service
- `mydreamcampus_staff` - Staff Service
- `mydreamcampus_student` - Student Service
- `mydreamcampus_catalog` - Catalog Service
- `mydreamcampus_enrollment` - Enrollment Service
- `mydreamcampus_attendance` - Attendance Service
- `mydreamcampus_meal` - Meal Service

## 🔧 Konfigürasyon

Her servisin konfigürasyon dosyası ilgili klasörde bulunur. İhtiyacınıza göre düzenleyebilirsiniz:

- **Traefik**: `traefik/traefik.yml` ve `traefik/dynamic.yml`
- **PostgreSQL**: `postgres/init-dbs.sql`
- **RabbitMQ**: `rabbitmq/rabbitmq.conf`
- **Redis**: `redis/redis.conf`

## 📝 Notlar

- Loki, Promtail ve Grafana servisleri şu anda comment'li durumda (Faz 1 sonrası kullanılacak)
- Production'da mutlaka güvenlik ayarları yapılmalı (şifreler, TLS, etc.)
- Volume'lar Docker tarafından yönetilir ve data persistence sağlar

## 🆘 Sorun Giderme

### Port Çakışması
Eğer portlar kullanımdaysa, `docker-compose.yml` içindeki port mapping'leri değiştirin.

### Container Başlamıyor
```bash
docker-compose logs <service-name>
```

### Health Check Başarısız
```bash
docker-compose ps
docker inspect <container-name>
```

## 📚 Kaynaklar

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Traefik v3 Documentation](https://doc.traefik.io/traefik/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [RabbitMQ Documentation](https://www.rabbitmq.com/docs)
- [Redis Documentation](https://redis.io/docs/)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/latest/)
