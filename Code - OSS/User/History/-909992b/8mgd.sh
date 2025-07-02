#!/bin/bash

# Получаем токен
LOGIN_RESPONSE=$(curl -k -s -X POST https://localhost:8443/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bestarokku@gufum.com", "password":"a21w9n703"}')

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

# Функция для отправки запросов
send_request() {
  local phone="7919$(printf "%07d" $1)"
  local status_code=$(curl -k -o /dev/null -s -w "%{http_code}\n" \
    -X POST https://localhost:8444/notify \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"channel\":\"whatsapp\", \"recipient\":\"$phone\", \"body\":\"Rate limit test $1\"}")

  echo "Request $1: HTTP $status_code"
  if [ "$status_code" -eq 429 ]; then
    echo "Rate limit triggered on request $i"
  fi
}

# Отправляем 20 запросов подряд
echo "Sending 20 requests to test rate limiting:"
for i in {1..20}; do
  send_request $i
  sleep 0.1
done

# Проверяем аудит
echo "Audit:"
curl -k -H "Authorization: Bearer $ACCESS_TOKEN" https://localhost:8444/audit?limit=5