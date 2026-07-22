# drive +add-reply

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理；`--content` 完整格式见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md)。

给已有评论添加一条回复。

## 命令

```bash
# 推荐：传完整 URL（docx/doc/sheet/file/slides/base/apps 都可）
lark-cli drive +add-reply --url "https://example.larksuite.com/docx/<DOCX_TOKEN>" --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'

# 电子表格 URL 保留 /sheets/ 路径
lark-cli drive +add-reply --url "https://example.larksuite.com/sheets/<SHEET_TOKEN>" --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'

# 妙搭 apps URL 使用 /page/<token>
lark-cli drive +add-reply --url "https://example.feishu.cn/page/<APPS_TOKEN>/" --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'

# wiki URL 自动解包
lark-cli drive +add-reply --url "https://example.larksuite.com/wiki/<WIKI_TOKEN>" --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'

# 裸 wiki token 必须显式声明 --type wiki
lark-cli drive +add-reply --token "<WIKI_TOKEN>" --type wiki --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'

# 裸 token 需声明对应类型（以 sheet 为例）
lark-cli drive +add-reply --token "<SHEET_TOKEN>" --type sheet --comment-id '<id>' --content '[{"type":"text","text":"回复内容"}]'
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` | 与 `--token` 二选一 | 推荐入口。支持 doc/docx/sheet/file/slides/base/bitable/apps/wiki URL；apps 妙搭 URL 使用 `/page/<token>`；wiki URL 会自动解析到真实文档。 |
| `--token` | 与 `--url` 二选一 | 裸 token 或 URL。裸 token 必须搭配 `--type`；wiki token 使用 `--type wiki`。 |
| `--type` | 裸 token 时必填 | 传 token 对应类型：`doc`、`docx`、`sheet`、`file`、`slides`、`bitable`、`base`、`apps`、`wiki`。wiki token 使用 `wiki`；传 `base` 时，CLI 会按 `bitable` 类型处理。 |
| `--comment-id` | 是 | 要回复的评论 ID；来自 `drive +list-comments` 的 `items[].comment_id` |
| `--content` | 是 | `reply_elements` JSON，`type=text` 文本自动转义；完整 schema、mention_user/link、10000 字符限制见 [`lark-drive-comment-content.md`](lark-drive-comment-content.md) |

## 回复限制

- `is_whole=true` 的全文评论、`is_solved=true` 的已解决评论都不能回复。
- 目标的 `is_whole` / `is_solved` 通常在上一步 `+list-comments` / `+batch-query-comments` 的结果里已有，据此判断即可；信息不足时再补查一次。
- 命中限制时如实提示（“全文评论不支持回复” / “该评论已被解决，无法回复”），不要自动替用户改回复到别的评论。

## 行为说明

- 需要 shortcut 未暴露的字段时才用原生 `drive file.comment.replys create` 兜底（先 `lark-cli schema drive.file.comment.replys.create` 查契约，按 schema 拼 body）；直接调原生时需自行转义文本，Base 的 `file_type` 传 `bitable`。

## 输出

```json
{
  "file_token": "docx_token",
  "file_type": "docx",
  "comment_id": "<comment_id>",
  "created": true,
  "reply_id": "<reply_id>"
}
```

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-comment-content](lark-drive-comment-content.md) -- `--content` 格式
- [lark-drive-batch-query-comments](lark-drive-batch-query-comments.md) -- 按 ID 查 is_whole/is_solved
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取回复
