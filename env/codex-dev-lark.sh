#!/usr/bin/env bash
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
#
# Launch Codex in this checkout with project-local Lark skills and a dev
# lark-cli shim. The global ~/.agents/skills install is left untouched.

set -euo pipefail

usage() {
	cat <<'USAGE'
Usage:
  env/codex-dev-lark.sh [options] [--] [codex args or initial prompt]

Options:
  --lane <lane>       BOE lane injected through LARK_LANE.
                      Default: boe_bitable_bk11
  --env <env>         larkenv target: boe, pre, or online.
                      Default: boe
  --skill <name>      Link only one local skill, e.g. lark-base.
                      Default: all lark-* skills under ./skills
  --no-build          Reuse the current ./lark-cli binary instead of rebuilding.
  -h, --help          Show this help.

Examples:
  env/codex-dev-lark.sh
  env/codex-dev-lark.sh --skill lark-base
  env/codex-dev-lark.sh -- "用 Base Skill 查一下这个 workspace"
USAGE
}

die() {
	echo "error: $*" >&2
	exit 1
}

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

lane="${LARK_LANE:-boe_bitable_bk11}"
target_env="${LARKENV_TARGET:-boe}"
skill_filter="all"
do_build=1
codex_args=()

while [ $# -gt 0 ]; do
	case "$1" in
	--lane)
		[ $# -ge 2 ] || die "--lane requires a value"
		lane="$2"
		shift 2
		;;
	--env)
		[ $# -ge 2 ] || die "--env requires a value"
		target_env="$2"
		shift 2
		;;
	--skill)
		[ $# -ge 2 ] || die "--skill requires a value"
		skill_filter="$2"
		shift 2
		;;
	--no-build)
		do_build=0
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	--)
		shift
		codex_args=("$@")
		break
		;;
	*)
		codex_args+=("$1")
		shift
		;;
	esac
done

case "$target_env" in
boe | pre | online) ;;
*) die "--env must be one of: boe, pre, online" ;;
esac

command -v codex >/dev/null 2>&1 || die "codex not found in PATH"
[ -x "$repo_root/env/larkenv" ] || die "missing executable env/larkenv"

bin_dir="$repo_root/.codex-dev/bin"
skills_dir="$repo_root/.agents/skills"
mkdir -p "$bin_dir" "$skills_dir"

cd "$repo_root"

if [ "$do_build" -eq 1 ]; then
	echo "==> Building dev lark-cli from $repo_root" >&2
	./build.sh
else
	echo "==> Reusing existing ./lark-cli" >&2
fi

[ -x "$repo_root/lark-cli" ] || die "missing ./lark-cli; run without --no-build first"

cp "$repo_root/lark-cli" "$bin_dir/lark-cli-env"
cp "$repo_root/env/larkenv" "$bin_dir/larkenv"
chmod +x "$bin_dir/lark-cli-env" "$bin_dir/larkenv"

link_skill() {
	local name="$1"
	local src="$repo_root/skills/$name"
	local dst="$skills_dir/$name"
	local rel="../../skills/$name"
	local current backup

	[ -f "$src/SKILL.md" ] || die "missing skill: skills/$name/SKILL.md"

	current="$(readlink "$dst" 2>/dev/null || true)"
	if [ "$current" = "$rel" ]; then
		return
	fi

	if [ -L "$dst" ]; then
		backup="$dst.bak.$(date +%Y%m%d%H%M%S)"
		mv "$dst" "$backup"
		echo "==> Backed up existing local skill symlink: $backup" >&2
	elif [ -e "$dst" ]; then
		backup="$dst.bak.$(date +%Y%m%d%H%M%S)"
		mv "$dst" "$backup"
		echo "==> Backed up existing local skill directory: $backup" >&2
	fi

	ln -s "$rel" "$dst"
}

if [ "$skill_filter" = "all" ]; then
	found=0
	for skill_path in "$repo_root"/skills/lark-*; do
		[ -d "$skill_path" ] || continue
		link_skill "${skill_path##*/}"
		found=1
	done
	[ "$found" -eq 1 ] || die "no local lark-* skills found under ./skills"
else
	link_skill "$skill_filter"
fi

cat >"$bin_dir/lark-cli" <<SHIM
#!/usr/bin/env bash
exec env \\
  LARK_CLI_ENV_BIN="$bin_dir" \\
  LARK_LANE="$lane" \\
  LARKSUITE_CLI_NO_UPDATE_NOTIFIER="\${LARKSUITE_CLI_NO_UPDATE_NOTIFIER:-1}" \\
  LARKSUITE_CLI_NO_SKILLS_NOTIFIER="\${LARKSUITE_CLI_NO_SKILLS_NOTIFIER:-1}" \\
  "$bin_dir/larkenv" "$target_env" "\$@"
SHIM
chmod +x "$bin_dir/lark-cli"

echo "==> Starting Codex with dev lark-cli and project-local skills" >&2
echo "    PATH prefix: $bin_dir" >&2
echo "    Skill root:   $skills_dir" >&2
echo "    lark-cli ->   LARK_LANE=$lane larkenv $target_env" >&2

export PATH="$bin_dir:$PATH"
export LARK_CLI_ENV_BIN="$bin_dir"
export LARK_LANE="$lane"
export LARKSUITE_CLI_NO_UPDATE_NOTIFIER="${LARKSUITE_CLI_NO_UPDATE_NOTIFIER:-1}"
export LARKSUITE_CLI_NO_SKILLS_NOTIFIER="${LARKSUITE_CLI_NO_SKILLS_NOTIFIER:-1}"

exec codex -C "$repo_root" -c shell_environment_policy.inherit=all "${codex_args[@]}"
