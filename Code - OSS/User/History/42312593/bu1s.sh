#!/bin/bash
cd ~/code/notify-platform

check_port() {
    if lsof -i :$1 > /dev/null; then
        echo "Порт $1 уже занят! Остановите мешающий процесс и попробуйте снова."
        exit 1
    fi
}

ports=(8080 8081 8082 8083 8084 8085 3306 6379 5672 15672 9090 3000 8000)
for port in "${ports[@]}"; do
    check_port $port
done

echo "Deleting old logs..."
rm -rf ~/code/notify-platform/logs/*

echo "Starting infrastructure services..."
docker-compose -f deployments/docker-compose.yml up -d --build

echo "Waiting for services to initialize (30 seconds)..."
sleep 30

echo "All services started:"
echo "----------------------------------------------------------"
echo "Frontend:              http://localhost:8000"
echo "User Service:          http://localhost:8080"
echo "Notification Service:  http://localhost:8081"
echo "RabbitMQ Management:   http://localhost:15672 (guest/guest)"
echo "Prometheus:            http://localhost:9090"
echo "Grafana:               http://localhost:3000 (admin/admin)"
echo "----------------------------------------------------------"
echo "Logs: docker-compose logs -f"
echo "To stop: ./down.sh"
