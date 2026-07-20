# Drive 评论内容格式（--content）

> **前置条件：** 先阅读 [`lark-drive-comments-guide.md`](lark-drive-comments-guide.md) 了解评论域路由。本文是写入类评论命令共享的内容格式说明，供 [`lark-drive-add-comment.md`](lark-drive-add-comment.md)、[`lark-drive-add-reply.md`](lark-drive-add-reply.md)、[`lark-drive-update-reply.md`](lark-drive-update-reply.md) 引用。

`drive +add-comment`、`drive +add-reply`、`drive +update-reply` 的 `--content` 使用同一套 `reply_elements` JSON 数组格式。本文集中说明 schema、元素类型、转义和长度限制，各命令 ref 只保留最常见的纯文本例子。

## Schema

`--content` 是一个 JSON 数组字符串，每个元素形如 `{"type": "<type>", ...}`。支持三种 `type`：

| type | 必填字段 | 含义 | 便捷写法 |
|---|---|---|---|
| `text` | `text` | 普通文本 | — |
| `mention_user` | `mention_user` | @某个用户，值传 open_id | 也可把 open_id 直接放在 `text` 字段 |
| `link` | `link` | 超链接，值传 URL | 也可把 URL 直接放在 `text` 字段 |

最常见就是单个纯文本元素：

```bash
--content '[{"type":"text","text":"评论正文"}]'
```

组合多种元素：

```bash
--content '[
  {"type":"text","text":"请 "},
  {"type":"mention_user","text":"ou_xxx"},
  {"type":"text","text":" 看下 "},
  {"type":"link","text":"https://example.com"}
]'
```

- `--content` 至少包含一个元素；`type=text` 的 `text` 不能为空。
- `mention_user` / `link` 若既没给专属字段也没给 `text`，会报 validation error。
- 未知 `type` 会被拒绝，只允许 `text` / `mention_user` / `link`。

## 转义

- `type=text` 文本里的 `<`、`>` 会被 shortcut 自动转义为 `&lt;`、`&gt;`，避免被评论渲染器当作标记解析。调用方不需要自己转义。
- 直接调用原生评论写入 API（如 `drive file.comments create_v2`）时没有这层兜底，必须自行传入已转义的内容。

## 长度限制

- 所有 `type=text` 元素的字符（rune）总和上限 10000，按原始输入的字符数计（中英文、符号一视同仁，不是字节数、也不是转义后的长度）。
- 这是对**总额**的限制：把一段长文本拆成多个 text 元素不能绕过，它们共用同一个 10000 字符预算。
- `mention_user` / `link` 不计入该长度。
- 超限时 shortcut 在发送前拒绝并指出累计超长的元素；服务端对超限返回不透明的 `[1069302]`，所以这是预检。

## shortcut → 原生 body 的转换边界

写入类 shortcut 会把上面的简化元素转换成 Drive v1 评论的 wire 形态再发送：

| 简化元素 | 原生 wire 元素 |
|---|---|
| `{"type":"text","text":"..."}` | `{"type":"text_run","text_run":{"text":"..."}}` |
| `{"type":"mention_user","mention_user":"ou_xxx"}` | `{"type":"person","person":{"user_id":"ou_xxx"}}` |
| `{"type":"link","link":"https://..."}` | `{"type":"docs_link","docs_link":{"url":"https://..."}}` |

回复类命令（`+add-reply` / `+update-reply`）最终 body 是 `{"content":{"elements":[<上表 wire 元素>]}}`。只有在需要 shortcut 未暴露的字段时，才直接手写这套原生 body 调用 `drive file.comments create_v2` / `drive file.comment.replys create|update`。

## 参考

- [lark-drive-comments-guide](lark-drive-comments-guide.md) -- 评论域二级路由
- [lark-drive-add-comment](lark-drive-add-comment.md) -- 添加评论
- [lark-drive-add-reply](lark-drive-add-reply.md) -- 回复评论
- [lark-drive-update-reply](lark-drive-update-reply.md) -- 更新回复
