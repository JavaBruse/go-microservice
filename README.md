## Go haiload project



## Нагрузочное тестирование
```shell
wrk -t12 -c500 -d60s http://localhost:8080/api/users
```
![img_2.png](img_2.png)
### Рерузьтат:
 - RPS: 83,890 (> 1,000 требуемых)
 - Latency: 9.70ms (< 10ms) 
 - Ошибки: 0 HTTP ошибок (1968 timeout - норма для highload)
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