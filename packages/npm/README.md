# skiller-cli

`skiller-cli` 是 Skiller 的 npm 安装包。它会根据当前电脑的系统和 CPU 架构下载对应的预编译二进制文件，因此安装 Skiller 不需要安装 Go。

```bash
npm install --global https://github.com/Kklyee/skiller/releases/download/v0.1.0/skiller-cli-0.1.0.tgz
skiller --version
skiller tui
```

如果包已经发布到 npm，也可以使用：

```bash
npm install --global skiller-cli
```

从 GitHub Release 安装不需要 npm 账号，只需要本机已经安装 Node.js 和 npm。

当前支持 Linux、macOS 和 Windows 的 x64、arm64。升级或卸载：

```bash
npm update --global skiller-cli
npm uninstall --global skiller-cli
```

该包只负责安装和启动 Skiller，Skill、Group、Profile 等数据仍然保存在 Skiller 的默认目录中。
