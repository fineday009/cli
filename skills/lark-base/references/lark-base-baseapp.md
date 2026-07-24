# BaseApp（应用模式）操作指引

> 先读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。接口和组件字段协议以 [App CLI RPC 协议](https://bytedance.larkoffice.com/docx/M2D4dKijgoVKP1xVxuhcvu7tn0v) 为唯一事实来源；不要从组件名称推断额外约束。

## Token 与命令

| 对象 | 标识 | 命令 |
|---|---|---|
| Workspace | `workspace_token` | `+workspace-create` / `+workspace-entity-list` / `+workspace-move-in` / `+workspace-entity-remove` |
| BaseApp | `app_token` | `+app-create/get/rename`；删除见下方 |
| Base | `base_token` | `+app-create` 返回；表、字段、记录命令使用它 |
| Page | `page_id` | `+app-page-list/get/create/update/delete` |
| Block | `block_id` | `+app-block-list/get/create/update` |

页面和组件命令使用 `app_token`；Base 数据命令使用 `base_token`。唯一例外是 `+app-block-get-data`，它使用 `base_token + block_id`。

## 创建应用

```bash
lark-cli base +app-create \
  --name "销售应用" \
  --workspace-token <workspace_token> \
  --as user
```

- `+app-create` 没有 `--base-token`。
- App 创建成功后，CLI 创建一个空 Base，并将其移入 App 所在 Workspace，作为备选关联 Base。
- 可用 `--base-name` 和 `--table-name` 设置新建 Base 与首张表的名称；不传则使用默认名称。
- 记录输出中的 `app_token`、`base_token` 和 `workspace_token`。

### 部分完成后的续跑

如果 App 已创建而 Base 创建或移动失败，CLI 返回 `status=partial`、`failed_step`、已得到的 token、说明和 `retry.command`。此时：

1. 明确告诉用户 App 已创建且不会回滚。
2. 不要再次执行 `+app-create`。
3. 用户要求继续时，执行输出中的 `retry.command`。
4. `failed_step=base_create` 时，先重试 `+base-create`，再用 `+workspace-move-in` 移入同一 Workspace。
5. `failed_step=base_move` 时，只重试 `+workspace-move-in`，不要重复创建 Base。

## 重命名应用

```bash
lark-cli base +app-rename --app-token <app_token> --name "新名称"
```

BaseApp 与 Base 在 Drive 文件接口中都使用 `type=bitable`。`+app-rename` 复用 Drive `files patch`；它不会重命名关联 Base。

## 删除应用

```bash
lark-cli drive +delete --file-token <app_token> --type baseapp --yes
```

- 删除 BaseApp 应用本体需要切到 `lark-drive`。
- CLI 参数对外使用 `--type baseapp` 表达意图；底层 Drive 请求发送 `type=bitable`，和删除 Base 本体一致。
- 这是高风险写操作；执行前先确认 `app_token` 来自 `+app-get` 或 `+workspace-entity-list`。

## Page

```bash
lark-cli base +app-page-list --app-token <app_token>
lark-cli base +app-page-create --app-token <app_token> --name "总览" --to-last
lark-cli base +app-page-update --app-token <app_token> --page-id <page_id> --name "经营总览"
lark-cli base +app-page-delete --app-token <app_token> --page-id <page_id> --yes
```

- 同一 App 内 Page 名称必须唯一。创建或更新名称前，CLI 会读取页面列表；更新时排除当前 Page。
- `--prev-page-id` 与 `--to-last` 互斥。
- 本期没有 Page arrange，也没有 Block delete；Block 的 `type/sub_type` 创建后不可修改。

## 列表组件

创建列表时使用 `--type list` 与 `--sub-type standard|grouped|collapsible|card|detail`。省略 `--sub-type` 时默认 `standard`。

```bash
lark-cli base +app-block-create \
  --app-token <app_token> \
  --page-id <page_id> \
  --name "待处理订单" \
  --type list \
  --sub-type standard \
  --data-config '{"base_token":"<base_token>","table_name":"订单","columns":[]}'
```

- `data_config.base_token` 是单值：每个列表最多选择一个 Base。
- Base 必须在当前 App 的同一个 Workspace；CLI 写入前校验。
- 完整字段协议读 [lark-base-baseapp-block-data-config.md](lark-base-baseapp-block-data-config.md)。

## 更新组件

`+app-block-update` 只发送显式传入的 `data_config` 字段。未传字段保持不变；数组或对象字段是否整体替换，以 RPC 协议为准。不要为了“补全”先读取并提交全量配置。

## 常见恢复

| 现象 | 动作 |
|---|---|
| `status=partial` | 告知已完成/失败步骤；用户要求继续时执行 `retry.command` |
| Page 重名 | 先 `+app-page-list`，选择唯一名称后重试 |
| 列表 Base 不在同一 Workspace | 用 `+workspace-entity-list` 核对；选择同 Workspace Base |
| 列表协议校验失败 | 读取组件协议文档；不要推断 title、group_by 数量或 field role |
| Block 类型选错 | 本期无法删除且类型不可改，只能在 UI 处理后重新创建 |
