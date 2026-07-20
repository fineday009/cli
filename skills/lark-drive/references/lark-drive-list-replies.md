# drive +list-replies

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由、目标定位（URL/wiki/apps/base）与根回复（root reply）概念。

分页获取某条评论下的回复。

## 命令

```bash
lark-cli drive +list-replies --url '<DOC_URL>' --comment-id '<id>'

# 分页续跑
lark-cli drive +list-replies --url '<DOC_URL>' --comment-id '<id>' \
  --page-size 100 --page-token '<NEXT_PAGE_TOKEN>'

# 需要 reaction 数据
lark-cli drive +list-replies --url '<DOC_URL>' --comment-id '<id>' --need-reaction
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-id` | 是 | 评论 ID；来自 `drive +list-comments` 的 `items[].comment_id` |
| `--page-size` | 否 | 1-100，默认 50 |
| `--page-token` | 否 | 上次输出的 `page_token`；`has_more=true` 时用它续拉 |
| `--need-reaction` | 否 | 在回复上返回 reaction 数据，见 [`lark-drive-reactions.md`](lark-drive-reactions.md) |
| `--user-id-type` | 否 | 控制 `items[].user_id` 形态：`open_id`（默认）、`union_id` |

## 行为说明

- 根回复承载评论正文本身，是回复列表中创建最早的一条：**仅第一页（未传 `--page-token`）的 `items[0]` 是根回复**；翻页后（传了 `--page-token`）返回的 `items[0]` 只是普通回复，不要按位置当作根回复去更新或删除。
- 输出字段：`items[].reply_id` / `user_id` / `create_time` / `update_time` / `content.elements`，供 `+update-reply`、`+delete-reply` 使用。
- 检查回复归属（更新/删除前）：默认返回 open_id，持有 union_id 时传 `--user-id-type union_id` 对齐后再比对 `items[].user_id`。
- 输出的 `items` 始终是 JSON 数组（服务端省略时归一化为 `[]`）。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comment.replys list`（先 `lark-cli schema drive.file.comment.replys.list` 查契约）。直接调原生时 Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-update-reply](lark-drive-update-reply.md) -- 更新回复
- [lark-drive-delete-reply](lark-drive-delete-reply.md) -- 删除回复
- [lark-drive-reactions](lark-drive-reactions.md) -- reaction 查询与写入
