# Skiller

Skiller 是面向 AI 编程 Agent 的本地 Skill Environment Controller，提供 CLI 和
TUI 两种方式管理已安装 Skill 的工作环境。

它可以决定哪些 Skill 当前对 Agent 可用，哪些 Skill 暂时停用，并按分组、Profile
快速切换。Skill 内容仍由原有安装器管理，Skiller 负责环境编排。

## 为什么开发

Skill 数量增多后，所有项目共用同一套环境容易变得混乱，也容易误启用或误删除。
Skiller 把环境选择保存为可重复的配置，让切换和恢复更简单。

## 核心能力

- 查看并切换 `active`、`disabled`、`conflict`、`broken`、`invalid` 状态；
- 使用 Group、Profile 组织 Skill 环境；
- 用 pin 保证关键 Skill 始终保持 active；
- 永久删除本机不再需要的 Skill 目录，并自动清理相关引用；
- 查看 Skill 来源、安装器、版本和实际位置；
- 通过 `npx skills` 桥接安装、更新和移除；
- 用 `doctor` 检查问题。

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

永久删除 Skill：

```bash
skiller delete
```

`delete` 会先列出当前电脑上真实存在的全部 Skill，并显示 `active`、`disabled` 和 `pinned`
状态。输入编号即可单选或用逗号多选，例如 `1,3,5`；确认后会删除对应的 Skill 目录，并同步
清理 pin、Group、Profile 和来源记录。这是不可逆操作；也可以用 `--yes` 跳过确认，例如
`skiller delete --yes code-review`，命令仍会先打印本机安装清单并校验 Skill 是否存在。

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

`pin` 只接受当前电脑上已安装的 Skill。省略 Skill ID 执行 `skiller pin` 时，工具会列出尚未 pin 的本机 Skill，输入编号并用逗号分隔即可多选，例如输入 `1,3,5`；也可以输入 `a` 全选或 `q` 取消。省略参数执行 `skiller unpin` 时，工具会列出当前已经 pin 的 Skill，并显示其状态，方便选择移除。

安装器桥接：

```bash
skiller install owner/repo --skill code-review
skiller update code-review
skiller remove code-review
```

## TUI

```bash
skiller tui
```

主界面按 `Groups / Skills / Details` 展示环境；`p` 打开 Profiles，`:` 打开命令面板。顶部的
`Group` 或 `Profile` 表示最后一次应用的目标，
不是当前光标所在的项目。

```text
↑↓ / jk    移动
Tab         切换 Groups / Skills 焦点
Space       切换当前 Skill
a           激活/禁用全部可见 Skill
x           标记 Skill
b           批量操作
Delete      永久删除当前或已标记的 Skill
/           搜索
g           Groups
u           预览并应用当前 Group/Profile
Enter       进入分组或查看 Details
?           帮助
q           退出
```

Details 面板为只读信息；只有 Skills 获得焦点时，`Space`、`a` 等状态操作才生效。

### 环境状态

Skill 的 `active` 和 `disabled` 是电脑上的实际状态。Group 和 Profile 是目标环境：

- `●` 表示目标已应用，实际 Skill 状态与目标一致；
- `◐` 表示目标仍然是当前目标，但实际状态被手动修改过；
- `!` 表示目标缺少 Skill、Group 或配置存在问题；
- `○` 表示该目标当前没有被应用。

手动启用或禁用 Skill 不会悄悄切换当前目标，只会让目标进入 `Modified` 状态。再次使用目标
即可重新同步。当前目标保存在 `~/.skiller/state.toml`，因此重新打开 TUI 后仍能识别最后使用的
Group 或 Profile。

## 数据位置

| 数据 | 默认位置 |
| --- | --- |
| Active Skills | `~/.agents/skills` |
| Disabled Skills | `~/.skiller/disabled` |
| Groups | `~/.skiller/groups` |
| Profiles | `~/.skiller/profiles` |
| Pins | `~/.skiller/pins.toml` |
| Provenance | `~/.skiller/provenance.toml` |
| Applied environment | `~/.skiller/state.toml` |

各路径可通过对应的 `SKILLER_*` 环境变量覆盖。

## 开发检查

```bash
go fmt ./...
go vet ./...
go test ./...
```
