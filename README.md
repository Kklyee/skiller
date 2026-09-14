# Skiller

Skiller is a CLI/TUI for controlling which installed AI skills are visible to coding agents.

## Project configuration

Place a `.skiller.toml` file in a project directory. Use one profile:

```toml
profile = "go-backend"
```

or define the project's desired skills directly:

```toml
skills = ["code-review", "tdd"]
```

Run `skiller sync` to apply the project configuration. To synchronize and
launch an installed coding agent, use `skiller run codex`, `skiller run
gemini`, or `skiller run opencode`.
