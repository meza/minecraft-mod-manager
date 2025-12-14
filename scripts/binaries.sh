#!/bin/bash

VERSION=$1

zip -j "dist/minecraft-mod-manager-${VERSION}-linux.zip" build/linux/mmm
zip -j "dist/minecraft-mod-manager-${VERSION}-macos.zip" build/darwin/mmm
zip -j "dist/minecraft-mod-manager-${VERSION}-windows.zip" build/windows/mmm.exe
