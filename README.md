# GoPulse Messages API

Otomatik mesaj gönderim sistemi. Background scheduler, Redis cache, PostgreSQL ve REST API ile webhook entegrasyonu.

## 🚀 Özellikler

- **Otomatik Mesaj Gönderimi**: 2 dakikalık aralıklarla background scheduler
- **Webhook Entegrasyonu**: Mesajları harici servislere gönderme
- **Redis Cache**: Gönderilen mesajların cache'lenmesi
- **REST API**: Mesaj yönetimi endpoints
- **APM Monitoring**: Jaeger ile distributed tracing
- **Docker Desteği**: Container tabanlı deployment

## 🚀 Hızlı Başlangıç

```bash
# Projeyi klonla
git clone https://github.com/muratdemir0/gopulse-messages.git
cd gopulse-messages

# Docker Compose ile başlat
docker compose up -d --build
```

## 🌐 Servisler

- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/
- **Jaeger UI**: http://localhost:16686 (APM)
- **PostgreSQL**: localhost:5433
- **Redis**: localhost:6379

## 📡 API Kullanımı

```bash
# Health check
curl http://localhost:8080/health

# Mesajları listele
curl "http://localhost:8080/messages?limit=5"

# Otomatik gönderimi başlat/durdur
curl -X POST http://localhost:8080/messages/start
curl -X POST http://localhost:8080/messages/stop
```

## 🔄 Sistem Akışı

1. **Data Producer** → 30s'de bir fake mesaj üret → DB'ye kaydet (pending)
2. **Message Scheduler** → 2dk'da bir pending mesajları al → Webhook'a gönder
3. **Cache** → Gönderilen mesajlar Redis'te cache'lenir

## ⚙️ Konfigürasyon

Temel environment variables (Docker Compose varsayılanları yeterli):

```bash
DATABASE_DSN=postgres://postgres:postgres@localhost:5432/gopulse?sslmode=disable
REDIS_ADDR=localhost:6379
WEBHOOK_HOST=https://webhook.site
WEBHOOK_PATH=/your-webhook-id
TELEMETRY_ENABLED=true
```

## 📊 Monitoring

- **Jaeger UI**: http://localhost:16686 - Request tracing, performance monitoring
- **Health Endpoint**: http://localhost:8080/health - Sistem durumu

### 🚧 APM Eksikleri (TODO)

- [ ] **Custom Instrumentation**: Business logic için custom span'lar eklenmeli
- [ ] **Error Tracking**: Structured error logging ve alerting eksik
- [ ] **Performance Dashboards**: Dashboard yapılmadı
- [ ] **Alert Rules**: Critical metric'ler için alert rule'ları eksik

## 🚀 CI/CD Pipeline

Mevcut pipeline Docker build, security scan ve deployment içeriyor.

### 🚧 Pipeline Eksikleri (TODO)

- [ ] **Code Coverage**: Test coverage reporting ve gate eksik
- [ ] **Performance Testing**: Load testing ve benchmark'lar eksik
- [ ] **Quality Gates**: SonarQube integration eksik
- [ ] **Dependency Updates**: Automated dependency bump'ları eksik
- [ ] **Container Registry**: Private registry setup eksik

## 📋 Gereksinimler

- Docker 20.10+
- Docker Compose 2.0+