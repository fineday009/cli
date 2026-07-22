# drive +batch-query-comments

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理。

按评论 ID 批量获取评论卡片。已知 comment_id 时用它精确取；要分页遍历、全量统计或找最新/最早评论，用 [`lark-drive-list-comments.md`](lark-drive-list-comments.md)。

## 命令

```bash
# 按 ID 批量取（逗号分隔或重复 --comment-ids）
lark-cli drive +batch-query-comments --url '<DOC_URL>' --comment-ids '<id1>,<id2>'

# 需要 reaction 数据
lark-cli drive +batch-query-comments --url '<DOC_URL>' --comment-ids '<id>' --need-reaction

# docx 需要评论定位关系（非 docx 静默忽略）
lark-cli drive +batch-query-comments --url '<DOCX_URL>' --comment-ids '<id>' --need-relation

# 裸 wiki token
lark-cli drive +batch-query-comments --token '<WIKI_TOKEN>' --type wiki --comment-ids '<id>'
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-ids` | 是 | 评论 ID，逗号分隔或重复传，单次最多 100 个；来自 `drive +list-comments` 的 `items[].comment_id` |
| `--need-reaction` | 否 | 返回评论卡片上的 reaction 数据，见 [`lark-drive-reactions.md`](lark-drive-reactions.md) |
| `--need-relation` | 否 | docx 评论定位关系；仅 docx 生效，非 docx 静默忽略，见 [`lark-drive-comment-location.md`](lark-drive-comment-location.md) |

## 行为说明

- `--need-relation` 通过请求 **body** 发送（`+list-comments` 是 query param），只在解析后的目标是 docx 时发送；该参数未收录于平台 metadata，但服务端支持，返回 `items[].relation` 及块位置。
- 输出的 `items` 始终是 JSON 数组（服务端省略时归一化为 `[]`），外层补 `file_token`、`file_type`、`count`。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comments batch_query`（先 `lark-cli schema drive.file.comments.batch_query` 查契约）。直接调原生时 Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-list-comments](lark-drive-list-comments.md) -- 分页获取评论列表
- [lark-drive-comment-location](lark-drive-comment-location.md) -- `need_relation` 评论定位
