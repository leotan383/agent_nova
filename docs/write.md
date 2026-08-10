# Write 底层实现原理

本文说明 `nova write` 如何把「卷纲中的一章」变成「正文 + 摘要 + 索引/事实沉淀」。只覆盖 CLI 与 `internal/` 核心路径，不涉及桌面端。

核心代码：

| 模块 | 路径 |
|------|------|
| CLI 入口 | `cmd/nova/write.go` |
| 工作流 | `internal/workflows/write.go`（`WriteChapter`） |
| 润色抽取 | `internal/workflows/chapter_body.go` |
| 事实提取 | `internal/workflows/extract.go` |
| Gate / Ledger / 落盘 | `internal/pipeline/pipeline.go` |
| 上下文快照 | `internal/context/builder.go`、`recall.go` |
| Prompt | `internal/prompts/prompts.go` |
| Agent | `internal/agent/agent.go` |
| 工具绑定 | `internal/tools/registry.go` → `BindProject` |
| 版本快照 | `internal/version/version.go` |
| 路径布局 | `internal/project/chapter_layout.go` |

---

## 1. 问题定义

Write 不是单次「让模型写一段正文」，而是一条**多阶段流水线**：

1. 写前检查连续性前提（phase、卷纲、上一章摘要、索引）。
2. 用 Go 组装可机械复用的上下文快照（章纲、近章摘要、设定、记忆、伏笔、FTS）。
3. 先让 LLM 产出**写作任务书**，再据此起草正文。
4. 同一次审查调用里完成评分 + 润色正文，再由代码从报告中抽出正文写回。
5. 生成摘要（供后续章 gate / 上下文链），更新进度与索引，提取实体/伏笔/爽点/记忆。

实现分层：

- **编排层（WriteWorkflow）**：步骤顺序、ledger 断点、何时落盘、失败软硬分级。
- **材料层（Builder）**：从磁盘 + SQLite 预组装 Snapshot；**主路径不靠 tool call 取材料**。
- **生成层（Agent）**：多次纯 chat（`Tools: false`），只产出文本；写文件全是 Go。

```
CLI nova write [章号]
        │
        ▼
app.LoadContext → Config + Project + Store
        │
        ▼
NewWriteWorkflow → BindProject + agent.New
        │
        ▼
WriteChapter
  gate → ledger → Snapshot
    → ContextSystem（任务书）
    → WriteSystem（起草）→ 正文.md
    → ReviewSystem（审查+润色）→ 审查.md + 覆盖正文.md
    → SummarySystem → 摘要.md
    → progress + index + ExtractAndPersistFacts
    → Report
```

与 Plan 的对比见 §10。

---

## 2. 入口：CLI 如何接到 Workflow

### 2.1 命令与标志

```text
nova write [章号] [--resume] [--stream] [--volume N] [--continue-on-error]
           [--project PATH] [--format text|json]
```

| Flag | 作用 |
|------|------|
| 位置参数 | `ParseChapterRange`：`"5"` 或 `"5-8"` |
| `--resume` | 读 `.nova/run_ledger.json`，从断点步骤续跑 |
| `--stream` | 起草阶段流式写 stdout（`OnDelta`） |
| `--volume` | 上下文用的卷号；默认 `Meta.CurrentVolume`，再默认 `1` |
| `--continue-on-error` | 批量写章时单章失败不中断 |

`WriteOptions` 上还有 `SkipReview`、`PinnedMemoryIDs`、`ExcludedMemoryIDs`、`OnStep`——CLI **未暴露**，仅供其它调用方传入。

### 2.2 CLI 循环

```
ParseChapterRange
LoadContext + RequireAPIKey
NewWriteWorkflow
vol = --volume | CurrentVolume | 1
for ch in chapters:
    WriteChapter(..., WriteOptions{Chapter, Volume: vol, Resume, Stream, OnDelta})
    若 err 或 Status ∈ {needs_action, failed}：
        无 --continue-on-error → 停止
    Report.Print
```

每章独立一次完整流水线；批量时共享同一个 `WriteWorkflow` 实例（同一 Agent/Registry）。

### 2.3 Workflow 构造

```go
func NewWriteWorkflow(cfg, p, st) *WriteWorkflow {
    reg := tools.NewRegistry()
    reg.BindProject(p.Root, st)  // 含 write_file / update_memory
    return &WriteWorkflow{Agent: agent.New(...), Config: cfg}
}
```

**重要不对称：** Registry 绑了完整写工具集，但 `WriteChapter` 里所有 `Agent.Run` 都**不传** `Tools: true`。材料靠 Builder 预注入；落盘靠 `pipeline` / `os.WriteFile`。工具绑定对主路径是「备而不用」（独立 `nova review` 才会开 Tools）。

---

## 3. `WriteChapter` 逐步执行

入口：`(*WriteWorkflow).WriteChapter(ctx, p, st, opts)`。全程共用一个 `UsageAccumulator` 累加各轮 token。

### 3.0 写前 Gate（`pipeline.RunGate` · `GatePrewrite`）

| 检查项 | 条件 |
|--------|------|
| phase | 必须是 `planning` 或 `writing` |
| 卷纲文件 | `VolumeOutlinePath(Meta.CurrentVolume 或 1)` 存在 |
| 上一章摘要 | `chapter > 1` 时 `SummaryPath(chapter-1)` 存在 |
| 索引 | `CheckIndexStale`；若 stale 先尝试 `RebuildChapters(0)`，仍 stale 则失败 |

失败时返回 `Report{Status: needs_action}`，**error 为 nil**（软失败）。不跑 LLM，不写 ledger。

**卷号陷阱：** Gate 用 `Meta.CurrentVolume` 查卷纲；Builder / 任务书用 `opts.Volume`（可来自 `--volume`）。两者不一致时可能「gate 过了但抽到错误卷的章纲」，或反过来。

### 3.1 Ledger 与断点（`.nova/run_ledger.json`）

```go
ledger, _ := LoadLedger(RunLedgerPath)
ledger.Chapter = opts.Chapter   // 覆盖字段；不校验旧 ledger 是否同章
startStep := "draft"
if opts.Resume {
    startStep = ledger.ResumeStep()
    if startStep == "done" { startStep = "draft" }  // 已完成则重跑起草
}
```

`ResumeStep` 只看 **Steps 最后一条**：

| 最后一步 | Status | 续跑起点 |
|----------|--------|----------|
| （空） | — | `draft` |
| `draft` | done | `review` |
| `review` / `polish` | done | `summary` |
| 其它名 | done | `done`（CLI 会再当成 draft） |
| 任意名 | ≠ done | 重试该 `Name` |

`Record` 是 append-only，不改写历史条目。`IsResumable`：`Chapter>0` 且最后一步不是 `(commit, done)`。

**续跑粒度粗：** 不校验 ledger.Chapter 与本次 `opts.Chapter` 是否一致；`summary`/`extract` 失败后的续跑行为见 §3.5–3.6。

### 3.2 组装 Snapshot（`context.Builder.Build`）

无 LLM。产出内存结构 `Snapshot`：

| 字段 | 来源 | 预算/策略 |
|------|------|-----------|
| BookAnchor | `nova.yaml` Meta → `prompts.BookAnchor` | 简介截 200 rune；含 antiDriftRules |
| ChapterOutline | `大纲/第VV卷.md` → `ExtractChapterSection` | 找不到则后续 prompt 用卷纲截断兜底 |
| VolumeOutline | 同文件全文 | 截断 4000 rune |
| RecentSummary | `SummaryPath(ch-1..)` 最多 3 章 | 从近到远收集，拼回旧→新 |
| Settings | `设定集/**/*.md` | 文件名关键词优先级；单文件截 800；合计约 3500 rune |
| OpenForeshadows | `store.ListForeshadows("open")` | 全量列表进字符串 |
| Memories | `RecallMemories`（规则 + 可选语义 RRF） | Top-K≈10，rune 预算≈2400；空则近期记忆兜底；再应用 pin/exclude |
| FTSHits | `SearchFTS(keywords, 5)` | 拼进检索命中小节 |

优先级在任务书 prompt 里写死：**章纲 > 近章摘要 > 设定 > 记忆**。

`ToContextPrompt` / `ToWriteUserPrompt(taskBook)` 把 Snapshot 编成 user 侧 Markdown；书籍锚点进 **system**（`ContextSystem` / `WriteSystem` / `ReviewSystem`），避免重复占 user 位。

### 3.3 起草（LLM #1 + #2）

仅当 `startStep == "draft"`：

**#1 任务书** — `ContextSystem(anchor)` + `ToContextPrompt()` + 「请输出写作任务书」

强制小节：本章目标、必含要素、爽点、伏笔操作、衔接、禁忌清单；300–600 字。

**#2 正文** — `WriteSystem(anchor)` + `ToWriteUserPrompt(taskBook)`

- 目标字数硬编码 `"2500-4000"`，**不读** `Meta.ChapterWords`。
- 可 `Stream` + `OnDelta`（无 tools 时走 `runStream`）。
- 成功后：`ParseChapterTitle`（取首个 `#` 行）→ `SaveChapterWithVersion(..., SourceWriteDraft, "写章起草")`。
- Ledger：`draft` done（失败则 `draft` failed 并 Save ledger，硬错误返回）。

非 draft 续跑：`loadChapterFile` 读已有 `正文.md`。

**落盘路径：** `正文/第NNN章-标题/正文.md`。`ChapterDirFor` 若已有同章号目录则复用，避免标题变化另起目录时丢附属文件（审查/摘要按章号找目录）。

**版本：** `version.BeforeSave` —— 若旧正文非空且与新内容不同，先快照到 `.nova/versions/chapter-NNN/`，再写新文件。

### 3.4 审查 + 润色（LLM #3，一次调用记两步）

条件：`!SkipReview && (startStep == "review" || startStep == "polish")`。

User 材料：

```
【本章章纲】ChapterOutline 或 VolumeOutline
【Open 伏笔】snap.OpenForeshadows
【正文】当前 content
```

`ReviewSystem` 要求输出：维度评分、问题清单、**`## 润色版正文`**（完整正文）、末尾 JSON（`hook_score` 等）。

随后 Go 侧：

1. 整份报告写入 `审查.md`。
2. `persistReviewRecord`：抽 JSON → `store.UpsertReview`。
3. `extractPolishedBody(reviewed, content)`：按标记 / 章标题 / `---` 分段启发式抽取润色正文；失败则回退原文；并 `normalizeChapterBody` 去掉前言、附录、尾部 metrics JSON。
4. 再次 `SaveChapterWithVersion(..., SourceWriteReview, "审查润色")` 覆盖 `正文.md`。
5. Ledger：连续记 `review` done、`polish` done（**同一次 LLM**）。

审查 LLM 失败 → 硬错误；起草文件保留。

独立命令 `ReviewWorkflow.ReviewChapter`（`nova review`）不同：开 `Tools: true`，user 里塞全设定拼接，结束后也会 `ExtractAndPersistFacts`，并把 chapter status 标 `reviewed`。Write 内嵌审查是「轻量同管线 pass」。

### 3.5 摘要（LLM #4）

**几乎总是执行**（无 `if startStep == "summary"` 跳过）。续跑到 `summary` 时：跳过起草与审查，仍会重新跑摘要与后续沉淀。

- System：`SummarySystem()`（无 BookAnchor）—— 200–400 字事实摘要。
- 失败：返回 `StatusPartial`，「正文已保存，摘要失败」；**不**更新 progress / index / extract；ledger 可能停在此前的 review/polish。
- 成功：`SaveSummary` → `摘要.md`；ledger `summary` done。

下一章 gate 依赖本文件存在，故摘要失败会阻断 `nova write N+1`。

### 3.6 写后沉淀（无 / 有 LLM #5）

顺序固定：

```
UpdateProjectProgress(p, chapter)
  → CurrentChapter = max(...)
  → phase planning|init_done → writing
  → Save nova.yaml

PostWriteIndex(p, st, chapter, chapterPath)
  → UpsertChapter（字数、路径、SummaryPath、InferChapterStatus）
  → index.RebuildChapters(chapter)   // FTS

ExtractAndPersistFacts(ctx, Agent, st, chapter, content, summary)
  → LLM ExtractSystem → JSON
  → entities + history / foreshadows / cool_points / memories

ledger: extract done|failed；始终 commit done；Save ledger
usage.AddWriteRun → .nova/usage_stats.json
Report completed
```

**Extract 失败不失败整章：** 仍 `StatusDone` + `commit` done。仅 ledger 记 `extract failed`。

`extractStoryFacts`：正文截前 8000 rune；JSON 解析失败会**再调一次** LLM 强调严格 JSON；仍失败才返回 error。

实体：规范名 → `EntityID` → Upsert + `RecordEntityStateHistory`。  
伏笔：`action=resolve` 或 `status=resolved` 时记 `ResolvedChapter`。  
无 id 时用描述前 32 rune 生成 `fs:…`。

---

## 4. 数据流

```
                    ┌─ nova.yaml (Meta)
                    ├─ 大纲/第VV卷.md ──ExtractChapterSection──┐
磁盘/DB 输入 ───────┼─ 近章 摘要.md (≤3)                      │
                    ├─ 设定集/*.md (优先级+预算)                 ├─► Snapshot
                    └─ nova.db: memories / foreshadows / FTS ──┘
                                      │
          ┌───────────────────────────┼───────────────────────────┐
          ▼                           ▼                           ▼
   ContextSystem                 WriteSystem                 ReviewSystem
   → taskBook (内存)             → 正文.md                   → 审查.md
                                 (+ versions)                → 抽润色 → 覆盖正文.md
                                                                      │
                                                                      ▼
                                                              SummarySystem
                                                              → 摘要.md
                                                                      │
                    ┌─────────────────────────────────────────────────┤
                    ▼                     ▼                           ▼
              nova.yaml            nova.db chapters              ExtractSystem
              phase/chapter        + FTS                         → entities
                                                                 → foreshadows
                                                                 → cool_points
                                                                 → memories
                    │
                    └─► .nova/run_ledger.json
                        .nova/usage_stats.json
                        Report (artifacts: 正文路径, 摘要路径)
```

### 读写清单

| 读 | 写 |
|----|-----|
| `nova.yaml` | `nova.yaml`（进度 / phase） |
| `大纲/第VV卷.md` | `正文/.../正文.md`（起草 + 可能润色覆盖） |
| `设定集/**/*.md` | `正文/.../审查.md` |
| 近章 `摘要.md` | `正文/.../摘要.md` |
| `.nova/nova.db` | 同上 DB 多表 + FTS |
| `.nova/run_ledger.json`（resume） | ledger / usage_stats / versions |

章节目录形态：

```
正文/第00N章-标题/
  正文.md
  审查.md      # 审查+润色完整报告
  摘要.md      # 连续性真源之一
```

---

## 5. 失败语义与续跑

| 失败点 | 对外表现 | 磁盘状态 | 续跑建议 |
|--------|----------|----------|----------|
| Gate | `needs_action`，err=nil | 无变更 | 先 plan / 补摘要 / rebuild index |
| 任务书 LLM | 硬 error | 无正文 | 直接重跑 |
| 起草 LLM / 保存 | 硬 error；ledger `draft failed` | 可能无正文 | `--resume` 重试 draft |
| 审查 LLM | 硬 error；ledger `review failed` | 起草正文已在 | `--resume` → review |
| 摘要 LLM | `partial`，err=nil | 正文+审查在，无摘要 | 重跑会再走摘要（resume 到 summary 时跳过起草审查） |
| Extract | 仍 `completed` | 正文摘要进度都在；事实可能缺 | 可另走 review/extract 类命令补 |

批量：`--continue-on-error` 吞掉单章失败继续；否则返回首个硬 error 或打印 needs_action/failed 后停。

Agent 层：写路径不开 tools，无 tool loop；流式仅起草。无自动重试（除 extract JSON 一次）。

---

## 6. 卷纲如何进入写章

1. Plan 产出 `大纲/第%02d卷.md`，章标题需匹配 `outline.chapterHeaderRe`。
2. Gate：只检查 **CurrentVolume** 对应文件是否存在。
3. Builder：按 `opts.Volume` 读文件，`ExtractChapterSection` 得到本章块 → 任务书「必含要素」的主要来源。
4. 找不到本章标题：prompt 提示参考卷纲截断（2000 rune），写章易跑题但不阻断。
5. 审查阶段同样对照章纲（或整卷截断）。

Plan 与 Write 的契约是 **Markdown 标题格式**，不是共享结构化 API。

---

## 7. Prompt 与 Agent 调用一览

| 阶段 | System | Tools | Stream |
|------|--------|-------|--------|
| 任务书 | `ContextSystem` + BookAnchor | 否 | 否 |
| 起草 | `WriteSystem` + BookAnchor | 否 | 可选 |
| 审查润色 | `ReviewSystem` + BookAnchor | 否 | 否 |
| 摘要 | `SummarySystem` | 否 | 否 |
| 提取 | `ExtractSystem` | 否 | 否 |

均通过 `withUsage` 挂同一 `UsageAccumulator`，最终进 Report 与 `usage.AddWriteRun`。

---

## 8. 与工具集的关系

`BindProject` 相对 `BindProjectPlan` 多了 `write_file`、`update_memory`。

| | Plan 主路径 | Write 主路径 |
|--|-------------|--------------|
| Tools 开关 | `true`（只读集） | `false` |
| 材料获取 | 预注入 + 可选 tool | **仅** Builder 预注入 |
| 文件写入 | Go `WriteFile` 卷纲 | Go `pipeline` / `WriteFile` |
| 记忆写入 | 无 | Extract → `UpsertMemory`（非 tool） |

设计意图：写章上下文要稳定、可测、可预算控制；避免模型在写正文时随意 `write_file` 打乱流水线。

---

## 9. 关键数据结构

```go
type WriteOptions struct {
    Chapter, Volume int
    Resume, SkipReview, Stream bool
    PinnedMemoryIDs, ExcludedMemoryIDs []string
    OnDelta func(string) error
    OnStep  func(step, message string) error
}

type Snapshot struct {
    Chapter, Volume int
    BookAnchor, ChapterOutline, VolumeOutline string
    RecentSummary, Settings, Memories, OpenForeshadows, FTSHits string
    MemoryRecalls []MemoryRecallHit
}

type RunLedger struct {
    Chapter int
    Steps   []RunStep  // Name, Status, Started, Finished, Message
    Updated time.Time
}
```

任务书、审查报告、摘要都是 **非结构化 Markdown**（提取阶段才要求 JSON）。润色正文靠字符串启发式从审查报告切出，不是独立 API 字段。

---

## 10. 与 Plan 的衔接与差异

```
init → plan → 大纲/第NN卷.md + phase=planning
                │
                ▼
         write（gate 要卷纲 + 可 planning）
                │
                ▼
         正文/摘要/DB + phase=writing
                │
                ▼
         下一章 write（gate 要上一章摘要）
```

| | Plan | Write |
|--|------|-------|
| LLM 次数（典型） | 1 | 4–5 |
| 工具 | 开（只读） | 关 |
| 落盘时机 | 生成后立即 | 分步多次 |
| 断点 | 无 | run_ledger |
| 软失败 | 少 | gate / 摘要 / extract 分级 |
| 改 phase | init_done→planning | planning/init_done→writing |

---

## 11. 实现取舍

1. **任务书两阶段**：先约束再散文，降低跑题；代价是多一轮延迟与 token。
2. **审查与润色合并**：省一轮；ledger 仍记两步名，便于 resume 语义，但无法「只重跑润色」。
3. **工具绑定却关闭**：与独立 review 共用构造函数简单；主路径行为靠 `Tools: false` 保证。
4. **Gate 卷号 vs opts.Volume**：历史简化，调用方需保持一致。
5. **字数目标写死在 prompt**：与 `nova.yaml` 的 `chapter_words` 脱节。
6. **摘要是硬连续性纽带**：文件缺失直接挡下一章；extract 失败则不挡。
7. **润色抽取启发式**：依赖模型遵守 `## 润色版正文`；否则回退草稿，可能把报告噪声写入正文（normalize 尽量剥离）。
8. **Ledger 单文件单章语义弱**：全局一个 `run_ledger.json`，切换章号续跑可能错接步骤。

---

## 12. 端到端调用链

```text
writeCmd.RunE
  → ParseChapterRange
  → LoadContext / RequireAPIKey
  → NewWriteWorkflow → BindProject → agent.New
  → WriteChapter
       → RunGate(GatePrewrite)
       → LoadLedger / ResumeStep
       → Builder.Build
            → ExtractChapterSection / recentSummaries / settingsDigest
            → RecallMemories / SearchFTS
       → Agent.Run(ContextSystem)              // 任务书
       → Agent.Run(WriteSystem [, Stream])     // 起草
       → SaveChapterWithVersion(write_draft)
            → version.BeforeSave → Snapshot?
            → SaveChapter → 正文.md
       → Agent.Run(ReviewSystem)               // 审查+润色文本
       → WriteFile(审查.md) / persistReviewRecord
       → extractPolishedBody
       → SaveChapterWithVersion(write_review)
       → Agent.Run(SummarySystem)
       → SaveSummary → 摘要.md
       → UpdateProjectProgress
       → PostWriteIndex → UpsertChapter + RebuildChapters
       → ExtractAndPersistFacts
            → extractStoryFacts (+ JSON retry)
            → UpsertEntity / RecordEntityStateHistory
            → UpsertForeshadow / UpsertCoolPoint / UpsertMemory
       → ledger commit + usage.AddWriteRun
       → Report
  → Report.Print
```

---

## 13. 相关文档

- [Plan 底层实现](./plan.md) — 卷纲如何生成，Write 的上游契约
- [Init 目录与 SQLite](./init.md) — 项目骨架与 DB
- [Memory](./memory.md) — 记忆召回与沉淀细节（Write 的 Memories / Extract 侧）
