#!/bin/bash

FILE=$1

VERSION=$(node -p "require('./package.json').version")
cat > "$FILE" <<EOF
export const version = '$VERSION';
EOF
