# drive +react-reply

> **前置条件：** 先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和权限处理。reaction 查询规则、语义联想与完整 `reaction_type` 枚举见跨切面专题 [`lark-drive-reactions.md`](lark-drive-reactions.md)。

给一条回复添加或删除表情回应（reaction）。操作对象始终是 `reply_id`。

## 命令

```bash
# 给某条 reply 添加点赞
lark-cli drive +react-reply --url '<DOC_URL>' --reply-id '<id>' --emoji THUMBSUP --action add

# 删除某条 reply 上的 DONE（wiki URL 自动解包）
lark-cli drive +react-reply --url '<WIKI_URL>' --reply-id '<id>' --emoji DONE --action delete
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--url` / `--token` + `--type` | 是（二选一） | 目标定位，见 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md)；wiki 自动解包 |
| `--reply-id` | 是 | 要操作的回复 ID；来自 `drive +list-replies` 的 `items[].reply_id`。给“这条评论”加/删表情时取该评论根回复（第一页 `items[0]`）的 `reply_id` |
| `--emoji` | 是 | `reaction_type` 值，大小写敏感；本地按平台枚举校验。完整列表与语义映射见 [`lark-drive-reactions.md`](lark-drive-reactions.md) |
| `--action` | 是 | `add` 添加；`delete` 删除当前身份自己加的 reaction |

## 行为说明

- `--emoji` 大小写敏感（如 `THUMBSUP` 与 `ThumbsDown`），并做本地枚举校验兜底。服务端不校验 `reaction_type`：任意字符串都会被接受并持久化成一条损坏的 reaction，所以本地校验是唯一防线；直接调原生命令时必须自行保证取值合法。
- add / delete 幂等：重复添加已有 reaction、删除不存在的 reaction 都会成功返回且无副作用；delete 只取消当前身份自己加的 reaction。
- 对根回复操作等价于给评论本身加 / 删表情。
- 读回 reaction：在 `drive +list-replies` / `drive +batch-query-comments` 上带 `--need-reaction`；`count=0` 的条目是已删除 reaction 的残留，判断存在与否按 `count>0` 过滤。

## 原生兜底

只有需要 shortcut 未暴露的字段时，才用原生 `drive file.comment.reply.reactions update_reaction`（先 `lark-cli schema` 查契约）。原生路径没有本地枚举校验，`reaction_type` 取值须自行保证；直接调原生时 Base 的 `file_type` 传 `bitable`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-reactions](lark-drive-reactions.md) -- reaction 查询规则、语义与完整枚举
- [lark-drive-list-replies](lark-drive-list-replies.md) -- 获取 reply_id
