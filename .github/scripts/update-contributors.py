#!/usr/bin/env python3
"""Regenerate the contributor avatar wall inside the ITSM README files.

The wall is rendered between the markers
    <!-- CONTRIBUTORS-WALL:START -->
    <!-- CONTRIBUTORS-WALL:END -->
in every README path passed as a CLI argument.

Only *human* contributors are shown (GitHub Apps / bots such as
dependabot and github-actions are filtered out). The section updates
itself on every run, so the README never goes stale as the project
attracts new contributors.
"""
import json
import os
import re
import subprocess
import sys
import urllib.request

REPO = os.environ.get("GITHUB_REPOSITORY", "heidsoft/itsm")
TOKEN = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
START = "<!-- CONTRIBUTORS-WALL:START -->"
END = "<!-- CONTRIBUTORS-WALL:END -->"


def fetch_contributors():
    """Return the list of contributor dicts from the GitHub API."""
    url = f"https://api.github.com/repos/{REPO}/contributors?per_page=100"
    if TOKEN:
        req = urllib.request.Request(url)
        req.add_header("Accept", "application/vnd.github+json")
        req.add_header("Authorization", f"Bearer {TOKEN}")
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.load(resp)
    # Fallback to the gh CLI when no token is present (local runs).
    try:
        out = subprocess.check_output(
            ["gh", "api", f"repos/{REPO}/contributors?per_page=100"],
            stderr=subprocess.DEVNULL,
        )
        return json.loads(out)
    except Exception as exc:  # pragma: no cover - network/tooling edge case
        sys.stderr.write(f"failed to fetch contributors: {exc}\n")
        return []


def build_wall(contributors):
    humans = [c for c in contributors if not c.get("login", "").endswith("[bot]")]
    humans.sort(key=lambda c: c.get("contributions", 0), reverse=True)
    if not humans:
        return "_No human contributors yet — be the first to open a PR 🙂_"
    cells = []
    for c in humans:
        login = c["login"]
        url = c["html_url"]
        avatar = c["avatar_url"]
        commits = c.get("contributions", 0)
        cells.append(
            f'<a href="{url}" title="{login} · {commits} commits">'
            f'<img src="{avatar}&amp;s=64" width="56" height="56" alt="{login}" '
            f'style="border-radius:50%;margin:4px;vertical-align:middle;"></a>'
        )
    return "\n".join(cells)


def inject(path, wall):
    with open(path, encoding="utf-8") as fh:
        text = fh.read()
    pattern = re.compile(re.escape(START) + r".*?" + re.escape(END), re.DOTALL)
    if not pattern.search(text):
        sys.stderr.write(f"markers not found in {path}, skipping\n")
        return False
    replacement = f"{START}\n{wall}\n{END}"
    new_text = pattern.sub(replacement, text)
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(new_text)
    return True


def main():
    if len(sys.argv) < 2:
        sys.stderr.write("usage: update-contributors.py README.md [README.en.md ...]\n")
        sys.exit(2)
    contributors = fetch_contributors()
    wall = build_wall(contributors)
    human_count = sum(1 for c in contributors if not c.get("login", "").endswith("[bot]"))
    for path in sys.argv[1:]:
        if inject(path, wall):
            print(f"updated {path} ({human_count} human contributor(s))")


if __name__ == "__main__":
    main()
