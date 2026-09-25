# lazyfile 技术栈

本文档记录 lazyfile 沿用 lazygit 工程思想后的技术基线。这里复用的是
“Go 原生终端应用、状态驱动 UI、明确分层、外部命令适配”的方法，而不是复制
lazygit 的全部业务代码或全部依赖。

## 1. 核心选型

| 层次 | 选型 | 说明 |
| --- | --- | --- |
| 主语言 | Go | 适合系统调用、文件系统 API、并发任务和单二进制发布。 |
| 终端渲染 | `github.com/gdamore/tcell/v3` | 负责屏幕绘制、颜色、Unicode 宽度、终端能力和键盘事件。 |
| UI 模型 | 状态驱动 + 事件循环 | 输入事件改变应用状态，再由渲染层根据状态刷新视图。 |
| 文件系统 | Go 标准库 `io/fs`、`os`、`path/filepath` | 目录遍历、元数据读取和基础文件操作优先不依赖外部命令。 |
| 外部命令 | `os/exec` 封装层 | Git、rsync、预览器等外部能力必须经过统一适配，不在 UI 中直接拼接命令。 |
| 配置 | XDG 目录约定 + YAML 或 JSON | 配置读取、默认值和持久化集中在配置包中。 |
| 测试 | Go `testing`，必要时使用 `testify` | 业务和命令构造逻辑可单测；TUI 通过状态转换和渲染模型测试。 |
| 格式化 | `gofumpt` | 统一 Go 源码格式。 |
| 质量检查 | `go vet` + `golangci-lint` | 提交前检查类型、常见错误和静态质量问题。 |

## 2. 对 lazygit 的继承与适配

### 继承

- 使用 Go 构建跨平台终端应用。
- 将 UI、应用状态、领域逻辑和命令执行分层。
- 通过事件驱动交互，而不是在输入处理函数中直接堆叠所有业务逻辑。
- 对长耗时操作使用后台任务，并通过消息或状态变更通知 UI。
- 保持核心功能可测试，避免让终端 I/O 渗透到所有业务代码。

### 适配

- lazyfile 的核心领域是目录、文件、选择集和文件任务，不是 Git 工作树。
- 基础文件操作使用 Go 标准库，只有同步、Git 和用户配置的外部能力才通过进程适配器接入。
- 任何删除、覆盖和批量修改操作都必须由领域层生成明确的操作计划，并由 UI 提供确认反馈。
- 预览、复制、移动和同步任务应支持取消、错误展示和进度更新。

## 3. 推荐目录结构

```text
cmd/lazyfile/              # 进程入口和 CLI 参数
internal/app/               # 应用状态、事件分发和生命周期
internal/ui/                # tcell 终端绘制、布局、输入映射
internal/filesystem/        # 目录读取、文件元数据和基础操作
internal/tasks/             # 复制、移动、删除、同步等后台任务
internal/commands/          # Git、rsync 等外部命令适配
internal/config/            # 配置文件、默认值和持久化
internal/preview/           # 文件预览和内容检测
internal/testutil/          # 测试夹具和虚拟文件系统辅助代码
```

UI 层只能依赖应用状态和领域接口；`filesystem`、`tasks` 和 `commands`
不能依赖具体 UI 控件。

## 4. 依赖边界

### 必选

- Go 标准库
- `gdamore/tcell/v3`

### 按需求引入

- `afero`：需要可替换文件系统或更丰富测试隔离时使用。
- `xdg`：需要统一处理各平台配置、缓存和数据目录时使用。
- `testify`：测试断言和 mock 场景复杂后使用。
- `golangci-lint`、`gofumpt`：作为开发工具，不作为运行时依赖。

### 不作为默认依赖

- Bubble Tea、gocui：本项目已固定为直接使用 tcell，不同时引入多个 TUI
  框架。
- shell 字符串拼接库：命令参数必须使用 `exec.CommandContext` 的参数列表，
  不能把用户输入拼接成 shell 命令。

## 5. 并发与错误处理

- 所有后台任务都必须绑定 `context.Context`，支持取消和超时。
- 任务错误必须回传到应用状态并显示给用户，不能静默吞掉。
- UI 刷新由主事件循环统一协调；后台 goroutine 不直接写终端。
- 文件操作前检查路径、权限、目标存在性和覆盖策略。
- 外部命令记录可诊断的命令名、退出状态和 stderr；不记录敏感环境变量。

## 6. 平台目标

首要目标为 Linux，随后兼容 macOS 和 Windows。平台差异集中在文件路径、
终端能力、打开文件命令和外部同步工具适配层，不扩散到 UI 和领域层。

## 7. 参考来源

- [lazygit README](https://github.com/jesseduffield/lazygit/blob/master/README.md)
- [lazygit go.mod](https://github.com/jesseduffield/lazygit/blob/master/go.mod)
- [tcell](https://github.com/gdamore/tcell)
