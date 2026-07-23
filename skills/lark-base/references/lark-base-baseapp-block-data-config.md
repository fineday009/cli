# BaseApp page block data_config SSOT

应用页面上的 block 有三类：图表类、富文本、列表类。图表类的 `data_config` 与仪表盘同构，直接复用 [dashboard-block-data-config.md](dashboard-block-data-config.md)；本文是应用页面 block 的 SSOT，重点写清三类的差异、列表类的独有结构，以及不可回退的约束。

## 支持的 `type` 枚举

| type 值 | 说明 | data_config 形态 |
|---|---|---|
| `column` | 柱状图 | 图表类 |
| `bar` | 条形图 | 图表类 |
| `line` | 折线图 | 图表类 |
| `pie` | 饼图 | 图表类 |
| `ring` | 环形图 | 图表类 |
| `area` | 面积图 | 图表类 |
| `combo` | 组合图 | 图表类 |
| `scatter` | 散点图 | 图表类 |
| `funnel` | 漏斗图 | 图表类 |
| `wordCloud` | 词云 | 图表类 |
| `radar` | 雷达图 | 图表类 |
| `statistics` | 指标卡 | 图表类 |
| `richText` | 富文本 | 富文本 |
| `standardList` | 普通列表 | 列表类 |
| `detailList` | 详情列表 | 列表类 |
| `cardList` | 卡片列表 | 列表类 |
| `groupedList` | 聚合列表 | 列表类 |
| `collapsibleList` | 折叠列表 | 列表类 |

**类型创建后不可修改，本期又没有删除组件的命令**，所以 `--type` 必须一次选对，否则只能到界面上人工处理。仪表盘的 `text` 在应用页面写作 `richText`。

## 图表类：复用仪表盘结构

```json
{
  "table_name": "订单表",
  "series": [{ "field_name": "金额", "rollup": "SUM" }],
  "count_all": false,
  "group_by": [{ "field_name": "月份", "mode": "integrated" }],
  "filter": { "conjunction": "and", "conditions": [] }
}
```

规则与仪表盘完全一致，逐条见 [dashboard-block-data-config.md](dashboard-block-data-config.md)：

- `series` 与 `count_all` 二选一、互斥；`rollup` 只支持 `SUM` / `MAX` / `MIN` / `AVERAGE`，计数用 `count_all`。
- `table_name`、`field_name` 用**名称**不用 ID，且必须来自 `+table-list` / `+field-list` 的真实返回。
- `group_by` 最多 2 个维度；给了 `sort` 就必须同时给 `order`（`asc` / `desc`）。
- `filter.conditions[].operator` 的取值和字段类型的对应关系见仪表盘文档的操作符速查表。

## 富文本：`richText`

```json
{ "text": "## 本月概览\n销售额环比增长 12%" }
```

`text` 必填，支持 Markdown。富文本组件没有数据源，不要写 `table_name` / `series` / `group_by` / `filter`。

## 列表类：应用模式独有结构

```json
{
  "table_id": "tblxxx",
  "view_id": "viwxxx",
  "fields": ["fldxxx", "fldyyy"],
  "filter": {
    "conjunction": "and",
    "conditions": [{ "field_id": "fldxxx", "operator": "is", "value": "进行中" }]
  },
  "sort": [{ "field_id": "fldxxx", "order": "asc" }],
  "display": {
    "show_title": true,
    "primary_field_id": "fldxxx"
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `table_id` | string | 数据表；与 `table_name` 至少提供其一 |
| `table_name` | string | 数据表名称，作为 `table_id` 的替代 |
| `view_id` | string | 可选。指定视图后列表继承该视图的可见字段与顺序 |
| `fields` | string[] | 可选。展示哪些字段，元素是字段 ID 或字段名 |
| `filter` | object | 可选。结构同图表类，条件里可以用 `field_id` 或 `field_name` |
| `sort` | object[] | 可选。每项要有 `field_id`（或 `field_name`）和 `order`（`asc` / `desc`） |
| `display` | object | 可选。列表 UI 配置，如 `show_title`、`primary_field_id` |

列表类**不接受** `series` / `count_all` / `group_by`：它展示的是记录行，不是聚合结果。要聚合请改用图表类组件或 `+data-query`。

字段引用优先用 ID：先跑 `+field-list` 拿到 `field_id` 再填，比字段名更稳（改名不会打断配置）。

## 标题显隐：`show_title`

`show_title` 写在 `--data-config` 里，CLI 会把它提到请求体顶层：

```bash
--data-config '{"table_name":"订单表","count_all":true,"show_title":true}'
```

读回时它也在 block 顶层（`+app-block-get` 的 `block.show_title`），不在 `data_config` 内。

## 位置：`--position`

`--position` 是独立 flag，不属于 `data_config`：

```bash
--position '{"x":0,"y":0,"w":12,"h":8}'
```

四个值都必须是数字。**不传就让平台自动排布**——本期没有页面级 arrange 命令，位置一旦不合适只能改 `--position` 重新 update 或到界面调整。

## 本地校验与 `--no-validate`

`+app-block-create` 在发请求前会按 `--type` 做本地校验：图表类走图表规则，列表类走列表规则，`richText` 只查 `text`。校验失败会直接报错并列出全部问题，不会发出请求。

`+app-block-update` 不带 `--type`，因此**只做归一化不做强类型校验**（`rollup` 转大写、`sort.type` / `order` 转小写），具体字段交给服务端验证。

确认是本地校验误判时才加 `--no-validate`：它会跳过校验和归一化，把 `data_config` 原样发出去。

## 可复制模板

```bash
# 指标卡：总记录数
--type statistics --data-config '{"table_name":"订单表","count_all":true}'

# 指标卡：求和
--type statistics --data-config '{"table_name":"订单表","series":[{"field_name":"金额","rollup":"SUM"}]}'

# 折线图：按月趋势
--type line --data-config '{"table_name":"订单表","series":[{"field_name":"金额","rollup":"SUM"}],"group_by":[{"field_name":"月份","mode":"integrated"}]}'

# 饼图：品类占比
--type pie --data-config '{"table_name":"订单表","series":[{"field_name":"金额","rollup":"SUM"}],"group_by":[{"field_name":"品类","mode":"integrated"}]}'

# 富文本
--type richText --data-config '{"text":"## 本月概览\n销售额环比增长 12%"}'

# 普通列表：按视图展示若干字段
--type standardList --data-config '{"table_id":"tblxxx","view_id":"viwxxx","fields":["fldxxx","fldyyy"]}'

# 卡片列表：带筛选和排序
--type cardList --data-config '{"table_id":"tblxxx","filter":{"conjunction":"and","conditions":[{"field_id":"fldstatus","operator":"is","value":"进行中"}]},"sort":[{"field_id":"fldtime","order":"desc"}]}'
```
