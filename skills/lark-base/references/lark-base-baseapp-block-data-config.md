# BaseApp Block `data_config`

本文件说明 BaseApp 组件 data_config 的 CLI 映射与操作约束，不复制完整字段 Schema。具体字段以服务端返回和校验为准。

## 类型映射

- 图表：`--type column|bar|line|pie|ring|area|combo|scatter|funnel|wordCloud|radar|statistics`
- 富文本：`--type richText`
- 列表：`--type list --sub-type standard|grouped|collapsible|card|detail`
- 列表省略 `--sub-type` 时默认 `standard`
- `type/sub_type` 创建后不可修改

## 列表配置

列表公共数据源字段为单值 `base_token` 和 `table_name`。每个列表最多关联一个 Base，且该 Base 必须位于 App 所在的同一 Workspace。

按服务端协议，各 subtype 使用以下字段组：

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

字段的具体对象结构与必填性直接查服务端协议，不在这里猜测或复制。

## 更新语义

```bash
lark-cli base +app-block-update \
  --app-token <app_token> --page-id <page_id> --block-id <block_id> \
  --data-config '{"filter":{"conjunction":"and","conditions":[]}}'
```

- CLI 只发送用户显式传入的字段。
- 未传字段由服务端保持不变。
- 不为 update 注入 create 默认值，不先读取后拼成全量配置。
- 数组/对象字段的替换粒度以服务端协议为准。

## 图表与富文本

**App 图表是多数据源结构（`ChartDataConfig`），与 Dashboard 的扁平单源结构不同。** 顶层用一个 `base_token`（所有数据源共用），`table_name` / `series` / `count_all` / `group_by` / `filter` 下沉到每个 `data_sources[]` 元素里；顶层另有可选的 `data_source_mode` 和 `sort`。每个数据源内部各字段的取值逻辑与 [dashboard-block-data-config.md](dashboard-block-data-config.md) 完全一致（`series[].rollup` 大写、`group_by[].sort` 小写等），CLI 对每个 `data_sources[]` 元素复用同一套规范化与校验。富文本按服务端协议使用 `richText` 配置，无数据源。

顶层参数：

| 参数 | 必填 | 取值 | 说明 |
|-|-|-|-|
| `base_token` | 是 | `string` | 数据所在 Base 的 token；所有数据源共用同一个值。App 命令不带 `--base-token`，只能写在 data_config 内 |
| `data_sources` | 是 | `ChartDataSourceConfig[]` | 有序数组，至少一项 |
| `data_source_mode` | 否 | `aggregate` / `compare` | `aggregate`（默认）在横轴聚合数据源；`compare` 按数据源拆分系列 |
| `sort` | 否 | `{type: group\|value\|record, order?: asc\|desc}` | 顶层排序；`statistics` 不允许 |

每个 `data_sources[]` 元素：`table_name`（必填）、`series` 与 `count_all=true` 二选一、`group_by`（最多 2 项，`statistics` 不允许）、`filter`。

```json
{
  "base_token": "A2f5boKjfazMzesI9zKbmugTc4T",
  "data_sources": [
    {
      "table_name": "数据表",
      "count_all": true,
      "group_by": [
        { "field_name": "文本", "mode": "integrated", "sort": { "type": "value", "order": "desc" } }
      ]
    }
  ]
}
```

对应命令（单数据源计数柱状图）：

```bash
lark-cli base +app-block-create \
  --app-token <app_token> --page-id <page_id> \
  --name "文本分布" --type column \
  --data-config '{"base_token":"A2f5boKjfazMzesI9zKbmugTc4T","data_sources":[{"table_name":"数据表","count_all":true,"group_by":[{"field_name":"文本","mode":"integrated","sort":{"type":"value","order":"desc"}}]}]}'
```

多数据源示例（两张表各出一条系列，按数据源拆分）：

```bash
lark-cli base +app-block-create \
  --app-token <app_token> --page-id <page_id> \
  --name "销售与成本" --type combo \
  --data-config '{"base_token":"bas_xxx","data_source_mode":"compare","data_sources":[{"table_name":"销售表","group_by":[{"field_name":"月份","sort":{"type":"group","order":"asc"}}],"series":[{"field_name":"销售额","rollup":"SUM"}]},{"table_name":"成本表","group_by":[{"field_name":"月份","sort":{"type":"group","order":"asc"}}],"series":[{"field_name":"成本","rollup":"SUM"}]}],"sort":{"type":"group","order":"asc"}}'
```

Update 语义：传入 `data_sources` 即全量替换整个有序数组；修改 `base_token` 时必须同时传入完整 `data_sources`。请求不得包含 `sub_type`（平滑/堆积/百分比等展示变体走产品默认值）。`position` 是独立 flag。具体请求字段仍以服务端协议为准。
