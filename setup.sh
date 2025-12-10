#!/bin/bash

set -e

echo "🔥 СТАРТ УСТАНОВКИ НА DEBIAN"

# 1. СИСТЕМНЫЕ ЗАВИСИМОСТИ
echo "📦 Обновление системы..."
sudo apt update && sudo apt install -y \
    curl \
    wget \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common

# 2. УСТАНОВКА DOCKER
echo "🐳 Установка Docker..."
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# 3. УСТАНОВКА GO 1.24
echo "⚙️ Установка Go 1.24..."
wget -q https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
rm go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin

# 4. УСТАНОВКА MINIKUBE И KUBECTL
echo "☸️ Установка Minikube..."
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
rm minikube-linux-amd64

echo "🔧 Установка kubectl..."
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
rm kubectl

# 5. ЗАПУСК MINIKUBE
echo "🚀 Запуск Minikube..."
sudo sysctl fs.protected_regular=0
minikube start --driver=docker --cpus=4 --memory=8g
minikube addons enable metrics-server
minikube addons enable ingress
eval $(minikube docker-env)

# 6. ИСПРАВЛЕНИЕ go.mod ДЛЯ DOCKER
echo "🔨 Исправление go.mod..."
sed -i 's/go 1.23.0/go 1.24/' go.mod 2>/dev/null || true
sed -i '/toolchain go1.24.1/d' go.mod 2>/dev/null || true

# 7. СБОРКА DOCKER ОБРАЗА
echo "🏗️ Сборка Docker образа..."
docker build -t go-microservice:latest .

# 8. РАЗВЕРТЫВАНИЕ В KUBERNETES
echo "📦 Развертывание в K8s..."
kubectl create namespace iot-analytics --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f k8s/ -n iot-analytics

# 9. ОЖИДАНИЕ ЗАПУСКА
echo "⏳ Ожидание запуска сервиса..."
sleep 30
if kubectl wait --for=condition=ready pod -l app=go-microservice -n iot-analytics --timeout=120s; then
    echo "✅ Сервис запущен"
else
    echo "⚠️ Сервис не запустился, проверяем логи..."
    kubectl get pods -n iot-analytics
    kubectl describe pods -l app=go-microservice -n iot-analytics
fi

# 10. ЗАПУСК ТЕСТА
echo "🔥 Запуск нагрузочного теста..."
kubectl apply -f k8s/load-test-job.yaml -n iot-analytics

# 11. ФИНАЛЬНЫЕ КОМАНДЫ
echo "
==================================================
✅ УСТАНОВКА ЗАВЕРШЕНА!
==================================================

Для мониторинга автоскейлинга выполните:
watch -n 1 'kubectl get hpa,pods -n iot-analytics'

Для проверки логов сервиса:
kubectl logs -f deployment/go-microservice -n iot-analytics

Для остановки всего:
minikube stop
=================================================="