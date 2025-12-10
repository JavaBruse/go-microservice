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



## Kubernetes
### 1. Minikube
```bash
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
minikube version
# Запуск кластера
minikube start --cpus=6 --memory=10000mb --nodes=2
minikube status
#	Конфигурация
curl -LO https://dl.k8s.io/v1.34.1/bin/linux/amd64/kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
kubectl cluster-info
kubectl get nodes
#	Подготовка кластера:
kubectl get pods -n kube-system
minikube addons enable ingress
minikube addons enable metrics-server
#	Проверка
kubectl get pods -A
```
![img.png](img.png)
### 2. Docker build
```shell
docker build -t go-microservice:latest .
minikube image load go-microservice:latest
```
![img_1.png](img_1.png)

### 3. Deploy
```shell
kubectl create namespace iot-analytics
kubectl apply -f k8s/redis-deployment.yaml -n iot-analytics
kubectl apply -f k8s/configmap.yaml -n iot-analytics
kubectl apply -f k8s/deployment.yaml -n iot-analytics
kubectl apply -f k8s/hpa.yaml -n iot-analytics
kubectl get all -n iot-analytics
```
![img_3.png](img_3.png)

## 4. Пробрасываем порты
```shell
kubectl port-forward svc/go-microservice 8080:80 -n iot-analytics &
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
