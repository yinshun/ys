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
| `publish.ps1` | 發布：adb ＋公開資料庫頁（index／manifest）＋官網，一次做完 |
| `build.ps1` | 一鍵重建：`y2adb` → `prep` → `calibrate` → `addj` → `pack` |
| `paths.ts` | 所有外部路徑（可用環境變數覆寫） |
| `cmd/y2adb/` | CBETA Y 部 TEI → Accelon3 文字（Go，只用標準庫） |
| `prep.ts` | 併入附錄與後記，補 `<章>/<節>/<項>` 的 `u="文"` |
| `calibrate.ts` | 依對照表把大正藏引文包成 `<聯 i q>`（字元級鏈結） |
| `addj.ts` | 把 `q` 對回大正頁面，補 `j="起-訖"`（長引文只取前 80 字；只加 `j`，`q` 與正文不動） |
| `pack.ts` | 用 `adb-converter` 的 `authorAdb` 打包 `ys.adb`（含 2 張 logo ＋ 75 張插圖） |
| `images.ts` | 一次性：CBR2X-figures／舊 corpus 的圖 → `Y-images`（資源名 `svg/YNN/…`） |
| `extract-appendix.ts` | 一次性：從舊 `yinshun.adb` 重抽附錄 |
| `data/` | `term-overrides.tsv`（作者手動指定的術語）、`appendix.txt`（舊版卷首五頁）、`ys-epilog26.txt`（後記）、`u-attrs.txt`（u= 規則）、`aln-quotes2.tsv`（大正引文） |
| `alignments/` | `yinshun-taisho.tsv`（印順 → 大正藏對照表，14,128 列；2026-10-30 從 offtext 移入） |
| `logo/` | `ys_logo.png`（封面）、`yinshunfoundation.png`（年表頁插圖） |
| `docs/ys-steps.md` | 完整步驟、標記對照、預期數字與陷阱 |
| `docs/hanfan-summary.md` | 漢梵對照（`<术>`）工作總結：規則、詞表、限制、閱讀器與西文檢索交接 |

## 快速重建

```powershell
pwsh D:\2026\ys\build.ps1
```

前置資料（第一次才要準備）：

1. **CBETA Y**：`git clone --filter=blob:none --sparse --depth 1 https://github.com/cbeta-org/xml-p5.git D:\2026\xml-p5` 然後 `git sparse-checkout set Y`
2. **圖檔**：`bun run images.ts`（需要 `cbeta-git/CBR2X-figures` 與 `yinshun/yinshun-corpus` 的 sparse clone，見 `docs/ys-steps.md` §2.4）
3. **舊 corpus XML**：`D:\2023\yinshun-corpus\xml`（頁元與引文定位）
4. **對照表**：`D:\2026\ys\alignments\yinshun-taisho.tsv`（已在本 repo）

## 產物

`build/ys-text.txt` → `build/ys-text-full.txt` → `build/ys-linked.txt`（含 `j`）→ 直接寫
`D:\accelon3\ys.adb`（可用 `YS_DEPLOY` 改；工作區不留 adb，跑 `pack.ts` 就會覆寫它）。

預期：`dbname ys`、`linecount 318227`、`26.8 MiB`、`xrev` 20,752 筆、`resources 77`（2 logo ＋ 75 圖）、
`<聯>` 23,679（含 `place="inline"` 的行內大正／南傳對照）、`j` 9,213、`<引>` 49,103、`<术 sk|pi|tb>` 8,976（跨行 459）、`<表>` 259 張文字表格、`<圖>` 368 張（t="圖版"）、`glossary` 4,097 筆（含 line／page／off／hanline）、`<註>` 11,168（`<釋>` 10,773）、`andtagid 0`；CB 缺字 `gaiji=484 missing=0`（`build/ys-missing-gaiji.tsv`）。

## 特色：漢梵對照（下次發布要提）

`ys.adb` 把書裡「漢詞（梵語）」的對照做成結構化資料：`<术 sk|pi|tb="…">漢詞</术>`，
共 **8,852** 條（梵 8,336、巴 440、藏 5）。這件事 **CBETA 沒有提供**——CBETA 的 TEI 只用
`<foreign xml:lang="sa">` 標出外語，沒有「漢詞邊界＋外語」的對照表，也沒有詞表。

我們多做了一步：

1. **外語反查**：同一個外語值在整份語料裡應該對到同一個漢詞；不一致的就是邊界抓錯，
   用最常見的後綴當正解（同義異譯不動）。
2. **辭典交叉檢查**：從佛光大辭典抽出 7,502 組「漢詞—梵／巴」對照
   （`build/fgdict-terms.tsv`）比對，確認邊界沒有偏離。

用處：

- **斷詞**：詞界有外語可以驗證，不必只靠標點與虛詞。
- **詞表**：`build/term-table.tsv`（2,679 組，含同義異譯與出現次數）可直接給使用者，
  或做成術語索引、辭典附錄。
- **閱讀器**：`<术>` 可以選擇行內顯示或 hover 顯示原文（accelon26 待實作）。

發布 `ys.adb` 時，`ys.html`／公告要寫這一段；這是 ys 相對 CBETA 的差異化功能。
