# Claude Code Requirements Builder

## Опис проєкту
Система для структурованого збору вимог через Claude Code. Автоматизує процес discovery через yes/no питання з розумними дефолтами.

## Основні компоненти
- `/requirements-start` - початок збору вимог
- `/requirements-status` - перевірка прогресу
- `/requirements-end` - завершення збору
- `/requirements-list` - список всіх вимог

## Структура файлів
```
requirements/
├── .current-requirement     # Активна вимога
├── index.md                # Зведення всіх вимог
└── YYYY-MM-DD-HHMM-name/  # Окремі вимоги
    ├── metadata.json
    ├── 00-initial-request.md
    ├── 01-discovery-questions.md
    ├── 02-discovery-answers.md
    ├── 03-context-findings.md
    ├── 04-detail-questions.md
    ├── 05-detail-answers.md
    └── 06-requirements-spec.md
```

## Робочий процес
1. Аналіз кодової бази
2. 5 yes/no питань для контексту
3. Автономний аналіз на основі відповідей
4. 5 експертних питань
5. Генерація детальної специфікації

## Зв'язок з Protocol Engineering
Цей проєкт демонструє структурований підхід до роботи з AI, що перегукується з ідеями Protocol Engineering - замість ad-hoc промптів використовуються чіткі протоколи та workflows.