# Лабораторная работа №5: Kubernetes — Самостоятельные задания

## О проекте

Выполненные задания для самостоятельной работы по лабораторной работе №5 «Основы Kubernetes».

| Задание | Описание |
|---------|----------|
| **Задание 1** | ConfigMap и Secret, подключённые к Deployment через `valueFrom` |
| **Задание 2** | Nginx Ingress Controller и Ingress ресурс с маршрутизацией |
| **Задание 3** | Prometheus мониторинг + метрики в backend (`/metrics`) |
| **Задание 4** | Horizontal Pod Autoscaler + генератор нагрузки |

---

## Структура проекта

```
lab5/
├── examples/
│   ├── frontend/               # Node.js Express frontend
│   │   ├── app.js
│   │   ├── package.json
│   │   ├── Dockerfile
│   │   └── public/index.html
│   └── backend/                # Go REST API backend
│       ├── main.go
│       ├── go.mod
│       └── Dockerfile
├── k8s-manifests/              # Базовые манифесты
│   ├── namespace.yaml
│   ├── frontend-deployment.yaml
│   ├── frontend-service.yaml
│   ├── backend-deployment.yaml
│   └── backend-service.yaml
├── k8s-selfwork/               # Задания для самостоятельной работы
│   ├── task1-configmap.yaml
│   ├── task1-secret.yaml
│   ├── task1-deployments-with-config.yaml
│   ├── task2-ingress.yaml
│   ├── task3-monitoring.yaml
│   ├── task3-backend-with-metrics.go
│   ├── task4-hpa.yaml
│   └── task4-load-test-job.yaml
└── README.md
```

---

## Требования

- Docker Desktop 4.15+ с включённым Kubernetes
- `kubectl` (входит в Docker Desktop)
- Kubernetes включён: Settings → Kubernetes → Enable Kubernetes

Проверить что кластер работает:

```bash
kubectl cluster-info
kubectl get nodes
# Должен быть узел docker-desktop в статусе Ready
```

---

## Быстрый старт

### 1. Собрать Docker-образы

```bash
# Frontend
cd examples/frontend
docker build -t k8s-frontend:1.0 .

# Backend (--no-cache чтобы не подхватить старый слой)
cd ../backend
docker build --no-cache -t k8s-backend:1.0 .

# Проверить
docker images | grep k8s
```

### 2. Развернуть базовую инфраструктуру

```bash
cd ../../

kubectl apply -f k8s-manifests/namespace.yaml
kubectl apply -f k8s-manifests/backend-deployment.yaml
kubectl apply -f k8s-manifests/backend-service.yaml
kubectl apply -f k8s-manifests/frontend-deployment.yaml
kubectl apply -f k8s-manifests/frontend-service.yaml
```

### 3. Дождаться готовности подов

```bash
kubectl get pods -n lab5 -w
# Ждать пока все STATUS = Running, READY = 1/1
# Выйти: Ctrl+C
```

### 4. Открыть приложение

```
http://localhost:30080
```

Или через port-forward:

```bash
kubectl port-forward service/frontend-service 8080:80 -n lab5
# http://localhost:8080
```

---

## Задание 1: ConfigMap и Secret

### Применить

```bash
kubectl apply -f k8s-selfwork/task1-configmap.yaml
kubectl apply -f k8s-selfwork/task1-secret.yaml
kubectl apply -f k8s-selfwork/task1-deployments-with-config.yaml
```

### Проверить

```bash
# Показывает откуда берутся переменные
kubectl set env deployment/backend-deployment -n lab5 --list

# Значения ConfigMap в открытом виде
kubectl get configmap app-config -n lab5 -o jsonpath='{.data}' | python3 -m json.tool

# Значение Secret (декодировать из base64)
kubectl get secret app-secret -n lab5 -o jsonpath='{.data.API_KEY}' | base64 --decode
```

Ожидаемый результат:

```
# APP_ENV from configmap app-config, key APP_ENV
# LOG_LEVEL from configmap app-config, key LOG_LEVEL
# API_KEY from secret app-secret, key API_KEY
```

```json
{
    "APP_ENV": "production",
    "BACKEND_URL": "http://backend-service:5000",
    "LOG_LEVEL": "info"
}
```

### Изменить значение Secret

```bash
# Закодировать новое значение
echo -n "new-secret-value" | base64

# Обновить Secret
kubectl edit secret app-secret -n lab5

# Перезапустить поды чтобы подхватили новое значение
kubectl rollout restart deployment/backend-deployment -n lab5
kubectl rollout restart deployment/frontend-deployment -n lab5
```

---

## Задание 2: Ingress

### Применить

```bash
# Установить Nginx Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Дождаться готовности контроллера
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

# Добавить запись в hosts
# macOS / Linux:
echo "127.0.0.1  k8s-lab.local" | sudo tee -a /etc/hosts
# Windows (PowerShell от администратора):
# Add-Content C:\Windows\System32\drivers\etc\hosts "127.0.0.1  k8s-lab.local"

# Применить Ingress
kubectl apply -f k8s-selfwork/task2-ingress.yaml
```

### Проверить

```bash
kubectl get ingress -n lab5

curl http://k8s-lab.local/
curl http://k8s-lab.local/api/info
```

---

## Задание 3: Мониторинг (Prometheus)

### Применить

```bash
# Добавить репозиторий и установить стек
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace

# Дождаться готовности (~2-3 минуты)
kubectl get pods -n monitoring -w

# Применить ServiceMonitor и алерты
kubectl apply -f k8s-selfwork/task3-monitoring.yaml
```

### Проверить

```bash
# Prometheus UI
kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring
# http://localhost:9090

# Grafana
kubectl port-forward svc/prometheus-grafana 3001:80 -n monitoring
# http://localhost:3001  логин: admin  пароль: prom-operator
```

Полезные PromQL-запросы:

```promql
# Количество запросов к backend
rate(backend_http_requests_total[5m])

# CPU по подам в namespace lab5
sum(rate(container_cpu_usage_seconds_total{namespace="lab5"}[5m])) by (pod)
```

---

## Задание 4: Horizontal Pod Autoscaler

### Применить

```bash
# Установить Metrics Server (в Docker Desktop не встроен)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Добавить флаг --kubelet-insecure-tls (обязательно для Docker Desktop)
kubectl patch deployment metrics-server -n kube-system \
  --type='json' \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

# Дождаться готовности
kubectl wait deployment metrics-server -n kube-system --for=condition=Available --timeout=60s

# Проверить
kubectl top nodes

# Применить HPA
kubectl apply -f k8s-selfwork/task4-hpa.yaml

# Проверить (TARGETS показывают текущее/пороговое значение CPU)
kubectl get hpa -n lab5
```

### Создать нагрузку и наблюдать масштабирование

Открыть два терминала:

```bash
# Терминал 1 — запустить генератор нагрузки
kubectl apply -f k8s-selfwork/task4-load-test-job.yaml

# Терминал 2 — наблюдать за HPA и подами
kubectl get hpa -n lab5 -w
kubectl get pods -n lab5 -w
```

```bash
# Остановить нагрузку и наблюдать scale-down (~2-3 минуты)
kubectl delete -f k8s-selfwork/task4-load-test-job.yaml
```

---

## Полезные команды

```bash
# Все ресурсы в namespace
kubectl get all -n lab5

# Статус rollout после обновления
kubectl rollout status deployment/backend-deployment -n lab5

# Логи пода
kubectl logs -n lab5 -l app=backend --tail=50

# Описание пода (события, ошибки)
kubectl describe pod -n lab5 <pod-name>

# Войти в контейнер
kubectl exec -it -n lab5 <pod-name> -- /bin/sh

# Переменные окружения deployment
kubectl set env deployment/backend-deployment -n lab5 --list
```

---

## Удаление

```bash
# Удалить всё одной командой
kubectl delete namespace lab5

# Удалить Ingress Controller
kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Удалить Prometheus
helm uninstall prometheus -n monitoring
kubectl delete namespace monitoring
```
