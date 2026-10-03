#!/usr/bin/env python3
"""
scripts/docs-gate/find-frontend-orphans.py

Detect frontend files in src/components/** and src/lib/api/** that are not
reachable from any src/app/** page/layout entry point.

Algorithm:
  1. Seed: all .tsx/.ts files under src/app/ + root layout.tsx + middleware.ts
  2. For each seed file, parse imports (static + dynamic) and follow transitively
  3. Build a set of all reachable files
  4. Scan target directories for all .ts/.tsx files
  5. Report files not in the reachable set

Import resolution:
  - @/ alias → src/
  - Relative imports resolved against current file
  - External packages (no ./ or @/) skipped
  - Dynamic imports (next/dynamic, React.lazy, import()) extracted

Output: one orphan file path per line (relative to frontend root), sorted.
Exit 0 always — caller decides advisory vs blocking.
"""
import os
import re
import sys
from pathlib import Path
from collections import deque

# --- Configuration ---
# Directories to scan for orphans
ORPHAN_DIRS = ["src/components", "src/lib/api"]
# File extensions to consider
EXTENSIONS = (".ts", ".tsx")
# Directories to skip during import traversal
SKIP_DIRS = {"node_modules", ".next", "dist", "__mocks__", "__tests__"}

# --- Import extraction patterns ---
# Static: from 'path' / import 'path' / export ... from 'path'
RE_STATIC_IMPORT = re.compile(
    r"""(?:import|export)\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]""",
    re.MULTILINE,
)
# Dynamic: import('path') / dynamic(() => import('path'))
RE_DYNAMIC_IMPORT = re.compile(
    r"""import\(\s*['"]([^'"]+)['"]\s*\)""",
    re.MULTILINE,
)
# require('path')
RE_REQUIRE = re.compile(
    r"""require\(\s*['"]([^'"]+)['"]\s*\)""",
    re.MULTILINE,
)


def is_external(specifier: str) -> bool:
    """Return True if the import specifier is an external package."""
    if specifier.startswith(("./", "../", "@/")):
        return False
    return True


def resolve_import(specifier: str, source_file: Path, src_root: Path) -> Path | None:
    """Resolve an import specifier to a concrete file path, or None."""
    if is_external(specifier):
        return None

    if specifier.startswith("@/"):
        base = src_root / specifier[2:]
    elif specifier.startswith(("./", "../")):
        base = (source_file.parent / specifier).resolve()
    else:
        return None

    # Try exact match, then with extensions, then as directory with index
    candidates = [
        base,
        base.with_suffix(".ts"),
        base.with_suffix(".tsx"),
        base / "index.ts",
        base / "index.tsx",
    ]
    for c in candidates:
        if c.is_file():
            return c.resolve()
    return None


def extract_imports(file_path: Path) -> list[str]:
    """Extract all import specifiers from a file."""
    try:
        content = file_path.read_text(encoding="utf-8", errors="ignore")
    except Exception:
        return []

    specifiers = []
    for pattern in (RE_STATIC_IMPORT, RE_DYNAMIC_IMPORT, RE_REQUIRE):
        for m in pattern.finditer(content):
            specifiers.append(m.group(1))
    return specifiers


def build_reachable_set(frontend_root: Path) -> set[Path]:
    """BFS from all entry points to build the set of reachable files."""
    src_root = frontend_root / "src"
    app_dir = src_root / "app"
    reachable = set()
    queue = deque()

    # Seed: all .ts/.tsx files under src/app/
    if app_dir.is_dir():
        for root, dirs, files in os.walk(app_dir):
            dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
            for f in files:
                if f.endswith(EXTENSIONS):
                    p = Path(root) / f
                    resolved = p.resolve()
                    if resolved not in reachable:
                        reachable.add(resolved)
                        queue.append(resolved)

    # Seed: root layout.tsx
    root_layout = src_root / "app" / "layout.tsx"
    if root_layout.is_file():
        resolved = root_layout.resolve()
        if resolved not in reachable:
            reachable.add(resolved)
            queue.append(resolved)

    # Seed: middleware.ts
    middleware = frontend_root / "src" / "middleware.ts"
    if middleware.is_file():
        resolved = middleware.resolve()
        if resolved not in reachable:
            reachable.add(resolved)
            queue.append(resolved)

    # BFS: follow imports
    while queue:
        current = queue.popleft()
        for spec in extract_imports(current):
            resolved = resolve_import(spec, current, src_root)
            if resolved and resolved not in reachable:
                # Only follow files within src/
                try:
                    resolved.relative_to(src_root)
                except ValueError:
                    continue
                reachable.add(resolved)
                queue.append(resolved)

    return reachable


def collect_target_files(frontend_root: Path) -> list[Path]:
    """Collect all .ts/.tsx files in target directories."""
    files = []
    for target_dir in ORPHAN_DIRS:
        full = frontend_root / target_dir
        if not full.is_dir():
            continue
        for root, dirs, filenames in os.walk(full):
            dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
            for f in filenames:
                if f.endswith(EXTENSIONS):
                    files.append((Path(root) / f).resolve())
    return files


def main():
    if len(sys.argv) < 2:
        print("Usage: find-frontend-orphans.py <frontend-root>", file=sys.stderr)
        sys.exit(2)

    frontend_root = Path(sys.argv[1]).resolve()
    if not (frontend_root / "src").is_dir():
        print(f"Error: {frontend_root}/src not found", file=sys.stderr)
        sys.exit(2)

    reachable = build_reachable_set(frontend_root)
    targets = collect_target_files(frontend_root)

    orphans = sorted(
        p for p in targets if p not in reachable
    )

    # Output relative paths
    for p in orphans:
        print(p.relative_to(frontend_root))


if __name__ == "__main__":
    main()
