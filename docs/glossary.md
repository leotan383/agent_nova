# 目录说明与专有名词

本文解释两套「目录」：

1. **仓库代码目录**（开发 agent_nova 时）
2. **小说项目工作区目录**（`nova init` 之后每本书的磁盘布局）

以及代码与创作流程里反复出现的**专有名词**。实现细节见 [init](./init.md) / [plan](./plan.md) / [write](./write.md) / [memory](./memory.md)。

---

## 一、仓库代码目录

```
agent_nova/
├── cmd/nova/          # CLI 入口（每个子命令一个文件）
├── internal/          # 核心业务包（不可被外部 Go 模块 import）
├── desktop/           # Nova Studio（Wails + React）
├── docs/              # 设计与说明文档
├── web/dashboard/     # 只读 Web 面板静态资源
├── bin/               # 本地编译产物（如 nova 二进制）
├── go.mod / go.sum
└── README.md
```

### `cmd/nova/`

Cobra 命令注册与参数解析，尽量薄：加载 `app.Context` 后调用 `internal/workflows` 或其它包。

| 文件（示例） | 对应命令 |
|--------------|----------|
| `init.go` / `plan.go` / `write.go` | 立项、卷纲、写章 |
| `review.go` / `coach.go` | 审查、改稿讨论 |
| `gate.go` / `status.go` / `doctor.go` | 写前检查、健康报告、体检 |
| `memory.go` / `index.go` / `query.go` | 记忆、索引、检索 |
| `serve.go` / `dashboard.go` | Daemon API、Web 面板 |

### `internal/` 包一览

| 包 | 作用 |
|----|------|
| `agent` | OpenAI 兼容客户端：`Run`（可带 tool loop）、`Chat`、流式、JSON 抽取 |
| `app` | `LoadContext`：拼装 Config + Project + Store |
| `config` | 全局 API Key / BaseURL / Model（环境变量或配置文件） |
| `project` | 小说项目：`nova.yaml`、路径、Init、章节目录布局、阶段常量 |
| `store` | SQLite：章节/实体/伏笔/记忆/审查/爽点/FTS/向量等 |
| `workflows` | 业务工作流：Init/Plan/Write/Review/Discover/Coach/Extract… |
| `workflow` | 另一套「每日活动/工作流」相关（与复数 `workflows` 不同） |
| `pipeline` | 写章管线辅助：Gate、RunLedger、存正文/摘要、进度、写后索引 |
| `context` | 写章上下文 `Builder` / `Snapshot`、记忆召回 |
| `prompts` | 各阶段 System Prompt（Plan/Write/Review/Extract…） |
| `tools` | Agent 可调工具注册表（`BindProject` / `BindProjectPlan`） |
| `outline` | 卷纲 Markdown 解析、章切片、卷边界、规划矩阵、结构健康 |
| `memory` | 设定回填记忆、记忆沉淀到设定等 |
| `index` | 章节 FTS 重建、embeddings |
| `rag` | 语义检索辅助（记忆召回可选用） |
| `version` | 正文版本快照与 diff（写前备份、Replan diff） |
| `report` | 统一命令结果结构（Status / Artifacts / NextSteps / Token） |
| `status` / `doctor` | 创作状态、健康度、体检项 |
| `consistency` | 一致性相关检查 |
| `wiki` | 设定/百科类读写辅助 |
| `library` | 多书书库 `library.json`（桌面端常用） |
| `paths` | 全局 `~/.config/nova`、`~/.local/share/nova` |
| `inspiration` | 灵感/探讨立项映射 |
| `chapterops` / `chapterdocs` | 章节结构操作、文档读写 |
| `export` / `backup` / `usage` | 导出、备份、token 用量统计 |
| `api` / `daemon` / `jobs` | HTTP 服务、守护进程、异步任务 |
| `search` | 检索封装 |
| `logger` | 日志 |

### `desktop/`

Wails 桌面应用：Go bindings（`plan.go`/`write.go`/…）+ `frontend/`（React）。与 CLI 共用 `internal/` 工作流，不另起一套领域逻辑。详见 [desktop.md](./desktop.md)。

### `docs/`

设计文档：`init` / `plan` / `write` / `memory` / `desktop` 等。

### `web/dashboard/`

`nova dashboard` 用的只读前端静态资源。

### 全局用户目录（不在仓库内）

由 `internal/paths` 管理（可用 `NOVA_HOME` 覆盖）：

| 路径 | 作用 |
|------|------|
| `~/.config/nova/config.yaml` | 持久化 API 配置 |
| `~/.config/nova/current` | CLI「当前项目」路径（`nova use`） |
| `~/.local/share/nova/` | 数据目录（含书库等，视功能启用情况） |

---

## 二、小说项目工作区目录

`nova init --dir ./my-novel` 之后，**一本书 = 一个目录**。架构原则：**Markdown 是真源，SQLite 是读模型/索引**。

```
my-novel/
├── nova.yaml                 # 项目元数据真源
├── .nova/
│   ├── nova.db               # SQLite 索引库
│   ├── backups/              # 备份落点
│   ├── versions/             # 章节正文历史版本（写章覆盖前快照）
│   ├── run_ledger.json       # 写章断点续跑（写章时才有）
│   ├── usage_stats.json      # token 用量累计（写章后）
│   └── discovery/            # 仅 --discover：探讨笔记
│       └── notes.md
├── 设定集/
│   ├── 角色/                 # 主角卡、反派设计…
│   ├── 世界/                 # 世界观、力量体系、金手指…
│   ├── 势力/ 地点/ 物品/ 其他/
├── 大纲/
│   ├── 总纲.md               # 全书级大纲（Plan 上游）
│   ├── 爽点规划.md           # 爽点结构模板（Init 后常不自动改）
│   └── 第01卷.md             # 卷纲（Plan 产出）
└── 正文/
    └── 第001章-标题/
        ├── 正文.md
        ├── 摘要.md
        ├── 审查.md
        └── AI味.md           # 可选
```

> 旧版曾把审查/摘要平铺在顶层 `审查/`、`摘要/`；现行布局是**章节子目录内并列**。顶层旧目录名仅迁移兼容。

### 根与元数据

| 路径 | 作用 |
|------|------|
| `nova.yaml` | 书名、题材、`phase`、当前卷/章、风格、字数目标、简介、主角、金手指等。几乎所有命令都会读。 |
| `.nova/nova.db` | 章节行、实体、伏笔、记忆、审查指标、爽点、FTS、embeddings 等。 |
| `.nova/run_ledger.json` | 单章写流水线步骤日志，供 `--resume`。 |
| `.nova/versions/` | `version.BeforeSave` 在覆盖正文前存的历史稿。 |
| `.nova/discovery/` | Discover 探讨 transcript 与提炼摘要。 |
| `.nova/backups/` | `nova backup` 等备份输出。 |

### `设定集/`

稳定设定的人类可读真源。Init 按题材生成模板文件；Plan/Write 会拼接或摘要注入 prompt；Enrich / wiki / 记忆回填会读写此处。

子目录（`角色/世界/势力/地点/物品/其他`）用于分类；文件名参与设定优先级排序（如含「主角」「金手指」更靠前）。

### `大纲/`

| 文件 | 作用 |
|------|------|
| `总纲.md` | 全书梗概、冲突、分卷表等；`nova plan` 的主要输入之一 |
| `爽点规划.md` | 大/中/微爽点与追读力提纲 |
| `第NN卷.md` | **卷纲**：该卷逐章章纲；`nova plan` 写入；`write` 抽本章段落 |

### `正文/`

每章一个目录 `第NNN章-标题/`：

| 文件 | 作用 |
|------|------|
| `正文.md` | 章节正文真源 |
| `摘要.md` | 本章事实摘要；下一章 gate 与近章上下文依赖它 |
| `审查.md` | 审查+润色报告全文 |
| `AI味.md` | 可选的 AI 味检测报告 |

---

## 三、专有名词（创作域）

网文创作语境下的业务词，在 prompt、文件名与 UI 中频繁出现。

| 名词 | 含义 |
|------|------|
| **总纲** | 全书级大纲（`大纲/总纲.md`），管分卷与主线，不管单章细节。 |
| **卷纲** | 某一卷的详细章纲（`大纲/第NN卷.md`），每章含冲突/爽点/伏笔等。 |
| **章纲** | 卷纲里**单章**那一段；写章时用 `ExtractChapterSection` 抽出，优先于整卷。 |
| **设定集** | 世界观、人物、金手指等稳定设定的 Markdown 集合。 |
| **金手指** | 主角核心外挂/能力（`cheat` 字段 / `金手指.md`）。 |
| **爽点** | 读者爽感设计点：微/中/大（micro/medium/major）；可记在卷纲或 `cool_points` 表。 |
| **伏笔** | 埋设与回收的情节线索；库中有 `open` / `resolved` 等状态。 |
| **追读力 / 章末钩子** | 章末悬念、促使点下一章的设计；审查里常有 `hook_score`。 |
| **OOC** | Out Of Character，人物言行偏离已设定性格。 |
| **AI 味** | 正文像大模型套话/同质化的痕迹；可选专项检测。 |
| **Discover** | 立项前与 AI 多轮探讨方向（`nova init --discover`），不是独立命令。 |
| **Replan** | 已有连载正文后，按事实重规划卷纲（`nova plan replan`）。 |
| **任务书** | 写章前由 Context Agent 生成的可执行写作说明（目标/必含/禁忌等），再交给写手 Agent。 |
| **改稿 / Coach** | 针对已写章节的讨论与 `/revise` 出修改稿。 |

---

## 四、专有名词（工程 / 流水线）

代码与 CLI 里的结构概念。

### 阶段与项目

| 名词 | 含义 |
|------|------|
| **Project** | 一本书的工作区；根目录含 `nova.yaml`。 |
| **Meta / nova.yaml** | 项目元数据真源。 |
| **phase** | 创作阶段机：`empty`（概念上）→ `init_done` → `planning` → `writing`（另有 `paused`）。Init 直接写 `init_done`。 |
| **CurrentVolume / CurrentChapter** | 元数据里记录的「进行到的卷/已写到的章」。 |
| **Markdown 真源 + SQLite 读模型** | 作者可读可改的是 md；库负责检索、状态、记忆注入，不替代正文文件。 |

### Agent 与工具

| 名词 | 含义 |
|------|------|
| **Agent** | `internal/agent`：一次或多次 Chat Completion；可挂 Tools。 |
| **Tool loop / MaxToolLoops** | 模型连续 function call 的上限轮数；超限失败。 |
| **BindProject** | 写章等用的工具集（含 `write_file` / `update_memory`）。 |
| **BindProjectPlan** | 规划用只读工具集（无写文件）。 |
| **BookAnchor / 书籍锚点** | 塞进 system prompt 的书名/题材/风格/主角/金手指/简介摘要。 |
| **antiDriftRules / 一致性铁律** | 禁止擅自改设定、OOC、乱收伏笔等的固定约束文案。 |
| **Enrich / EnrichSettings** | Init 后用 LLM 完善设定文件与总纲。 |
| **`===FILE:路径===`** | Init Enrich 时模型批量输出多文件的分隔约定。 |

### 写章流水线

| 名词 | 含义 |
|------|------|
| **WriteWorkflow / WriteChapter** | 单章完整流水线编排入口。 |
| **Snapshot** | `context.Builder` 组装的写章上下文快照（章纲、近摘要、设定、记忆、伏笔、FTS…）。 |
| **Gate / 写前检查** | `pipeline.RunGate`：phase、卷纲存在、上一章摘要、索引是否过期等。 |
| **prewrite / precommit / postcommit** | Gate 三阶段：写前 / 提交前 / 提交后。 |
| **RunLedger / 账本** | `.nova/run_ledger.json`，记录 draft→review→polish→summary→extract→commit。 |
| **`--resume`** | 按 ledger 最后一步决定从哪步续跑。 |
| **SaveChapterWithVersion** | 写正文前先版本快照，再落盘。 |
| **extractPolishedBody** | 从审查报告里启发式抽出「润色版正文」。 |
| **ExtractAndPersistFacts** | 写后从正文/摘要抽实体、伏笔、爽点、记忆并写入 SQLite。 |
| **PostWriteIndex** | 更新 chapters 行 + 重建该章 FTS。 |
| **Report** | 命令结束时的结构化结果：`completed` / `partial` / `needs_action` / `failed`。 |

### 记忆与检索

| 名词 | 含义 |
|------|------|
| **Memory / 长期记忆** | `memories` 表中可跨章注入的条目（style/plot/character/world 等）。 |
| **BootstrapFromSettings** | 从设定 md 种子化 memories（Init Enrich 后常见）。 |
| **Recall / 召回** | 写章前按规则 + 可选语义检索选出要注入的记忆。 |
| **RRF** | Reciprocal Rank Fusion，融合规则排序与语义排序。 |
| **FTS** | SQLite 全文检索，章节/设定索引。 |
| **Entity / 实体** | 角色/地点/物品等及 `state` JSON；可有历史快照。 |
| **Foreshadow** | 伏笔记录：planted/resolved chapter、status。 |
| **CoolPoint** | 结构化爽点是否兑现。 |
| **pin / exclude** | 写章时强制纳入或排除某些 memory id（CLI 较少暴露）。 |

### 卷纲解析与规划

| 名词 | 含义 |
|------|------|
| **PlanStatus** | 卷纲章条目状态：`planned` / `done` / `deviated` / `abandoned`（对应「已完成/偏离/废弃」标记）。 |
| **Outline Matrix** | 章纲状态与正文是否存在等对照矩阵。 |
| **Volume bounds** | 从各卷纲章标题推断章号归属区间。 |
| **chapterHeaderRe** | 识别 `### 第N章 · 标题` 一类行的正则。 |

### 其它模块名

| 名词 | 含义 |
|------|------|
| **Library / Registry** | 多本小说注册表（桌面书库）。 |
| **Doctor / Preflight / Status** | 体检、写前预检、创作健康/待办报告。 |
| **Daemon / serve** | 后台 HTTP + 异步写章等 API。 |
| **Usage / TokenUsage** | LLM token 与粗估费用统计。 |
| **Inspiration** | 灵感讨论到立项字段的映射链路。 |

---

## 五、主流程里名词怎么串起来

```
Discover/Init
  → 设定集 + 总纲 + phase=init_done
       │
       ▼
Plan（卷纲）
  → 大纲/第NN卷.md + phase=planning
       │
       ▼
Write
  Gate → Snapshot（章纲+摘要+设定+记忆+伏笔）
    → 任务书 → 正文 → 审查/润色 → 摘要
    → 进度/索引 → Extract 事实
    → phase=writing
       │
       ▼
（可选）Replan / Review / Coach / Memory 维护
```

---

## 六、相关文档

| 文档 | 内容 |
|------|------|
| [init.md](./init.md) | 立项命令实现 |
| [plan.md](./plan.md) | 卷纲规划实现 |
| [write.md](./write.md) | 写章流水线实现 |
| [memory.md](./memory.md) | 长期记忆模块 |
| [desktop.md](./desktop.md) | 桌面端 |
| 根目录 `README.md` | 安装、命令速查、快速开始 |
