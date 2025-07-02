#!/bin/bash

TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bestarokku@gufum.com","password":"a21w9n703"}' | jq -r .access_token)

if [[ "$TOKEN" == "null" || -z "$TOKEN" ]]; then
  echo "Ошибка авторизации: проверь email и пароль"
  exit 1
fi

for i in {1..50}; do
  echo "Отправка уведомления #$i"

  curl -s -X POST http://localhost:8081/notify \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "channel": "email",
      "recipient": "bestarokku@gufum.com",
      "subject": "Test '$i'",
      "body": "This is a test message '$i'"
    }' > /dev/null

  sleep 0.2
done

echo "Готово!"