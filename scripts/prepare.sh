#!/bin/bash

FILE=$1
VERSION=$2

cat > "$FILE" <<EOF
export const version = '$VERSION';
EOF

sed -i "s|REPL_VERSION|$VERSION|" internal/environment/environment.go
sed -i "s|REPL_HELP_URL|$HELP_URL|" internal/environment/environment.go

echo "Set version to $VERSION"
