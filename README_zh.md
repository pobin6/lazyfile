# lazyfile

lazyfile 是一个使用 Go 编写的终端文件管理器，提供三栏式文件浏览界面，用于管理目录集合、浏览和预览文件，并执行常见文件操作。

## 功能

- 将已保存的目录条目整理到不同集合中。
- 使用三栏浏览集合/目录条目、文件和预览内容。
- 使用 `h` 和 `l` 在当前目录条目的根目录范围内返回上级或进入选中的子目录。
- 预览文本文件（最多读取 1 KiB）和目录内容。
- 创建文件和目录；删除前需要确认。
- 复制或剪切文件和目录，支持多项选择。
- 在底部窗格查看文件元数据、操作日志和当前复制/剪切选择。
- 最底部无边框行会根据当前焦点显示快捷键，内容过长时会按终端宽度截断。
- 持久化集合、目录条目以及每个条目的当前浏览路径。

## 环境要求

- Go 1.27 或更高版本
- Linux 或 Windows 10/11
- 支持 Unicode 方框字符和颜色的终端；Windows 推荐使用 Windows Terminal

## 构建和运行

Linux：

```sh
go build -o lazyfile ./cmd/lazyfile
./lazyfile
```

Windows PowerShell：

```powershell
go build -o lazyfile.exe ./cmd/lazyfile
.\lazyfile.exe
```

也可以在任一平台直接运行：

```sh
go run ./cmd/lazyfile
```

程序启动后会恢复已保存的集合和浏览状态。若要从空配置开始，请移动或删除下文所述的配置文件。

## 快捷键

| 按键 | 操作 |
| --- | --- |
| `q` | 退出 |
| `0` | 聚焦顶部路径栏 |
| `1` | 聚焦集合/目录条目栏 |
| `2` | 聚焦 Files 栏 |
| `3` | 聚焦 Preview 栏 |
| `[` / `]` | 在集合页和目录条目页之间切换 |
| `j` / `k` | 在当前焦点窗格中向下/向上移动 |
| `/` | 在当前焦点的第一列或 Files 列搜索，输入时自动匹配 |
| `n` / `N` | 跳转到下一个/上一个匹配项 |
| `Enter` / `Esc` | 搜索结束并选中首个匹配项/清除搜索 |
| `h` | 在 Files 栏返回上级目录 |
| `Enter` / `l` | Files 栏选中目录时进入下一级；选中文件时使用系统默认程序打开 |
| `a` | 根据焦点窗格新增集合、目录条目、文件或目录 |
| `e` | 修改选中的目录条目 |
| `d` | 删除选中的目录条目或 Files 项目；删除前需确认 |
| `y` / `x` | 选择当前 Files 项目作为复制/剪切源 |
| `Ctrl+y` / `Ctrl+x` | 将当前项目追加到复制/剪切选择 |
| `Shift+y` / `Shift+x` | 范围选择复制/剪切项目 |
| `p` | 将所选项目粘贴到当前 Files 目录 |
| `Enter` | Files 栏进入目录或打开文件；其他场景确认输入、删除操作或搜索 |
| `Esc` | 取消对话框；没有对话框时清除 Files 栏选中项 |
| `Ctrl+c` | 退出 |
| `Backspace` | 删除输入内容中的最后一个字符 |

在 Files 栏创建项目时，名称不含路径分隔符会创建空文件；名称包含 `/` 会创建目录路径，Windows 也接受 `\`。

Linux 上打开文件需要安装 `xdg-open`（通常由 `xdg-utils` 提供）。

## 配置文件

lazyfile 会将集合、目录条目和浏览状态保存到：

```text
%AppData%\lazyfile\entries.json
```

（Windows）或：

```text
$XDG_CONFIG_HOME/lazyfile/entries.json
```

如果 Linux 未设置 `XDG_CONFIG_HOME`，程序会使用操作系统默认的用户配置目录（通常为 `~/.config/lazyfile/entries.json`）。程序通过 Go 的 `os.UserConfigDir` 获取平台标准配置目录。

## 开发

运行测试、静态检查和构建：

```sh
go test ./...
go vet ./...
go build ./...
```

CI 会在 Linux 和 Windows 上运行这些检查。Windows 交互冒烟测试请在 Windows Terminal 中启动程序，并检查键盘输入、终端缩放/重绘、文件操作以及退出时终端状态是否正常恢复。

## 许可证

lazyfile 使用 Apache License 2.0，详见 [LICENSE](LICENSE)。
