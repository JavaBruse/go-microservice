#!/bin/bash

# 1. УСТАНОВКА ЗАВИСИМОСТЕЙ (Ubuntu/Debian)
echo "📦 Установка зависимостей..."
sudo apt update
sudo apt install -y docker.io curl wget

# Установка Go 1.24 (минимально требуемая)
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Установка Minikube и kubectl
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
rm minikube-linux-amd64

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
rm kubectl

# 2. ЗАПУСК MINIKUBE
echo "🚀 Запуск Minikube..."
minikube start --cpus=4 --memory=8g --driver=docker
minikube addons enable metrics-server
minikube addons enable ingress
eval $(minikube docker-env)

# 3. СБОРКА DOCKER ОБРАЗА
echo "🐳 Сборка Docker образа..."
# Исправь go.mod перед сборкой
sed -i 's/go 1.23.0/go 1.24/' go.mod 2>/dev/null || true
sed -i '/toolchain go1.24.1/d' go.mod 2>/dev/null || true

docker build -t go-microservice:latest .

# 4. РАЗВЕРТЫВАНИЕ В KUBERNETES
echo "☸️ Развертывание в K8s..."
kubectl create namespace iot-analytics --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f k8s/ -n iot-analytics

# 5. УСТАНОВКА MONITORING STACK
echo "📊 Установка мониторинга..."
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring

# 6. ОЖИДАНИЕ ЗАПУСКА
echo "⏳ Ожидание запуска pod..."
sleep 30
kubectl wait --for=condition=ready pod -l app=go-microservice -n iot-analytics --timeout=120s

# 7. ПОРТ-ФОРВАРД
echo "🔗 Порт-форвард сервиса..."
kubectl port-forward -n iot-analytics svc/go-microservice 8080:80 &
PORT_FORWARD_PID=$!
sleep 3

# 8. ЗАПУСК НАГРУЗОЧНОГО ТЕСТА
echo "🔥 Запуск нагрузочного теста..."
kubectl apply -f k8s/load-test-job.yaml -n iot-analytics

# 9. МОНИТОРИНГ
echo "
✅ ГОТОВО!
=========================================
API сервис: http://localhost:8080/health
Нагрузка генерируется...

Для мониторинга выполни в новом окне:
watch -n 1 'kubectl get hpa,pods -n iot-analytics'

Для остановки: kill $PORT_FORWARD_PID && minikube stop
========================================="