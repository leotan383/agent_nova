# Nova Studio 桌面端

跨平台桌面应用（Wails v2 + React），提供小说书库管理与完整创作工作室。

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

## 架构

```
desktop/
├── main.go          # Wails 入口
├── app.go           # Go ↔ 前端 bindings（书库、状态、索引）
├── write.go         # 写章任务 + 流式缓冲 + token 统计
├── plan.go          # 卷纲规划
├── review.go        # 章节审查
├── coach.go         # 改稿讨论
├── discover.go      # AI 探讨立项
└── frontend/        # React + Tailwind UI
    └── src/
        ├── pages/LibraryPage.tsx    # 书库
        └── pages/StudioPage.tsx     # 工作室（五 Tab）
```

## 工作室功能（StudioPage）

| Tab | 功能 |
|-----|------|
| **概览** | 进度、新手引导、健康待办、API 用量累计 |
| **规划** | 卷纲查看/编辑、AI 生成卷纲 |
| **章节** | AI 写章（流式）、阅读/编辑正文/审查/摘要、Coach 改稿、版本历史、选区快捷改写 |
| **记忆** | 长期记忆 CRUD、伏笔管理、冲突检测 |
| **设定** | Wiki 统一浏览设定集/大纲/实体 |

## Go Bindings（前端 `window.go.main.App`）

### 书库

| 方法 | 说明 |
|------|------|
| `ListNovels` | 书库列表 |
| `GetActiveNovel` / `SwitchNovel` | 当前书 / 切换 |
| `RegisterNovel` / `PickNovelDirectory` | 打开已有项目 |
| `CreateNovel` / `PickCreateDirectory` | 新建小说 |
| `RemoveFromLibrary` / `SetNovelPinned` / `SetNovelArchived` | 书库管理 |
| `RevealInFolder` | 系统文件管理器 |

### 创作数据

| 方法 | 说明 |
|------|------|
| `GetStatus` / `GetProjectHealth` | 创作状态 / 健康待办 |
| `ListChapters` / `GetChapterContent` | 章节列表与正文 |
| `GetChapterDocument` / `SaveChapterDocument` | 正文/审查/摘要读写 |
| `GetWriteContext` / `GetWriteGate` | 写章上下文与门禁 |
| `GetProjectTokenUsage` | 项目累计 LLM token 用量 |

### AI 工作流

| 方法 | 说明 |
|------|------|
| `StartDiscover` / `SendDiscoverMessage` / `FinishDiscover` / `CreateNovelFromDiscover` | AI 探讨立项 |
| `StartPlanVolume` / `CancelPlanVolume` | AI 生成卷纲 |
| `StartWriteChapter` / `CancelWriteChapter` | 流式写章 |
| `GetWriteJob` / `GetWriteJobState` / `GetActiveWriteJob` | 写章任务状态与流式缓冲恢复 |
| `StartReviewChapter` / `CancelReviewChapter` | 章节审查 |
| `SendChapterCoachMessage` / `StartChapterRevision` | 改稿讨论与修订 |
| `StartSelectionTransform` | 选区快捷改写 |

### 记忆与导出

| 方法 | 说明 |
|------|------|
| `ListMemories` / `CreateMemory` / `UpdateMemory` / `ArchiveMemory` | 记忆管理 |
| `ListForeshadows` / `ResolveForeshadow` / `UpdateForeshadow` | 伏笔管理 |
| `FindMemoryConflicts` / `MergeMemories` | 记忆冲突检测与合并 |
| `LearnFromFeedback` | 从反馈学习（nova learn） |
| `GetConsistencyReport` | 一致性仪表盘数据 |
| `RunProjectDoctor` / `RunPreflight` | 项目体检 / 写前预检 |
| `CreateProjectBackup` / `ListProjectBackups` / `RestoreProjectBackup` | 备份管理 |
| `BootstrapMemories` | 设定集→记忆回填 |
| `ExportProject` | 导出 Markdown / TXT / EPUB |
| `ListWikiEntries` / `GetWikiContent` / `SaveWikiContent` | Wiki 设定浏览 |

### 写章流式事件

前端通过 `window.runtime.EventsOn` 订阅：

| 事件 | payload 字段 | 说明 |
|------|----------------|------|
| `write:delta` | `job_id`, `chapter`, `delta` | 起草正文流式片段 |
| `write:step` | `job_id`, `chapter`, `step`, `message` | 流水线步骤 |
| `write:status` | `job_id`, `chapter`, `status`, `message` | pending / running / done / failed / cancelled |
| `write:done` | `job_id`, `chapter`, `report`, `batch_complete`, `batch_index`, `total_in_batch` | 完成报告（含 token_usage；批量写章时中间章 `batch_complete=false`） |
| `write:error` | `job_id`, `chapter`, `error` | 错误 |

切换 Tab 后可通过 `GetWriteJobState` / `GetActiveWriteJob` 恢复已输出的流式正文。

## CLI vs Desktop 能力对照

| 能力 | CLI | Desktop |
|------|-----|---------|
| 探讨立项 Discover | ✅ | ✅ |
| 初始化 / 创建小说 | ✅ | ✅ |
| 卷纲规划 | ✅ | ✅ |
| AI 写章（流式） | ✅ | ✅ |
| 章节审查 | ✅ | ✅ |
| Coach 改稿 | ✅ | ✅ |
| 记忆 / 伏笔管理 | ✅ | ✅ |
| 导出 EPUB | ✅ | ✅ |
| 全文搜索 | ✅ | ✅ |
| `nova learn` 反馈学记忆 | ✅ | ✅ MemoryPanel |
| `nova backup` / `doctor` / `preflight` | ✅ | ✅ Overview 项目工具 |
| `memory bootstrap` | ✅ | ✅ Overview 设定→记忆 |
| 批量连续写章 | ✅ | ✅ WritePanel 连续写到第 N 章 |
| 新手引导 | — | ✅ |
| Token 用量统计 | — | ✅ |

完整 CLI 命令见 [README](../README.md)。
