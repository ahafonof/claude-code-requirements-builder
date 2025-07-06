# План тестування Protocol Engineering

## Тестова задача
"Add rate limiting to API endpoints. Maximum 100 requests per minute per IP."

## Test 1: Session Continuity
1. Сесія 1: 
   - Почати задачу
   - Зберегти прогрес в work-in-progress memory
   - Закрити Claude

2. Сесія 2:
   - Перевірити чи AI згадає де зупинився
   - Очікуваний результат: "Continuing rate limiting implementation..."

## Test 2: Protocol Following
Перевірити чи AI:
1. Читає memories на початку (project-overview, work-in-progress)
2. Шукає існуючі middleware patterns через find_symbol
3. Пише тести першими (TDD)
4. Використовує символічні операції для змін
5. Зберігає нові patterns в memory

## Test 3: Proactiveness
- Чи скаже що продовжує з минулої сесії
- Чи запропонує наступні кроки
- Чи використає контекст з memories

## Метрики успіху
- Час на контекст: < 1 хв
- Правильність з першого разу: > 90%
- Використання memories: 100% задач
- Символічні операції: > 80% едитів

## Команда для старту тесту
```
serena.activate_project("your-go-project")
Add rate limiting to API endpoints. Maximum 100 requests per minute per IP.
```