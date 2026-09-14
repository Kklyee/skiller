# Skiller

Skiller 是一个面向 AI 编程 Agent 的本地 Skill Environment Controller，提供
CLI 和 TUI 两种使用方式。

它负责管理已安装 Skill 的工作环境：决定哪些 Skill 当前处于 Agent 可用的
active 环境，哪些 Skill 暂时放入 disabled 环境，并支持按分组、Profile、项目
配置和 pinned baseline 快速切换。

## 为什么开发

Skill 数量增长后，所有项目共用同一套 Skill 会让环境变得混乱。Skiller 把
Skill 环境选择保存成可重复的配置，减少手动移动目录、反复查找和误删 Skill
的操作。Skill 本身仍由原有安装器负责，Skiller 只管理本地工作环境。

## 核心能力

- 查看 Skill 的 `active`、`disabled`、`conflict`、`broken`、`invalid` 状态；
- 启用、禁用和批量切换 Skill；
- 创建分组并将分组应用为当前环境；
- 使用 Profile 组合分组、单独 Skill 和排除项；
- 使用项目目录中的 `.skiller.toml` 同步环境；
- 使用 pin 保证关键 Skill 在任何分组或项目同步后仍保持 active；
- 查看 Skill 的实际位置、目录/链接来源和 `SKILL.md` 路径；
- 通过 `doctor` 检查目录、事务和失效 pin；
- 使用 TUI 浏览 Groups、Skills 和 Details。

## 安装

需要 Go 1.27 或更高版本：

```bash
go install github.com/Kklyee/skiller/cmd/skiller@latest
```

也可以从源码构建：

```bash
git clone https://github.com/Kklyee/skiller.git
cd skiller
go build -o skiller ./cmd/skiller
```

请确保 Go 的安装目录已经加入系统 `PATH`。

## 快速开始

```bash
skiller doctor
skiller status
skiller list
skiller tui
```

单独管理 Skill：

```bash
skiller enable code-review
skiller disable code-review
```

## 分组

```bash
skiller group create backend
skiller group add backend code-review diagnosing-bugs tdd
skiller group list
skiller use backend --dry-run
skiller use backend
```

`--dry-run` 会先显示启用、禁用和问题项，正常执行时会请求确认。

## Profile

```bash
skiller profile create go-backend
skiller profile edit go-backend --groups backend --skills tdd --exclude wizard
skiller profile use go-backend --dry-run
skiller profile use go-backend
```

## Pin baseline

Pin 是始终需要保持 active 的 Skill。它会被自动加入 group、Profile 和项目
同步的最终环境中：

```bash
skiller pin code-review tdd
skiller pins
skiller unpin tdd
```

不存在的 Skill 也可以先 pin，`skiller doctor` 会报告失效 pin。

## Skill 来源

```bash
skiller provenance code-review
```

该命令用于确认 Skill 当前来自 active 还是 disabled 目录，以及它是普通目录、
符号链接还是 Windows junction。若检测到 active 和 disabled 两份副本，会同时
列出两份位置，方便处理外部安装器重新创建副本造成的冲突。

## 项目同步

在项目目录创建 `.skiller.toml`，选择一个 Profile 或直接列出 Skill：

```toml
profile = "go-backend"
```

或者：

```toml
skills = ["code-review", "tdd"]
```

在项目目录执行：

```bash
skiller sync --dry-run
skiller sync
```

## 启动 Agent

```bash
skiller run codex
skiller run gemini
skiller run opencode
skiller run codex --restore
```

`run` 会先根据当前项目配置同步 Skill 环境，再启动对应 Agent。Agent 需要
已经安装，并且可执行文件位于系统 `PATH` 中。使用 `--restore` 时，Skiller
会在 Agent 退出后恢复启动前的 Skill 环境，即使 Agent 返回错误也会执行恢复。

## TUI 快捷键

```text
↑↓ / jk    移动选择
Tab         切换 Groups / Skills 焦点
Space       切换当前 Skill（仅 Skills 焦点）
a           激活全部可见 Skill，再按一次则禁用非 pinned Skill
x           标记或取消标记当前 Skill
b           打开标记 Skill 的批量操作
c           清空标记
/           搜索
Enter       Groups 进入分组页，Skills 查看详情
g           打开分组管理
u           应用当前分组
d           运行 doctor
?           帮助
q           退出
```

Details 面板只展示当前 Skill 的 name、description、status、groups 和 pinned
状态，不参与焦点切换。批量操作支持启用、禁用以及加入或移出分组。

## 状态说明

| 状态 | 含义 |
| --- | --- |
| `active` | Skill 在 Agent 可用的 active 目录中。 |
| `disabled` | Skill 已安装但暂时不在 Agent 可用环境中。 |
| `conflict` | active 和 disabled 中同时存在同一 Skill。 |
| `broken` | Skill 条目存在，但链接目标不可用。 |
| `invalid` | Skill 目录结构不符合预期。 |

Skiller 不删除或修改 Skill 的 `SKILL.md`。批量切换使用事务和锁；如果发现
未完成事务或 conflict，请先运行 `skiller doctor`。

## 默认数据目录

| 用途 | 默认路径 | 环境变量 |
| --- | --- | --- |
| Active Skill | `~/.agents/skills` | `SKILLER_ACTIVE_DIR` |
| Disabled Skill | `~/.skiller/disabled` | `SKILLER_DISABLED_DIR` |
| Groups | `~/.skiller/groups` | `SKILLER_GROUPS_DIR` |
| Profiles | `~/.skiller/profiles` | `SKILLER_PROFILES_DIR` |
| Pins | `~/.skiller/pins.toml` | `SKILLER_PINS_FILE` |
| Transaction journal | `~/.skiller/transaction.json` | `SKILLER_TRANSACTION_JOURNAL` |
| Operation lock | `~/.skiller/lock` | `SKILLER_LOCK` |

## 常用命令

```text
skiller doctor
skiller list
skiller status
skiller enable <skill>
skiller disable <skill>
skiller provenance <skills...>
skiller pin <skills...>
skiller unpin <skills...>
skiller pins
skiller group <list|create|delete|show|add|remove>
skiller use <group> [--dry-run]
skiller profile <list|show|create|edit|delete|use>
skiller sync [--dry-run]
skiller run <codex|gemini|opencode> [--restore]
skiller tui
```

查看完整帮助：

```bash
skiller --help
skiller <command> --help
```

## 开发检查

```bash
go fmt ./...
go vet ./...
go test ./...
```
