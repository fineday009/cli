# drive +batch-query-comments

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理。

按评论 ID 批量获取评论卡片。已知 comment_id 时用它精确取；要分页遍历、全量统计或找最新/最早评论，用 [`lark-drive-list-comments.md`](lark-drive-list-comments.md)。

## 命令

```bash
# 推荐：传完整 URL（docx/doc/sheet/file/slides/base/apps 都可），逗号分隔或重复 --comment-ids
lark-cli drive +batch-query-comments --url "https://example.larksuite.com/docx/<DOCX_TOKEN>" --comment-ids '<id1>,<id2>'

# 电子表格 URL 保留 /sheets/ 路径，原样传入
lark-cli drive +batch-query-comments --url "https://example.larksuite.com/sheets/<SHEET_TOKEN>" --comment-ids '<id>'

# 妙搭 apps URL 使用 /page/<token>，识别为 file_type=apps
lark-cli drive +batch-query-comments --url "https://example.feishu.cn/page/<APPS_TOKEN>/" --comment-ids '<id>'

# wiki URL 自动解包到底层文档
lark-cli drive +batch-query-comments --url "https://example.larksuite.com/wiki/<WIKI_TOKEN>" --comment-ids '<id>'

# 裸 wiki token 必须显式声明 --type wiki
lark-cli drive +batch-query-comments --token "<WIKI_TOKEN>" --type wiki --comment-ids '<id>'

# 裸 token 需声明对应类型；不要默认当作 docx。这里以 sheet 为例
lark-cli drive +batch-query-comments --token "<SHEET_TOKEN>" --type sheet --comment-ids '<id>'

# 需要 reaction 数据加 --need-reaction；docx 需要评论定位加 --need-relation（非 docx 静默忽略）
lark-cli drive +batch-query-comments --url "https://example.larksuite.com/docx/<DOCX_TOKEN>" --comment-ids '<id>' --need-reaction --need-relation
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` | 与 `--token` 二选一 | 推荐入口。支持 doc/docx/sheet/file/slides/base/bitable/apps/wiki URL；apps 妙搭 URL 使用 `/page/<token>`；wiki URL 会自动解析到真实文档。 |
| `--token` | 与 `--url` 二选一 | 裸 token 或 URL。裸 token 必须搭配 `--type`；wiki token 使用 `--type wiki`。 |
| `--type` | 裸 token 时必填 | 传 token 对应类型：`doc`、`docx`、`sheet`、`file`、`slides`、`bitable`、`base`、`apps`、`wiki`。wiki token 使用 `wiki`；传 `base` 时，CLI 会按 `bitable` 类型处理。 |
| `--comment-ids` | 是 | 评论 ID，逗号分隔或重复传，单次最多 100 个；来自 `drive +list-comments` 的 `items[].comment_id` |
| `--need-reaction` | 否 | 返回评论卡片上的 reaction 数据，见 [`lark-drive-reactions.md`](lark-drive-reactions.md) |
| `--need-relation` | 否 | docx 评论定位关系；仅 docx 生效，非 docx 静默忽略，见 [`lark-drive-comment-location.md`](lark-drive-comment-location.md) |

## 行为说明

- `--need-relation` 通过请求 **body** 发送（`+list-comments` 是 query param），只在解析后的目标是 docx 时发送；该参数未收录于平台 metadata，但服务端支持，返回 `items[].relation` 及块位置。
- 输出的 `items` 始终是 JSON 数组（服务端省略时归一化为 `[]`），外层补 `file_token`、`file_type`、`count`。
- 需要 shortcut 未暴露的字段时才用原生 `drive file.comments batch_query` 兜底（先 `lark-cli schema drive.file.comments.batch_query` 查契约）；直接调原生时 Base 的 `file_type` 传 `bitable`。

## 输出

```json
{
  "file_token": "docx_token",
  "file_type": "docx",
  "items": [],
  "count": 0
}
```

`items` 是命中的评论卡片数组（外层补 `file_token`/`file_type`，wiki 输入再加 `wiki_token`）；`count` 是命中数。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-list-comments](lark-drive-list-comments.md) -- 分页获取评论列表
- [lark-drive-comment-location](lark-drive-comment-location.md) -- `need_relation` 评论定位
