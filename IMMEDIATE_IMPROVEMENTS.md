# Негайні покращення для системи Requirements

## 🔥 Що можна зробити прямо зараз (без кодування)

### 1. Виправити існуючі проблеми

#### Оновити index.md
```bash
# Додати завершену activity-feed вимогу до індексу
echo "## Completed Requirements

### 2025-01-06-2109-activity-feed
- **Status**: Complete ✅
- **Summary**: Real-time activity feed with SSE
- **Spec**: [View](requirements/2025-01-06-2109-activity-feed/06-requirements-spec.md)
" > requirements/index.md
```

#### Створити helper scripts
```bash
# requirements-new.sh
#!/bin/bash
TIMESTAMP=$(date +%Y-%m-%d-%H%M)
SLUG=$(echo "$1" | tr ' ' '-' | tr '[:upper:]' '[:lower:]')
DIR="requirements/$TIMESTAMP-$SLUG"

mkdir -p "$DIR"
echo "$1" > "$DIR/00-initial-request.md"
echo "$DIR" > requirements/.current-requirement
echo "Created new requirement: $DIR"
```

### 2. Додаткові інструкції для Claude

#### `.claude/requirements-helper.md`
```markdown
# Requirements Helper Instructions

When working with requirements:

1. **Auto-create structure**: When user runs /requirements-start, automatically:
   - Create the timestamped folder
   - Generate all phase files
   - Update .current-requirement
   - Add entry to index.md

2. **Track implementation**: After generating spec, create:
   - 07-implementation-tasks.md with TODO checklist
   - 08-test-cases.md with test scenarios
   - 09-implementation-log.md for progress tracking

3. **Link to code**: In requirements spec, add:
   - Affected files section with exact paths
   - Code snippets showing integration points
   - Test file paths that need creation/update

4. **Generate extras**: Automatically create:
   - Draft PR description
   - Testing checklist
   - Documentation updates needed
```

### 3. Розширені команди (як Markdown інструкції)

#### `/requirements-checklist`
Генерує чеклісти для поточної вимоги:
```markdown
## Implementation Checklist
- [ ] Create feature branch
- [ ] Write unit tests (TDD)
- [ ] Implement core functionality
- [ ] Add integration tests
- [ ] Update documentation
- [ ] Run linters
- [ ] Check test coverage
- [ ] Create PR
```

#### `/requirements-test-plan`
Створює детальний план тестування:
```markdown
## Test Plan for [Requirement]

### Unit Tests
- [ ] Test case 1: [description]
- [ ] Test case 2: [description]

### Integration Tests
- [ ] Scenario 1: [description]
- [ ] Scenario 2: [description]

### Edge Cases
- [ ] Edge case 1: [description]
- [ ] Edge case 2: [description]
```

### 4. Git Integration (manual але структурований)

#### Naming Convention
```
feature/req-[YYYY-MM-DD]-[slug]
# Example: feature/req-2025-01-06-activity-feed
```

#### Commit Message Template
```
feat(req-[id]): implement [summary]

Requirements: requirements/[folder-name]/
Spec: requirements/[folder-name]/06-requirements-spec.md

- Implemented [component 1]
- Added tests for [feature]
- Updated documentation
```

### 5. Метрики та трекінг

#### `requirements/metrics.md`
```markdown
# Requirements Metrics

## Velocity
- Average time from start to spec: X days
- Average questions per requirement: Y
- Implementation success rate: Z%

## Current Sprint
- In Progress: [list]
- Blocked: [list]
- Completed: [list]
```

### 6. Шаблони для швидкого старту

#### `requirements/templates/`

**api-endpoint.yaml**
```yaml
type: api_endpoint
default_questions:
  - "Will this endpoint require authentication?"
  - "Should it support pagination?"
  - "Will it need rate limiting?"
  - "Should responses be cached?"
  - "Will it handle file uploads?"
```

**ui-component.yaml**
```yaml
type: ui_component  
default_questions:
  - "Will this component be reusable?"
  - "Should it support dark mode?"
  - "Will it need loading states?"
  - "Should it be accessible (ARIA)?"
  - "Will it work on mobile?"
```

### 7. VSCode Snippets

`.vscode/requirements.code-snippets`
```json
{
  "New Requirement": {
    "prefix": "req-new",
    "body": [
      "# Requirement: ${1:name}",
      "",
      "## Context",
      "${2:description}",
      "",
      "## Acceptance Criteria",
      "- [ ] ${3:criteria1}",
      "- [ ] ${4:criteria2}",
      "",
      "## Technical Notes",
      "${5:notes}"
    ]
  }
}
```

## 🚀 Наступні кроки

1. **Сьогодні**: Виправити index.md, створити helper scripts
2. **Цей тиждень**: Додати розширені інструкції для Claude
3. **Наступний тиждень**: Почати роботу над Go CLI
4. **Через 2 тижні**: Інтегрувати з git workflow

Ці покращення можна імплементувати поступово, одразу отримуючи користь від кожного кроку.