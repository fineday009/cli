# BaseApp Block `data_config`

接口信封、Widget 公共结构、列表字段与更新语义以 [App CLI RPC 协议](https://bytedance.larkoffice.com/docx/M2D4dKijgoVKP1xVxuhcvu7tn0v) 为唯一事实来源。本文件只说明 CLI 映射与操作约束，不复制 Schema，避免双份维护。

## 类型映射

- 图表：`--type column|bar|line|pie|ring|area|combo|scatter|funnel|wordCloud|radar|statistics`
- 富文本：`--type richText`
- 列表：`--type list --sub-type standard|grouped|collapsible|card|detail`
- 列表省略 `--sub-type` 时默认 `standard`
- `type/sub_type` 创建后不可修改

## 列表配置

列表公共数据源字段为单值 `base_token` 和 `table_name`。每个列表最多关联一个 Base，且该 Base 必须位于 App 所在的同一 Workspace。

按 RPC 协议，各 subtype 使用以下字段组：

- 公共：`base_token`、`table_name`、`filter`、`sort_by`
- `standard/grouped/collapsible`：`columns`、`group_by`
- `card`：`fields`、`card_config`
- `detail`：`fields`、`detail_config`

不要添加协议未定义的语义校验，尤其不要假设：

- detail/card 必须有 title；
- grouped/collapsible 必须或只能有一个 group_by；
- fields 存在 role 或 visible 属性。

未知顶层字段会被本地校验拒绝；只有确认 CLI 校验与最新协议不一致时才使用 `--no-validate`。

## 创建示例

```bash
lark-cli base +app-block-create \
  --app-token <app_token> --page-id <page_id> \
  --name "订单列表" \
  --type list --sub-type standard \
  --data-config '{"base_token":"<base_token>","table_name":"订单","columns":[]}'
```

字段的具体对象结构与必填性直接查 RPC 协议，不在这里猜测或复制。

## 更新语义

```bash
lark-cli base +app-block-update \
  --app-token <app_token> --page-id <page_id> --block-id <block_id> \
  --data-config '{"filter":{"conjunction":"and","conditions":[]}}'
```

- CLI 只发送用户显式传入的字段。
- 未传字段由服务端保持不变。
- 不为 update 注入 create 默认值，不先读取后拼成全量配置。
- 数组/对象字段的替换粒度以 RPC 协议为准。

## 图表与富文本

图表沿用 [dashboard-block-data-config.md](dashboard-block-data-config.md) 的既有协议与规范化逻辑。列表使用独立实现，不修改 Dashboard 校验器。富文本按 RPC 协议使用 `richText` 配置。

`show_title` 与 `position` 属于 Block 顶层映射；`--position` 是独立 flag。具体请求字段仍以 RPC 协议为准。
