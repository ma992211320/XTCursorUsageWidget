# XTCursorUsageWidget

本机查看 Cursor 用量的开源客户端。

- **Windows**：XT-cursor 用量小工具（Go，GDI 自绘，当前 1.2.4）
- **macOS**：桌面应用 + 小组件（Swift / SwiftUI，macOS 14+）

成品安装包请到下载页获取，本仓库只含源码，不含服务端。

- 仓库：https://github.com/ma992211320/XTCursorUsageWidget
- 下载：https://cursor.kj1001.fun

## 目录

```
windows/    Windows 主程序与安装器源码
macos/      macOS App 与 Widget 源码
```

## Windows

需要 Go 1.24 或更高版本。可在 Windows 本机编译，也可在 Linux / macOS 交叉编译到 Windows amd64。

### 绿色版 / 主程序

在 `windows/` 目录：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui" -o "XT-cursor用量小工具.exe" .
```

图标与版本信息可用 [go-winres](https://github.com/tc-hib/go-winres) 根据 `windows/winres/winres.json` 生成 `rsrc_windows_amd64.syso`，放在源码目录后会自动链入。

### 安装程序

1. 先编译出 `XT-cursor用量小工具.exe`
2. 复制为 `windows/installer/payload.exe`
3. 将 `rsrc_windows_amd64.syso` 复制到 `windows/installer/`
4. 在 `windows/installer/` 执行：

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui" -o "../XT-cursor用量小工具-安装程序.exe" .
```

`payload.exe` 与 `.syso` 仅用于本地编译，不要提交。

Windows 客户端会向 `https://cursor.kj1001.fun` 发送在线心跳（仅设备编号和版本，不传 Cookie）。

界面图标来自 [Lucide](https://lucide.dev)（ISC License）。

## macOS

需要 macOS 14+ 与 Xcode。用 Xcode 打开 `macos/CursorUsageWidget.xcodeproj`，或按 `macos/project.yml` 用 XcodeGen 生成工程。

- 调试安装：`macos/install.sh`（写入 `/Applications/Cursor用量.app`）
- 签名并打 DMG：`macos/release.sh`（需本机已有 Developer ID 证书）

编译产物在 `macos/build/`、`macos/dist/`，已忽略，不要提交。

## 许可

MIT。Lucide 图标遵循其 ISC License。
