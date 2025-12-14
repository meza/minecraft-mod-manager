#!/bin/bash

FILE=$1
VERSION=$2

cat > "$FILE" <<EOF
export const version = '$VERSION';
EOF

sed -i "s|REPL_VERSION|$VERSION|" internal/environment/environment.go
sed -i "s|REPL_CURSEFORGE_API_KEY|$CURSEFORGE_API_KEY|" internal/environment/environment.go
sed -i "s|REPL_MODRINTH_API_KEY|$MODRINTH_API_KEY|" internal/environment/environment.go
sed -i "s|REPL_POSTHOG_API_KEY|$POSTHOG_API_KEY|" internal/environment/environment.go
sed -i "s|REPL_HELP_URL|$HELP_URL|" internal/environment/environment.go

echo "Set version to $VERSION"
