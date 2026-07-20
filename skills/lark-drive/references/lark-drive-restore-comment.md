# drive +restore-comment

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由与目标定位（URL/wiki/apps/base）。

恢复 / 重新打开一条已解决的评论（发送 `is_solved=false`）。反向操作——把评论标记为已解决——是独立命令 [`lark-drive-resolve-comment.md`](lark-drive-resolve-comment.md)，两者共享同一个 patch 端点。

用户说“重新打开 / 取消解决 / 恢复这条评论”对应本命令。

## 命令

```bash
lark-cli drive +restore-comment --url '<DOC_URL>' --comment-id '<id>'

# 裸 wiki token
lark-cli drive +restore-comment --token '<WIKI_TOKEN>' --type wiki --comment-id '<id>'

# 预览调用链
lark-cli drive +restore-comment --url '<DOC_URL>' --comment-id '<id>' --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--comment-id` | 是 | 要恢复的评论 ID；来自 `drive +list-comments`。已解决评论需传 `--solved-status true` 或 `all` 才能查到 |

## 行为说明

- 这是写操作。
- 对同一条评论连续翻转解决状态可能触发服务端限流（HTTP 429）；连续调用之间留间隔或短暂延迟后重试。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comments patch`（先 `lark-cli schema drive.file.comments.patch` 查契约）。直接调原生时 Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-resolve-comment](lark-drive-resolve-comment.md) -- 解决（标记已解决）评论
