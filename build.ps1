# ys.adb 建置流程（印順法師佛學著作集）
#
# 需要：Go、Bun，以及旁邊的 adb-converter 工作區（通用的 adb 打包工具，第一步要一起存在）
# 資料：CBETA Y 的 sparse clone（預設 D:\2026\xml-p5\Y）＋圖檔 Y-images ＋大正藏對照表
#
# 路徑集中在 paths.ts；要換位置就設環境變數（YS_XML／YS_IMAGES／YS_CORPUS_XML／YS_ALIGN／YS_DEPLOY）
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

if (-not $env:YS_XML) { $env:YS_XML = 'D:\2026\xml-p5\Y' }
if (-not $env:YS_IMAGES) { $env:YS_IMAGES = 'D:\2026\xml-p5\Y-images' }
New-Item -ItemType Directory -Force build | Out-Null

go run ./cmd/y2adb -src $env:YS_XML -out 'build\ys-text.txt' -lbs 'build\ys.lbs.tsv' -img $env:YS_IMAGES
bun run prep.ts        # 附錄（data/appendix.txt）＋後記（data/ys-epilog26.txt）
bun run calibrate.ts   # 依對照表插入 <聯>
bun run addj.ts        # 補字元級 j（把 q 對回大正頁面）
bun run pack.ts        # 打包 ys.adb（含 logo 與正文插圖）
Write-Host 'ys.adb rebuilt'
