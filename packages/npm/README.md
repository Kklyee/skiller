# skiller-cli

`skiller-cli` 是 Skiller 的 npm 安装包。它会根据当前电脑的系统和 CPU 架构下载对应的预编译二进制文件，因此安装 Skiller 不需要安装 Go。

```bash
npm install --global skiller-cli
skiller --version
skiller tui
```

当前支持 Linux、macOS 和 Windows 的 x64、arm64。升级或卸载：

```bash
npm update --global skiller-cli
npm uninstall --global skiller-cli
```

该包只负责安装和启动 Skiller，Skill、Group、Profile 等数据仍然保存在 Skiller 的默认目录中。
