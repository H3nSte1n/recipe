#!/bin/bash
# Fails if component styles set font properties directly instead of using the
# global text styles in src/styles/design-system.css. Runs as part of
# `npm run lint`.
#
# Allowed in component CSS:
#   font: var(--type-…);                      a text style
#   font-size: var(--icon-…);                 an icon glyph size
#   line-height: …; /* reason */              layout-driven, with a comment
set -euo pipefail

cd "$(dirname "$0")/.."

# design-system.css defines the tokens; ThemeExplorer is a temporary tool, off-system.
css_files=$(find src -name '*.css' ! -name 'design-system.css' ! -name 'ThemeExplorer.css')

violations=""
add() { violations+="$1"$'\n'; }

for f in $css_files; do
    while IFS= read -r hit; do add "$f:$hit  (use font: var(--type-…))"; done < <(
        grep -nE '^\s*(font-family|font-weight|font-style|letter-spacing)\s*:' "$f" || true)
    while IFS= read -r hit; do add "$f:$hit  (use font: var(--type-…), or var(--icon-…) for icon glyphs)"; done < <(
        grep -nE '^\s*font-size\s*:' "$f" | grep -vE 'font-size\s*:\s*var\(--icon-[a-z]+\)' || true)
    while IFS= read -r hit; do add "$f:$hit  (only font: var(--type-…) is allowed)"; done < <(
        grep -nE '^\s*font\s*:' "$f" | grep -vE 'font\s*:\s*var\(--type-[a-z]+\)' || true)
    while IFS= read -r hit; do add "$f:$hit  (line-height overrides need a /* reason */ comment)"; done < <(
        grep -nE '^\s*line-height\s*:' "$f" | grep -v '/\*' || true)
done

# Inline React styles bypass the stylesheets entirely.
while IFS= read -r hit; do add "$hit  (move font styling to CSS and use a text style)"; done < <(
    grep -rnE '\b(fontFamily|fontSize|fontWeight|fontStyle|lineHeight|letterSpacing)\s*:' src \
        --include='*.tsx' --include='*.ts' || true)

if [ -n "$violations" ]; then
    echo "ERROR: font styling outside the global type system (src/styles/design-system.css):" >&2
    printf '%s' "$violations" >&2
    exit 1
fi

echo "OK: all font styling comes from design-system.css."
