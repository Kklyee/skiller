# Skiller

Skiller is a CLI/TUI for controlling which installed AI skills are visible to coding agents.

Use `skiller --version` or `skiller version` to inspect the build version,
commit, and build date.

Generate completion scripts with `skiller completion bash`, `zsh`, `fish`, or
`powershell`.

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

Skiller rescans the active and disabled directories on every operation, so
external installers remain compatible. If an installer recreates an active
copy of a disabled skill, Skiller reports a conflict and preserves both
copies.
