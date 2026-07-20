# drive +delete-reply

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由与目标定位（URL/wiki/apps/base）。

删除某条回复。**高风险写操作**：真实执行需要按 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 的高风险审批协议向用户确认后追加 `--yes`；删除不可恢复。

## 命令

```bash
# 预览（不需要 --yes）
lark-cli drive +delete-reply --url '<DOC_URL>' --comment-id '<id>' --reply-id '<id>' --dry-run

# 确认后真实删除
lark-cli drive +delete-reply --url '<DOC_URL>' --comment-id '<id>' --reply-id '<id>' --yes
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-id` | 是 | 回复所属的评论 ID；来自 `drive +list-comments` |
| `--reply-id` | 是 | 要删除的回复 ID；来自 `drive +list-replies` 的 `items[].reply_id`，或 `drive +list-comments` 的 `items[].reply_list.replies[].reply_id` |
| `--yes` | 真实执行时是 | 高风险确认；`--dry-run` 预览不需要 |

## 行为说明

- 删除永久生效，回复没有回收站或撤销。
- 评论卡片的首条（根）reply 就是“评论本身”，删除根 reply 会删除整张评论卡片；删除前先和用户确认删的是回复还是整条评论。这也是删除整条评论的唯一 API 途径。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comment.replys delete`（先 `lark-cli schema drive.file.comment.replys.delete` 查契约）。直接调原生时 Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取回复与 reply_id
