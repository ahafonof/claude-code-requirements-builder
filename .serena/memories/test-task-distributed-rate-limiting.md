# Test Task: Distributed Rate Limiting з Redis

## Завдання
Розшир існуючий rate limiter для роботи в distributed environment:
1. Використати Redis для синхронізації лічильників між серверами
2. Додати graceful degradation - якщо Redis недоступний, fallback на local
3. Додати метрики: кількість запитів, відхилень, латентність Redis
4. Створити endpoint /metrics для моніторингу

## Природний поділ на сесії

### Сесія 1: Research & Design
1. Дослідити існуючий rate limiter код
2. Вибрати Redis клієнт для Go (go-redis vs redigo)
3. Спроектувати архітектуру:
   - Redis data structures (sorted sets vs strings)
   - TTL стратегія
   - Fallback механізм
4. Написати design document
5. Створити тести для Redis інтеграції

**Точка зупинки**: Після design та тестів, перед implementation

### Сесія 2: Implementation & Metrics
1. Прочитати design з попередньої сесії
2. Імплементувати Redis rate limiter
3. Додати fallback логіку
4. Створити metrics collector
5. Додати /metrics endpoint
6. Інтеграційне тестування

## Перевірка Protocol Engineering

### Continuity Test
- Сесія 2 має почати з: "Continuing distributed rate limiting. Design completed, starting implementation..."
- Має використати decisions з сесії 1

### Memory Test  
Після сесії 1 має бути:
- work-in-progress з планом implementation
- decisions-log з вибором Redis структур
- code-patterns з Redis integration pattern

### Proactiveness Test
- В сесії 2 має сам запропонувати почати з implementation
- Має згадати дизайн рішення без нагадування