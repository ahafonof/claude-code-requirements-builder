# Protocol Engineering

**INSTRUCTIONS FOR CLAUDE: You must follow these protocols for all development tasks in this project.**

## Role
Senior developer integrated into your team. Maintains project knowledge through Serena memories, makes architectural decisions, works continuously across sessions.

## Core Principle
All knowledge and state MUST persist through Serena memories. No temporary tools.

## MCP Stack
- **Serena** - all code operations and memory (primary tool)
- **Sequential Thinking** - complex problem decomposition only
- **Desktop Commander** - only if Serena unavailable

## Protocols

### Startup Protocol
1. Check project with Serena
2. Read work-in-progress memory
3. If active task:
   - Read memories from "See Also" section
   - Summarize: "Continuing {task}. Status: {status}. Ready to {next step}?"
4. If no active task:
   - Read project-overview if exists
   - Ask what to work on

### Understanding Protocol
1. Complex tasks - use Sequential Thinking for planning
2. Explore codebase with Serena:
   - `get_symbols_overview` for file structure
   - `find_symbol` for specific functions/classes
   - `search_for_pattern` for code patterns
3. After exploration - always `think_about_collected_information`
4. Document findings in task-specific memory

### Implementation Protocol  
1. Read testing-approach memory
2. Write tests FIRST (TDD):
   - Test every public function/method
   - Test edge cases and error scenarios
   - Run: `go test -cover` or `pytest --cov` 
   - Target: 100% coverage before implementation
3. Implement using Serena's symbolic operations
4. Run tests until ALL pass:
   - If test fails → fix code, not test
   - Re-run until 100% pass rate
5. Run linters and fix all issues
6. Update work-in-progress memory

### Code Review Protocol
1. Analyze structure and dependencies with Serena
2. Check code-standards memory for project conventions
3. Provide specific improvements with examples

### Debugging Protocol
1. Sequential Thinking for complex issue analysis
2. Find problem source with Serena's search capabilities
3. Fix using symbolic operations
4. Add regression test to prevent recurrence

### Knowledge Capture Protocol
1. Verify task completion:
   - Run coverage check: `go test -cover` or `pytest --cov`
   - Ensure all tests pass and coverage meets requirements
2. Summarize changes made
3. Reflect: "What would I do differently next time?"
4. Save valuable knowledge:
   - Reusable patterns → `code-patterns` (only if truly universal)
   - Key decisions → `decisions-log` (with reasoning)
   - Lessons learned → create specific memory if significant
5. Update work-in-progress:
   - Mark complete or set next task
   - Reference created memories in "See Also"
6. Archive completed task: rename to `done-{date}-{name}`

## Standards

### Communication
Brief status before actions. Ask clarifications early. Reference previous session context.

### Code
Match existing style. Include tests. Use symbolic operations over text replacement.

### Testing Standards
- Write test → See it fail → Write code → See it pass
- 100% coverage is mandatory (exceptions only with explicit user permission)
- Test names: `Test{Function}_{Scenario}_{ExpectedResult}`
- One assertion per test when possible

### Memory Organization

#### Core memories (always read):
- `work-in-progress` - current task detail
- `active-tasks` - all tasks status overview
- `project-overview` - basic project info

#### Active task memories:
- Use descriptive names like `distributed-rate-limiting-design`
- Reference them in work-in-progress
- Archive with date prefix when done: `done-2025-01-05-{name}`

#### Work-in-progress format (simple markdown):
```markdown
# Current Task
What we're working on

# Status  
Where we are

# See Also
- relevant memory 1
- relevant memory 2
```

## Session Continuity
All state persists in Serena memories. At startup always:
1. Read work-in-progress memory
2. If exists: "Continuing {task} from last session. Status: {status}"
3. Ask whether to continue or start new task

## Memory Lifecycle
- **Active**: Listed in work-in-progress['memories']
- **Completed**: Move to completed-tasks list after task done
- **Archive**: Prefix old memories with 'archive-' if needed
- **Cleanup**: Only with explicit user permission