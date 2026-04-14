#!/bin/bash
# scripts/update-internal-deps.sh
# Обновляет в go.mod только dev- и beta-версии указанного пакета до новой beta-версии.

set -e

PKG="$1"
VERSION="$2"
MODULE_PATH="github.com/ssoeasy-dev/pkg/${PKG}"

echo "Updating internal dependencies for ${MODULE_PATH} to ${VERSION} (only dev/beta)"

find . -name "go.mod" -not -path "./${PKG}/*" | while read -r modfile; do
    dir=$(dirname "$modfile")
    if grep -q "${MODULE_PATH}" "$modfile"; then
        # Проверяем, какая версия сейчас указана
        CURRENT=$(grep "${MODULE_PATH}" "$modfile" | head -n1 | awk '{print $2}')
        # Если версия содержит "-dev-" или "-beta.", обновляем
        if [[ "$CURRENT" == *-dev-* ]] || [[ "$CURRENT" == *-beta.* ]]; then
            echo "  Updating $modfile (was $CURRENT)"
            sed -i "s|${MODULE_PATH} ${CURRENT}|${MODULE_PATH} ${VERSION}|g" "$modfile"
            (cd "$dir" && go mod tidy)
            git add "$modfile" "$dir/go.sum"
        else
            echo "  Skipping $modfile (version $CURRENT is stable)"
        fi
    fi
done

if ! git diff --cached --quiet; then
    git commit -m "chore(deps): update ${PKG} to ${VERSION}"
    echo "Committed dependency updates."
else
    echo "No dependency updates needed."
fi
