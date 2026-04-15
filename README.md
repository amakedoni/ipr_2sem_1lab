# Лабораторная работа №5: Kubernetes — Самостоятельные задания

## О проекте

Этот репозиторий содержит выполненные задания для самостоятельной работы по лабораторной работе №5 «Основы Kubernetes».

Реализованы все четыре задания:

| Задание | Описание | Файлы |
|---------|----------|-------|
| **Задание 1** | ConfigMap и Secret с обновлёнными Deployment | `k8s-selfwork/task1-*.yaml` |
| **Задание 2** | Nginx Ingress Controller и Ingress ресурс | `k8s-selfwork/task2-ingress.yaml` |
| **Задание 3** | Prometheus мониторинг + метрики в backend | `k8s-selfwork/task3-*.yaml`, `task3-*.go` |
| **Задание 4** | Horizontal Pod Autoscaler + генератор нагрузки | `k8s-selfwork/task4-*.yaml` |

---

## Структура проекта

```
lab5/
├── examples/
│   ├── frontend/          # Node.js Express frontend
│   │   ├── app.js
│   │   ├── package.json
│   │   ├── Dockerfile
│   │   └── public/
│   │       └── index.html
│   └── backend/           # Go REST API backend
│       ├── main.go
│       ├── go.mod
│       └── Dockerfile
├── k8s-manifests/         # Базовые манифесты (из основной части лабы)
│   ├── namespace.yaml
│   ├── frontend-deployment.yaml
│   ├── frontend-service.yaml
│   ├── backend-deployment.yaml
│   └── backend-service.yaml
├── k8s-selfwork/          # Задания для самостоятельной работы
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
- kubectl (входит в Docker Desktop)
- Базовые знания Docker и Kubernetes из лабораторных работ №1–4

---

## Быстрый старт

### 1. Убедиться, что Kubernetes работает

```bash
kubectl cluster-info
kubectl get nodes
# Должен быть узел docker-desktop в статусе Ready
```

### 2. Собрать Docker-образы

```bash
# Frontend
cd examples/frontend
docker build -t k8s-frontend:1.0 .

# Backend
cd ../backend
docker build -t k8s-backend:1.0 .

# Проверить
docker images | grep k8s
```

### 3. Развернуть базовую инфраструктуру

```bash
cd ../../

# Namespace
kubectl apply -f k8s-manifests/namespace.yaml

# Базовые deployments и services
kubectl apply -f k8s-manifests/backend-deployment.yaml
kubectl apply -f k8s-manifests/backend-service.yaml
kubectl apply -f k8s-manifests/frontend-deployment.yaml
kubectl apply -f k8s-manifests/frontend-service.yaml

# Проверить
kubectl get pods -n lab5
kubectl get services -n lab5
```

### 4. Открыть приложение

```
http://localhost:30080
```

Либо через port-forward:

```bash
kubectl port-forward service/frontend-service 8080:80 -n lab5
# http://localhost:8080
```

---

## Задание 1: ConfigMap и Secret

### Что сделано
- `task1-configmap.yaml` — ConfigMap с ключами `APP_ENV`, `LOG_LEVEL`, `BACKEND_URL`
- `task1-secret.yaml` — Secret с ключом `API_KEY` (base64-закодированное значение)
- `task1-deployments-with-config.yaml` — оба Deployment обновлены: значения подтягиваются через `valueFrom.configMapKeyRef` и `valueFrom.secretKeyRef`

### Как применить

```bash
# Создать ConfigMap и Secret
kubectl apply -f k8s-selfwork/task1-configmap.yaml
kubectl apply -f k8s-selfwork/task1-secret.yaml

# Обновить Deployment
kubectl apply -f k8s-selfwork/task1-deployments-with-config.yaml

# Проверить, что переменные попали в Pod
kubectl exec -n lab5 \
  $(kubectl get pod -n lab5 -l app=backend -o jsonpath='{.items[0].metadata.name}') \
  -- env | grep -E "APP_ENV|LOG_LEVEL|API_KEY"
```

### Как изменить значение Secret

```bash
# Закодировать новый ключ в base64
echo -n "new-secret-value" | base64

# Отредактировать Secret
kubectl edit secret app-secret -n lab5

# Перезапустить Pods для применения нового значения
kubectl rollout restart deployment/backend-deployment -n lab5
kubectl rollout restart deployment/frontend-deployment -n lab5
```

---

## Задание 2: Ingress

### Что сделано
- `task2-ingress.yaml` — Ingress ресурс, маршрутизирующий трафик:
  - `/` → `frontend-service:80`
  - `/api` → `backend-service:5000`

### Как применить

```bash
# Шаг 1: Установить Nginx Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Дождаться готовности (~1-2 минуты)
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

# Шаг 2: Добавить в hosts-файл
# Windows: C:\Windows\System32\drivers\etc\hosts
# macOS/Linux: /etc/hosts
# Добавить строку:
#   127.0.0.1  k8s-lab.local

# Шаг 3: Применить Ingress
kubectl apply -f k8s-selfwork/task2-ingress.yaml

# Шаг 4: Проверить
kubectl get ingress -n lab5
curl http://k8s-lab.local/
curl http://k8s-lab.local/api/info
```

---

## Задание 3: Мониторинг и логирование

### Что сделано
- `task3-monitoring.yaml` — `ServiceMonitor` (сбор метрик с backend и frontend) + `PrometheusRule` с алертами
- `task3-backend-with-metrics.go` — расширенная версия `main.go` с Prometheus-метриками (`/metrics` endpoint, счётчики запросов, гистограмма времени ответа)

### Как применить

```bash
# Шаг 1: Установить Prometheus через Helm
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace

# Шаг 2: Дождаться готовности
kubectl get pods -n monitoring -w

# Шаг 3: Применить ServiceMonitor
kubectl apply -f k8s-selfwork/task3-monitoring.yaml

# Шаг 4: Открыть Prometheus UI
kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring
# http://localhost:9090

# Шаг 5: Открыть Grafana
kubectl port-forward svc/prometheus-grafana 3001:80 -n monitoring
# http://localhost:3001  (логин: admin / пароль: prom-operator)
```

### Полезные PromQL запросы

```promql
# Количество запросов к backend
rate(backend_http_requests_total[5m])

# Среднее время ответа
rate(backend_http_request_duration_seconds_sum[5m])
  / rate(backend_http_request_duration_seconds_count[5m])

# CPU по namespace
sum(rate(container_cpu_usage_seconds_total{namespace="lab5"}[5m])) by (pod)
```

---

## Задание 4: Horizontal Pod Autoscaler

### Что сделано
- `task4-hpa.yaml` — HPA для backend (2–10 реплик, CPU threshold 50%) и frontend (2–6 реплик, CPU 60%)
- `task4-load-test-job.yaml` — Kubernetes Job с 5 параллельными воркерами, генерирующими нагрузку

### Как применить

```bash
# Убедиться, что Metrics Server работает
kubectl top nodes

# Применить HPA
kubectl apply -f k8s-selfwork/task4-hpa.yaml

# Проверить HPA (TARGETS должны отображаться)
kubectl get hpa -n lab5

# Запустить генератор нагрузки
kubectl apply -f k8s-selfwork/task4-load-test-job.yaml

# Наблюдать за масштабированием (в отдельных терминалах)
kubectl get hpa -n lab5 -w
kubectl get pods -n lab5 -w

# Остановить нагрузку и наблюдать за scale-down (~2-3 минуты)
kubectl delete -f k8s-selfwork/task4-load-test-job.yaml
```

---

## Полезные команды

```bash
# Все ресурсы в namespace lab5
kubectl get all -n lab5

# Логи конкретного Pod
kubectl logs -n lab5 <pod-name> --follow

# Описание ресурса (события, ошибки)
kubectl describe pod -n lab5 <pod-name>

# Войти в Pod
kubectl exec -it -n lab5 <pod-name> -- sh

# Удалить все ресурсы лабы
kubectl delete namespace lab5
```

---

## Удаление ресурсов

```bash
# Удалить только самостоятельные задания
kubectl delete -f k8s-selfwork/ --ignore-not-found

# Удалить весь namespace (включая базовые манифесты)
kubectl delete namespace lab5

# Удалить Ingress Controller
kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Удалить Prometheus
helm uninstall prometheus -n monitoring
kubectl delete namespace monitoring
```
