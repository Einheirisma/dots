#!/bin/bash

EMAIL="bestarokku@gufum.com"
PASSWORD="strongpassword"
NEW_PASSWORD="newpassword"
TELEGRAM_CHAT_ID="903968693"
WHATSAPP_NUMBER="79192492745"

# Функция для выполнения запросов
run_curl() {
    local description=$1
    local command=$2

    echo -e "\n\033[1;34m$description\033[0m"
    echo "Выполняем: $command"

    response=$(eval "$command")
    exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo -e "\033[1;31mОшибка выполнения запроса ($exit_code)\033[0m"
        return 1
    fi

    echo -e "\033[1;32mУспех! Ответ:\033[0m"
    echo "$response" | jq

    return 0
}

# 1. Регистрация пользователя
run_curl "1. Регистрация пользователя" \
"curl -k -s -X POST https://localhost:8443/register \
  -H \"Content-Type: application/json\" \
  -d '{\"email\":\"$EMAIL\", \"password\":\"$PASSWORD\"}'"

echo -e "\n\033[1;33mВведите токен подтверждения из письма:\033[0m"
read VERIFICATION_TOKEN

# 2. Подтверждение email
run_curl "2. Подтверждение email" \
"curl -k -s -X GET \"https://localhost:8443/verify-email?token=$VERIFICATION_TOKEN\""

# 3. Вход в систему
run_curl "3. Вход в систему" \
"curl -k -s -X POST https://localhost:8443/login \
  -H \"Content-Type: application/json\" \
  -d '{\"email\":\"$EMAIL\", \"password\":\"$PASSWORD\"}'"

ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token')
if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo -e "\033[1;31mОшибка: не удалось получить токен доступа\033[0m"
    exit 1
fi

echo -e "\n\033[1;35mТокен доступа: $ACCESS_TOKEN\033[0m"

# 4. Отправка уведомлений
## Email
run_curl "4.1 Отправка email" \
"curl -k -s -X POST https://localhost:8444/notify \
  -H \"Authorization: Bearer $ACCESS_TOKEN\" \
  -H \"Content-Type: application/json\" \
  -d '{\"channel\":\"email\", \"recipient\":\"$EMAIL\", \"subject\":\"Test\", \"body\":\"Hello from API\"}'"

## Telegram
run_curl "4.2 Отправка Telegram" \
"curl -k -s -X POST https://localhost:8444/notify \
  -H \"Authorization: Bearer $ACCESS_TOKEN\" \
  -H \"Content-Type: application/json\" \
  -d '{\"channel\":\"telegram\", \"recipient\":\"$TELEGRAM_CHAT_ID\", \"body\":\"Hello Telegram\"}'"

## WhatsApp
run_curl "4.3 Отправка WhatsApp" \
"curl -k -s -X POST https://localhost:8444/notify \
  -H \"Authorization: Bearer $ACCESS_TOKEN\" \
  -H \"Content-Type: application/json\" \
  -d '{\"channel\":\"whatsapp\", \"recipient\":\"$WHATSAPP_NUMBER\", \"body\":\"Hello WhatsApp\"}'"

# 5. Просмотр истории
run_curl "5. Просмотр истории уведомлений" \
"curl -k -s -H \"Authorization: Bearer $ACCESS_TOKEN\" https://localhost:8444/history?limit=5"

# 6. Просмотр статистики
run_curl "6. Просмотр статистики" \
"curl -k -s -H \"Authorization: Bearer $ACCESS_TOKEN\" https://localhost:8444/stats"

# 7. Восстановление пароля
## Запрос на сброс
run_curl "7.1 Запрос на сброс пароля" \
"curl -k -s -X POST https://localhost:8443/forgot-password \
  -H \"Content-Type: application/json\" \
  -d '{\"email\":\"$EMAIL\"}'"

echo -e "\n\033[1;33mВведите токен сброса из письма:\033[0m"
read RESET_TOKEN

## Сброс пароля
run_curl "7.2 Сброс пароля" \
"curl -k -s -X POST https://localhost:8443/reset-password \
  -H \"Content-Type: application/json\" \
  -d '{\"token\":\"$RESET_TOKEN\", \"password\":\"$NEW_PASSWORD\", \"confirm_password\":\"$NEW_PASSWORD\"}'"

# 8. Вход с новым паролем
run_curl "8. Вход с новым паролем" \
"curl -k -s -X POST https://localhost:8443/login \
  -H \"Content-Type: application/json\" \
  -d '{\"email\":\"$EMAIL\", \"password\":\"$NEW_PASSWORD\"}'"

ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token')
if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo -e "\033[1;31mОшибка: не удалось получить токен доступа после сброса пароля\033[0m"
    exit 1
fi

echo -e "\n\033[1;35mНовый токен доступа: $ACCESS_TOKEN\033[0m"

# 9. Проверка состояния сервисов
run_curl "9.1 Проверка User Service Health" \
"curl -k -s https://localhost:8443/health"

run_curl "9.2 Проверка Notification Service Health" \
"curl -k -s https://localhost:8444/health"

# 10. Проверка доставки уведомлений
echo -e "\n\033[1;33mОжидаем 20 секунд для обработки уведомлений...\033[0m"
sleep 20

run_curl "10. Проверка доставки уведомлений" \
"curl -k -s -H \"Authorization: Bearer $ACCESS_TOKEN\" https://localhost:8444/history?limit=3"

echo -e "\n\033[1;32mТестирование завершено!\033[0m"
echo "Для просмотра логов используйте: docker-compose logs -f"