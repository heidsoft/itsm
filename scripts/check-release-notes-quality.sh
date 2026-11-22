#!/usr/bin/env bash
# 发布说明文案门禁：禁止无证据的宣传语和"全部通过"式结论进入对外 Release。
# 与 docs-gate/check-release-claims.sh 的分工：那边校验工程审计报告，
# 这里只校验即将发布给用户的 Release 正文。
set -euo pipefail

notes="${1:?Usage: check-release-notes-quality.sh <release-notes-file>}"

if [ ! -s "$notes" ]; then
  printf '发布说明为空，拒绝发布。\n' >&2
  exit 1
fi

# 只列无证据即可判定的空洞表达；带 commit/digest/日期的测试结论由
# docs-gate/check-release-claims.sh 在审计侧处理，不在发布正文里放行。
readonly BANNED='史诗级升级|全面赋能|革命性突破|企业级能力全面增强|深度打通业务闭环|持续夯实技术底座|实现全链路智能升级|全面提升企业数字化体验|全部通过|全部 OK|全部完成|立即上线|可以发布|零阻断|零问题|无 P0|完美|彻底解决|100% 覆盖|覆盖率达成|no blocker|all green|ready to ship|production ready'

if grep -Ein "$BANNED" "$notes"; then
  printf '发布说明包含无证据的宣传语或结论式表述，请改为可验证的用户影响描述。\n' >&2
  exit 1
fi

printf '发布说明文案检查通过。\n'
