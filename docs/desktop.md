# Nova Studio 桌面端

跨平台桌面应用（Wails v2 + React），提供小说书库管理与创作工作室壳层。

## 环境要求

- Go 1.20+
- Node.js 18+
- [Wails CLI v2.7.1](https://wails.io/docs/gettingstarted/installation)（兼容 Go 1.20，勿用 v2.8+）

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.7.1
```

macOS 还需 Xcode Command Line Tools。

## 开发

```bash
cd desktop

# 安装前端依赖
cd frontend && npm install && cd ..

# 热重载开发（推荐）
wails dev
```

## 构建

```bash
cd desktop
wails build
```

产物位于 `desktop/build/bin/`。

仅构建前端：

```bash
cd desktop/frontend
npm run build
```

## 架构

```
desktop/
├── main.go          # Wails 入口
├── app.go           # Go ↔ 前端 bindings
├── wails.json
└── frontend/        # React + Tailwind UI
    └── src/
        ├── pages/LibraryPage.tsx   # 书库
        └── pages/StudioPage.tsx    # 工作室壳

internal/library/    # 书库 registry (~/.config/nova/library.json)
```

## Go Bindings（前端 window.go.main.App）

| 方法 | 说明 |
|------|------|
| `ListNovels` | 书库列表 |
| `GetActiveNovel` / `SwitchNovel` | 当前书 / 切换 |
| `RegisterNovel` / `PickNovelDirectory` | 打开已有项目 |
| `CreateNovel` / `PickCreateDirectory` | 新建小说 |
| `RemoveFromLibrary` / `SetNovelPinned` / `SetNovelArchived` | 书库管理 |
| `RevealInFolder` | 系统文件管理器 |
| `GetStatus` / `ListChapters` / `GetChapterContent` | 工作室数据 |
| `StartWriteChapter` / `CancelWriteChapter` | 流式写章（见下方事件） |
| `GetWriteJob` / `IsWriteRunning` | 任务状态 |

### 写章流式事件

前端通过 `window.runtime.EventsOn` 订阅：

| 事件 |  payload 字段 | 说明 |
|------|----------------|------|
| `write:delta` | `job_id`, `chapter`, `delta` | 起草正文流式片段 |
| `write:step` | `job_id`, `chapter`, `step`, `message` | 流水线步骤 |
| `write:status` | `job_id`, `chapter`, `status`, `message` | pending / running / done / failed / cancelled |
| `write:done` | `job_id`, `chapter`, `report` (JSON) | 完成报告 |
| `write:error` | `job_id`, `chapter`, `error` | 错误 |

## 后续迭代

- [ ] Discover 构思对话页
- [ ] 设定集 / 大纲 Markdown 编辑
- [ ] 审查、记忆、导出
