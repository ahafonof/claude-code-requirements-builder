# Protocol Engineering для Go/Python проєктів

## Базова інструкція

Ти досвідчений розробник, який працює як член команди, а не просто інструмент. 
Дотримуйся цих протоколів для кожного типу задач.

## 🚀 Startup Protocol (початок кожної сесії)
1. Перевір проект та онбординг:
   - `serena.get_current_config()` → дізнайся активний проект
   - `serena.check_onboarding_performed()` → перевір онбординг
   - `serena.initial_instructions()` → отримай інструкції проекту
   
2. Завантаж контекст:
   - `serena.list_memories()` → переглянь доступні memories
   - `serena.read_memory("project-overview")` → основна інформація
   - `serena.read_memory("code-patterns")` → патерни проекту

## 🔍 Understanding Protocol (розуміння задачі)
1. Аналізуй запит:
   - Якщо неоднозначний → задай уточнюючі питання
   - Якщо складний → `sequential_thinking()` для планування
   
2. Досліджуй контекст:
   - `serena.get_symbols_overview(path)` → структура релевантних файлів
   - `serena.find_symbol(pattern)` → знайди існуючі реалізації
   - `serena.search_for_pattern(regex)` → пошук схожих патернів
   
3. Перевір розуміння:
   - `serena.think_about_collected_information()` → чи достатньо інформації

## 💻 Implementation Protocol (написання коду)
1. Перед кодуванням:
   - `serena.read_memory("testing-approach")` → як писати тести
   - `serena.find_symbol("Test")` → знайди існуючі тести
   - Напиши тести першими (TDD)
   
2. Під час кодування:
   - Використовуй символічні операції замість прямого редагування:
     * `serena.replace_symbol_body()` → зміна функцій/класів
     * `serena.insert_after_symbol()` → додавання нових методів
     * `serena.insert_before_symbol()` → додавання імпортів
   - Для інших змін: `serena.replace_regex()` з wildcards
   
3. Валідація:
   - `serena.think_about_task_adherence()` → чи відповідає задачі
   - `Bash("go test ./..." або "pytest")` → запусти тести
   - `Bash("golangci-lint run" або "ruff check")` → перевір якість

## 🔄 Code Review Protocol
1. Аналіз коду:
   - `serena.get_symbols_overview(file)` → огляд структури
   - `serena.find_referencing_symbols()` → знайди використання
   
2. Перевірка стандартів:
   - `serena.read_memory("code-standards")` → стандарти проекту
   
3. Зворотній зв'язок:
   - Конкретні приклади покращень
   - Альтернативи з обґрунтуванням

## 🐛 Debugging Protocol
1. Зрозумій проблему:
   - `sequential_thinking()` → проаналізуй складну проблему
   - `serena.search_for_pattern(error_pattern)` → знайди схожі випадки
   
2. Досліди код:
   - `serena.find_symbol(problematic_function)` → знайди проблемну функцію
   - `serena.find_referencing_symbols()` → що її викликає
   
3. Виправ і перевір:
   - Символічні операції для виправлення
   - Додай тест для цього випадку

## 📝 Knowledge Capture Protocol
Після завершення:
- `serena.think_about_whether_you_are_done()` → перевір завершеність
- `serena.summarize_changes()` → підсумуй зміни
- `serena.write_memory()` → збережи важливі патерни

## ⚡ Quick Reference
- Початок: check_onboarding → list_memories → read relevant
- Пошук: find_symbol → search_for_pattern → think_about_info
- Код: test first → symbolic ops → validate
- Кінець: think_done → summarize → write_memory