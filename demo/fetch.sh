#!/usr/bin/env bash
# Fetch the demo book used by demo.tape.
#
# 西遊記 (Journey to the West) by 吳承恩 is in the public domain; the EPUB is
# exported from Chinese Wikisource (zh.wikisource.org). The file is not
# committed to this repository (see .gitignore).
set -euo pipefail

cd "$(dirname "$0")"

out="xiyouji.epub"
url="https://ws-export.wmcloud.org/?lang=zh&page=%E8%A5%BF%E9%81%8A%E8%A8%98&format=epub"

echo "downloading $out ..."
curl -sL --http1.1 --retry 3 -o "$out" "$url"
echo "wrote $(pwd)/$out"
