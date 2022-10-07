#!/bin/bash

mkdir -p dist/pkg/linux
mkdir -p dist/pkg/win
mkdir -p dist/pkg/macos

mv dist/pkg/mmm-linux dist/pkg/linux/mmm
mv dist/pkg/mmm-macos dist/pkg/macos/mmm
mv dist/pkg/mmm-win.exe dist/pkg/win/mmm.exe
