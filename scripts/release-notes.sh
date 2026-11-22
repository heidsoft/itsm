#!/usr/bin/env bash
# Print the public notes for a tag. Older tags use their full changelog section.
set -euo pipefail

version="${1:?Usage: release-notes.sh <tag> [changelog]}"
changelog="${2:-CHANGELOG.md}"

awk -v heading="## [${version#v}]" '
  /^## / {
    if (found) exit
    if ($0 == heading || index($0, heading " ") == 1) found = 1
    next
  }
  !found { next }
  $0 == "### 发布说明" { has_public = in_public = 1; next }
  /^### / { in_public = 0 }
  {
    full = full $0 ORS
    if (in_public) public = public $0 ORS
  }
  END {
    notes = has_public ? public : full
    sub(/^[[:space:]]+/, "", notes)
    sub(/[[:space:]]+$/, "", notes)
    if (!has_public) {
      sub(/(^|\n)---$/, "", notes)
      sub(/[[:space:]]+$/, "", notes)
    }
    if (notes == "") exit 1
    print notes
  }
' "$changelog" || {
  printf '找不到 %s 的有效发布说明，请检查 %s。\n' "$version" "$changelog" >&2
  exit 1
}
