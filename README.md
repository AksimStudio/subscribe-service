## REST-сервис , запуск сервиса

```bash
git clone https://github.com/AksimStudio/subscribe-service.git
cd subscribe-service

# Сборка и запуск контейнеров
docker-compose up -d --build

# Просмотр логов
docker-compose logs -f app

# Просмотр логов базы данных
docker-compose logs -f postgres
