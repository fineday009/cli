# drive +delete-reply

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理。

删除某条回复。**高风险写操作**：真实执行需要按 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 的高风险审批协议向用户确认后追加 `--yes`；删除不可恢复。

## 命令

```bash
# 预览（不需要 --yes）；传完整 URL（docx/doc/sheet/file/slides/base/apps 都可）
lark-cli drive +delete-reply --url "https://example.larksuite.com/docx/<DOCX_TOKEN>" --comment-id '<id>' --reply-id '<id>' --dry-run

# 电子表格 URL 保留 /sheets/ 路径
lark-cli drive +delete-reply --url "https://example.larksuite.com/sheets/<SHEET_TOKEN>" --comment-id '<id>' --reply-id '<id>' --dry-run

# 妙搭 apps URL 使用 /page/<token>
lark-cli drive +delete-reply --url "https://example.feishu.cn/page/<APPS_TOKEN>/" --comment-id '<id>' --reply-id '<id>' --dry-run

# wiki URL 自动解包
lark-cli drive +delete-reply --url "https://example.larksuite.com/wiki/<WIKI_TOKEN>" --comment-id '<id>' --reply-id '<id>' --dry-run

# 裸 wiki token 必须显式声明 --type wiki
lark-cli drive +delete-reply --token "<WIKI_TOKEN>" --type wiki --comment-id '<id>' --reply-id '<id>' --dry-run

# 裸 token 需声明对应类型（以 sheet 为例）
lark-cli drive +delete-reply --token "<SHEET_TOKEN>" --type sheet --comment-id '<id>' --reply-id '<id>' --dry-run

# 确认后真实删除（把 --dry-run 换成 --yes）
lark-cli drive +delete-reply --url "https://example.larksuite.com/docx/<DOCX_TOKEN>" --comment-id '<id>' --reply-id '<id>' --yes
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` | 与 `--token` 二选一 | 推荐入口。支持 doc/docx/sheet/file/slides/base/bitable/apps/wiki URL；apps 妙搭 URL 使用 `/page/<token>`；wiki URL 会自动解析到真实文档。 |
| `--token` | 与 `--url` 二选一 | 裸 token 或 URL。裸 token 必须搭配 `--type`；wiki token 使用 `--type wiki`。 |
| `--type` | 裸 token 时必填 | 传 token 对应类型：`doc`、`docx`、`sheet`、`file`、`slides`、`bitable`、`base`、`apps`、`wiki`。wiki token 使用 `wiki`；传 `base` 时，CLI 会按 `bitable` 类型处理。 |
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
