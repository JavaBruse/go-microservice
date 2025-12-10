# Высоконагруженный IoT сервис с AI-оптимизацией на Go

## О проекте
Сервис для обработки потоковых метрик от IoT-устройств с аналитикой нагрузки, обнаружением аномалий и автоматическим масштабированием в Kubernetes. Обрабатывает более 1000 RPS с latency < 50мс.

## Технические требования
- Go 1.24+
- Docker & Docker Compose
- Kubernetes (Minikube/Kind)
- Redis
- Prometheus + Grafana
- Обработка >= 1000 RPS
- Точность детекции >70%
- False positive <10%

## Архитектура
```mermaid
graph TB
    IoT -->|JSON| API
    API --> Analytics
    Analytics --> Redis[(Redis)]
    Analytics --> Prometheus
    Prometheus --> Grafana
    API --> K8S
    subgraph Analytics
        Analytics --> Rolling
        Analytics --> ZScore
        Analytics --> Goroutines
    end
```

## Быстрый старт
### 1. Setup
```
chmod +x setup.sh && ./setup.sh
```

## Kubernetes
### 1. Minikube
```
minikube start --cpus=4 --memory=8g --driver=docker
minikube addons enable metrics-server
minikube addons enable ingress
eval $(minikube docker-env)
```

### 2. Docker build
```
docker build -t go-microservice:latest .
minikube image load go-microservice:latest
```

### 3. Deploy
```
kubectl create namespace iot-analytics
kubectl apply -f k8s/ -n iot-analytics
kubectl get all -n iot-analytics
```

## Prometheus метрики
```
http_requests_total
http_request_duration_seconds
metrics_processed_total
anomalies_detected_total
rolling_average_rps
processing_queue_size
```

## Нагрузочное тестирование
```
kubectl get hpa,pods -n iot-analytics
```

## Тестирование точности
--------------------------
--------------------------

## Структура проекта
```
go-microservice/
├── main.go                      # Точка входа
├── handlers/                    # HTTP обработчики
│   ├── metrics_handler.go       # Обработчик метрик
│   ├── user_handler.go          # CRUD пользователей
│   └── integration_handler.go   # Внешние API
├── services/                    # Бизнес-логика
│   ├── analytics_service.go     # Аналитика + Redis
│   ├── user_service.go          # Пользователи
│   └── integration_service.go   # Интеграции
├── models/                      # Модели данных
│   ├── metric.go                # Модель метрики
│   └── user.go                  # Модель пользователя
├── utils/                       # Утилиты
│   ├── logger.go                # Логирование
│   └── rate_limiter.go          # Rate limiting
├── metrics/                     # Prometheus метрики
│   └── prometheus.go
├── k8s/                         # Kubernetes манифесты
│   ├── deployment.yaml
│   ├── hpa.yaml
│   ├── load-test-job.yaml
│   ├── metrics-server.yaml
│   ├── redis-deployment.yaml
│   ├── configmap.yaml
│   └── ingress.yaml
├── docker-compose.yml           # Локальная разработка
├── Dockerfile                   # Production сборка
├── prometheus.yml               # Конфиг Prometheus
├── setup.sh                     # Запуск проекта
└── README.md                    # Документация
```
