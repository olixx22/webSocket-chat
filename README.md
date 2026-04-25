# messenger-server

## Запуск

### 1. Применить миграции

```bash
go run ./cmd/migrator -config ./config/local.yaml
```

### 2. Запустить сервер

```bash
go run ./cmd/wsserver -config ./config/local.yaml
```

По умолчанию сервер слушает адрес из `config/local.yaml`, сейчас это `0.0.0.0:8080`.

## Аутентификация и идентификация

Полноценной auth-схемы в проекте пока нет.

Сейчас идентификация пользователя передаётся одним из способов:

- через cookie `userID`
- через query-параметр `user_id`
- через JSON-поле `user_id`

Это зависит от конкретного endpoint. Для локальной разработки это удобно, но для production такой подход нужно заменить на нормальную аутентификацию.

## Middleware

В проекте используются:

- logger middleware
- `isChatMember` — проверяет, что пользователь состоит в чате
- `isChatOwner` — проверяет, что пользователь является владельцем чата

`isChatOwner` навешан на:

- изменение чата
- удаление чата
- добавление участников
- удаление участников

`isChatMember` навешан на:

- получение участников чата
- получение истории сообщений чата
- подключение к WebSocket

## HTTP API

Ниже перечислены актуальные маршруты из [internal/app/websocket/app.go](/Users/mark/coding_projects/webSocket-chat/messenger-server/internal/app/websocket/app.go:108).

### Health

- `GET /health`

Пример ответа:

```json
{
  "status": "OK"
}
```

### Users

- `POST /users`
  Создать пользователя.

  Пример body:
  ```json
  {
    "user_id": "11111111-1111-1111-1111-111111111111",
    "username": "mark",
    "avatar_url": "https://example.com/avatar.png",
    "bio": "hello"
  }
  ```

- `GET /users/{userID}`
  Получить пользователя по id.

- `GET /users/by-name?username=mark`
  Получить пользователя по имени.

- `PUT /users/{userID}`
  Обновить пользователя.

- `DELETE /users/{userID}`
  Удалить пользователя.

### Chats

- `GET /chats?user_id={userID}`
  Получить все чаты пользователя.

- `GET /chats-with-messages`
  Получить чаты текущего пользователя с последним сообщением.
  Требует cookie `userID`.

- `POST /chats`
  Создать групповой чат.
  Требует cookie `userID` владельца.

  Пример body:
  ```json
  {
    "title": "team chat",
    "is_group": true
  }
  ```

- `POST /chats/private`
  Создать приватный чат.
  Требует cookie `userID` текущего пользователя.

  Пример body:
  ```json
  {
    "user_id": "22222222-2222-2222-2222-222222222222"
  }
  ```

- `PUT /chats/{chatID}`
  Изменить название чата.
  Требует, чтобы пользователь из cookie `userID` был владельцем чата.

- `DELETE /chats/{chatID}`
  Удалить чат.
  Требует владельца чата.

- `GET /chats/{chatID}/members?user_id={userID}`
  Получить список участников.
  Требует участия в чате.

- `POST /chats/{chatID}/members`
  Добавить участников.
  Требует владельца чата.

  Пример body:
  ```json
  {
    "members": [
      {
        "UserID": "33333333-3333-3333-3333-333333333333",
        "Role": "member"
      }
    ]
  }
  ```

- `DELETE /chats/{chatID}/members`
  Удалить участников.
  Требует владельца чата.

  Пример body:
  ```json
  {
    "user_ids": [
      "33333333-3333-3333-3333-333333333333"
    ]
  }
  ```

- `DELETE /chats/{chatID}/leave?user_id={userID}`
  Выйти из чата.

- `GET /chats/{chatID}/user-role`
  Получить роль пользователя в чате.
  В текущей реализации endpoint ждёт JSON body с `user_id`, несмотря на `GET`.

- `GET /chats/{chatID}/messages?user_id={userID}&limit=50&before=2026-04-25T12:00:00Z`
  Получить историю сообщений.
  Требует участия в чате.

### Messages

- `POST /messages`
  Создать сообщение.
  Требует cookie `userID`.
  В body передаётся `content` и `message_type`.
  В текущей реализации хэндлер также ожидает `chatID` в path-параметре, но маршрут зарегистрирован как `/messages`, без `{chatID}`. Это несоответствие в коде стоит поправить.

- `PUT /messages/{messageID}`
  Обновить текст сообщения.

- `DELETE /messages/{messageID}`
  Удалить сообщение.

- `POST /messages/{messageID}/mark-as-read`
  Отметить сообщение как прочитанное.

  Пример body:
  ```json
  {
    "user_id": "11111111-1111-1111-1111-111111111111"
  }
  ```

- `GET /messages/{messageID}/status`
  Получить статусы сообщения.

## WebSocket

Маршрут:

- `GET /ws?user_id={userID}&chat_id={chatID}`

Подключение доступно только участнику чата.

### Входящие события

Создание сообщения:

```json
{
  "type": "message.create",
  "content": "Привет",
  "message_type": "text"
}
```

Ping:

```json
{
  "type": "ping"
}
```

### Исходящие события

Созданное сообщение:

```json
{
  "type": "message.created",
  "message": {
    "id": "00000000-0000-0000-0000-000000000000",
    "chat_id": "00000000-0000-0000-0000-000000000000",
    "sender_id": "00000000-0000-0000-0000-000000000000",
    "content": "Привет",
    "type": "text",
    "created_at": "2026-04-25T12:00:00Z"
  }
}
```

Ответ на ping:

```json
{
  "type": "pong"
}
```

Ошибка:

```json
{
  "type": "error",
  "error": "content is required"
}
```

## Как проходит сообщение

1. Клиент получает `chatID` и `userID`.
2. Клиент подключается к `/ws?user_id=...&chat_id=...`.
3. Middleware проверяет, что пользователь состоит в чате.
4. Клиент отправляет `message.create`.
5. Сервер сохраняет сообщение в PostgreSQL.
6. Hub рассылает `message.created` всем активным подключениям этого чата.
7. Историю можно получить через `GET /chats/{chatID}/messages`.

## Известные особенности

- В проекте пока нет нормальной аутентификации.
- Часть endpoint использует cookie `userID`, часть — query/body `user_id`.
- `GET /chats/{chatID}/user-role` сейчас читает `user_id` из JSON body, что нетипично для `GET`.
- `POST /messages` сейчас зарегистрирован без `chatID` в пути, хотя хэндлер ожидает `chatID` как path-параметр.

## Что улучшить дальше

- ввести нормальную auth-схему
- унифицировать передачу текущего пользователя
- выровнять контракты HTTP endpoint
- добавить OpenAPI/Swagger
- покрыть handlers и middleware тестами
