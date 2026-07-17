# Drive 评论指南

> **前置条件：** 认证、全局参数与高风险确认协议见 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。本文只做场景引导和跨命令口径；单个命令的参数与行为细节以“意图路由”表所指的专题文档为准。

## 何时读取

- 用户要对云文档评论做任何操作：添加评论、查评论列表、按 ID 批量取评论、回复、获取/更新/删除回复、解决/恢复评论、给回复加/删表情回应。
- 用户要统计评论数/回复数、找“最新/最早评论”，或根据评论定位文档正文位置。

## 意图路由

| 用户意图 | 优先命令 | 详见 |
|---|---|---|
| 添加评论（全文 / 局部 / review 逐条批注） | `drive +add-comment` | [`lark-drive-add-comment.md`](lark-drive-add-comment.md) |
| 获取评论列表、全量统计、找最新/最早评论 | `drive +list-comments` | [`lark-drive-list-comments.md`](lark-drive-list-comments.md) |
| 已知评论 ID 批量查询 | `drive +batch-query-comments` | [`lark-drive-comment-ops.md`](lark-drive-comment-ops.md) |
| 解决 / 恢复（重新打开）评论 | `drive +resolve-comment` / `drive +restore-comment` | [`lark-drive-comment-ops.md`](lark-drive-comment-ops.md) |
| 回复评论、获取 / 更新 / 删除回复 | `drive +add-reply` / `+list-replies` / `+update-reply` / `+delete-reply` | [`lark-drive-comment-ops.md`](lark-drive-comment-ops.md) |
| 给回复加 / 删表情回应，查询 reaction | `drive +react-reply`、查询加 `--need-reaction` | [`lark-drive-reactions.md`](lark-drive-reactions.md) |
| 根据评论定位正文位置（仅 docx） | `+list-comments --need-relation` / `+batch-query-comments --need-relation` | [`lark-drive-comment-location.md`](lark-drive-comment-location.md) |

- 优先使用上表的 shortcut，不要优先手写 `drive file.comments *` / `drive file.comment.replys *` 原生命令；只有需要 shortcut 未暴露的字段时才用原生命令兜底（先 `lark-cli schema` 查契约）。
- 查询默认口径：`drive +list-comments` 默认仅查未解决评论；即使用户说“所有评论”“全部评论”，只要没明确提到包含已解决评论，仍按默认口径，明确要求时才传 `--solved-status all`。
- 回复、更新回复、解决恢复、删除回复都有产品级限制（全文评论不可回复、根回复即评论正文等），操作前先读 [`lark-drive-comment-ops.md`](lark-drive-comment-ops.md) 对应命令一节。

## 目标定位通用规则

- 评论家族命令的目标定位方式一致：传 `--url`（包括 wiki URL）或 `--token` + `--type`；wiki 场景内部会自动解析出真实资源的 type 和 token。
- 支持 `doc`/`docx`/`sheet`/`file`/`slides`/`bitable`/`apps`；Base 传 `/base/` URL 或 `--type bitable`（`base` 为兼容别名）；妙搭 apps 传 `/page/<token>` URL 或裸 token + `--type apps`。
- 例外：`+add-comment` 不支持 apps（妙搭不支持新增评论，其余评论管理能力支持 apps）。

## 评论卡片模型与统计口径

- 评论列表（`drive +list-comments` / 原生 `file.comments list`）返回的 `items` 是评论卡片列表，每个 `item` 对应用户界面中的一张评论卡片，不是平铺的互动消息列表。
- 创建第一条评论时会同时创建该卡片里的第一条 reply；真正承载正文的是 `item.reply_list.replies`，其中第一条 reply 在用户视角下就是这张卡片里的“评论本身”。
- 统计“评论数”或“评论卡片数”：统计 `items` 长度；全量统计时对所有分页返回的 `items` 长度累加。
- 统计“回复数”：统计所有 `item.reply_list.replies` 长度之和，再减去 `items` 长度。
- 统计“总互动数”：统计所有 `item.reply_list.replies` 长度之和，包含每张评论卡片里的首条评论。
- 如果 `item.has_more=true`，说明该评论卡片下还有更多回复未包含在当前返回中；需要继续用 `drive +list-replies --comment-id <id>` 分页拉全后，再做全量回复数或总互动数统计。

## 排序

- 只有当用户明确提到“最新评论”“最后评论”“最早评论”时，才需要按 `create_time` 排序。
- 排序前必须拉完所有评论分页，不能只取第一页。
- “最新评论”/“最后评论”：按 `create_time` 降序取第一条。
- “最早评论”：按 `create_time` 升序取第一条。
- 用户只说“第一条评论”时，直接使用 `drive +list-comments` 返回的第一条，不需要额外排序。
