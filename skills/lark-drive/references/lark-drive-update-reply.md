# drive +update-reply

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由与目标定位（URL/wiki/apps/base）；`--content` 完整格式见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md)。

整体替换某条回复的内容。

## 命令

```bash
lark-cli drive +update-reply --url '<DOC_URL>' --comment-id '<id>' --reply-id '<id>' \
  --content '[{"type":"text","text":"新内容"}]'
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-id` | 是 | 回复所属的评论 ID；来自 `drive +list-comments` |
| `--reply-id` | 是 | 要更新的回复 ID；来自 `drive +list-replies` 的 `items[].reply_id` |
| `--content` | 是 | 新的 `reply_elements` JSON，`type=text` 文本自动转义；完整 schema 见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md) |

## 行为说明

- 更新是整体替换：新 `content` 完全覆盖旧内容，没有局部修改语义。
- **只能更新当前身份自己创建的回复**；更新他人回复返回 API 错误 `1069303 forbidden`。执行前先用 `+list-replies` 核对 `items[].user_id`（默认 open_id，持有 union_id 时传 `--user-id-type union_id` 对齐），并用创建该回复的同一个 `--as` 身份执行。
- 更新评论卡片的根回复（第一页 `items[0]`，即创建最早的一条 reply）等价于改写这条评论的正文本身；改写前先和用户确认改的是回复还是评论正文。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comment.replys update`（先 `lark-cli schema drive.file.comment.replys.update` 查契约）。直接调原生时需自行转义文本、自行拼 `content.elements` wire 形态（见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md)），Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-comment-content](lark-drive-comment-content.md) -- `--content` 格式
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取回复与 reply_id
