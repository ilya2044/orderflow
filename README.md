# OrderFlow — Микросервисная система обработки заказов

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-14-000000?logo=next.js)](https://nextjs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://postgresql.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-7-47A248?logo=mongodb)](https://mongodb.com)
[![Kafka](https://img.shields.io/badge/Apache_Kafka-7.6-231F20?logo=apache-kafka)](https://kafka.apache.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)](https://redis.io)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docker.com)

Дипломный проект — масштабируемое приложение для обработки заказов, построенное на **микросервисной архитектуре** с использованием современного стека технологий уровня enterprise.

## Архитектура

```
┌─────────────┐     ┌──────────────────────────────────────────────────────┐
│   Frontend  │────▶│                    Nginx (Load Balancer)             │
│  Next.js 14 │     └──────────────────────┬───────────────────────────────┘
└─────────────┘                            │
                                    ┌──────▼──────┐
                                    │ API Gateway  │  Rate Limiting · JWT Validation
                                    │   (Gin)      │  Reverse Proxy · Metrics
                                    └──────┬───────┘
                     ┌─────────────────────┼─────────────────────┐
              ┌──────▼──────┐      ┌───────▼──────┐      ┌───────▼──────┐
              │ Auth Service │      │ User Service  │      │Product Service│
              │ JWT · Redis  │      │  PostgreSQL   │      │ MongoDB · ES  │
              └─────────────┘      └──────────────┘      │ MinIO        │
                                                          └──────────────┘
              ┌──────────────┐      ┌───────────────┐      ┌─────────────┐
              │ Order Service │─────▶│  Kafka Topics  │────▶│  Inventory  │
              │  PostgreSQL  │      │  order.created │      │  Service    │
              └──────────────┘      │  order.status  │      └─────────────┘
                                    │  payment.proc  │      ┌─────────────┐
              ┌──────────────┐      │  inventory.*   │────▶│  Payment    │
              │ Notification │◀─────│  notification  │      │  Service    │
              │   Service    │      └───────────────┘      └─────────────┘
              │ Redis · SMTP │
              └──────────────┘
```

## Технологический стек

### Backend (Go 1.22)
| Сервис | Технологии | Порт |
|--------|-----------|------|
| API Gateway | Gin, Redis (rate limit), JWT validation | 8080 |
| Auth Service | Gin, PostgreSQL, Redis, JWT, bcrypt | 8081 |
| User Service | Gin, PostgreSQL | 8082 |
| Product Service | Gin, MongoDB, Elasticsearch, MinIO | 8083 |
| Order Service | Gin, PostgreSQL, Kafka Producer | 8084 |
| Inventory Service | Gin, PostgreSQL, Kafka Consumer | 8085 |
| Payment Service | Gin, PostgreSQL, Kafka | 8086 |
| Notification Service | Kafka Consumer, Redis | 8087 |

### Инфраструктура
| Сервис | Назначение | Порт |
|--------|-----------|------|
| PostgreSQL 16 | Реляционные данные (auth, users, orders, inventory, payments) | 5432 |
| MongoDB 7 | Каталог товаров | 27017 |
| Redis 7 | Кэш, сессии, rate limiting, уведомления | 6379 |
| Apache Kafka | Event streaming между сервисами | 9092 |
| Elasticsearch 8 | Полнотекстовый поиск товаров | 9200 |
| MinIO | Object storage для изображений товаров | 9000/9001 |
| Jaeger | Distributed tracing | 16686 |
| Prometheus | Сбор метрик | 9090 |
| Grafana | Визуализация метрик | 3001 |
| Nginx | Load balancer, reverse proxy | 80 |

### Frontend (Next.js 14)
- **TypeScript** — строгая типизация
- **Tailwind CSS** — utility-first стили
- **shadcn/ui** — компонентная библиотека
- **Framer Motion** — анимации
- **Recharts** — графики и аналитика
- **TanStack Query** — server state management
- **Zustand** — client state management
- **Axios** — HTTP клиент с автообновлением токенов

## Kafka топики и Event-driven flow

```
order.created       → Inventory Service (резервирование)
                    → Notification Service (уведомление)

order.status_updated → Notification Service

inventory.reserved  → Payment Service (инициация оплаты)

payment.processed   → Order Service (обновление статуса)
                    → Notification Service

notification.send   → Notification Service (универсальный)
```

## Быстрый старт

### Требования
- Docker + Docker Compose
- Make (опционально)

### Запуск

```bash
git clone https://github.com/ilya2044/orderflow.git
cd orderflow

# Запустить все сервисы
make up

# Или напрямую
docker compose up -d

```

### Доступные адреса

| Сервис | URL |
|--------|-----|
| Frontend | http://localhost:3000 |
| API Gateway | http://localhost:8080 |
| Kafka UI | http://localhost:8090 |
| MinIO Console | http://localhost:9001 |
| Jaeger UI | http://localhost:16686 |
| Grafana | http://localhost:3001 |
| Prometheus | http://localhost:9090 |

### Тестовые данные
```
Email: admin@order.dev
Password: admin123
```

## API Reference

### Аутентификация
```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

### Заказы
```http
GET    /api/v1/orders          # Список заказов (с пагинацией)
POST   /api/v1/orders          # Создать заказ
GET    /api/v1/orders/:id      # Получить заказ
PUT    /api/v1/orders/:id/status  # Обновить статус (admin)
DELETE /api/v1/orders/:id      # Отменить заказ
```

### Товары
```http
GET    /api/v1/products        # Список товаров
GET    /api/v1/products/search?q=query  # Поиск (Elasticsearch)
POST   /api/v1/products        # Создать товар (admin)
PUT    /api/v1/products/:id    # Обновить товар (admin)
DELETE /api/v1/products/:id    # Удалить товар (admin)
POST   /api/v1/products/:id/images  # Загрузить изображение
```

### Пользователи
```http
GET    /api/v1/users           # Список пользователей (admin)
GET    /api/v1/users/:id       # Получить пользователя
PUT    /api/v1/users/:id       # Обновить профиль
```

### Платежи
```http
GET  /api/v1/payments         # История платежей
POST /api/v1/payments         # Создать платёж
GET  /api/v1/payments/:id     # Получить платёж
```

## Структура проекта

```
order-processing/
├── api-gateway/              # Точка входа, маршрутизация
├── auth-service/             # Аутентификация и авторизация
├── user-service/             # Управление пользователями
├── product-service/          # Каталог товаров
├── order-service/            # Управление заказами
├── inventory-service/        # Управление складом
├── payment-service/          # Обработка платежей
├── notification-service/     # Уведомления
├── pkg/                      # Общие пакеты
│   ├── jwt/                  # JWT утилиты
│   ├── kafka/                # Kafka producer/consumer
│   ├── logger/               # Structured logging (zap)
│   └── response/             # HTTP response helpers
├── frontend/                 # Next.js приложение
├── infra/
│   ├── nginx/                # Nginx конфигурация
│   ├── prometheus/           # Prometheus конфигурация
│   └── grafana/              # Grafana dashboards
├── scripts/                  # Init скрипты
├── docker-compose.yml        # Docker Compose конфигурация
├── go.work                   # Go workspace
├── Makefile                  # Команды разработчика
└── .env.example              # Пример переменных окружения
```

## Ключевые паттерны и практики

- **Clean Architecture** — domain, repository, usecase, delivery слои
- **Event-driven** — async коммуникация через Kafka
- **CQRS** — разделение операций чтения и записи
- **Circuit Breaker** — защита от каскадных сбоев в API Gateway
- **Rate Limiting** — защита от перегрузки (Redis sliding window)
- **JWT + Refresh Tokens** — stateless аутентификация
- **Multi-stage Docker builds** — минимальные образы (scratch)
- **Graceful Shutdown** — корректное завершение работы
- **Structured Logging** — JSON логи через uber/zap
- **Prometheus Metrics** — метрики для каждого сервиса
- **Distributed Tracing** — Jaeger/OpenTelemetry
- **Database per Service** — изолированные БД

## Локальная разработка

```bash
# Только инфраструктура
make infra

# Запустить отдельный сервис
cd auth-service
go run ./cmd/server

# Запустить фронтенд в dev режиме
cd frontend
npm install
npm run dev

# Сгенерировать proto файлы
make proto

# Запустить тесты
make test

# Линтинг
make lint
```

## Мониторинг

Grafana доступна на http://localhost:3001 (admin/admin).  
Jaeger UI на http://localhost:16686 для трассировки запросов.  
Kafka UI на http://localhost:8090 для мониторинга топиков.
