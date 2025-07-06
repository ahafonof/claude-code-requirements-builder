# Requirements Protocol Best Practices

## Оптимізований протокол Requirements → Implementation

### Phase 0: Task Classification (NEW)
```
User: "Додай feature X"
AI: Classify complexity:
- Simple (< 2 hours) → Skip to Phase 3
- Medium (2-8 hours) → Start from Phase 1 
- Complex (> 8 hours) → Full requirements process
```

### Phase 1: Rapid Context Gathering (замість 5 discovery questions)
```
AI: "Відповідь на 3 ключові питання одним повідомленням:
1. Хто користувачі цієї функції?
2. Який основний workflow?
3. Які існуючі компоненти використати?"
```

### Phase 2: Requirements Specification (спрощено)
```markdown
## Quick Spec Template
**Problem**: [1-2 речення]
**Solution**: [Основні компоненти]
**Acceptance**: [3-5 критеріїв]
**Technical**: [Ключові рішення]
```

### Phase 3: Handoff to Implementation
```
AI: "Requirements готові. Запускаю Protocol Engineering:
- Посилання: requirements/{folder}/quick-spec.md
- План: [TDD approach з основними кроками]
- Memories: створено task-{name}-requirements"
```

### Phase 4: Continuous Implementation
```
Session 1: Backend (Protocol Engineering)
Session 2: Frontend (Protocol Engineering)
Session 3: Testing & Polish (Protocol Engineering)
```

## Best Practices Integration

### 1. Use Conditional Flows
```python
if task.complexity == "simple":
    skip_to_implementation()
elif task.has_ui():
    add_ux_requirements()
```

### 2. Parallel Information Gathering
```
Замість послідовних питань:
"Опишіть: 1) користувачів, 2) основний workflow, 3) інтеграції"
```

### 3. Template Library
```
- CRUD operations → crud-template.md
- API endpoints → api-template.md  
- UI components → ui-template.md
```

### 4. Smart Defaults
```
Питання: "Потрібна авторизація?"
Default: Так (безпечніше)
```

### 5. Context Preservation
```
Work-in-progress memory:
- Current requirements link
- Implementation progress
- Decisions made
```

## Metrics for Success
- Time to requirements: < 10 min for medium tasks
- Context switches: < 3 per task
- Requirements changes during implementation: < 20%