# Serena Documentation Summary

## What is Serena
- Free & open-source coding agent toolkit
- Turns LLM into fully-featured coding agent
- Semantic code retrieval and editing via Language Server Protocol (LSP)
- Works with MCP (Model Context Protocol) or Agno framework

## Language Support
Out-of-the-box: Python, TypeScript/JS, PHP, Go, Rust, C#, Java, Elixir, Clojure, C/C++

## Key Features
1. **Semantic Analysis** - understands code structure, not just text
2. **Symbol-level Operations** - find/edit classes, functions, methods precisely
3. **Project Memory** - persistent knowledge between sessions
4. **Multiple Integration Options** - MCP server, Agno agent, custom frameworks

## Modes (Dynamic Switching)
- **interactive** - asks clarifications, step-by-step
- **editing** - can modify files  
- **planning** - analysis only, NO code generation
- **one-shot** - autonomous completion
- **onboarding** - initial project learning

## Contexts (Fixed at Startup)
- **desktop-app** - for Claude Desktop (default)
- **ide-assistant** - for IDEs (VSCode, Cursor, Cline)
- **agent** - for autonomous agents

## Important Tools
- `switch_modes` - dynamically change modes
- `find_symbol` - semantic search
- `replace_symbol_body` - safe symbol editing
- `get_symbols_overview` - file structure
- Memory operations (read/write/list)

## Mode Effects
1. Changes system prompt
2. Can exclude certain tools
3. Defines interaction style

## Configuration Hierarchy
1. ~/.serena/serena_config.yml - global
2. Command line args - per client
3. .serena/project.yml - per project
4. Active modes - dynamic