# BaseApp（应用模式）与 Workspace 模块指引

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

BaseApp 是建立在 Base 之上的**应用模式**：数据仍然存在 Base 的表里，应用负责把数据组织成一个个**页面（Page）**，页面上摆放**组件（Block）**——图表、指标卡、列表和富文本。Workspace 是应用和 Base 的容器目录。

## 核心概念与 token

| 概念 | 公开标识 | 从哪里拿 |
|---|---|---|
| Workspace | `workspace_token` | `+workspace-create` 返回 |
| Base | `base_token` | `+baseapp-create` 返回，或 `+baseapp-get` 的 `base_tokens` |
| BaseApp | `app_token` | `+baseapp-create` / `+workspace-entity-list` 返回 |
| Page | `page_id` | `+baseapp-page-list` / `+baseapp-page-create` 返回 |
| Block | `block_id` | `+app-block-list` / `+app-block-create` 返回 |

一条铁律：**页面和组件命令吃 `app_token`，表/字段/记录命令吃 `base_token`**，两者不能互换。唯一例外是 `+app-block-get-data`，见下方「唯一的参数例外」。

## 能力速览

| 你想做什么 | 用这些命令 |
|---|---|
| 新建 Workspace | `+workspace-create` |
| 看 Workspace 里有哪些 Base / 应用 | `+workspace-entity-list` |
| 把已有 Base 或应用挂进 / 移出 Workspace | `+workspace-entity-add` / `+workspace-entity-remove` |
| 新建一个空白应用（含空白 Base 和空白表） | `+baseapp-create` |
| 看应用信息、页面列表 | `+baseapp-get`（可加 `--with-pages`） |
| 应用改名 | `+baseapp-rename` |
| 页面增删改查 | `+baseapp-page-list/get/create/rename/delete` |
| 页面上的组件增改查 | `+app-block-list/get/create/update` |
| 读图表组件的计算结果 | `+app-block-get-data`（注意参数是 `--base-token`） |
| 组件配置怎么写 | 读 [lark-base-baseapp-block-data-config.md](lark-base-baseapp-block-data-config.md) |

## 本期没有的能力

这些必须在第一次向用户解释方案时讲清楚，不要照搬仪表盘的用法：

- **没有 `+app-block-delete`**。组件建错只能到界面上删。
- **组件的 `type` 建后不可改**。加上没有删除命令，意味着**创建前必须确认类型选对**，否则只能人工到界面处理。
- **没有页面版本的 arrange**。页面布局由平台默认排布；`--position` 可以给建议位置，但没有「一键美化」命令。
- 不支持复制应用、复制页面、修改页面图标、角色成员管理、主题与外观配置。

## 典型工作流：从零搭一个应用

```bash
# 1. 建一个空白应用；平台会同时创建一个空白 Base 和一张空白表
lark-cli base +baseapp-create --name "销售看板" --workspace-token <workspace_token> --as user
# 记录返回里的 app_token 和 base_token —— 后面两条线都要用

# 2. 往 Base 里补数据结构（用 base_token，不是 app_token）
lark-cli base +table-list  --base-token <base_token>
lark-cli base +field-list  --base-token <base_token> --table-id <table_id>

# 3. 建页面（用 app_token）
lark-cli base +baseapp-page-create --app-token <app_token> --name "总览" --to-last
# 记录返回的 page_id

# 4. 串行创建组件；创建前先读 data-config 文档，确认 type 选对
lark-cli base +app-block-create \
  --app-token <app_token> --page-id <page_id> \
  --name "总销售额" --type statistics \
  --data-config '{"table_name":"订单表","series":[{"field_name":"金额","rollup":"SUM"}]}'

lark-cli base +app-block-create \
  --app-token <app_token> --page-id <page_id> \
  --name "月度趋势" --type line \
  --data-config '{"table_name":"订单表","series":[{"field_name":"金额","rollup":"SUM"}],"group_by":[{"field_name":"月份","mode":"integrated"}]}'

# 5. 验证：读图表计算结果（这里换回 base_token）
lark-cli base +app-block-get-data --base-token <base_token> --block-id <block_id>
```

要点：

- 组件**必须串行创建**，不要对同一个页面并发建多个组件。
- 表名和字段名必须来自 `+table-list` / `+field-list` 的真实返回，不要按用户口述猜。
- 建组件前先确认类型；本期没有删除命令，类型错了只能人工处理。

## 唯一的参数例外：`+app-block-get-data`

`+app-block-get-data` 直接复用仪表盘的计算端点，入参只有 `base_token` + `block_id`：

```bash
lark-cli base +app-block-get-data --base-token <base_token> --block-id <block_id>
```

- **不要传 `--app-token` / `--page-id`**，这条命令不需要它们（传了会被忽略）。
- `--base-token` 是应用背后的那个 Base，从 `+baseapp-get` 的 `base_tokens` 或 `+baseapp-create` 的返回里取。
- 返回的是图表计算协议 JSON，和 `+dashboard-block-get-data` 完全一致；要 block 的名称、类型、布局、`data_config` 用 `+app-block-get`。
- 列表组件和富文本组件没有计算结果，不要对它们调这条命令。

## 与仪表盘的边界

应用页面的 block 和仪表盘的 block 是同一套底层实体，但**两组命令的 ID 体系不通用**：

- `+dashboard-block-*` 用 `dashboard_id`，`+app-block-*` 用 `app_token` + `page_id`。
- 从 `+app-block-list` 拿到的 `block_id` 不要拿去打 `+dashboard-block-get`，反之亦然；混用会得到难以诊断的 4xx。
- 只有 `+app-block-get-data` 与 `+dashboard-block-get-data` 是同一个端点、同一套参数，这是有意为之。
- 图表类组件的 `data_config` 两边同构，所以仪表盘那套写法可以直接迁移过来；列表和富文本是应用模式独有的。

## Workspace 操作

```bash
# 新建
lark-cli base +workspace-create --name "增长团队"

# 看里面有什么（--type 可选 base / baseapp，不传则两类都列）
lark-cli base +workspace-entity-list --workspace-token <workspace_token> --type baseapp

# 把已有 Base 挂进来
lark-cli base +workspace-entity-add --workspace-token <workspace_token> --type base --token <base_token> --to-last

# 移出（高风险，需要 --yes）
lark-cli base +workspace-entity-remove --workspace-token <workspace_token> --entity-id <entity_id> --yes
```

- `--prev-entity-id` 和 `--to-last` 都控制排序，**只能二选一**。
- `+workspace-entity-remove` 只解除目录关系，**不会删除底层的 Base 或应用**；`--entity-id` 是 `+workspace-entity-list` 返回的 `entity_id`，不是 token。

## 常见错误与恢复

| 现象 | 恢复动作 |
|---|---|
| 对 `+app-block-*` 传了 `--base-token` 报缺 `--app-token` | 除 `get-data` 外都用 `app_token` + `page_id`；`base_token` 只给表/字段/记录命令 |
| `+app-block-get-data` 报缺 `--base-token` | 这条命令不吃 `app_token`；从 `+baseapp-get` 的 `base_tokens` 取 `base_token` |
| 组件类型选错 | 本期无删除命令且类型不可改，只能到界面上处理；下次创建前先确认类型 |
| `data_config 校验失败` | 按报错逐条修正，再读 [lark-base-baseapp-block-data-config.md](lark-base-baseapp-block-data-config.md)；确认本地校验误判时才用 `--no-validate` |
| 想重排页面布局 | 本期没有页面 arrange 命令；只能在创建时给 `--position`，或让用户到界面调整 |
