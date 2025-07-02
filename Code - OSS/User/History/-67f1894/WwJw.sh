#!/bin/bash
cd ~/code/notify-platform
echo "Stopping all services..."
docker-compose -f deployments/docker-compose.yml down
echo "All services stopped."