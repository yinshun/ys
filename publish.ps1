# ys.adb 發布（2026-11-02）——**2026-11-06 已停用**：發布交給 adb-converter 統一處理。
#
# 「發布」＝ adb 本體 ＋ 公開資料庫頁（index/manifest）＋ 官網，一次做完；
# 不能只上傳 adb，否則 https://shutonggui.cn/accelon26/adb/index.html 的日期會停在舊版。
#
#   pwsh D:\2026\ys\publish.ps1              # adb + index/manifest + 官網
#   pwsh D:\2026\ys\publish.ps1 -SkipAdb     # adb 已上傳，只更新資料庫頁與官網
#
# 前置：deploy\adb-notes.json 要有本機 ys.adb 這個 sha256 的「改版說明」，
# 沒有的話這支會停下並告訴你要補哪個 sha。
$ErrorActionPreference = 'Stop'

if ($env:YS_LEGACY_PUBLISH -ne '1') {
    Write-Host 'publish.ps1 已停用：ys 只產出 adb，部署由 D:\2026\adb-converter 統一處理（程序＋adb 一起上）。' -ForegroundColor Yellow
    Write-Host '若真的要手動發布，先設 $env:YS_LEGACY_PUBLISH='1' 再跑。'
    exit 2
}
Set-Location $PSScriptRoot

$adb = 'D:\accelon3\ys.adb'
if (-not (Test-Path $adb)) { throw "找不到 $adb" }
$sha = (Get-FileHash $adb -Algorithm SHA256).Hash.ToLower()

$notesPath = '..\adb-converter\deploy\adb-notes.json'
$notes = Get-Content $notesPath -Raw -Encoding UTF8 | ConvertFrom-Json
$entry = $notes.'ys.adb'
if (-not $entry -or $entry.sha256 -ne $sha) {
    throw "deploy\adb-notes.json 沒有 ys.adb 這個 sha 的說明：$sha`n請先在該檔補上 notes（sha256 + notes），再發布。"
}

Push-Location ..\adb-converter\ts
try {
    if (-not $SkipAdb) {
        Write-Host '── 1/3 adb 本體 → 主站＋鏡像'
        bun run tools/publish-adb.ts --files ys.adb --allow-large
        if ($LASTEXITCODE -ne 0) { throw 'adb 上傳失敗' }
    } else {
        Write-Host '── 1/3 跳過 adb 上傳（-SkipAdb）'
    }

    Write-Host '── 2/3 公開資料庫頁（index ＋ manifest）'
    bun run tools/make-adb-index.ts --adb-dir D:\accelon3
    if ($LASTEXITCODE -ne 0) { throw 'make-adb-index 失敗' }
    bun run tools/make-adb-manifest.ts
    if ($LASTEXITCODE -ne 0) { throw 'make-adb-manifest 失敗' }

    Write-Host '── 3/3 官網（www/，含 adb/index.html 與 manifest.json）'
    bun run tools/deploy.ts --site-only
    if ($LASTEXITCODE -ne 0) { throw '官網上傳失敗' }
} finally {
    Pop-Location
}

Write-Host ''
Write-Host "完成：ys.adb $sha 已發布（adb＋公開資料庫頁＋官網）"
