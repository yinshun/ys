# 漢梵對照（`<术>`）工作總結（2026-10-31）

## 0. 一句話

`ys.adb` 把書裡「漢詞（梵語／巴利語／藏語）」的對照做成結構化資料
`<术 sk|pi|tb="外語">漢詞</术>`，共 **8,966** 條（梵 8,488、巴 473、藏 5）。
CBETA 的 TEI 只用 `<foreign xml:lang="sa">` 標外語，**沒有**「漢詞邊界＋外語」的對照表，
也沒有詞表；這是 ys 相對 CBETA 的差異化功能，發布時 `ys.html`／公告要寫這段。

## 1. 目前產物（2026-10-31 21:22 已發布到主站＋鏡像，sha256 驗證一致）

| 項目 | 值 |
|---|---|
| `D:\accelon3\ys.adb` | 28,037,302 bytes、SHA256 `4B95DF48714BC1D0…` |
| linecount | 328,601 |
| `<聯>`／`j` | 20,757／9,167 |
| `<引>` / `<註>` / `<釋>` | 49,103 / 16,355 / 15,960 |
| `<术>` | 8,966（跨行 459、單行 1 字 377） |
| CB 缺字 | gaiji=484、missing=0（`build/ys-missing-gaiji.tsv`） |
| `xrev` | 20,752 筆 |

## 2. 資料流

```text
CBETA Y TEI ──y2adb──▶ build/ys-text.txt ──prep──▶ ys-text-full ──calibrate──▶ ys-linked ──addj──▶ pack ──▶ ys.adb
```

`<术>` 在 **y2adb** 產生；後面的步驟只加 `<聯>`／`j`，不碰術語。
`build.ps1` 在 `addj` 之後、`pack` 之前跑 `tools/check-links.py`，數字不符就中止、不寫 deployment。

## 3. 抽取規則（現行，全部在 `cmd/y2adb/main.go`）

1. **來源**：`<foreign xml:lang="sa|pi|bo">` → `sk`／`pi`／`tb`；其他語言（en、fr、ja…）不包。
2. **詞界**：括號（`（` 或 `(`）往左找；停止字集見 `termStops`；靜態前綴黑名單見
   `termStripPrefixes`（朝代、虛詞、`有作`／`或作`／`亦作`／`譯作`、`佛化`…）。
   - `junkLastCut`：切在詞裡最後一個虛詞之後（虛詞不能在詞首，保護「有為」「說一切有部」）。
   - 例外：去掉的前綴是 `無`／`不`／`非` 時不縮（梵語 `a-` 前綴的否定）。
3. **書名號**：`《吠陀》（Veda）` → `《<术 sk="Veda">吠陀</术>》`（書名號留在外面）。
4. **漢字括號注**：`盧遮（舍）那（Rocana）` 這種整條跳過（作者指示，實例不多）。
5. **梵語含 `-`**：漢詞不可能 1 字；往左多吃 2–3 字，若被標籤擋住就整條不包。
6. **反查正解**：同（語言＋外語）取最常見的短後綴（≥3 次）當正解；同義異譯（摩竭陀／摩揭陀、
   巴利／巴利語）不動。
7. **手動覆寫**：`data/term-overrides.tsv`（32 條），格式 `foreign ⇥ term ⇥ note`；
   `term` 填 `-` 表示「整條不標記」。覆寫優先於所有自動規則。
8. **跨行**：`<lb>` 切斷、括號在行首（`閻浮提畔\n（Jambudvīpaśīrṣa）`）、跨 `<頁>`／`<註>`、
   `《往｜世書》（Purāṇa）` 都有專門處理；`<术>` 因此可以跨行（開在上一行、收在下一行）。
9. **CB 缺字**：讀 teiHeader 的 `<charDecl>`（`<char xml:id>` ＋ `<mapping type="unicode">`／
   `normal_unicode`）；`<g>` 內文 1 字就用它，否則查 charDecl，都沒有才留 `&CB…;` 並記表。
10. **硬補丁**：`fixChayatan()` 只處理「六〈頁〉（內）處」這個樣式（作者指定，parser 不動）。

## 4. 詞表與工具

| 檔案／工具 | 內容 |
|---|---|
| `build/term-table.tsv` | 依（語言＋外語）分組：`lang / foreign / term / count / variants / total` |
| `build/term-list.md`、`build/term-list.tsv` | 人看／機器看的詞表 |
| `build/fgdict-terms.tsv` | 佛光大辭典抽出的 7,502 組「漢詞—梵／巴」對照（驗證用） |
| `build/remaining-one-char-terms.tsv` | 仍只有 1 字的 377 筆（含前後 40 字與原因分類） |
| `build/one-char-*.tsv` | 上表的分類版：stop 248、han 88、crossline-miss 29、tag 16、linestart 8（會隨建置變動） |
| `build/long-terms.tsv`／`long-terms-strict.tsv` | 疑似過長（366／34）與建議正解 |
| `tools/fgdict-terms.py` | 從 `D:\2023\fgdict-corpus\xml` 抽對照表（`--check` 會比對 ys） |
| `tools/term-boundary-report.py` | 產生 1 字清單與分類 |
| `tools/term-long-report.py` | 產生過長清單 |
| `tools/check-links.py` | 建置後鏈結健檢（20,754／9,703／9,166） |

## 5. 已知限制（誠實清單）

- **377 個單字詞**沒有逐筆人工確認；其中 248 筆是「前一字是標點／停止字」。
- **縮短規則**只防 `無/不/非`；`大/小/舊/新` 這類語義前綴沒有防。
- **停止字**是長尾工程：這輪動了 `不`（移除）、`到/至`（加入），其餘沿用舊值。
- **跨標記／空行**的個案靠覆寫或 `fixChayatan()` 這類硬補丁，沒有通則。
- **佛光大辭典**只用來驗證，沒有反向修正邊界。
- **`<术>` 跨行**需要閱讀器支援；目前 accelon26 尚未實作 `<术>` 的顯示。
- 統計數字會隨規則調整而變（跨行 432→459、單行 1 字 449→377），文件數字以最新建置為準。

## 6. 閱讀器端待辦（`D:\2026\adb-converter`，accelon26）

1. `<术>` 渲染：漢詞照常、外語依設定「行內顯示／hover 顯示／關閉」。
2. 三種語言不同顏色（`sk`／`pi`／`tb`），建議 class `term-sk`／`term-pi`／`term-tb`。
3. 支援跨行 `<术>`（開在上一行、收在下一行）。
4. 邊界：屬性值可能含 `&quot;`；同一外語可能對多個漢詞（同義異譯），不要假設一對一。

## 7. 西文檢索交接（下一個工作區）

`ys.adb` 裡每一條 `<术>` 都帶著外語屬性，等於**現成的「西文—漢詞」索引**，不必再跑對齊或 OCR：

- 最快：`glossary` 容器（2026-10-31 起）已經是去重好的「語言／外語／漢詞／次數」清單（4,107 筆），直接讀它，不必掃全文。`glossary.ts` 已提供 `loadGlossary()`／`foldGlossary()`／`matchGlossary()`，閱讀器「詞彙」分頁就是用它。
  建 `外語 → (頁 id, 漢詞, 位置)`；可依語言分開索引。
- 查詢：使用者輸入拉丁字母時走「西文」模式；命中後跳到該頁並高亮漢詞（不是外語）。
- 詞表：`build/term-table.tsv` 可直接當離線詞典（含同義異譯與次數）。
- 驗證：`build/fgdict-terms.tsv`（7,502 組）可當第二來源比對。
- 已知風險：同一外語多漢詞（多對多）、`sk` 值可能有 `&quot;`、跨行 `<术>` 的屬性在開標籤上。

## 8. 發布檢查清單

1. `build/term-table.tsv`、`build/fgdict-terms.tsv` 是最新版的。
2. `ys.html`／公告寫「漢梵對照」這段（條數、三語言、斷詞／詞表用途、CBETA 沒有提供）。
3. `tools/check-links.py` 通過、`gaiji=… missing=0`。
4. 未 commit、未發布（等作者指示）。


## 9. 詞彙表與閱讀器（2026-10-31 補）

`glossary` 容器已改成 8 欄，每列都帶第一次出現的位置：

```text
["sk","Ābhāsvāra","光音天",1,279653,"38p0077a",148,6]
```

`lang`／`foreign`／`han`／`count`／`line`／`page`／`off`／`hanline`。

閱讀器（`D:\2026\adb-converter` 的 `docs/glossary.md` 有完整說明）：

- 左欄「詞彙」分頁；漢語／外語排序、任意／開頭／中間／結尾比對、梵／巴／藏圓圈。
- **點整列＝直接跳到該詞第一次出現**（行號＋行內漢字，精準且整詞塗滿）；
  列尾 🔍＝全文檢索。
- 選取文字 → 查看：一個對應語一顆「查 …」，用詞典查（不是改詞彙輸入區）。
- 全文搜尋打西文：先用詞彙表換成漢詞再查，摘要附註換詞。
- 完整說明另見 `D:\2026\adb-converter\docs\glossary.md`。
