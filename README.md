# Skiller

Skiller is a local CLI/TUI skill visibility manager for AI coding agents.
It lets you decide which installed skills are visible to an agent at a given
time, then switch that selection for a project, task, or working style.

## Why Skiller exists

AI coding agents can discover skills from a shared skills directory. That is
convenient when there are only a few skills, but a growing toolkit creates a
different problem:

- every project sees skills that are unrelated to its current work;
- finding the right skill becomes harder as the collection grows;
- switching from a backend task to a documentation or frontend task requires
  repeated manual file operations;
- deleting and reinstalling skills is slow and risks losing configuration;
- a project has no clear, repeatable description of which skills it expects.

Skiller adds a reversible visibility layer between installed skills and the
agent. Skills remain on disk, while Skiller controls whether each skill is in
the agent's active directory or in a managed disabled directory. Groups,
profiles, and project configuration make those choices repeatable instead of
making them a sequence of manual moves.

## What it does

Skiller provides:

- inspection of installed skills and their active, disabled, conflict,
  broken, or invalid states;
- direct enable/disable operations for individual skills;
- groups for reusable collections of skill IDs;
- profiles that combine groups, individual skills, and exclusions;
- project-local `.skiller.toml` files for reproducible skill selection;
- dry-run plans before a visibility change is applied;
- a three-panel TUI for browsing groups, skills, and details;
- `doctor` checks for paths, permissions, filesystem compatibility, and
  interrupted transactions;
- optional synchronization followed by launching Codex, Gemini CLI, or
  OpenCode.

Skiller manages skill visibility and does not edit a skill's `SKILL.md`
contents. It also does not install skills; use the installer or package
manager appropriate for your agent, then let Skiller organize visibility.

## How the model works

A valid skill is a directory containing `SKILL.md`. By default, Skiller scans
two locations:

```text
~/.agents/skills/          visible to the agent
~/.skiller/disabled/       installed but hidden from the agent
```

Enabling and disabling a skill moves its complete directory between those two
locations. The skill is not deleted, rewritten, or copied into a second
installation.

The catalog reports one of these states:

| State | Meaning |
| --- | --- |
| `active` | The skill has a usable copy in the active directory. |
| `disabled` | The skill has a usable copy in the disabled directory. |
| `conflict` | Both locations contain a copy of the same skill ID. |
| `broken` | A discovered entry cannot be used normally. |
| `invalid` | The entry does not satisfy the expected skill structure. |

`Installed` is the number of skill records discovered across both locations.
`Active` and `Disabled` describe the normal single-copy states; conflicts and
other catalog problems are reported separately.

### Groups and `All`

A group is metadata containing a list of skill IDs. Creating or editing a
group does not move any files. Applying a group reconciles the filesystem so
that the group's skills are active and other manageable installed skills are
disabled.

The TUI also provides a virtual `All` group. It is always available, is not a
group file on disk, and represents every installed skill. Selecting `All` and
using it is the way to return to an “everything visible” setup. If a CLI
workflow needs a named reusable equivalent, create a normal group and add the
desired skills to it.

### Profiles

A profile is a named saved selection. Its desired set is calculated as:

```text
(skills listed directly) + (skills from its groups) - (excluded skills)
```

This makes it possible to share a broad group while excluding one skill for a
particular workflow.

## Installation

### Download a release

Published releases and platform binaries are available on the
[GitHub Releases page](https://github.com/Kklyee/skiller/releases). The
release workflow builds Linux, macOS, and Windows binaries for amd64 and
arm64.

### Install with Go

With Go 1.27 or newer:

```bash
go install github.com/Kklyee/skiller/cmd/skiller@latest
```

Make sure Go's install directory is on your `PATH`.

### Build from source

```bash
git clone https://github.com/Kklyee/skiller.git
cd skiller
go build -o skiller ./cmd/skiller
```

On Windows, use `go build -o skiller.exe ./cmd/skiller` if you want an
`.exe` output name.

The `run` command expects the selected agent executable to be installed and
available on `PATH`.

## Quick start

Start by checking the local setup and catalog:

```bash
skiller doctor
skiller status
skiller list
```

Open the interactive interface when you want to browse and switch selections:

```bash
skiller tui
```

The simplest reversible manual workflow is:

```bash
skiller disable code-review
skiller enable code-review
```

`disable` hides the skill from the agent while preserving it under the
disabled directory. `enable` makes it visible again.

## Common workflows

### Create and apply a group

```bash
skiller group create backend
skiller group add backend code-review diagnosing-bugs tdd
skiller group show backend
skiller use backend --dry-run
skiller use backend
```

The dry run prints the planned enables, disables, and any catalog issues. The
normal command asks for confirmation before moving anything. Use `group list`
to see all saved groups:

```bash
skiller group list
```

Groups are selections, not nested directories or copies of skills. A skill may
belong to more than one group.

### Save a profile

```bash
skiller profile create go-backend
skiller profile edit go-backend --groups backend --skills tdd --exclude wizard
skiller profile show go-backend
skiller profile use go-backend --dry-run
skiller profile use go-backend
```

Useful inspection commands are:

```bash
skiller profile list
skiller profile delete go-backend
```

### Configure a project

Create `.skiller.toml` in a project directory. A project configuration must
choose exactly one of `profile` or `skills`.

Use a profile:

```toml
profile = "go-backend"
```

Or list the project's desired skills directly:

```toml
skills = ["code-review", "tdd"]
```

From the project directory, preview and apply the selection:

```bash
skiller sync --dry-run
skiller sync
```

Skiller searches the current directory and its parents for the nearest
`.skiller.toml`, so the same project configuration works from a nested
directory.

### Synchronize and launch an agent

When the project configuration should be applied immediately before starting
an agent, use:

```bash
skiller run codex
skiller run gemini
skiller run opencode
```

The command shows the synchronization plan, asks for confirmation when a
change is needed, applies it, and then launches the selected executable.

## TUI

Launch the interface with:

```bash
skiller tui
```

The main screen has three panels:

- `Groups` filters the visible skill list. `All` is the virtual all-installed
  selection.
- `Skills` shows each skill's state and is the only panel where enable/disable
  toggles are performed.
- `Details` displays the selected skill's name, description, status, and
  groups. It is informational and does not take focus.

Common keys:

| Key | Action |
| --- | --- |
| `↑` / `↓`, `j` / `k` | Move the selection. |
| `Tab` | Switch focus between the interactive panels. |
| `Space` | Toggle the selected skill when the Skills panel is focused. |
| `a` | Activate all visible skills; press it again to disable all visible skills. |
| `/` | Search visible skills. |
| `Enter` | Open group management from Groups, or view details from Skills. |
| `g` | Open group management. |
| `u` | Apply the selected group, including the virtual `All` group. |
| `d` | Run diagnostics. |
| `?` | Show help. |
| `q` | Quit. |

Focus matters: `Space` and `a` only change skill visibility when the Skills
panel is focused. Focusing Groups and pressing `Enter` opens group management;
it does not toggle a skill.

## Diagnostics and recovery

Run:

```bash
skiller doctor
```

`doctor` checks that the configured directories exist or can be created, are
writable, and support the filesystem operations Skiller needs. It also scans
skill states and reports an unfinished transaction journal.

Bulk changes use a lock and a transaction journal. Skiller records each move,
verifies the source and destination, and rolls completed moves back when a
transaction fails. It refuses to overwrite an existing destination.

If an operation reports an unfinished transaction or catalog issue, stop and
inspect it with `doctor` and `list` before retrying. Do not manually delete a
journal unless you have confirmed that no move is still in progress.

Skiller rescans both locations for every operation, so external skill
installers remain compatible. If an installer recreates an active copy while a
disabled copy still exists, Skiller reports a conflict and preserves both
copies rather than silently choosing one.

## Data locations

The default paths are:

| Purpose | Default path | Environment override |
| --- | --- | --- |
| Active skills | `~/.agents/skills` | `SKILLER_ACTIVE_DIR` |
| Disabled skills | `~/.skiller/disabled` | `SKILLER_DISABLED_DIR` |
| Groups | `~/.skiller/groups` | `SKILLER_GROUPS_DIR` |
| Profiles | `~/.skiller/profiles` | `SKILLER_PROFILES_DIR` |
| Transaction journal | `~/.skiller/transaction.json` | `SKILLER_TRANSACTION_JOURNAL` |
| Operation lock | `~/.skiller/lock` | `SKILLER_LOCK` |

On Windows, `~` means the current user's home directory. The active and
disabled directories should be on the same volume so that directory moves are
reliable; `doctor` checks this requirement.

## Command reference

```text
skiller completion <bash|zsh|fish|powershell>
skiller disable <skill>
skiller doctor
skiller enable <skill>
skiller group <list|create|delete|show|add|remove>
skiller list
skiller profile <list|show|create|edit|delete|use>
skiller run <codex|gemini|opencode>
skiller status
skiller sync [--dry-run]
skiller tui
skiller use <group> [--dry-run]
skiller version
```

Use `skiller --help` or `skiller <command> --help` for command-specific
arguments and examples. Inspect build metadata with:

```bash
skiller --version
skiller version
```

Generate shell completion scripts with, for example:

```bash
skiller completion bash
skiller completion zsh
skiller completion fish
skiller completion powershell
```

## Development

Skiller is written in Go and uses Bubble Tea for the TUI and Cobra for the
CLI. After making a change, run the repository checks:

```bash
go fmt ./...
go vet ./...
go test ./...
```

The module targets Go 1.27. The repository's integration tests use temporary
directories and do not require modifying a real user skills directory.
