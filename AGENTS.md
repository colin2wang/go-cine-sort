# Cine Sort — AI 代理项目指令

Wails v2.17.1 Windows 桌面应用（Go 1.27 后端 + Vue 3.5/TS/pnpm 前端），产物为单个便携式 exe。

## 命令

**不要主动编译、测试和打包，用户会自己做这些事情**；以下命令仅在用户明确要求时执行。

- Go 测试与检查：`go test ./...`、`go vet ./...`（仓库根执行）
- 前端类型检查：`pnpm -C frontend exec vue-tsc --noEmit`
- 完整构建：`.\build.ps1`（可 `-SkipInstall`），产出 `build/bin/cine-sort.exe`；图标单独由 `.\generate-icon.ps1` 生成，构建只校验其存在。
- 前端开发：`wails dev`（前后端联动）或 `pnpm -C frontend run dev`

## 架构

`app.go`（Wails 绑定）→ `internal/service` 流水线 → `scanner` 扫描清洗 → `douban`/`imdb` 抓取 → `organizer` 建目录/归档 → `store` SQLite 持久化；`internal/youtube` 提供「YouTube 归档」页的按上传者分类移动（`PlanYouTubeSort`/`StartYouTubeSort` 绑定，后台任务 + 进度事件）。`internal/netclient` 是所有外部 HTTP 的统一出口，经 `internal/cache`（go-cache，TTL 1h）记录并回放响应；`internal/model` 是前后端 JSON 契约的唯一来源；`frontend/wailsjs/go` 是生成的绑定代码且已提交，修改 Go 绑定后必须同步更新。

## 项目特有注意事项

- 配置、`cine_sort.db`、日志全部生成在**可执行程序同目录**（便携式，`config.Dir()` 基于 `os.Executable()`），不是 AppData；不要改回 `os.UserConfigDir()`。
- 清洗规则嵌入 `internal/config/defaults.yml`（go:embed）；用户 config.yml 可覆盖。Go 用 RE2：`\b`、`\w` 只识别 ASCII，涉及中文片段的规则已改写为 Unicode 等价形式，不得照搬 Python 正则。
- 新增配置字段需同步四处：`internal/config`（struct + defaults）、`frontend/src/types/index.ts`、`frontend/src/views/SettingsView.vue`（开关与 toggle 联合类型）、`frontend/wailsjs/go/models.ts`。
- `ConfigPath`/`DataDir` 为运行时解析（`yaml:"-"`），永不信任前端传来的路径。
- SQLite 用纯 Go 的 modernc.org/sqlite（无 CGO），Windows 可直接 `go build`。
- 全仓库 LF，`*.ps1` 强制 CRLF（.gitattributes）；`build.ps1` 调用 `pnpm.cmd` 而非 pnpm.ps1 shim（PowerShell 5.1 的 stderr 问题）。Wails CLI 固定 v2.17.1。

## 维护规则

项目结构、构建测试命令、架构边界、开发约定或本文件记录的事实发生变化时，必须在同一次改动中同步更新本文件，用当前事实替换过期表述；不逐次追加变更历史。事故复盘、性能对比、实验过程等详细材料放独立文档并在此留链接，仅在任务相关时读取。
