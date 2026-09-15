# Skiller

Skiller 是面向 AI 编程 Agent 的本地 Skill Environment Controller，提供 CLI 和
TUI 两种方式管理已安装 Skill 的工作环境。

它可以决定哪些 Skill 当前对 Agent 可用，哪些 Skill 暂时停用，并按分组、Profile
或项目配置快速切换。Skill 内容仍由原有安装器管理，Skiller 负责环境编排。

## 为什么开发

Skill 数量增多后，所有项目共用同一套环境容易变得混乱，也容易误启用或误删除。
Skiller 把环境选择保存为可重复的配置，让切换和恢复更简单。

## 核心能力

- 查看并切换 `active`、`disabled`、`conflict`、`broken`、`invalid` 状态；
- 使用 Group、Profile、项目配置组织 Skill 环境；
- 用 pin 保证关键 Skill 始终保持 active；
- 查看 Skill 来源、安装器、版本和实际位置；
- 通过 `npx skills` 桥接安装、更新和移除；
- 导出/导入环境，并用 `doctor` 检查问题。

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

## 快速开始

```bash
skiller doctor
skiller status
skiller tui
```

## 常用命令

启用、禁用和查看 Skill：

```bash
skiller enable <skill>
skiller disable <skill>
skiller info <skill>
```

管理分组：

```bash
skiller group create backend
skiller group add backend code-review tdd
skiller use backend
```

管理 Profile 和固定 Skill：

```bash
skiller profile create go-backend
skiller profile edit go-backend --groups backend --skills tdd
skiller profile use go-backend
skiller pin code-review
skiller unpin code-review
```

安装器桥接：

```bash
skiller install owner/repo --skill code-review
skiller update code-review
skiller remove code-review
```

导出、导入和项目同步：

```bash
skiller export environment.toml
skiller import environment.toml --install-missing
skiller sync
skiller sync --check
```

`sync --check` 只检查项目环境，发现漂移时返回退出码 `2`，适合 CI。

## 项目配置

在项目目录放置 `.skiller.toml`，选择 Profile 或直接列出 Skill：

```toml
profile = "go-backend"
include = ["research"]
exclude = ["frontend-design"]
```

也可以使用：

```toml
skills = ["code-review", "tdd"]
```

最终环境会叠加 pin，并排除 `exclude` 中的 Skill。

## TUI

```bash
skiller tui
```

主界面按 `Groups / Skills / Details` 展示环境；`p` 打开 Profiles，`o` 打开当前
Project，`:` 打开命令面板。

```text
↑↓ / jk    移动
Tab         切换 Groups / Skills 焦点
Space       切换当前 Skill
a           激活/禁用全部可见 Skill
x           标记 Skill
b           批量操作
/           搜索
g           Groups
u           使用当前 Group/Profile/Project
Enter       进入分组或查看 Details
?           帮助
q           退出
```

Details 面板为只读信息；只有 Skills 获得焦点时，`Space`、`a` 等状态操作才生效。

## 数据位置

| 数据 | 默认位置 |
| --- | --- |
| Active Skills | `~/.agents/skills` |
| Disabled Skills | `~/.skiller/disabled` |
| Groups | `~/.skiller/groups` |
| Profiles | `~/.skiller/profiles` |
| Pins | `~/.skiller/pins.toml` |
| Provenance | `~/.skiller/provenance.toml` |

各路径可通过对应的 `SKILLER_*` 环境变量覆盖。

## 开发检查

```bash
go fmt ./...
go vet ./...
go test ./...
```
