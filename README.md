# ys — 印順法師佛學著作集（ys.adb）的建置來源

這個 repo 只放**印順獨有**的東西：來源資料、轉換與打包腳本。通用的 adb 讀寫／打包工具
在隔壁的 [`adb-converter`](../adb-converter) 工作區。

> **第一步（現在）**：建置 `ys.adb` 需要 **這個 repo ＋ `D:\2026\adb-converter`** 兩個都在，
> 而且放在同一個上層資料夾（預設 `D:\2026\`）。
> **未來**：等 `adb-converter` 的建置工具整理成可發佈的套件，客戶只要這個 repo 就能
> 在 CBETA 更新後自行重建 `ys.adb`。

## 目錄

| 路徑 | 說明 |
|---|---|
| `build.ps1` | 一鍵重建：`y2adb` → `prep` → `calibrate` → `addj` → `pack` |
| `paths.ts` | 所有外部路徑（可用環境變數覆寫） |
| `cmd/y2adb/` | CBETA Y 部 TEI → Accelon3 文字（Go，只用標準庫） |
| `prep.ts` | 併入附錄與後記，補 `<章>/<節>/<項>` 的 `u="文"` |
| `calibrate.ts` | 依對照表把大正藏引文包成 `<聯 i q>`（字元級鏈結） |
| `addj.ts` | 把 `q` 對回大正頁面，補 `j="起-訖"`（長引文只取前 80 字；只加 `j`，`q` 與正文不動） |
| `pack.ts` | 用 `adb-converter` 的 `authorAdb` 打包 `ys.adb`（含 2 張 logo ＋ 75 張插圖） |
| `images.ts` | 一次性：CBR2X-figures／舊 corpus 的圖 → `Y-images`（資源名 `svg/YNN/…`） |
| `extract-appendix.ts` | 一次性：從舊 `yinshun.adb` 重抽附錄 |
| `data/` | `appendix.txt`（舊版卷首五頁）、`ys-epilog26.txt`（後記）、`u-attrs.txt`（u= 規則）、`aln-quotes2.tsv`（大正引文） |
| `logo/` | `ys_logo.png`（封面）、`yinshunfoundation.png`（年表頁插圖） |
| `docs/ys-steps.md` | 完整步驟、標記對照、預期數字與陷阱 |

## 快速重建

```powershell
pwsh D:\2026\ys\build.ps1
```

前置資料（第一次才要準備）：

1. **CBETA Y**：`git clone --filter=blob:none --sparse --depth 1 https://github.com/cbeta-org/xml-p5.git D:\2026\xml-p5` 然後 `git sparse-checkout set Y`
2. **圖檔**：`bun run images.ts`（需要 `cbeta-git/CBR2X-figures` 與 `yinshun/yinshun-corpus` 的 sparse clone，見 `docs/ys-steps.md` §2.4）
3. **舊 corpus XML**：`D:\2023\yinshun-corpus\xml`（頁元與引文定位）
4. **對照表**：`D:\2026\offtext\alignments\yinshun-taisho.tsv`

## 產物

`build/ys-text.txt` → `build/ys-text-full.txt` → `build/ys-linked.txt`（含 `j`）→ 直接寫
`D:\accelon3\ys.adb`（可用 `YS_DEPLOY` 改；工作區不留 adb，跑 `pack.ts` 就會覆寫它）。

預期：`dbname ys`、`linecount 329391`、`27.6 MiB`、`resources 77`（2 logo ＋ 75 圖）、
`<聯>` 29,624（正文 18,180＋註 11,444）、`j` 17,035、`andtagid 0`。
