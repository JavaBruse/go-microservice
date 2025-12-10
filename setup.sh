#!/bin/bash

# 1. Установка зависимостей
echo "Установка зависимостей..."
brew install go docker minikube kubectl helm  # MacOS
# sudo apt install -y golang docker.io minikube kubectl helm  # Ubuntu

# 2. Запуск Minikube
echo "Запуск Minikube..."
minikube start --cpus=4 --memory=8g --driver=docker
minikube addons enable metrics-server
minikube addons enable ingress
eval $(minikube docker-env)

# 3. Сборка Docker образа
echo "Сборка Docker образа..."
docker build -t go-microservice:latest .

# 4. Развертывание в Kubernetes
echo "Развертывание в K8s..."
kubectl create namespace iot-analytics
kubectl apply -f k8s/ -n iot-analytics

# 5. Ожидание запуска
echo "Ожидание запуска pod..."
kubectl wait --for=condition=ready pod -l app=go-microservice -n iot-analytics --timeout=120s

# 6. Порт-форвард для тестирования
echo "Порт-форвард сервиса..."
kubectl port-forward -n iot-analytics svc/go-microservice 8080:80 &
PORT_FORWARD_PID=$!

# 7. Запуск нагрузочного теста
echo "Запуск нагрузочного теста..."
kubectl apply -f k8s/load-test-job.yaml -n iot-analytics

# 8. Мониторинг
echo "Мониторинг автоскейлинга..."
echo "Откройте новый терминал и выполните:"
echo "watch -n 1 'kubectl get hpa,pods -n iot-analytics'"
echo "Для остановки: kill $PORT_FORWARD_PID && minikube stop"