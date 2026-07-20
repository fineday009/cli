# drive +add-reply

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由与目标定位（URL/wiki/apps/base）；`--content` 完整格式见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md)。

给已有评论添加一条回复，走 `POST .../comments/:comment_id/replies` 回复端点。

## 命令

```bash
lark-cli drive +add-reply --url '<DOC_URL>' --comment-id '<id>' \
  --content '[{"type":"text","text":"回复内容"}]'
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-id` | 是 | 要回复的评论 ID；来自 `drive +list-comments` 的 `items[].comment_id` |
| `--content` | 是 | `reply_elements` JSON，`type=text` 文本自动转义；完整 schema、mention_user/link、10000 字符限制见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md) |

## 回复前先检查目标评论状态

`is_whole=true` 的全文评论、`is_solved=true` 的已解决评论都不能回复。回复前先确认目标可回复：

- 已知 comment_id：`drive +batch-query-comments --url '<DOC_URL>' --comment-ids '<id>'`，看返回项的 `is_whole` / `is_solved`。
- 用 `drive +list-comments` 查找目标：**必须带 `--solved-status all`**，否则默认只查未解决评论，会漏掉已解决的目标；再核对 `is_whole` / `is_solved`。

命中限制时的提示口径：全文评论 → “全文评论不支持回复”；已解决评论 → “该评论已被解决，无法回复”。当目标评论不能回复时，只提示限制，不要自动替用户改回复到别的评论。

## 为什么不用“添加评论”接口回复

不要用 `POST .../comments` 传 body `comment_id` 来回复——尽管官方文档如此描述，该写法实际不会挂到目标评论下，而是创建一条新的独立评论。本 shortcut 用的是专门的 replies 端点。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comment.replys create`（先 `lark-cli schema drive.file.comment.replys.create` 查契约）。直接调原生时需自行转义文本、自行拼 `content.elements` wire 形态（见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md)），Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-comment-content](lark-drive-comment-content.md) -- `--content` 格式
- [lark-drive-batch-query-comments](lark-drive-batch-query-comments.md) -- 按 ID 查 is_whole/is_solved
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取回复
