#!/bin/bash

VERSION=$1

mkdir -p dist/pkg/linux
mkdir -p dist/pkg/win
mkdir -p dist/pkg/macos

mv dist/pkg/mmm-linux dist/pkg/linux/mmm
mv dist/pkg/mmm-macos dist/pkg/macos/mmm
mv dist/pkg/mmm-win.exe dist/pkg/win/mmm.exe

zip -j "dist/minecraft-mod-manager-${VERSION}-linux.zip" dist/pkg/linux/mmm
zip -j "dist/minecraft-mod-manager-${VERSION}-macos.zip" dist/pkg/macos/mmm
zip -j "dist/minecraft-mod-manager-${VERSION}-windows.zip" dist/pkg/win/mmm.exe
