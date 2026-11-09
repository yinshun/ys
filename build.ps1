# ys.adb 建置流程（印順法師佛學著作集）
#
# 需要：Go、Bun，以及旁邊的 adb-converter 工作區（通用的 adb 打包工具，第一步要一起存在）
# 資料：CBETA Y 的 sparse clone（預設 D:\2026\xml-p5\Y）＋圖檔 Y-images ＋大正藏對照表（repo 內 alignments\）
#
# 路徑集中在 paths.ts；要換位置就設環境變數（YS_XML／YS_IMAGES／YS_CORPUS_XML／YS_ALIGN／YS_DEPLOY）
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

if (-not $env:YS_XML) { $env:YS_XML = 'D:\2026\xml-p5\Y' }
if (-not $env:YS_IMAGES) { $env:YS_IMAGES = 'D:\2026\xml-p5\Y-images' }
New-Item -ItemType Directory -Force build | Out-Null

go run ./cmd/y2adb -src $env:YS_XML -out 'build\ys-text.txt' -lbs 'build\ys.lbs.tsv' -img $env:YS_IMAGES -gaiji 'build\ys-missing-gaiji.tsv'
bun run prep.ts        # 附錄（data/appendix.txt）＋後記（data/ys-epilog26.txt）
bun run calibrate.ts   # 依對照表插入 <聯>
bun run addj.ts        # 補字元級 j（把 q 對回大正頁面）

# 補圖版：CBETA <graphic> 只有 75 個，Y-images 有 377 個；依頁碼把沒引用的插在 <頁> 後
python tools\inject-images.py build\ys-linked.txt
if ($LASTEXITCODE -ne 0) { throw '補圖失敗（tools/inject-images.py）' }

# 從 yinshun.adb 抽頁圖 PNG，補 Y-images 沒有的頁（只補還沒有 <圖> 的頁）
bun run tools\import-yinshun-images.ts
if ($LASTEXITCODE -ne 0) { throw '補 yinshun 頁圖失敗（tools/import-yinshun-images.ts）' }

# 術語斷詞修正：用 adb-converter 的共用詞表與 CLI（只修 <术>，不動 <聯>/q/j）
# canonical：D:\2026\adb-converter\data\terms.tsv、data\term-fixes.tsv
bun run ..\adb-converter\ts\tools\terms.ts fix --text build\ys-linked.txt --write
if ($LASTEXITCODE -ne 0) { throw '術語斷詞修正失敗（tools/terms.ts）' }

# 鏈結健檢：<术> 插入若動到 <聯>/q/j 的數量，就在打包前中止（不寫 deployment）
python tools\check-links.py
if ($LASTEXITCODE -ne 0) { throw '鏈結健檢失敗（tools/check-links.py）：不打包、不寫 deployment。' }

bun run pack.ts        # 打包 ys.adb（含 logo 與正文插圖）
Write-Host 'ys.adb rebuilt'
