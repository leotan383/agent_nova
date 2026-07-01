# nova

命令行长篇网文创作 Agent。提供从**初始化设定 → 卷纲规划 → 写章 → 审查 → 记忆沉淀 → 状态查询 → 可视化面板**的一条龙工作流，用 Markdown 存正文、SQLite 存索引与记忆，缓解 AI 长篇连载中的「遗忘」和「幻觉」问题。

设计参考 [webnovel-writer](https://github.com/lingfengQAQ/webnovel-writer)，采用更轻量的 **Markdown 真源 + SQLite 读模型** 架构。

## 环境要求

- Go 1.20+
- OpenAI 兼容 API（OpenAI / OpenRouter / 本地 Ollama 等）

## 安装

```bash
git clone <repo-url> agent_nova
cd agent_nova

# 编译
go build -o bin/nova ./cmd/nova/

# 可选：加入 PATH
export PATH="$PWD/bin:$PATH"
```

### Nova Studio 桌面端（Wails）

跨平台 GUI：小说书库、切换、工作室概览与章节阅读。详见 [docs/desktop.md](docs/desktop.md)。

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.7.1
cd desktop/frontend && npm install && cd ..
cd desktop && wails dev    # 开发
cd desktop && wails build  # 打包
```

> 桌面端 pinned 在 Wails **v2.7.1**，因 v2.8+ 的 `go.mod` 含 `toolchain` 指令，需要 Go 1.21+。本项目保持 Go 1.20。

## 配置

复制环境变量模板：

```bash
cp .env.example .env
```

| 变量 | 说明 |
|------|------|
| `OPENAI_API_KEY` | API 密钥（必填，写章/规划时需要） |
| `OPENAI_BASE_URL` | 兼容端点，如 `https://api.openai.com/v1` 或 Ollama 地址 |
| `NOVA_MODEL` | 默认模型，如 `gpt-4o` |
| `NOVA_HOME` | 可选，覆盖全局配置目录 |

也可用 CLI 持久化配置：

```bash
nova config set api_key sk-...
nova config set base_url https://api.openai.com/v1
nova config set model gpt-4o
nova config show
```

写章前建议跑一次预检：

```bash
nova preflight
nova preflight --format json
```

## 快速开始

### 1. 初始化新书

```bash
# 只有模糊想法？先和 AI 探讨方向（不必想好书名/主角）
nova init --dir ./my-novel --discover

# 表单式补全（字段可留空）
nova init --dir ./my-novel --genre 玄幻 --interactive

# 非交互，仅建骨架（不调 LLM）
nova init --dir ./my-novel --title "我的修仙路" --genre 玄幻 \
  --protagonist 林风 --cheat 签到系统 --skip-llm

# 绑定为当前默认项目
nova use ./my-novel
```

`--discover` 模式下与 AI 多轮对话，输入 `/done` 后自动提炼书名、题材、主角等并创建设定；探讨记录保存在 `.nova/discovery/notes.md`。

之后所有命令可在项目目录内直接运行，或用 `-p ./my-novel` 指定项目根目录。

### 2. 规划卷纲

```bash
nova plan 1          # 生成第 1 卷章纲 → 大纲/第01卷.md
nova plan 2-3        # 批量生成
nova plan show 1     # 查看卷纲
```

### 3. 写章

```bash
nova write 1                    # 完整流水线
nova write 1 --stream           # 流式输出正文到终端
nova write 45 --resume          # 断点续写
nova write 1-3                  # 连续多章
```

写章内部流程：**组装上下文 → 起草 → 审查/润色 → 生成摘要 → 沉淀记忆 → 更新索引 → 自动备份**。

### 3.1 改稿讨论（已写章节）

```bash
nova coach 3              # 与 AI 讨论第 3 章哪里不好、怎么改
nova coach 3 --stream     # /revise 时流式输出修改稿
```

交互命令：`/revise` 生成修改稿 · `/apply` 写回正文 · `/save` 保存草稿 · `/quit` 退出

### 4. 审查与查询

```bash
nova review 1
nova review 1-5
nova review show 1

nova query 林风
nova query 伏笔 --status open
nova status
nova status --focus urgency
```

### 5. 记忆与学习

```bash
nova learn "本章危机钩设计很有效，悬念拉满"

nova memory stats
nova memory query --subject 主角
nova memory dump
```

### 6. 可视化面板

```bash
nova dashboard --port 8765
# 浏览器打开 http://127.0.0.1:8765
```

面板为只读，展示进度、章节、实体、open 伏笔等。

## 推荐工作流

```bash
nova init --dir ./demo-novel --genre 玄幻 --interactive
nova use ./demo-novel
nova plan 1
nova write 1 --stream
nova review 1
nova query 主角
nova status
nova dashboard
```

## 项目目录结构

初始化后，每本书是一个独立 workspace：

```
my-novel/
├── nova.yaml              # 书名、题材、phase、当前卷/章
├── .nova/
│   ├── nova.db            # SQLite：章节索引、实体、伏笔、记忆、FTS
│   ├── run_ledger.json    # 写章断点续跑状态
│   └── backups/           # 自动/手动备份
├── 设定集/                # 世界观、主角卡、金手指等
├── 大纲/                  # 总纲、爽点规划、第NN卷.md
├── 正文/                  # 第001章-标题.md
├── 审查/                  # 第001章.review.md
└── 摘要/                  # 第001章.summary.md
```

## 命令参考

### 全局

| 命令 | 说明 |
|------|------|
| `nova init` | 初始化新书（`--discover` / `--interactive` / `--skip-llm`） |
| `nova use <path>` | 绑定当前默认项目 |
| `nova config set/show` | 管理 API 配置 |
| `nova doctor [--deep]` | 项目体检 |
| `nova preflight` | 写章前预检 |
| `nova status [--focus urgency]` | 创作健康报告 |
| `nova version` | 版本号 |

### 创作

| 命令 | 说明 |
|------|------|
| `nova plan [卷号]` | 生成卷纲 |
| `nova plan show [卷号]` | 查看卷纲 |
| `nova write [章号]` | 写章（支持 `--resume` `--stream` `--volume` `--continue-on-error`） |
| `nova coach [章号]` | 改稿讨论（`/revise` `/apply` `/save`，支持 `--stream`） |
| `nova review [范围]` | 审查章节 |
| `nova review show [章号]` | 查看审查报告 |

### 记忆与检索

| 命令 | 说明 |
|------|------|
| `nova query [关键词]` | 混合检索实体 / FTS / 记忆 |
| `nova learn [内容]` | 手动沉淀写作模式 |
| `nova memory stats/query/dump` | 长期记忆管理 |
| `nova memory bootstrap` | 从设定集回填初始记忆 |
| `nova memory conflicts` | 查看同 subject 记忆冲突 |
| `nova index stats/rebuild/embed` | FTS 索引 + 向量索引 |

### 运维

| 命令 | 说明 |
|------|------|
| `nova gate --chapter N --stage prewrite\|precommit\|postcommit` | 写章边界校验 |
| `nova backup create/list/restore` | 备份管理 |
| `nova context extract --chapter N` | 调试写章上下文 |
| `nova export [--format markdown\|epub]` | 导出合集 |

### 服务

| 命令 | 说明 |
|------|------|
| `nova dashboard [--port 8765]` | 只读 Web 面板 |
| `nova serve [--port 8787]` | 后台 Daemon（含异步写章 API + SSE） |

Daemon 异步写章：

```bash
# 启动
nova serve --port 8787

# 提交写章任务
curl -X POST http://127.0.0.1:8787/api/write -d '{"chapter":1,"volume":1}'

# SSE 订阅进度
curl -N "http://127.0.0.1:8787/api/write/write-1-xxx/events?stream=1"
```

## 全局参数

所有命令均支持：

```bash
-p, --project string   # 指定项目根目录（含 nova.yaml）
    --format text|json # 输出格式
    --debug            # 调试日志
```

示例：

```bash
nova -p ./my-novel status --format json
nova -p ./my-novel context extract --chapter 12 --format json
```

## 常见问题

**未找到 nova.yaml**

在项目目录内运行，或加 `-p` 指定路径，或先执行 `nova use ./my-novel`。

**OPENAI_API_KEY 未配置**

设置环境变量或 `nova config set api_key ...`。若只想建目录骨架，用 `nova init --skip-llm`。

**写章 gate 未通过**

```bash
nova gate --chapter 1 --stage prewrite
nova doctor
```

常见原因：未生成卷纲（`nova plan 1`）、上一章摘要缺失、项目 phase 未进入 writing。

**中断后续写**

```bash
nova write 12 --resume
```

## Nova Studio 桌面端

跨平台 GUI 覆盖书库、规划、写章、审查、改稿、记忆、Wiki、导出等核心流程。详见 [docs/desktop.md](docs/desktop.md)。

产品路线图见 [docs/roadmap.md](docs/roadmap.md)。

### CLI vs Desktop 能力对照

| 能力 | CLI | Desktop |
|------|-----|---------|
| 探讨立项 / 创建小说 | ✅ | ✅ |
| 卷纲规划 / AI 写章 | ✅ | ✅ |
| 审查 / Coach 改稿 | ✅ | ✅ |
| 记忆 / 伏笔 / 导出 | ✅ | ✅ |
| `nova learn` / backup / doctor | ✅ | — |
| 批量连续写章 | ✅ | — |
| 新手引导 / Token 用量 | — | ✅ |

## 开发与测试

```bash
go test ./...
go build -o bin/nova ./cmd/nova/
```

## 许可证

待定
