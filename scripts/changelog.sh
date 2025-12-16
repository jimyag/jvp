#!/bin/bash
# 生成 changelog 脚本
# 找到上一个正式版本（不含 -rc, -beta, -alpha 等后缀），生成 changelog

set -e

CURRENT_TAG="${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "")}"

if [ -z "$CURRENT_TAG" ]; then
    echo "No current tag found"
    exit 1
fi

# 找到上一个正式版本（排除 pre-release）
# 正式版本格式: vX.Y.Z（不含 - 后缀）
get_previous_stable_tag() {
    local current="$1"
    git tag --sort=-version:refname | while read -r tag; do
        # 跳过当前 tag
        if [ "$tag" = "$current" ]; then
            continue
        fi
        # 检查是否是正式版本（不含 -）
        if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo "$tag"
            return
        fi
    done
}

PREVIOUS_TAG=$(get_previous_stable_tag "$CURRENT_TAG")

if [ -z "$PREVIOUS_TAG" ]; then
    echo "No previous stable tag found, using first commit"
    RANGE="$CURRENT_TAG"
else
    RANGE="${PREVIOUS_TAG}..${CURRENT_TAG}"
fi

# 生成 changelog
OUTPUT_FILE="${2:-CHANGELOG.tmp.md}"

cat > "$OUTPUT_FILE" << EOF
## What's Changed

EOF

# 按类型分组提交
declare -A commits
commits["feat"]=""
commits["fix"]=""
commits["docs"]=""
commits["refactor"]=""
commits["perf"]=""
commits["chore"]=""
commits["other"]=""

while IFS= read -r line; do
    if [ -z "$line" ]; then
        continue
    fi

    hash=$(echo "$line" | cut -d'|' -f1)
    subject=$(echo "$line" | cut -d'|' -f2-)
    short_hash="${hash:0:7}"

    # 解析 conventional commit 类型
    if [[ "$subject" =~ ^feat(\(.+\))?:\ (.+) ]]; then
        commits["feat"]+="* ${subject} (${short_hash})"$'\n'
    elif [[ "$subject" =~ ^fix(\(.+\))?:\ (.+) ]]; then
        commits["fix"]+="* ${subject} (${short_hash})"$'\n'
    elif [[ "$subject" =~ ^docs(\(.+\))?:\ (.+) ]]; then
        commits["docs"]+="* ${subject} (${short_hash})"$'\n'
    elif [[ "$subject" =~ ^refactor(\(.+\))?:\ (.+) ]]; then
        commits["refactor"]+="* ${subject} (${short_hash})"$'\n'
    elif [[ "$subject" =~ ^perf(\(.+\))?:\ (.+) ]]; then
        commits["perf"]+="* ${subject} (${short_hash})"$'\n'
    elif [[ "$subject" =~ ^chore(\(.+\))?:\ (.+) ]]; then
        commits["chore"]+="* ${subject} (${short_hash})"$'\n'
    else
        commits["other"]+="* ${subject} (${short_hash})"$'\n'
    fi
done < <(git log --pretty=format:"%H|%s" "$RANGE" 2>/dev/null || git log --pretty=format:"%H|%s")

# 输出分组
if [ -n "${commits["feat"]}" ]; then
    echo "### ✨ Features" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["feat"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["fix"]}" ]; then
    echo "### 🐛 Bug Fixes" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["fix"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["perf"]}" ]; then
    echo "### ⚡ Performance" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["perf"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["refactor"]}" ]; then
    echo "### ♻️ Refactor" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["refactor"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["docs"]}" ]; then
    echo "### 📚 Documentation" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["docs"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["chore"]}" ]; then
    echo "### 🔧 Chores" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["chore"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

if [ -n "${commits["other"]}" ]; then
    echo "### 📝 Other Changes" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    echo -n "${commits["other"]}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
fi

# 添加 footer
cat >> "$OUTPUT_FILE" << EOF
---

**Full Changelog**: https://github.com/jimyag/jvp/compare/${PREVIOUS_TAG:-"initial"}...${CURRENT_TAG}
EOF

echo "Changelog generated: $OUTPUT_FILE (${PREVIOUS_TAG:-"initial"} -> ${CURRENT_TAG})"
