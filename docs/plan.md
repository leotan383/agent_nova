# Plan 底层实现原理

本文说明 `nova plan` 如何把「总纲 + 设定」变成「卷纲 Markdown」，以及 Replan 如何在已写正文约束下重算卷纲。只覆盖 CLI 与 `internal/` 核心路径，不涉及桌面端。

核心代码：

| 模块 | 路径 |
|------|------|
| CLI 入口 | `cmd/nova/plan.go` |
| 工作流 | `internal/workflows/init.go`（`PlanVolume`）、`replan.go`（`ReplanVolume`） |
| Agent 循环 | `internal/agent/agent.go` |
| 只读工具 | `internal/tools/registry.go` → `BindProjectPlan` |
| Prompt | `internal/prompts/prompts.go` → `PlanSystem` / `ReplanSystem` |
| 卷纲解析 | `internal/outline/parse.go` |
| 写章消费 | `internal/context/builder.go`、`internal/pipeline/pipeline.go` |

---

## 1. 问题定义

Plan 解决的不是「随便写个大纲」，而是产出一份**可被下游机械消费**的卷级章纲：

1. 文件路径固定：`大纲/第%02d卷.md`（`Project.VolumeOutlinePath`）。
2. 章条目用固定标题正则可解析（`outline.chapterHeaderRe`）。
3. 写前 gate 用文件是否存在做硬门槛；写章时用 `ExtractChapterSection` 抽本章段落进任务书。

因此 Plan 的实现分两层：

- **编排层（Workflow）**：读哪些材料、怎么拼 prompt、Agent 跑完后如何落盘 / 是否落盘、如何改 `nova.yaml`。
- **执行层（Agent + Tools）**：OpenAI 兼容 chat + tool calling 循环；工具集刻意只有只读能力，文件写入由 Go 侧完成。

```
CLI (plan / show / replan)
        │
        ▼
app.LoadContext  →  Config + Project(nova.yaml) + Store(nova.db)
        │
        ▼
NewPlanWorkflow  →  BindProjectPlan + agent.New
        │
        ├─ PlanVolume     → Agent.Run → WriteFile → Save meta → Report
        └─ ReplanVolume   → Agent.Run → ReplanResult（不写文件）
                              │
                              ▼
                         CLI: Diff → 可选 --apply WriteFile
```

---

## 2. 入口：CLI 如何接到 Workflow

### 2.1 命令拆分

| 命令 | 函数 | LLM | 副作用 |
|------|------|-----|--------|
| `nova plan [卷号]` | `runPlan` | 是 | 立即覆盖写卷纲；可能改 phase / current_volume |
| `nova plan show [卷号]` | 内联 `RunE` | 否 | 只读打印 |
| `nova plan replan [卷号]` | `runPlanReplan` | 是 | 默认不写；`--apply` 后覆盖写 |

卷号解析：`project.ParseVolumeRange`，与章节范围同一实现（`"1"` 或 `"2-3"`）。Replan **强制单卷**（`len(vols) != 1` 报错）。

### 2.2 公共前置：`LoadContext` + API Key

```go
actx, err := app.LoadContext(projectRoot)
// Config: 全局配置（API Key / BaseURL / Model）
// Project: 读 nova.yaml
// Store:   打开 .nova/nova.db
app.RequireAPIKey(actx.Config)  // 空 key 直接失败
```

`plan show` 不需要 API Key；`plan` / `replan` 需要。

### 2.3 Workflow 构造

```go
func NewPlanWorkflow(cfg, p, st) *PlanWorkflow {
    reg := tools.NewRegistry()
    reg.BindProjectPlan(p.Root, st)   // 只绑定只读工具
    return &PlanWorkflow{
        Agent: agent.New(agent.Options{Config: cfg, Registry: reg}),
        store: st,
    }
}
```

要点：每次 `NewPlanWorkflow` 新建独立 `Registry` 与 `Agent`，工具闭包绑死当前项目 `root` 与 `store`。与写章用的 `BindProject`（含 `write_file` / `update_memory`）是**两套绑定**，不是运行时开关。

---

## 3. `PlanVolume`：首次 / 常规生成

入口：`(*PlanWorkflow).PlanVolume(ctx, p, vol)`。

### 3.1 逐步在做什么

| 步骤 | 代码动作 | 目的 |
|------|----------|------|
| 1. 读总纲 | `os.ReadFile(大纲/总纲.md)`，失败则空串 | 卷级目标与分卷骨架 |
| 2. 拼设定 | `readDirConcat(设定集/)` | 把设定集下全部 `.md` 递归拼进 prompt |
| 3. 附加连续上下文 | `planVolumeContext(p, store, vol)` | 上一卷卷纲、已写摘要、实体、开放伏笔 |
| 4. 拼 User Prompt | 固定章格式模板 + 总纲 + 设定 + extra | 告诉模型「输出什么形状」和「材料在哪」 |
| 5. 拼 System Prompt | `prompts.PlanSystem(BookContext{...})` | 角色、书籍锚点、反漂移、输出约束 |
| 6. 跑 Agent | `Agent.Run(..., Tools:true, MaxToolLoops:14)` | 可多轮 tool call，最终要拿到完整 Markdown |
| 7. 落盘 | `WriteFile(VolumeOutlinePath(vol), content, 0644)` | **无校验、不解析**，原样写入模型返回文本 |
| 8. 改元数据 | `CurrentVolume` 上提；`init_done`→`planning`；`p.Save()` | 推进阶段机 |
| 9. 返回 Report | `StatusDone` + artifacts + next_steps | CLI `--format` 打印 |

注意第 7 步：Workflow **不**调用 `ParseVolumeOutline` 校验格式。格式契约靠 prompt 约束；下游解析失败时写章仍会跑，但章纲抽取会退化（见 §7）。

### 3.2 User Prompt 结构

逻辑结构如下（字符串模板在 `PlanVolume` 内）：

```
请为第 {vol} 卷生成详细卷纲 Markdown。
每章格式：
### 第N章 · 标题
- 核心冲突：
- 爽点：
- 伏笔：

下文已附总纲、设定与已写摘要/状态，请优先直接使用。
仅在明显缺信息时调用只读工具补充，不要用 write_file。
收集足够信息后，必须在本轮直接输出完整 Markdown 正文。

总纲：
{master}

设定摘要：
{settings}{extra}
```

`extra` 可能为空，也可能是若干 `##` 小节（见 §3.3）。设计意图：

- **主材料预注入**：减少无意义 tool call（设定集往往已够用）。
- **工具是补洞**：缺某章细节 / 查实体时才用。
- **禁止 write_file**：工具集里本来就没有，prompt 再强调一次，避免模型「假装写了文件」却只返回空内容。
- **强制最终文本输出**：对抗「只调工具、本轮无 content」的坏行为；Agent 循环在无 `ToolCalls` 时才把 `msg.Content` 当作成功返回值。

### 3.3 `planVolumeContext`：连续上下文怎么打包

```
vol > 1 ?
  └─ 读 大纲/第{vol-1}卷.md → "## 上一卷卷纲"

CurrentChapter > 0 ?
  ├─ collectWrittenSummaries(1..N, maxRunes=12000)
  ├─ formatEntities(store, limit=50)
  └─ formatOpenForeshadows(store)   # status=open
```

**摘要预算算法（`collectWrittenSummaries`）**——Plan 与 Replan 共用：

1. 从 `throughChapter` **倒序**读 `SummaryPath(i)`（缺文件则 skip）。
2. 每块按 rune 计数累加；若 `total + n > maxRunes`，记录 `omitted = i` 并停止。
3. 块以「从旧到新」顺序拼回（prepend），保证阅读顺序仍是第 1→N。
4. 若有省略，文首加：`> … 第 1–{omitted} 章摘要已省略（token 预算）`。

含义：优先保留**最近章节**事实，早期摘要可丢。实体每条 `StateJSON` 再截到约 280 runes。

### 3.4 设定集拼接（`readDirConcat`）

`filepath.Walk` 设定集目录：

- 跳过目录与非 `.md`；
- 相对路径作小标题：`### 角色/主角卡.md`；
- 文件之间用 `\n---\n` 分隔。

无上限截断：设定很大时会直接撑大 user prompt（依赖模型上下文窗口）。这是当前实现的明确取舍——规划阶段宁可材料全，也不先做摘要压缩。

### 3.5 System Prompt：`PlanSystem` + 书籍锚点

`PlanSystem` 组装顺序：

1. 角色声明：「网文规划助手，根据总纲和设定集生成卷级章纲」。
2. `BookAnchor(BookContext)`：书名 / 题材 / 风格 / 主角 / 金手指 / 简介（简介超过 200 rune 截断）。**注意**：`PlanVolume` 传入的 `BookContext` **不设 `Chapter`**，锚点里不会出现「当前任务：第 x 卷 · 第 y 章」。
3. `BookAnchor` 末尾固定附带 `antiDriftRules`（世界观/OOC/伏笔/视角等铁律）。
4. 输出要求：优先用已附上下文；必须输出完整 Markdown；每章含冲突/爽点/伏笔；可执行、不写「待定」；节奏 3–5 章小高潮、卷末大钩子。

`BookContext` 字段来自 `nova.yaml` 的 `Meta`（风格取自 `WritingStyle()` → `style`）。

### 3.6 元数据副作用

仅 `PlanVolume` 成功写盘后：

```go
if p.Meta.CurrentVolume < vol {
    p.Meta.CurrentVolume = vol
}
if p.Meta.Phase == project.PhaseInitDone {
    p.Meta.Phase = project.PhasePlanning
}
_ = p.Save()  // 错误被吞掉：卷纲文件已写，meta 失败不回滚
```

阶段机：`empty → init_done → planning → writing`。Plan 只负责 `init_done → planning`；进入 `writing` 由写章流水线完成。

Report 的 `NextSteps` 含 `nova write {(vol-1)*30+1}`——这是**启发式章号**（假设约 30 章/卷），不是从卷纲解析出来的真实起始章。真实章号区间由 `outline` 包扫描标题得到。

### 3.7 批量卷号

`runPlan` 对 `vols` **顺序** `for` 调用 `PlanVolume`。前一卷已落盘后若后一卷失败，不会回滚前一卷。每卷独立 `Report.Print`。

---

## 4. Agent 执行层：tool loop 原理

Plan / Replan 都调用同一个 `(*Agent).Run`。

### 4.1 请求构造

```go
messages := [
  {role: system, content: SystemPrompt},
  {role: user,   content: UserPrompt},
]
Tools: true → 使用 registry.Tools()（OpenAI function definitions）
```

客户端：`openai.Client`，`BaseURL` / `APIKey` / `Model` 来自 `config.Config`。

### 4.2 循环语义

```
limit = MaxToolLoops>0 ? MaxToolLoops : 8   // PlanVolume 显式传 14；Replan 用默认 8

for i in 0..limit-1:
    resp = ChatCompletion(messages, tools)
    msg  = resp.Choices[0].Message

    if msg.ToolCalls 为空:
        return msg.Content          // 成功：最终卷纲正文

    messages.append(assistant msg)  // 含 tool_calls
    for each tool_call:
        result = registry.Execute(name, args)
        // 失败则 result = {"error":"..."} 字符串，不中断循环
        messages.append(tool role, ToolCallID, result)

return error("tool loop exceeded maximum iterations")
```

关键推论：

1. **成功判据**是「某一轮模型不再要工具、且返回了 content」，不是「调用了某个 finish 工具」。
2. 工具失败被编码进 tool message，模型可自行改策略再试。
3. 超过上限直接失败，**不会**把中间某次 content 当结果；若模型一直只调工具，整次 Plan 失败且不写文件。
4. Plan 开 14 轮、Replan 默认 8 轮：生成卷时允许更多补充检索；Replan 材料已在 user prompt 里堆得很满，少轮次即可。

Plan 路径当前 **不传** `UsageAcc` / `Stream`，token 统计与流式输出未接入卷纲生成。

### 4.3 为什么工具不能写文件

`BindProjectPlan` 注册的 map **不含** `write_file`、`update_memory`。即便模型幻觉调用，`Execute` 返回 `unknown tool`，被包成 error JSON 喂回。

设计目的：

- 落盘时机可控（只有 `Agent.Run` 成功返回后才 `WriteFile`）。
- 避免模型写半成品文件或写到错误路径。
- Replan 可以「只生成草案、由调用方决定是否 apply」。

---

## 5. 只读工具集实现细节

`BindProjectPlan` 绑定 6 个工具：

| 工具 | 实现要点 |
|------|----------|
| `read_file` | `safePath` 禁止 `..` 逃逸；返回 JSON `{path, content}` |
| `search_project` | `store.SearchFTS`；store 空则 `{"results":[]}` |
| `query_entity` | `SearchEntities(query, 20)` |
| `query_foreshadow` | `ListForeshadows(status)`，status 可选 |
| `get_chapter_outline` | 读 `大纲/第%02d卷.md` **整文件**；`volume` 默认 1；返回里带 `chapter` 字段但**未做章节切片** |
| `list_chapters` | `store.ListChapters()` |

`safePath`：

```go
rel = Clean(rel)
if HasPrefix(rel, "..") → error
return Join(root, rel)
```

`get_chapter_outline` 名实不完全一致：参数要求 `chapter`，实现却返回整卷 outline。下游写章用的切片在 `outline.ExtractChapterSection`，不在这个 tool 里。对 Plan 来说，它主要用于「再看一眼已有卷纲文件」（例如规划第 2 卷时工具读第 1 卷——尽管 `planVolumeContext` 往往已预注入）。

与 `BindProject`（写章）对比：Plan 少了写能力，其余读工具实现函数共用同一套方法。

---

## 6. `ReplanVolume`：在事实约束下重规划

入口：`(*PlanWorkflow).ReplanVolume(ctx, p, st, opts)`。  
**本函数从不写盘**；写盘是 CLI（或其它调用方）的事。

### 6.1 前置条件（硬失败）

| 条件 | 错误 |
|------|------|
| `Volume <= 0` | 无效卷号 |
| `CurrentChapter <= 0` | 尚无已写章节 |
| `FromChapter <= 0` | 自动设为 `CurrentChapter+1` |
| `FromChapter <= CurrentChapter` | 起始章须大于已写章 |
| 卷纲文件空或不存在 | 请先 `nova plan N` |

「已写」以 `nova.yaml` 的 `current_chapter` 为准，不是扫正文目录。

### 6.2 上下文与 Plan 的差异

Replan **总是**注入：

- 规划范围（已写至 / 从哪章起 / 卷号）
- 作者 `Notes`（空则 `"(无)"`）
- **当前卷纲全文**（待调整对象）
- 总纲、设定摘要
- 摘要链、实体、开放伏笔（空则占位文案）

对比 `PlanVolume`：

| | Plan | Replan |
|--|------|--------|
| 旧卷纲 | 通常不注入本卷（本卷正要生成）；vol>1 才注入上一卷 | **必须**注入本卷旧稿 |
| 摘要/实体/伏笔 | 仅 `CurrentChapter>0` 时 | **强制要求**已有已写章，总是尝试注入 |
| 上一卷卷纲 | 有 | 无专门小节（可靠工具读） |
| 落盘 | 立即 | 否 |
| MaxToolLoops | 14 | 默认 8 |
| System | `PlanSystem` | `ReplanSystem`（多一组 Replan 铁律；`BookContext.Chapter=fromChapter`） |
| Report.Status | `completed` | `needs_action` |

### 6.3 Replan 的输出契约（靠 prompt，非代码校验）

User prompt 要求模型输出**完整新卷纲**，并：

1. 第 1–`written` 章：与正文一致，标 `> 状态：已完成`
2. 第 `fromChapter` 起：新规划，格式同原
3. 偏离 / 废弃用对应状态行
4. 不得与摘要链、实体、伏笔矛盾；禁止「待定」

这些标记事后由 `outline.parsePlanStatus` 解析成 `done` / `deviated` / `abandoned` / `planned`，供矩阵与健康检查使用——**不是** Replan 返回值的结构化字段。

### 6.4 CLI 如何 apply

```
ReplanVolume → DiffTexts(old, proposed) → printOutlineDiff（跳过 same，最多 80 行）
             → Report.Print
             → 无 --apply：退出（预览）
             → --apply 且无 -y：stdin 读 y/N
             → WriteFile(VolumeOutlinePath, ProposedContent)
```

Apply **不**更新 `phase` / `current_volume` / `current_chapter`。Diff 用 `version.DiffTexts`，按行标 add/del。

---

## 7. 产物如何被下游消费

卷纲真源是 Markdown 文件，不是 SQLite 表。

### 7.1 解析契约（`outline.ParseVolumeOutline`）

章标题正则（多级 `#`，章号可前导 0，标题分隔符多种）：

```
^#{1,4}\s*第\s*0*(\d+)\s*章(?:\s*[·•·\-—]\s*(.+))?\s*$
```

状态行：

```
^>\s*状态[：:]\s*(已完成|偏离|废弃)
```

无状态行 → `PlanStatus = "planned"`。  
`ExtractChapterSection(full, chapter)`：找到目标章标题到下一章标题之间的切片；找不到则退化为全文截断 1500 runes。

### 7.2 写前 Gate

`pipeline.RunGate(..., GatePrewrite)`：

- phase 必须是 `planning` 或 `writing`；
- `VolumeOutlinePath(CurrentVolume 或 1)` 必须存在（`os.Stat`）；
- 与 Plan 的耦合是**文件存在性**，不是解析是否成功。

### 7.3 写章上下文（`context.Builder`）

```go
full := Read(VolumeOutlinePath)
snap.ChapterOutline = ExtractChapterSection(full, chapter)  // 本章任务优先依据
snap.VolumeOutline  = truncate(full, 4000)                  // 卷级背景
```

任务书 prompt 里「本章章纲」优先；缺失时用卷纲截断兜底。故 Plan 输出的标题格式直接影响写章是否「对准章节」。

### 7.4 其它消费者（原理层）

- `outline` 卷边界：扫各卷标题推断章号归属。
- 结构健康 / doctor / status：缺卷纲或格式漂移时提示先 `nova plan`。
- 矩阵：对比 `plan_status` 与正文是否存在，检测偏离。

---

## 8. 数据流总览

### 8.1 Plan 成功路径

```
设定集/*.md ─┐
总纲.md ─────┼─► user prompt ─┐
extra(可选) ─┘                │
nova.yaml Meta ─► PlanSystem ─┼─► Agent tool loop (≤14)
store(实体/伏笔) ─(可选工具)───┘         │
                                         ▼
                              完整 Markdown content
                                         │
                    ┌────────────────────┼────────────────────┐
                    ▼                    ▼                    ▼
            大纲/第NN卷.md         nova.yaml 更新         report.Report
            (覆盖写)              (volume/phase)         (CLI 打印)
```

### 8.2 Replan 路径

```
旧卷纲 + 总纲 + 设定 + 摘要链 + 实体 + 伏笔 + notes
        │
        ▼
 ReplanSystem + user prompt → Agent (≤8) → ProposedContent
        │
        ▼
 Diff(旧, 新) + Report(needs_action)
        │
        ├─ 默认：结束（内存中的草案丢弃，除非调用方保存）
        └─ --apply：覆盖 大纲/第NN卷.md
```

### 8.3 与写章的交接

```
大纲/第NN卷.md
    │
    ├─ GatePrewrite: Stat 存在？
    └─ Builder: ExtractChapterSection → ContextSystem/WriteSystem
```

---

## 9. 实现上的关键取舍

1. **材料预注入 vs 工具检索**  
   总纲 + 全设定集先进 prompt；工具只补洞。优点：少轮次、行为稳。代价：设定膨胀时 prompt 巨大。

2. **写权限外置**  
   模型不能写文件 → Plan 可立即落盘、Replan 可安全预览，同一套 `Agent.Run`。

3. **格式靠约定不靠校验**  
   `PlanVolume` 写盘前不 parse。坏格式不会在 Plan 阶段失败，而在写章抽取 / 矩阵上暴露。

4. **摘要有硬预算，设定没有**  
   摘要 12k runes 从新到旧保留；设定 Walk 全量。规划期更怕「丢近期剧情事实」，而非「设定 token」。

5. **元数据与文件不一致窗口**  
   先写卷纲再 `Save()`；`Save` 错误被忽略。极端情况下文件已新、phase 仍为 `init_done`（gate 会因 phase 失败）。

6. **`get_chapter_outline` 不做章切片**  
   命名偏写章语义，实现是整卷文件读取；真正切片在 `outline` 包。

7. **Replan System 中 `antiDriftRules` 实际出现两次**  
   `BookAnchor` 已含一份，`ReplanSystem` 又拼一次——当前代码如此，属于 prompt 冗余而非独立规则集。

---

## 10. 文件与类型速查

### 路径

| API | 路径 |
|-----|------|
| `VolumeOutlinePath(vol)` | `{root}/大纲/第%02d卷.md` |
| 总纲（硬编码拼接） | `{root}/大纲/总纲.md` |
| `SettingsDir()` | `{root}/设定集` |
| `SummaryPath(n)` | 章节布局下的 `摘要.md`（见 chapter_layout） |

### 类型

```go
type PlanWorkflow struct {
    Agent *agent.Agent
    store *store.Store
}

type ReplanOptions struct {
    Volume, FromChapter int
    Notes string
}

type ReplanResult struct {
    ProposedContent, OldContent string
    FromChapter, WrittenThrough int
    Report *report.Report
}

// outline.Entry
// PlanStatus: done | deviated | abandoned | planned
```

### CLI 标志（replan）

| Flag | 作用 |
|------|------|
| `--from-chapter` | 重规划起点；默认 `CurrentChapter+1` |
| `--apply` | 写入文件 |
| `-y` / `--yes` | 跳过确认 |
| `--notes` | 注入作者备注 |

---

## 11. 相关文档

- [Init 目录与 SQLite](./init.md) — Plan 的上游骨架（总纲、设定集、`init_done`）
- [Memory](./memory.md) — 写章后实体 / 记忆如何进入 Store（Replan 会读）
