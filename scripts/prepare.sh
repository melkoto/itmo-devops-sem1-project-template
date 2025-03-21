#!/bin/bash
set -e

echo "== Установка прав на выполнение для всех скриптов =="
chmod +x ./scripts/*.sh

echo "== Подготовка базы данных PostgreSQL =="
# Конфигурация (значения по умолчанию, если переменные окружения не заданы)
# Можно значение по умолчанию удалить. Для этого нужно указать пустую строку
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export DB_NAME="${DB_NAME:-project-sem-1}"
export DB_USER="${DB_USER:-validator}"
export DB_PASSWORD="${DB_PASSWORD:-val1dat0r}"

echo "== Установка зависимостей Go =="
go mod tidy

# Создаём таблицу в базе данных
echo "== Создание таблицы 'prices' =="
PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" -d "$DB_NAME" -c "
CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,           -- Автоматически увеличиваемый идентификатор
    created_at DATE NOT NULL,        -- Дата создания продукта
    name VARCHAR(255) NOT NULL,      -- Название продукта
    category VARCHAR(255) NOT NULL,  -- Категория продукта
    price DECIMAL(10, 2) NOT NULL    -- Цена продукта с точностью до 2 знаков после запятой
);"

echo "== Подготовка завершена! =="
