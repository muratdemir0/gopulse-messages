# GoPulse Messages API

GoPulse Messages, otomatik mesaj gönderim sistemi için geliştirilmiş bir Go uygulamasıdır. Sistem, mesajları webhook aracılığıyla göndermek için background scheduler, Redis cache, PostgreSQL veritabanı ve REST API özelliklerini içerir.

## 🚀 Özellikler

- **Otomatik Veri Üretimi**: 30 saniyede bir fake mesaj verisi üreten background producer
- **Otomatik Mesaj Gönderimi**: 2 dakikalık aralıklarla çalışan background scheduler
- **Webhook Entegrasyonu**: Mesajları harici servislere webhook ile gönderme
- **Redis Cache**: Gönderilen mesajların cache'lenmesi
- **REST API**: Mesaj yönetimi için HTTP endpoints
- **Retry Mekanizması**: Başarısız mesajlar için tekrar deneme sistemi
- **Swagger Dokümantasyonu**: API endpoints için otomatik dokümantasyon
- **Clean Architecture**: Modüler ve sürdürülebilir kod yapısı
- **Docker Desteği**: Container tabanlı deployment
- **Health Check**: Sistem durumu kontrolü
- **Graceful Shutdown**: Güvenli uygulama kapanışı

## 🔄 Sistem Nasıl Çalışır

GoPulse Messages uygulaması başladığında 2 background process otomatik olarak çalışmaya başlar:

### 1. Data Producer (Veri Üretici)
- **Sıklık**: 30 saniyede bir
- **Görev**: Fake mesaj verisi üretir
- **Veri Türü**: Rastgele telefon numarası + mesaj içeriği
- **Durum**: Mesajlar "pending" status'unda veritabanına kaydedilir

### 2. Message Scheduler (Mesaj Zamanlayıcı)  
- **Sıklık**: 2 dakikada bir
- **Görev**: Pending durumundaki mesajları alır ve webhook'a gönderir
- **Cache**: Gönderilen mesajlar Redis'te cache'lenir
- **Retry**: Başarısız mesajlar için tekrar deneme mekanizması

### İş Akışı
```
1. Data Producer → Fake mesaj üret → Veritabanına kaydet (pending)
2. Message Scheduler → Pending mesajları al → Webhook'a gönder → Cache'le
3. API → Gönderilen mesajları listele
```

## 📋 Gereksinimler

- **Docker**: 20.10+
- **Docker Compose**: 2.0+

## 🐳 Kurulum

### 1. Projeyi Klonlama

```bash
git clone https://github.com/muratdemir0/gopulse-messages.git
cd gopulse-messages
```

### 2. Docker Compose ile Kurulum

En kolay yöntem Docker Compose kullanmaktır:

```bash
# Tüm servisleri başlat
docker-compose up -d 

# Logları takip et
docker-compose logs -f server
```

Bu komut aşağıdaki servisleri başlatır:
- `server`: GoPulse Messages API
- `db`: PostgreSQL veritabanı
- `redis`: Redis cache

### Servisler

- **API**: http://localhost:8080
- **PostgreSQL**: localhost:5433
- **Redis**: localhost:6379
- **Swagger UI**: http://localhost:8080/swagger/

### 3. Servisleri Durdurma

```bash
docker-compose down
```

### 4. Sistemi Test Etme

Uygulama başladıktan sonra data producer'ın çalıştığını görmek için:

```bash
# Data producer loglarını takip edin (30 saniyede bir mesaj üretir)
docker-compose logs -f server | grep "Successfully created a new message"

# Message scheduler loglarını takip edin (2 dakikada bir gönderim)
docker-compose logs -f server | grep "message sent"

# Gönderilen mesajları API ile kontrol edin
curl "http://localhost:8080/messages?limit=5"
```

### 5. Webhook Konfigürasyonu (Opsiyonel)

Eğer gerçek bir webhook endpoint'i kullanmak istiyorsanız, `compose.yaml` dosyasında webhook ayarlarını güncelleyin:

```yaml
environment:
  - WEBHOOK_HOST=https://your-webhook-host.com
  - WEBHOOK_PATH=/your-webhook-endpoint
```

## ⚙️ Konfigürasyon

### Environment Variables (Advanced)

Docker Compose varsayılan ayarlarla yeterlidir. İhtiyacınız varsa şu environment variables'ları özelleştirebilirsiniz:

- `DATABASE_DSN`: PostgreSQL bağlantı string'i
- `REDIS_ADDR`: Redis server adresi
- `REDIS_PASSWORD`: Redis şifresi
- `REDIS_DB`: Redis database numarası
- `WEBHOOK_HOST`: Webhook host adresi
- `WEBHOOK_PATH`: Webhook endpoint path

## 📖 API Dokümantasyonu

### Swagger UI

API dokümantasyonuna erişim için: http://localhost:8080/swagger/

### Endpoints

#### 1. Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 2. Otomatik Mesaj Gönderimini Başlat
```http
POST /messages/start
```

**Response:**
```json
{
  "message": "Automatic message sending started",
  "status": "active"
}
```

#### 3. Otomatik Mesaj Gönderimini Durdur
```http
POST /messages/stop
```

**Response:**
```json
{
  "message": "Automatic message sending stopped",
  "status": "inactive"
}
```

#### 4. Gönderilen Mesajları Listele
```http
GET /messages?limit=10&offset=0
```

**Query Parameters:**
- `limit` (optional): Döndürülecek mesaj sayısı (varsayılan: 10)
- `offset` (optional): Sayfalama için offset (varsayılan: 0)

**Response:**
```json
{
  "messages": [
    {
      "id": 1,
      "recipient": "+1234567890",
      "content": "Test message",
      "status": "sent",
      "sent_at": "2024-01-01T00:00:00Z",
      "retry_count": 0,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "count": 1
}
```

### Kod Yapısı

```
├── cmd/api/                 # Application entry point
├── internal/
│   ├── adapters/           # External adapters (DB, HTTP, Redis, Webhook)
│   ├── app/                # Application services
│   ├── config/             # Configuration management
│   ├── domain/             # Domain models and interfaces
│   └── infra/              # Infrastructure (handlers, middleware)
├── api/rest/               # REST API models
├── docs/                   # Swagger documentation
├── migrations/             # Database migrations
└── testdata/               # Test configuration files
```

### Database Migration

Migration'lar Docker Compose ile otomatik çalışır. Veritabanı başladığında tüm migration'lar uygulanır.

## 📦 Deployment

### Production Docker Build

```bash
# Docker image build
docker build -t gopulse-messages:latest .

# Container çalıştır
docker run -d \
  --name gopulse-messages \
  -p 8080:8080 \
  -e DATABASE_DSN="your_production_dsn" \
  -e REDIS_ADDR="your_redis_addr" \
  gopulse-messages:latest
```

### Production Konfigürasyonu

Production için environment variables kullanın:

```bash
export APP_ENV=prod
export DATABASE_DSN="your_production_postgres_dsn"
export REDIS_ADDR="your_production_redis_address"
export REDIS_PASSWORD="your_production_redis_password"
export WEBHOOK_HOST="your_production_webhook_host"
export WEBHOOK_PATH="your_production_webhook_path"
```


### Health Check

Sistem durumunu kontrol etmek için:

```bash
curl http://localhost:8080/health
```

## 📝 API Response Formatları

### Başarılı Response
```json
{
  "data": { ... },
  "status": "success"
}
```

### Hata Response
```json
{
  "error": "Error message",
  "status": "error"
}
```