# BaseApp（应用模式）操作指引

> 先读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。接口和组件字段以服务端返回和校验为准；不要从组件名称推断额外约束。

## Token 与命令

| 对象 | 标识 | 命令 |
|---|---|---|
| Workspace | `workspace_token` | `+workspace-create` / `+workspace-entity-list` / `+workspace-move-in` |
| BaseApp | `app_token` | `+app-create/get`；重命名和删除见下方 |
| Base | `base_token` | `+base-create` 返回；表、字段、记录命令使用它 |
| Page | `page_id` | `+app-page-list/get/create/update/delete` |
| Block | `block_id` | `+app-block-list/get/create/update` |

页面和组件命令使用 `app_token`；Base 数据命令使用 `base_token`。`+app-block-get-data` 使用 `app_token + base_token + chart_token`：CLI 参数名仍为 `--block-id`，但必须传组件返回的 `chart_token`，不能传普通 `block_id`。请求路径与仪表盘图表数据接口相同，并通过 `rpc-persist-x-base-apptoken` 请求头传递 `app_token`。

## 查询应用

```bash
lark-cli base +app-get --app-token <app_token>
```

- 响应中的 `pages` 是页面摘要。
- `ref` 的结构是 `Base token -> 当前组件引用的 Table 名称数组`。需要操作被引用 Base 时，使用 `ref` 的 key 作为 `base_token`。
- `ref` 只描述当前组件已经引用的数据源；没有被组件引用的 Base 不会出现在其中。

## 创建 Workspace

```bash
lark-cli base +workspace-create \
  --name "AppMode-空白评测空间" \
  --as user
```

- 创建成功后，最终答复必须同时给出 `workspace_token` 和可点击访问链接。
- CLI 输出中的 `workspace_url` / `url` 是访问链接；如果服务端响应缺少 URL，CLI 会按 `/base/workspace/<workspace_token>` 回填。
- 若创建后又执行 `+workspace-entity-list` 验证空目录，最终答复仍必须保留创建结果里的访问链接；不要只报告回读的 `entities` / `has_more`。

## 创建应用

```bash
lark-cli base +app-create \
  --name "销售应用" \
  --workspace-token <workspace_token> \
  --as user
```

- `+app-create` 没有 `--base-token`。
- `--workspace-token` 必填；`+app-create` 只调用 App 创建接口，不创建 Workspace、Base，也不移动资源。
- `--theme-style` 可选，支持 `default|cloudBlue|fresh|softLight|future|technology`。
- 记录输出中的 `app_token` 和 `workspace_token`。

### 创建应用的自然语言编排

先根据用户是否指定 Workspace 和现有 Base 选择流程，再调用原子 shortcut：

| 用户提供的信息 | 执行流程 |
|---|---|
| Workspace + 现有 Base | 确认 Base 位于该 Workspace → `+app-create`；不创建备用 Base |
| Workspace，未指定 Base | `+app-create` → `+base-create` 创建空 Base → `+workspace-move-in` |
| 未指定 Workspace，指定现有 Base | 先确认该 Base 所属 Workspace；能确定时在该 Workspace 执行 `+app-create`，不能确定时请用户提供 Workspace；不创建备用 Base |
| Workspace 和 Base 都未指定 | `+workspace-create` → `+app-create` → `+base-create` 创建空 Base → `+workspace-move-in` |

应用模式的列表组件只能引用同一 Workspace 内的一个 Base。用户指定现有 Base 时，不要因为 `+app-create` 没有接收 `base_token` 就额外创建 Base；后续在组件 `data_config.base_token` 中引用该 Base。

多步编排中，每个成功的 shortcut 都会立即产生资源且不自动回滚。后续步骤失败时，明确报告已经成功创建的 Workspace、App 或 Base 及其 token；用户要求继续时，只重试失败步骤，不要重复创建已经成功的资源。

## 读取图表计算结果

```bash
lark-cli base +app-block-get-data \
  --app-token <app_token> \
  --base-token <base_token> \
  --block-id <chart_token>
```

- `--block-id` 的值必须取图表组件摘要中的 `chart_token`，不能使用组件的普通 `block_id`。
- `base_token` 使用当前图表组件 `data_config.base_token`；一个 App 引用多个 Base 时，不要从 `+app-get ref` 中任意选择一个 key。
- `page_id` 不参与请求。
- 返回协议与 `+dashboard-block-get-data` 完全一致。

## 重命名应用

```bash
lark-cli drive +rename --file-token <app_token> --type bitable --name "新名称"
```

BaseApp 与 Base 在 Drive 文件接口中都使用 `type=bitable`。重命名应用不会重命名它引用的 Base。

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
lark-cli base +app-page-create --app-token <app_token> --name "总览"
lark-cli base +app-page-update --app-token <app_token> --page-id <page_id> --name "经营总览"
lark-cli base +app-page-delete --app-token <app_token> --page-id <page_id> --yes
```

- 同一 App 内 Page 名称必须唯一。创建或更新名称前，CLI 会读取页面列表；更新时排除当前 Page。
- 本期 `+app-page-create` 只支持创建顶级 Page，不支持 PageGroup 归属参数。
- 同一 Page 内组件名称必须唯一。`+app-block-create` 会分页读取该 Page 的全部组件并在创建前检查重名。
- 本期没有 Page arrange，也没有 Block delete；Block 的 `type/sub_type` 创建后不可修改。详见[本期不支持的能力](#本期不支持的能力)。

## 本期不支持的能力

下列能力本期不存在。用户提出时，直接说明不支持并给出可选的替代方向，不要用 Dashboard 或其他域的同名能力顶替。

| 用户诉求 | 本期状态 | 正确动作 |
|---|---|---|
| 自动排版 / 重新布局 / 美化页面组件 | 没有 App page arrange | 直接告知不支持；不要调用 `+dashboard-arrange` |
| 删除页面组件 | 没有 App block delete | 直接告知不支持，只能在 UI 处理；不要调用 `+dashboard-block-delete` |
| 修改组件位置 / 大小 / 置顶 | 布局、位置、尺寸不属于公开 Create/Update 协议 | 直接告知不支持；不要用 `+app-block-update` 做空更新伪装成移动 |
| 修改已存在 App 的主题 | `--theme-style` 只在 `+app-create` 时生效 | 直接告知不支持；如确有必要，说明只能新建 App 时指定主题 |

`+dashboard-*` 命令只作用于 Base 内的仪表盘，`dashboard_id` 是 `blk` 开头、组件 ID 是 `cht` 开头；AppMode 的 `pge` 页面和 `wgt` 组件不属于它们的作用域。缺少能力时不要用这些命令试探，包括 `--help` 和 `--dry-run`：一次调用就是一次错误的能力归属判断。

## 列表组件

创建列表时使用 `--type list` 与 `--sub-type standard|grouped|collapsible|card|detail`。省略 `--sub-type` 时默认 `standard`。

```bash
lark-cli base +app-block-create \
  --app-token <app_token> \
  --page-id <page_id> \
  --name "待处理订单" \
  --type list \
  --sub-type standard \
  --data-config '{"base_token":"<base_token>","table_name":"订单"}'
```

- `data_config.base_token` 是单值：每个列表最多选择一个 Base。
- Base 必须在当前 App 的同一个 Workspace；CLI 写入前校验。
- 完整字段协议读 [lark-base-baseapp-block-data-config.md](lark-base-baseapp-block-data-config.md)。

## 更新组件

`+app-block-update` 只发送显式传入的 `data_config` 字段。未传字段保持不变；数组或对象字段是否整体替换，以服务端协议为准。不要为了“补全”先读取并提交全量配置。

## 常见恢复

| 现象 | 动作 |
|---|---|
| `status=partial` | 告知已完成/失败步骤；用户要求继续时执行 `retry.command` |
| Page 重名 | 先 `+app-page-list`，选择唯一名称后重试 |
| 组件重名 | 先 `+app-block-list`，为该 Page 内的新组件选择唯一名称后重试 |
| 列表 Base 不在同一 Workspace | 用 `+workspace-entity-list` 核对；选择同 Workspace Base |
| 列表协议校验失败 | 读取组件协议文档；不要推断 title、group_by 数量或 field role |
| Block 类型选错 | 本期无法删除且类型不可改，只能在 UI 处理后重新创建 |
| 用户要 arrange / 删组件 / 调位置 / 改主题 | 按[本期不支持的能力](#本期不支持的能力)直接告知不支持；不要改用 `+dashboard-*` 命令 |
