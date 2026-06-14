import re

path = "/home/nautilus/Desktop/Playground/mydreamcampus/CLAUDE.md"
with open(path, "r") as f:
    content = f.read()

old_section = """## 13. Servis Portlari (referans)

| Servis | HTTP | DB |
|---|---|---|
| auth | 8001 | 5432 |
| staff | 8002 | 5433 |
| student | 8003 | 5434 |
| course-catalog | 8004 | 5435 |
| enrollment | 8005 | 5436 |
| attendance | 8006 | 5437 |
| grades | 8007 | 5438 |
| meal | 8008 | 5439 |
| payment | 50051 (gRPC) | 5440 |"""

new_section = """## 13. Servis Portlari (referans)

**Monolit Mimari**
Proje artik bir modular monolith yapisindadir. Tüm modüller tek bir backend process icinde birleştirilmiştir.

| Bilesen | Port | Aciklama |
|---|---|---|
| Monolit Backend | 8080 | Tum moduller (auth, student, meal vb.) tek process icinde calisir. |
| PostgreSQL | 5432 | Tüm modüller aynı DB instance üzerinde kendi schemalarında çalışır. |
| RabbitMQ | 5672/15672 | Asenkron event mesajlaşması için. |
| Redis | 6379 | Token blacklist ve rate limiting vb. işlemler için. |"""

content = content.replace(old_section, new_section)

with open(path, "w") as f:
    f.write(content)
print("CLAUDE.md fixed")
