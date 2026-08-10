# Init 底层实现原理

本文说明 `nova init` 如何从「立项参数」落到「可被 plan/write 消费的项目骨架」，以及可选的 Discover 探讨与 LLM 设定完善。只覆盖 CLI 与 `internal/` 核心路径。

核心代码：

| 模块 | 路径 |
|------|------|
| CLI 入口 | `cmd/nova/init.go` |
| 骨架创建 | `internal/project/init.go`（`InitProject`） |
| 设定子目录映射 | `internal/project/settings_layout.go` |
| LLM 完善设定 | `internal/workflows/init.go`（`EnrichSettings`） |
| 构思探讨 | `internal/workflows/discover.go` |
| 记忆回填 | `internal/memory/bootstrap.go` |
| SQLite | `internal/store/sqlite.go`（`Open` / `migrate` / `InitProject`） |
| Prompt | `internal/prompts/prompts.go`（`InitSystem` / `DiscoverSystem` / `DiscoverExtractSystem`） |

> 说明：仓库里没有独立的 `nova discover` 命令；Discover 是 `nova init --discover` 的一条分支。

---

## 1. 问题定义

Init 要解决的是：**在空目录上建立一本新书的可连载工程**，使后续 `nova plan` / `nova write` 有稳定契约可依：

1. **元数据真源**：`nova.yaml`（书名、题材、phase、字数目标、主角等）。
2. **设定骨架**：`设定集/{角色,世界,…}/` 下题材相关 Markdown 模板。
3. **大纲入口**：`大纲/总纲.md`、`大纲/爽点规划.md`（Plan 读总纲；爽点规划当前不被 Enrich 覆盖）。
4. **索引库**：`.nova/nova.db`（空表 + `project_meta` 一行）。
5. **阶段**：直接写成 `phase: init_done`（磁盘上不会出现 `empty` 阶段）。

实现分三层：

| 层 | 职责 |
|----|------|
| 参数采集 | flags / `--interactive` / `--discover` → `InitInput` |
| 骨架层 | `project.InitProject` + `store.Open`：纯本地，无 LLM |
| 完善层 | `EnrichSettings`（2 次 LLM）+ `BootstrapFromSettings`（可选） |

```
CLI flags | interactive | discover
              │
              ▼
         InitInput
              │
    ┌─────────┴─────────┐
    ▼                   ▼
store.Open          InitProject
(.nova/nova.db)     (目录 + nova.yaml + 模板 md)
    │                   │
    └─────────┬─────────┘
              ▼
     st.InitProject + SetCurrentProject
              │
     [--skip-llm 或 无 API Key?]
              │ yes → 结束（骨架可用）
              ▼ no
     EnrichSettings（设定文件 + 总纲）
              │
              ▼
     BootstrapFromSettings → memories
              │
              ▼
         Report.Print
```

---

## 2. 入口：CLI 与标志

### 2.1 命令面

```text
nova init --dir <路径>
          [--title] [--genre] [--style]
          [--target-words] [--chapter-words]
          [--synopsis] [--protagonist] [--cheat]
          [--interactive | --discover]
          [--skip-llm]
          [--format text|json]
```

| Flag | 默认 | 作用 |
|------|------|------|
| `--dir` | （必填） | 项目根目录 |
| `--title` | `""` | 书名；非 discover 时必填 |
| `--genre` | `玄幻` | 题材；决定设定模板集合 |
| `--style` | `热血` | 写作风格 |
| `--target-words` | `300000` | 目标总字数 |
| `--chapter-words` | `4000` | 单章目标字数 |
| `--synopsis` | `""` | 预填总纲「一句话梗概」与 yaml |
| `--protagonist` / `--cheat` | `""` | 预填主角卡 / 金手指 |
| `--interactive` | false | stdin 表单补全字段 |
| `--discover` | false | 多轮 AI 探讨后再立项 |
| `--skip-llm` | false | 只建骨架，不 Enrich |

**互斥 / 校验：**

- `--dir` 为空 → 直接报错。
- `--discover` 与 `--skip-llm` 不能同用（探讨本身要 LLM）。
- `--discover` 与 `--interactive` 二选一。
- 非 discover：`Title` 为空 → 报错；discover 提炼后若仍空则降为 `"未命名"`。

`InitInput.SkipLLM` / `Interactive` 只作传参；**真正分支在 CLI `RunE`**，`InitProject` 本身不读 `SkipLLM`。

### 2.2 三种参数采集路径

| 模式 | 函数 | 产出 |
|------|------|------|
| 纯 flags | 直接填 `InitInput` | 字段来自命令行 |
| `--interactive` | `runInteractiveInit` | stdin 逐项询问；回车保留当前值 |
| `--discover` | `RunDiscoverySession` | 多轮 Chat → `/done` → JSON 提炼 → 确认 Y/n → `InitInput` + `DiscoverResult` + transcript |

Discover 强制先 `RequireAPIKey`（硬失败）。Interactive 不要求 Key。

---

## 3. 逐步执行流程

入口：`initCmd.RunE`（`cmd/nova/init.go`）。

### 3.1 默认值归一

采集完成后：

```
Title 空（仅 discover 兜底）→ "未命名"
Genre 空 → "玄幻"
Style 空 → "热血"
TargetWords ≤0 → 300000
ChapterWords ≤0 → 4000
```

### 3.2 打开数据库（先于骨架）

```go
dbPath := "{initDir}/.nova/nova.db"
st, err := store.Open(dbPath)
```

`Open` 会 `MkdirAll(.nova)`、建库、`migrate()`（各业务表 + FTS）、`ensureTrigramFTS()`。  
此时**尚无** `nova.yaml`；若随后 `InitProject` 失败，可能留下空的 `.nova/`。

### 3.3 `project.InitProject` —— 骨架创建

```go
root = Abs(Dir)
若已存在 nova.yaml → 报错「项目已存在」（幂等拒绝）
```

**创建的目录：**

```
.nova/backups/
设定集/
设定集/{角色,世界,势力,地点,物品,其他}/   # DefaultSettingSubdirs
大纲/
正文/
```

**写入 `nova.yaml`：**

```yaml
title / genre / phase: init_done
style
target_words / chapter_words
synopsis / protagonist / cheat
# current_volume / current_chapter 默认 0（yaml 零值）
```

**Phase 语义：** 常量里有 `PhaseEmpty`，但 Init **从不写入** `empty`；立项成功即 `init_done`。下一阶段由 `nova plan` 改为 `planning`。

**设定模板文件**（按题材）：

| 题材 | 文件（经 `SettingFileSubdir` 落入子目录） |
|------|------------------------------------------|
| 玄幻（默认/未知题材回退） | 世界观、力量体系、金手指 → `世界/`；主角卡、反派设计 → `角色/` |
| 都市 | 世界观、金手指 → `世界/`；主角卡 → `角色/`；势力关系 → `势力/` |
| 科幻 | 世界观、科技体系 → `世界/`；主角卡 → `角色/`；势力关系 → `势力/` |

模板内容：统一页眉（书名/题材/风格）；`主角卡.md` / `金手指.md` 用 Meta 预填姓名/能力；其余 `## 待补充`。

**大纲模板：**

| 文件 | 内容要点 | 后续是否被 Enrich 覆盖 |
|------|----------|------------------------|
| `大纲/总纲.md` | 梗概、冲突、主线、创作目标、分卷表、基调 | **是**（整文件重写） |
| `大纲/爽点规划.md` | 大/中/微爽点、追读力空结构 | **否** |

### 3.4 Discover 附加：`SaveDiscoveryNotes`

仅 `--discover` 且骨架成功后：

1. 写 `.nova/discovery/notes.md`（pitch、synopsis、transcript）。
2. 若有 pitch/synopsis，用 `replaceSection` 改总纲的「一句话梗概」「核心冲突」。

注意：CLI 的 `extractDiscoverResult` **没有**把 `DiscoverResult.Synopsis` 填进 `InitInput.Synopsis`，故 `nova.yaml` 的 synopsis 常为空；梗概主要在 notes / 总纲补丁。随后若跑 Enrich，总纲会被 LLM **整文件覆盖**，补丁可能丢失（notes.md 仍保留）。

### 3.5 DB 元数据与「当前项目」

```go
st.InitProject(root, meta)   // INSERT OR REPLACE project_meta
project.SetCurrentProject(root)
  // → ~/.config/nova/current  （或 $NOVA_HOME/config/current）
```

`project_meta` 只镜像 root/title/genre/phase/updated_at；字数、主角等以 `nova.yaml` 为准。

### 3.6 分支：跳过完善 vs Enrich

| 条件 | 行为 |
|------|------|
| `--skip-llm` | 打印路径，退出；无 LLM、无记忆回填 |
| 无 API Key | **软成功**：提示骨架已创建，退出 0；不 Enrich |
| 有 Key | `LoadContext` → `NewInitWorkflow` → `EnrichSettings` → `BootstrapFromSettings` → `Report.Print` |

Discover 路径不允许 skip-llm，但无 Key 在探讨阶段已硬失败。

### 3.7 `EnrichSettings` —— 两次 LLM

`NewInitWorkflow`：`BindProject` + `agent.New`。Enrich 内 **`Tools` 默认 false**（工具绑定备而不用）；写文件靠解析模型输出。

**调用 1 — 完善设定集**

- System：`InitSystem(genre)`（立项助手：结构清晰、冲突潜力、金手指有边界等）。
- User：书名/题材/主角/金手指/基调 + Walk 得到的相对路径列表。
- 要求模型用 `===FILE:路径===` 分隔输出各文件完整 Markdown。
- Go：`writeSplitFiles` 按标记 `Split`，对每个 path `WriteFile(root/path)`（错误被忽略）。

路径须与磁盘相对路径一致（如 `设定集/世界/世界观.md`）；格式不对则该文件保持模板。

**调用 2 — 生成总纲**

- 同一 `InitSystem`。
- User：基于书名题材生成总纲（含分卷规划表）。
- 成功：覆盖写 `大纲/总纲.md`。
- 失败：返回 `StatusPartial`（设定可能已写；总纲仍是模板或 Discover 补丁版），err=nil。

**不触碰** `爽点规划.md`、不改 `nova.yaml` phase。

### 3.8 `BootstrapFromSettings`

Walk `设定集/**/*.md`，每文件：

- `category=world`，`subject=文件名去后缀`，`content` 截断约 600 rune。
- `UpsertMemory`；仅 **新插入** 计入返回条数。
- `SourceChapter=0`。

仅 Enrich 成功路径之后执行；`--skip-llm` / 无 Key 时 memories 表为空。

---

## 4. Discover 会话细节

`RunDiscoverySession(ctx, cfg, seedGenre)`：

1. 无工具 Agent；messages 以 `DiscoverSystem` 开头（短问答顾问）。
2. 若 seedGenre ≠ `玄幻`，先塞一条用户种子消息并 Chat 一轮。
3. 循环读 stdin：
   - 普通文本 → 追加 user → `Chat` → 打印顾问回复。
   - `/done` → `extractDiscoverResult`。
   - `/quit` → 取消，不建项目。
   - `/help` → 打印命令。
4. 提炼：`DiscoverExtractSystem` + 对话 transcript → JSON → `DiscoverResult`。
5. 映射为 `InitInput`（title/genre/style/protagonist/cheat；**不含 synopsis**）。
6. 打印摘要，stdin 确认；`n` 则取消。

CLI 强制 `in.Dir = initDir`（探讨结果不带目录）。

---

## 5. 产出物清单

### 5.1 `--skip-llm`（或无 Key 软退出）后

```
{root}/
├── nova.yaml                 # phase=init_done
├── .nova/
│   ├── nova.db               # 表结构 + project_meta 1 行
│   └── backups/              # 空
├── 设定集/…/*.md             # 题材模板
├── 大纲/
│   ├── 总纲.md
│   └── 爽点规划.md
└── 正文/                     # 空
```

另：`~/.config/nova/current` 指向 root。

**不会创建：** 章节目录、`审查.md`/`摘要.md`、`run_ledger.json`、`.nova/discovery/`、memories 数据行。

### 5.2 `--discover` 额外

```
.nova/discovery/notes.md
```

总纲可能在 Enrich 前被 pitch/synopsis 补丁；Enrich 后通常被覆盖。

### 5.3 完整 Enrich 后额外变化

| 产物 | 变化 |
|------|------|
| `设定集/**/*.md` | 被 LLM 内容覆盖（解析成功的文件） |
| `大纲/总纲.md` | 整文件重写 |
| `memories` 表 | 每设定文件最多 1 条 world 记忆 |
| Report | `completed` 或设定成功/总纲失败的 `partial` |

### 5.4 SQLite 初始行

| 表 | Init 后行数 |
|----|-------------|
| `project_meta` | 1 |
| `schema_meta` 等 migrate 元数据 | 按 migrate |
| chapters / entities / foreshadows / reviews / cool_points / … | 0 |
| memories | 0；Enrich+Bootstrap 后 ≈ 设定 md 数 |

---

## 6. 数据流

```
InitInput
   │
   ├──────────────────────────────► nova.yaml (Meta 真源)
   │
   ├─ genre ──► DefaultSettingFiles
   │                 │
   │                 ▼
   │            设定集/{子目录}/模板.md
   │                 │
   │                 │  (Enrich 调用 1)
   │                 ▼
   │            ===FILE:=== 解析写回同一路径
   │                 │
   │                 ▼
   │            BootstrapFromSettings → memories
   │
   ├─ synopsis/protagonist/cheat ──► 模板预填 / 总纲模板
   │                                      │
   │                    Discover 补丁 ────┤
   │                                      │ (Enrich 调用 2)
   │                                      ▼
   │                                 大纲/总纲.md（覆盖）
   │
   └─ Abs(Dir) ──► store.Open → migrate
                        │
                        ▼
                   project_meta 镜像
                        │
                        ▼
                   ~/.config/nova/current
```

下游消费：

```
init_done + 总纲 + 设定集
        │
        ▼
nova plan → 读 总纲.md + 设定集 Walk → 大纲/第NN卷.md → phase=planning
        │
        ▼
nova write → gate 要卷纲；Builder 再读设定/记忆（Bootstrap 种子）
```

---

## 7. Prompt 与 Agent 用法

| 场景 | System | Tools | 次数 |
|------|--------|-------|------|
| Discover 对话 | `DiscoverSystem` | 否（`Chat`） | N 轮 |
| Discover 提炼 | `DiscoverExtractSystem` | 否 | 1 |
| Enrich 设定 | `InitSystem(genre)` | 否 | 1 |
| Enrich 总纲 | `InitSystem(genre)` | 否 | 1 |

`===FILE:路径===` 协议是 Init Enrich 的核心约定：模型不调用 `write_file`，由 Go 解析后落盘——与 Plan「只读工具 + Go 写卷纲」、Write「Builder 预注入 + Go 写正文」同一哲学，但 Init 用**多分隔符 blob**，解析脆弱性更高。

---

## 8. 端到端调用链

```text
initCmd.RunE
  → 校验 --dir / 互斥 flags
  → [discover] RequireAPIKey → RunDiscoverySession
        → Chat 循环 → extractDiscoverResult → 确认
  → [interactive] runInteractiveInit
  → 默认值归一
  → store.Open(.nova/nova.db) → migrate
  → project.InitProject
        → MkdirAll 目录树
        → Save nova.yaml (phase=init_done)
        → 写设定模板 / 总纲 / 爽点规划
  → [discover] SaveDiscoveryNotes
  → st.InitProject(project_meta)
  → SetCurrentProject
  → [skip-llm] return
  → RequireAPIKey？无 → 软退出
  → LoadContext → NewInitWorkflow
  → EnrichSettings
        → Walk 设定路径
        → Agent.Run(InitSystem) → writeSplitFiles
        → Agent.Run(InitSystem) → WriteFile(总纲)
  → BootstrapFromSettings
  → Report.Print
```

---

## 9. 失败与边界

| 情况 | 行为 |
|------|------|
| 目录已有 `nova.yaml` | `InitProject` 硬失败；可能已有空 db |
| Discover `/quit` 或确认 n | 硬失败，不建项目 |
| Discover 无 API Key | 硬失败（探讨前） |
| 普通 init 无 API Key | 骨架成功，软提示 |
| Enrich 调用 1 失败 | 硬 error（整次 Enrich 失败） |
| Enrich 调用 2 失败 | `partial` Report，设定可能已改 |
| `writeSplitFiles` 路径无效 | 静默跳过该块 |
| Bootstrap 失败 | CLI 忽略 error（`n, _ :=`） |

无事务回滚：骨架写盘后 Enrich 失败，项目仍可用，可手改设定或日后补完善。

---

## 10. 实现取舍

1. **骨架与 LLM 解耦**：离线 / 无 Key 也能立项；LLM 是增强而非门槛（Discover 除外）。
2. **Phase 一步到位 `init_done`**：status/doctor 可立即建议 `nova plan 1`。
3. **题材驱动小模板集**：未知题材回退玄幻；子目录服务后续设定分类 UI/检索。
4. **Enrich 不开 tools**：用 `===FILE:===` 批量写；简单但依赖模型格式纪律。
5. **爽点规划不参与 Enrich**：留给作者或后续流程；总纲与设定优先。
6. **CLI 先 Open DB 再 InitProject**：与「先有 yaml 再有 db」的直觉相反，但保证立项成功时库已就绪。
7. **Discover 不把 synopsis 写入 InitInput**：yaml synopsis 可能空；依赖 notes / 总纲补丁 / Enrich。
8. **记忆回填只在 Enrich 后**：skip-llm 项目写章前无 world 种子记忆，仍可靠设定文件本身。

---

## 11. 产物目录速查（玄幻示例）

```
my-novel/
├── nova.yaml
├── .nova/
│   ├── nova.db
│   ├── backups/
│   └── discovery/notes.md    # 仅 --discover
├── 设定集/
│   ├── 角色/主角卡.md, 反派设计.md
│   ├── 世界/世界观.md, 力量体系.md, 金手指.md
│   ├── 势力/ 地点/ 物品/ 其他/   # 空目录
├── 大纲/
│   ├── 总纲.md
│   └── 爽点规划.md
└── 正文/
```

---

## 12. 相关文档

- [Plan 底层实现](./plan.md) — Init 的直接下游（消费总纲与设定）
- [Write 底层实现](./write.md) — 写章 gate / 上下文如何继续用设定与记忆
- [Memory](./memory.md) — Bootstrap 与写章召回的记忆模型
