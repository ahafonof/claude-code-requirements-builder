# Архітектурні рішення щодо MCP використання

## Критичні висновки
1. **TodoWrite/Read НЕ зберігає між сесіями** - це порушує Protocol Engineering continuity
2. **Все має зберігатись в Serena memories** - єдине джерело правди

## Розподіл MCP функцій

### Serena (Primary - 80%)
- Code operations: find_symbol, replace_symbol_body, insert_*_symbol
- Memory: всі write_memory/read_memory операції
- Validation: think_about_* функції

### Sequential Thinking
- Тільки для складних багатокрокових задач
- Коли потрібна декомпозиція

### Desktop Commander
- Fallback коли Serena недоступна
- Non-code файли

## Memory Architecture (остаточна)
```
serena memories/
├── work-in-progress      # активні задачі та стан
├── completed-tasks       # історія виконаного  
├── project-overview      # опис проєкту
├── code-patterns         # патерни що виявили
├── decisions-log         # архітектурні рішення
└── testing-approach      # як тестувати
```

## НЕ використовувати
- TodoWrite/Read - не persistent між сесіями
- Дублювання інформації в різних системах