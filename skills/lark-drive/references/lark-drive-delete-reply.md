# drive +delete-reply

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理。

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
- 删除按 reply 逐条生效：删一条 reply 只删那一条，卡片在其**最后一条 reply 被删掉时**才消失。评论卡片的首条（根）reply 就是“评论本身”，所以只有当根回复是该卡唯一 reply（评论没有其它回复）时，删根回复才等于删掉整张评论卡片；卡片下还有其它回复时，删根回复只删根回复、卡片带着剩余回复继续存在。
- **删除整条评论没有专门的命令，需要用本命令删光该卡片下的所有回复**（先用 `drive +list-replies` 拉全回复 id）。删除前先和用户确认删的是某条回复还是整条评论。
- 需要 shortcut 未暴露的字段时才用原生 `drive file.comment.replys delete` 兜底（先 `lark-cli schema drive.file.comment.replys.delete` 查契约）；直接调原生时 Base 的 `file_type` 传 `bitable`。

## 输出

```json
{
  "file_token": "docx_token",
  "file_type": "docx",
  "comment_id": "<comment_id>",
  "reply_id": "<reply_id>",
  "deleted": true
}
```

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取回复与 reply_id
