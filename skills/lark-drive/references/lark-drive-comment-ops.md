# Drive 评论操作：批量查询、解决恢复与回复

> **前置条件：** 先阅读 [`../SKILL.md`](../SKILL.md) 的“评论能力入口”；评论卡片模型与统计口径见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；认证、全局参数和高风险确认协议见 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。

本文覆盖已知评论 ID 之后的七个操作命令：`drive +batch-query-comments`、`drive +resolve-comment`、`drive +restore-comment`、`drive +add-reply`、`drive +list-replies`、`drive +update-reply`、`drive +delete-reply`。给回复加/删表情回应（`drive +react-reply`）见 [`lark-drive-reactions.md`](lark-drive-reactions.md)。

## 目标定位（全家族通用）

- 七个命令的目标定位方式一致：传 `--url`（包括 wiki URL）或 `--token` + `--type`；wiki 场景内部会自动解析出真实资源的 type 和 token。
- 支持 `doc`/`docx`/`sheet`/`file`/`slides`/`bitable`/`apps`；Base 传 `/base/` URL 或 `--type bitable`（`base` 为兼容别名）；妙搭 apps 传 `/page/<token>` URL 或裸 token + `--type apps`。
- 评论 ID 来自 `drive +list-comments` 输出的 `items[].comment_id`；回复 ID 来自 `+list-replies` 输出的 `items[].reply_id`，或 `+list-comments` 输出的 `items[].reply_list.replies[].reply_id`。
- 所有命令都支持 `--dry-run` 预览调用链，不发真实请求。

## 批量查询评论 +batch-query-comments

```bash
lark-cli drive +batch-query-comments --url '<DOC_URL>' --comment-ids '<id1,id2>'

# docx 需要评论定位时
lark-cli drive +batch-query-comments --url '<DOCX_URL>' --comment-ids '<id>' --need-relation
```

- `--comment-ids` 支持逗号分隔或重复传，单次最多 100 个 ID。
- 需要 reaction 数据时加 `--need-reaction`；docx 需要评论定位时加 `--need-relation`（非 docx 静默忽略），定位字段解读见 [`lark-drive-comment-location.md`](lark-drive-comment-location.md)。
- 分页遍历、全量统计用 `drive +list-comments`（见 [`lark-drive-list-comments.md`](lark-drive-list-comments.md)）；本命令只按已知 ID 取。

## 解决 / 恢复评论 +resolve-comment / +restore-comment

```bash
lark-cli drive +resolve-comment --url '<DOC_URL>' --comment-id '<id>'
lark-cli drive +restore-comment --url '<DOC_URL>' --comment-id '<id>'
```

- `+resolve-comment` 发送 `is_solved=true`；`+restore-comment` 发送 `is_solved=false`。两个命令共享同一个 patch 端点，只是方向相反。
- 用户说“把这条评论标记为已处理/已完成/关闭”对应 `+resolve-comment`；“重新打开/取消解决/恢复”对应 `+restore-comment`。

## 回复评论 +add-reply

```bash
lark-cli drive +add-reply --url '<DOC_URL>' --comment-id '<id>' \
  --content '[{"type":"text","text":"回复内容"}]'
```

- `--content` 与 `+add-comment` 同格式（text / mention_user / link）；`type=text` 文本自动转义 `<`、`>`。
- 该 shortcut 走 `POST .../comments/:comment_id/replies` 回复端点。不要试图用"添加评论"接口（`POST .../comments`）传 body `comment_id` 来回复——尽管官方文档如此描述，该写法实际不会挂到目标评论下，而是创建一条新的独立评论。
- 回复前先检查目标评论状态：`is_whole=true` 的全文评论不支持回复，遇到时提示“全文评论不支持回复”；`is_solved=true` 的已解决评论不支持回复，遇到时提示“该评论已被解决，无法回复”。
- 当目标评论不能回复时，只提示限制，不要自动替用户寻找其他可回复评论。

## 获取回复 +list-replies

```bash
lark-cli drive +list-replies --url '<DOC_URL>' --comment-id '<id>'

# 分页续跑
lark-cli drive +list-replies --url '<DOC_URL>' --comment-id '<id>' \
  --page-size 100 --page-token '<NEXT_PAGE_TOKEN>'
```

- 分页：`--page-size`（1-100，默认 50）+ `--page-token`（取上次输出的 `page_token`，`has_more=true` 时继续拉）；需要 reaction 数据时加 `--need-reaction`；`--user-id-type open_id|union_id` 控制 `items[].user_id` 的返回形态（不传时服务端默认 open_id）。
- 根回复承载评论正文本身，是回复列表中创建最早的一条：仅第一页（未传 `--page-token`）的 `items[0]` 是根回复；翻页后（传了 `--page-token`）返回的 `items[0]` 只是普通回复，不要按位置当作根回复去更新或删除。
- 输出字段：`items[].reply_id` / `user_id` / `create_time` / `update_time` / `content.elements`，供 `+update-reply`、`+delete-reply` 使用。

## 更新回复 +update-reply

```bash
lark-cli drive +update-reply --url '<DOC_URL>' --comment-id '<id>' --reply-id '<id>' \
  --content '[{"type":"text","text":"新内容"}]'
```

- `--content` 与 `+add-comment` 同格式（text / mention_user / link），`type=text` 自动转义 `<`、`>`。
- 更新是整体替换：新 `content` 完全覆盖旧内容，没有局部修改语义。
- 只能更新当前身份自己创建的回复；更新他人回复会返回 API 错误 `1069303 forbidden`。执行前先用 `+list-replies` 核对 `items[].user_id`（默认返回 open_id，持有 union_id 时传 `--user-id-type union_id` 对齐后再比对）。
- 更新评论卡片的根回复（第一页 `items[0]`，即创建最早的一条 reply）等价于改写这条评论的正文本身；改写前先和用户确认改的是回复还是评论正文。

## 删除回复 +delete-reply

```bash
lark-cli drive +delete-reply --url '<DOC_URL>' --comment-id '<id>' --reply-id '<id>' --yes
```

- 高风险写操作：真实执行需要按 `lark-shared` 高风险审批协议确认后追加 `--yes`；删除不可恢复。
- 评论卡片的首条（根）reply 就是“评论本身”，删除根 reply 会删除整张评论卡片；删除前先和用户确认删的是回复还是整条评论。这也是删除整条评论的唯一 API 途径。

## 原生命令

`drive file.comments batch_query / patch` 与 `drive file.comment.replys list / create / update / delete` 是对应的原生命令，只在需要 shortcut 未暴露的字段时使用；先用 `lark-cli schema <domain>.<resource>.<method>` 查看契约。直接调原生命令时注意：Base 的 `file_type` 传 `bitable` 而不是 `base`；评论写入文本需要自行转义 `<`、`>`。

## 参考

- [lark-drive](../SKILL.md) -- 云空间（云盘/云存储）全部命令
- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论场景路由、卡片模型与统计/排序口径
- [lark-drive-list-comments](lark-drive-list-comments.md) -- 分页获取评论列表
- [lark-drive-reactions](lark-drive-reactions.md) -- reaction 查询规则与 `+react-reply`
- [lark-drive-comment-location](lark-drive-comment-location.md) -- `need_relation` 评论定位字段
