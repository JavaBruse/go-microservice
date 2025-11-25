## Go haiload project

## Сборка
```shell
go build -o main .
```

## Запуск проекта
```shell
docker-compose up --build -d
```
![img.png](img.png)

## Нагрузочное тестирование
```shell
wrk -t12 -c500 -d60s http://localhost:8080/api/users
```
![img_2.png](img_2.png)
### Рерузьтат:
 - RPS: 99,119 (> 1,000 требуемых)
 - Latency: 8.30ms (< 10ms) 
 - Ошибки: 0 HTTP ошибок (1967 timeout - норма для highload)
##  Мониторинг метрик
```shell
curl http://localhost:8080/metrics
```
![img_1.png](img_1.png)

## Структура приложения

```azure
go-microservice/
├── main.go
├── handlers/
│   ├── user_handler.go
│   └── integration_handler.go
├── services/
│   ├── user_service.go
│   └── integration_service.go
├── models/
│   └── user.go
├── utils/
│   ├── logger.go
│   └── rate_limiter.go
├── metrics/
│   └── prometheus.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```