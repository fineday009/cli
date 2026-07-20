# Drive 评论指南

> **前置条件：** 认证、全局参数与高风险确认协议见 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。本文是评论域的二级路由：只保留跨命令的通用知识，单个命令的参数与行为细节以“意图路由”表所指的独立 ref 为准。

## 何时读取

- 用户要对云文档评论做任何操作：添加评论、查评论列表、按 ID 批量取评论、回复、获取/更新/删除回复、解决/恢复评论、给回复加/删表情回应。
- 用户要统计评论数/回复数、找“最新/最早评论”，或根据评论定位文档正文位置。

## 意图路由

| 用户意图 | 命令 | 详见 |
|---|---|---|
| 添加评论（全文 / 局部 / review 逐条批注） | `drive +add-comment` | [`lark-drive-add-comment.md`](lark-drive-add-comment.md) |
| 获取评论列表、全量统计、找最新/最早评论 | `drive +list-comments` | [`lark-drive-list-comments.md`](lark-drive-list-comments.md) |
| 已知评论 ID 批量查询 | `drive +batch-query-comments` | [`lark-drive-batch-query-comments.md`](lark-drive-batch-query-comments.md) |
| 解决评论（标记已解决） | `drive +resolve-comment` | [`lark-drive-resolve-comment.md`](lark-drive-resolve-comment.md) |
| 恢复 / 重新打开已解决评论 | `drive +restore-comment` | [`lark-drive-restore-comment.md`](lark-drive-restore-comment.md) |
| 回复评论 | `drive +add-reply` | [`lark-drive-add-reply.md`](lark-drive-add-reply.md) |
| 获取某条评论下的回复 | `drive +list-replies` | [`lark-drive-list-replies.md`](lark-drive-list-replies.md) |
| 更新回复内容 | `drive +update-reply` | [`lark-drive-update-reply.md`](lark-drive-update-reply.md) |
| 删除回复 | `drive +delete-reply` | [`lark-drive-delete-reply.md`](lark-drive-delete-reply.md) |
| 给回复加 / 删表情回应 | `drive +react-reply` | [`lark-drive-react-reply.md`](lark-drive-react-reply.md) |

跨切面专题（不替代单命令 ref）：

- 写入类命令（`+add-comment` / `+add-reply` / `+update-reply`）的 `--content` 格式 → [`lark-drive-comment-content.md`](lark-drive-comment-content.md)
- reaction 查询规则、语义联想、完整 `reaction_type` 枚举 → [`lark-drive-reactions.md`](lark-drive-reactions.md)
- 根据评论定位 docx 正文位置（`need_relation`）→ [`lark-drive-comment-location.md`](lark-drive-comment-location.md)

## 通用原则

- 优先使用上表的 shortcut，不要优先手写 `drive file.comments *` / `drive file.comment.replys *` 原生命令；原生命令只作为“shortcut 未暴露字段”的兜底，用前先 `lark-cli schema <domain>.<resource>.<method>` 查契约。
- 查询默认口径：`drive +list-comments` 默认仅查未解决评论；即使用户说“所有评论”“全部评论”，只要没明确提到包含已解决评论，仍按默认口径，明确要求时才传 `--solved-status all`。要检查/操作已解决评论（如恢复、或回复前核对状态）时也要带 `--solved-status all` 或 `true`，否则会漏掉目标。

## 目标定位（全命令通用）

- 目标传 `--url`（包括 wiki URL）或 `--token` + `--type`；wiki 场景内部会自动解析出真实资源的 type 和 token。
- 支持 `doc`/`docx`/`sheet`/`file`/`slides`/`bitable`/`apps`；Base 传 `/base/` URL 或 `--type bitable`（`base` 为兼容别名）；妙搭 apps 传 `/page/<token>` URL 或裸 token + `--type apps`。
- 例外：`+add-comment` 不支持 apps（妙搭不支持新增评论，其余评论管理能力支持 apps）。

## 评论卡片模型与统计口径

- 评论列表（`drive +list-comments` / `drive +batch-query-comments`）返回的 `items` 是评论卡片列表，每个 `item` 对应用户界面中的一张评论卡片，不是平铺的互动消息列表。
- 创建第一条评论时会同时创建该卡片里的第一条 reply；真正承载正文的是 `item.reply_list.replies`，其中第一条 reply（根回复）在用户视角下就是这张卡片里的“评论本身”。更新根回复即改写评论正文；删除按 reply 逐条生效，卡片在最后一条回复被删时才消失（删整条评论见 [`lark-drive-delete-reply.md`](lark-drive-delete-reply.md)）。
- 根回复的定位只在 `drive +list-replies` 第一页（未传 `--page-token`）的 `items[0]` 成立；翻页后 `items[0]` 只是普通回复。
- 统计“评论数”或“评论卡片数”：统计 `items` 长度；全量统计时对所有分页返回的 `items` 长度累加。
- 统计“回复数”：统计所有 `item.reply_list.replies` 长度之和，再减去 `items` 长度。
- 统计“总互动数”：统计所有 `item.reply_list.replies` 长度之和，包含每张评论卡片里的首条评论。
- 如果 `item.has_more=true`，说明该评论卡片下还有更多回复未包含在当前返回中；需要继续用 `drive +list-replies --comment-id <id>` 分页拉全后，再做全量回复数或总互动数统计。

## 排序

- 只有当用户明确提到“最新评论”“最后评论”“最早评论”时，才需要按 `create_time` 排序。
- 排序前必须拉完所有评论分页，不能只取第一页。
- “最新评论”/“最后评论”：按 `create_time` 降序取第一条。“最早评论”：按 `create_time` 升序取第一条。
- 用户只说“第一条评论”时，直接使用 `drive +list-comments` 返回的第一条，不需要额外排序。
