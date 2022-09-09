#!/bin/bash

FILE=$1
VERSION=$2

cat > "$FILE" <<EOF
export const version = '$VERSION';
EOF

sed -i "s/\"version\": \"[0-9.]*\"/\"version\": \"$VERSION\"/" package.json
sed -i "s/REPL_CURSEFORGE_API_KEY/$CURSEFORGE_API_KEY/" src/env.ts
sed -i "s/REPL_MODRINTH_API_KEY/$MODRINTH_API_KEY/" src/env.ts
