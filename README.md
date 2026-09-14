# Skiller

Skiller 是一个面向 AI 编程 Agent 的本地技能可见性管理工具，同时提供
CLI 和 TUI 两种使用方式。

它可以帮助你决定某个时间点哪些已安装的技能对 Agent 可见，并按照项目、
任务或工作方式快速切换这组技能。

## 为什么开发 Skiller

AI 编程 Agent 通常会从共享的 skills 目录中发现技能。技能数量较少时，
这种方式非常方便；但是随着技能库不断增长，会逐渐出现一些问题：

- 每个项目都会看到与当前工作无关的技能；
- 技能越来越多后，很难快速找到真正需要的技能；
- 从后端开发切换到文档、前端或其他任务时，需要反复手动移动文件；
- 为了暂时不用某个技能而删除并重新安装，会浪费时间，也可能丢失配置；
- 项目无法清晰、可重复地描述自己需要哪些技能。

Skiller 在已安装技能和 Agent 之间增加了一层可逆的可见性管理。技能仍然
保存在电脑上，Skiller 只负责控制它们位于 Agent 可以访问的 active 目录，
还是位于由 Skiller 管理的 disabled 目录中。

通过分组、Profile 和项目配置，你可以把原本需要手动移动文件的操作保存为
可重复使用的配置。

## Skiller 能做什么

Skiller 提供以下功能：

- 查看技能及其 `active`、`disabled`、`conflict`、`broken`、`invalid` 状态；
- 单独启用或禁用技能；
- 创建包含多个技能 ID 的分组；
- 使用分组、单独技能和排除项组合 Profile；
- 使用项目级 `.skiller.toml` 配置实现可重复的技能选择；
- 在实际修改前通过 dry-run 预览操作计划；
- 使用三栏 TUI 浏览分组、技能和详情；
- 使用 `doctor` 检查目录、权限、文件系统兼容性和未完成的事务；
- 同步项目技能配置后启动 Codex、Gemini CLI 或 OpenCode。

Skiller 只管理技能的可见性，不会修改技能的 `SKILL.md` 内容，也不会负责
安装技能。技能安装仍然使用对应 Agent 的安装器或包管理工具，安装完成后
再交给 Skiller 管理其可见性。

## 工作原理

一个有效的技能通常是一个包含 `SKILL.md` 的目录。默认情况下，Skiller
会扫描以下两个位置：

```text
~/.agents/skills/          Agent 可见的技能
~/.skiller/disabled/       已安装但对 Agent 隐藏的技能
```

启用或禁用技能时，Skiller 会在两个目录之间移动完整的技能目录。技能不会
被删除、重写，也不会被复制成另一份安装。

技能目录会被归类为以下状态：

| 状态 | 含义 |
| --- | --- |
| `active` | active 目录中存在可正常使用的技能副本。 |
| `disabled` | disabled 目录中存在可正常使用的技能副本。 |
| `conflict` | 两个目录中同时存在同一个技能 ID 的副本。 |
| `broken` | 找到了技能条目，但无法正常使用。 |
| `invalid` | 目录结构不符合预期的技能格式。 |

`Installed` 表示从两个目录中发现的技能记录总数。`Active` 和 `Disabled`
表示正常的单副本状态，冲突和其他目录问题会单独统计。

### 分组和 `All`

分组是由技能 ID 组成的元数据集合。创建或编辑分组不会移动文件；只有应用
分组时，Skiller 才会根据分组内容调整技能可见性：分组中的技能会被启用，
其他可管理的已安装技能会被禁用。

TUI 中还有一个虚拟的 `All` 分组。它始终可用，不对应磁盘上的分组文件，
表示所有已安装的技能。选中 `All` 并应用它，就可以恢复为“所有技能可见”
的状态。

如果 CLI 流程需要一个可以保存和复用的“全部技能”集合，可以创建一个普通
分组，并将需要的技能加入其中。

### Profile

Profile 是一个保存下来的技能选择方案。它最终需要的技能集合计算方式是：

```text
(直接列出的技能) + (分组中的技能) - (排除的技能)
```

因此，你可以复用一个较大的分组，同时针对某个工作流排除其中的个别技能。

## 安装

### 下载发布版本

可以从 [GitHub Releases](https://github.com/Kklyee/skiller/releases) 下载已
发布的版本和平台二进制文件。发布流程会构建 Linux、macOS 和 Windows 下的
amd64、arm64 版本。

### 使用 Go 安装

需要 Go 1.27 或更高版本：

```bash
go install github.com/Kklyee/skiller/cmd/skiller@latest
```

请确保 Go 的安装目录已经加入系统 `PATH`。

### 从源码构建

```bash
git clone https://github.com/Kklyee/skiller.git
cd skiller
go build -o skiller ./cmd/skiller
```

Windows 下如果希望生成 `.exe` 文件，可以使用：

```powershell
go build -o skiller.exe ./cmd/skiller
```

`run` 命令要求对应的 Agent 已经安装，并且其可执行文件位于系统 `PATH`
中。

## 快速开始

首先检查本地环境和技能目录：

```bash
skiller doctor
skiller status
skiller list
```

如果希望通过交互界面浏览和切换技能，可以启动 TUI：

```bash
skiller tui
```

最简单的单个技能操作如下：

```bash
skiller disable code-review
skiller enable code-review
```

`disable` 会让 Agent 看不到这个技能，但技能仍然保存在 disabled 目录中；
`enable` 会将它重新放回 active 目录。

## 常见使用方式

### 创建并应用分组

```bash
skiller group create backend
skiller group add backend code-review diagnosing-bugs tdd
skiller group show backend
skiller use backend --dry-run
skiller use backend
```

`--dry-run` 会显示计划启用、禁用的技能以及发现的问题。正常执行时，Skiller
会在移动文件前请求确认。

查看所有已保存的分组：

```bash
skiller group list
```

分组只是技能选择集合，不是嵌套目录，也不会复制技能。一个技能可以同时属于
多个分组。

### 创建并使用 Profile

```bash
skiller profile create go-backend
skiller profile edit go-backend --groups backend --skills tdd --exclude wizard
skiller profile show go-backend
skiller profile use go-backend --dry-run
skiller profile use go-backend
```

查看和删除 Profile：

```bash
skiller profile list
skiller profile delete go-backend
```

### 配置项目

在项目目录中创建 `.skiller.toml`。项目配置必须在 `profile` 和 `skills` 中
选择一个，不能同时配置，也不能两个都不配置。

使用 Profile：

```toml
profile = "go-backend"
```

或者直接列出项目需要的技能：

```toml
skills = ["code-review", "tdd"]
```

在项目目录中预览并应用配置：

```bash
skiller sync --dry-run
skiller sync
```

Skiller 会从当前目录开始向父目录查找最近的 `.skiller.toml`，因此在项目
子目录中执行命令时也会使用同一份项目配置。

### 同步配置并启动 Agent

如果希望在启动 Agent 前自动同步项目技能配置，可以使用：

```bash
skiller run codex
skiller run gemini
skiller run opencode
```

该命令会先显示同步计划；如果需要修改技能，会请求确认，完成同步后再启动
对应的 Agent 可执行文件。

## TUI 使用说明

启动界面：

```bash
skiller tui
```

主界面由三个面板组成：

- `Groups`：筛选要显示的技能。`All` 是虚拟的“全部已安装技能”选项；
- `Skills`：显示技能状态，也是唯一可以执行启用/禁用切换的面板；
- `Details`：展示当前技能的 name、description、status 和 groups，只用于
  查看信息，不参与焦点切换。

常用快捷键：

| 按键 | 操作 |
| --- | --- |
| `↑` / `↓`、`j` / `k` | 移动当前选择。 |
| `Tab` | 在可交互面板之间切换焦点。 |
| `Space` | Skills 面板获得焦点时，切换当前技能状态。 |
| `a` | 激活所有可见技能；再次按下则禁用所有可见技能。 |
| `/` | 搜索可见技能。 |
| `Enter` | Groups 面板进入分组管理，Skills 面板查看详情。 |
| `g` | 打开分组管理页面。 |
| `u` | 应用当前选中的分组，包括虚拟的 `All`。 |
| `d` | 执行诊断。 |
| `?` | 显示帮助。 |
| `q` | 退出。 |

焦点会影响操作范围：只有 Skills 面板获得焦点时，`Space` 和 `a` 才会修改
技能可见性。在 Groups 面板按 `Enter` 会进入分组管理，不会误操作技能状态。

## 诊断和故障恢复

执行：

```bash
skiller doctor
```

`doctor` 会检查配置的技能目录是否存在或可以创建、是否可写，以及文件系统
是否支持 Skiller 需要的移动操作。它还会扫描技能状态，并报告是否存在未完成
的事务日志。

批量修改会使用锁和事务日志。Skiller 会记录每一次移动，验证源目录已经消失、
目标目录已经出现；如果事务失败，会回滚已经完成的移动。它不会覆盖已经存在
的目标路径。

如果操作报告存在未完成事务或技能目录问题，请先使用 `doctor` 和 `list` 检查，
确认原因后再重试。除非已经确认没有移动操作正在进行，否则不要手动删除事务
日志。

Skiller 每次操作都会重新扫描 active 和 disabled 两个目录，因此可以和外部
技能安装器一起使用。如果安装器重新创建了某个技能的 active 副本，而 disabled
目录中仍然存在另一份，Skiller 会报告 `conflict`，并保留两份副本，不会静默
覆盖其中一份。

## 数据目录

默认路径和环境变量覆盖方式如下：

| 用途 | 默认路径 | 环境变量覆盖 |
| --- | --- | --- |
| Active 技能 | `~/.agents/skills` | `SKILLER_ACTIVE_DIR` |
| Disabled 技能 | `~/.skiller/disabled` | `SKILLER_DISABLED_DIR` |
| 分组 | `~/.skiller/groups` | `SKILLER_GROUPS_DIR` |
| Profile | `~/.skiller/profiles` | `SKILLER_PROFILES_DIR` |
| 事务日志 | `~/.skiller/transaction.json` | `SKILLER_TRANSACTION_JOURNAL` |
| 操作锁 | `~/.skiller/lock` | `SKILLER_LOCK` |

在 Windows 中，`~` 表示当前用户的主目录。active 和 disabled 目录最好位于
同一个磁盘卷中，以保证目录移动可靠；`doctor` 会检查这一点。

## 命令参考

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

使用以下命令查看详细帮助：

```bash
skiller --help
skiller <command> --help
```

查看构建版本信息：

```bash
skiller --version
skiller version
```

生成 Shell 补全脚本：

```bash
skiller completion bash
skiller completion zsh
skiller completion fish
skiller completion powershell
```

## 开发

Skiller 使用 Go 编写，TUI 基于 Bubble Tea，CLI 基于 Cobra。

完成代码修改后，请执行仓库要求的检查：

```bash
go fmt ./...
go vet ./...
go test ./...
```

项目目标 Go 版本为 1.27。仓库中的集成测试使用临时目录，不需要修改真实的
用户技能目录。
