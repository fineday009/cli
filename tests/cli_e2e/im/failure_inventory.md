# IM Failure Inventory

Seed bad cases for the IM CLI governance closeout. Each entry replays one
high-frequency failure and records whether the current error output lets an
agent decide its next action (PASS), needs a hint fix (FIX_HINT), or cannot be
fixed by hints at all (BLOCKED). Companion doc: [coverage.md](coverage.md).

Replay verdict rule — looking only at the stderr envelope
(`error.type/subtype/param/message/hint`) and `--help`, an agent must be able
to (1) identify the failing input, (2) understand why, (3) know the concrete
next action, (4) know how to verify it. All four → PASS.

## messages-send.audio.non_opus
- source: tests/cli_e2e/im/message_audio_dryrun_test.go
- user_task: send a local voice file as an audio message
- command: `lark-cli im +messages-send --chat-id <chat_id> --audio ./voice.mp3 --dry-run`
- observed: type=validation subtype=invalid_argument param=--audio; message says only Opus is supported; hint offers ffmpeg conversion and `--file` fallback
- verdict: PASS
- expected_next_action: convert to .opus and retry --audio, or resend with --file when voice semantics are not required
- lock: TestIM_MessagesSendAudioDryRunRejectsNonOpus

## im.search.read_after_write_index_delay
- source: tests/cli_e2e/im/coverage.md (blocked case)
- user_task: search a chat/message right after creating/sending it
- command: `lark-cli im +chat-search --query "<fresh name>"` / `lark-cli im +messages-search --query "<fresh text>"`
- observed: empty result set with exit 0 — not an error envelope; freshly written objects are not indexed deterministically in UAT
- verdict: BLOCKED (not_fixing_with_hint)
- expected_next_action: none via hint — do not assert read-after-write search visibility; needs a stable historical fixture or an explicit index-wait strategy
- lock: coverage.md blocked_case im.search.read_after_write_index_delay

## feed.head_tail.mutually_exclusive
- source: shortcuts/im/im_feed_shortcut_create.go resolveIsHeader
- user_task: add a chat to feed shortcuts while guessing position flags
- command: `lark-cli im +feed-shortcut-create --chat-id <chat_id> --head --tail --dry-run`
- observed (replayed, before fix): `{"ok":false,"identity":"user","error":{"type":"validation","subtype":"invalid_argument","message":"--head and --tail are mutually exclusive"}}` — names the conflict but gives no next action and no hint
- observed (after fix): same envelope plus `"hint":"pass only one of --head or --tail; omitting both inserts at the head"`
- verdict: FIX_HINT (fixed in this PR)
- expected_hint: pass only one of --head or --tail; omitting both inserts at the head
- expected_next_action: drop one of the two flags and retry
- lock: TestResolveIsHeaderMutualExclusionHint

## feed.chat_id.not_oc_prefix
- source: shortcuts/im/helpers.go collectChatIDs
- user_task: pass a message id (om_) or plain id where an open_chat_id is required
- command: `lark-cli im +feed-shortcut-create --chat-id om_test000 --dry-run`
- observed (replayed, before fix): `{"ok":false,"identity":"user","error":{"type":"validation","subtype":"invalid_argument","message":"invalid --chat-id \"om_test000\": must be an open_chat_id starting with oc_","param":"--chat-id"}}` — names what is required (an oc_ id) but gives no next action or ID-source hint
- observed (after fix): same envelope plus `"hint":"get the open_chat_id from im +chat-search (by name) or im +chat-list (my chats)"`
- verdict: FIX_HINT (fixed in this PR)
- expected_hint: get the open_chat_id from im +chat-search or im +chat-list
- expected_next_action: fetch the oc_ id via +chat-search / +chat-list and retry
- lock: shortcuts/im/im_feed_shortcut_test.go::TestCollectChatIDsHint

## chat-messages-list.bot_identity.user_id
- source: shortcuts/im/im_chat_messages_list.go (Validate), shortcuts/im/helpers.go resolveP2PChatID
- user_task: bot identity tries to list a P2P conversation by user open_id instead of a chat_id
- command: `lark-cli im +chat-messages-list --user-id <open_id> --as bot --dry-run`
- observed (replayed): `{"ok":false,"identity":"bot","error":{"type":"validation","subtype":"invalid_argument","message":"--user-id requires user identity (--as user); use --chat-id when calling with bot identity","param":"--user-id"}}`
- note: replay corrected the seed's target command — `im +messages-send --user-id <open_id> --as bot` is valid (a bot may DM a user by open_id) and returns a normal dry-run request, not an error; the "requires user identity" message only fires on `im +chat-messages-list`, which resolves --user-id via a P2P chat_id lookup that bot identity cannot perform
- verdict: PASS
- expected_next_action: switch to --as user, or target the chat via --chat-id
- lock: shortcuts/im/builders_test.go::TestShortcutValidateBranches/ImChatMessageList_rejects_user_target_for_bot_identity; shortcuts/im/coverage_additional_test.go::TestResolveChatIDForMessagesList/user_target_rejected_for_bot_identity

## messages-send.content.invalid_json
- source: shortcuts/im/im_messages_send.go content validation
- user_task: hand-writing --content JSON and getting it wrong
- command: `lark-cli im +messages-send --chat-id <chat_id> --content '{bad' --dry-run`
- observed (replayed): `{"ok":false,"identity":"bot","error":{"type":"validation","subtype":"invalid_argument","message":"--content is not valid JSON: {bad json\nexample: --content '{\"text\":\"hello\"}' or --text 'hello'","param":"--content"}}`
- note: replayed under `--as bot` because the local sandbox's user-identity token lacks `im:message.send_as_user`, which triggers an authorization/missing_scope envelope ahead of content validation under `--as user`; the invalid-JSON validation path itself is identity-independent
- verdict: PASS
- expected_next_action: prefer --text for plain text instead of hand-writing content JSON
- lock: shortcuts/im/builders_test.go::TestShortcutValidateBranches/ImMessagesSend_invalid_content_json
