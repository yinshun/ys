# ys.adb 製作步驟（CBETA Y → Accelon3）

`ys.adb` 是「印順法師佛學著作集」的另一個建置，來源是 **CBETA 官方 Y 部 XML**
（`cbeta-org/xml-p5`），不是舊的 `yinshun.adb`。

兩者定位：

| | `yinshun.adb` | `ys.adb` |
|---|---|---|
| 來源 | 舊的 accelon 版 corpus（`D:\2023\yinshun-corpus`） | CBETA Y 部 TEI P5 |
| 流通 | 很廣，**保持原樣** | 新能力都在這裡 |
| 內容 | 44 冊＋卷首附錄 | 44 冊（CBETA）＋附錄（舊版卷首）＋後記 |
| 大正鏈結 | 13,920 個（頁欄級為主） | 17,916 個（字元級，含 `q` 引文；見 §4.2） |

`ys.adb` 的檔頭（`<集>`、`<隱>`、`<樹>`、`<引>`、`<摘要頭>`、`<參考資料>`、
`<英文名>`）與 `yinshun.adb` 相同，其他標記不強求一致。

---

## 0. 前置需求

**兩個 repo（第一步）**：這個 repo（`D:\2026\ys`）＋ 通用的 adb 工具工作區
`D:\2026\adb-converter`，兩者放在同一個上層資料夾（預設 `D:\2026\`）。
`pack.ts` 會從隔壁 import `authorAdb`／`MAXLINELEN`。

- Go（`go run`）、Bun、PowerShell
- 網路（第一次抓 CBETA Y）
- 舊版 `D:\accelon3\yinshun.adb`（附錄來源，見 §4）
- 對照表 `D:\2026\ys\alignments\yinshun-taisho.tsv`（已在本 repo）
- 語料 `D:\2023\yinshun-corpus\xml\y*.xml`（頁元與引文定位用）

工作目錄一律是 `D:\2026\ys`。

---

## 1. 取得 CBETA Y

只需要 `Y/` 一個目錄，用 sparse clone（整庫 1.4 GB，Y 只有 49 MB）：

```powershell
cd D:\2026
git clone --filter=blob:none --sparse --depth 1 https://github.com/cbeta-org/xml-p5.git xml-p5
cd xml-p5
git sparse-checkout set Y
```

結果：`D:\2026\xml-p5\Y\Y01\Y01n0001.xml` … `Y44\…`（44 檔）。

CBETA 更新後，只要在同一個 clone 裡 `git pull`，再跑 §7 即可。

---

## 2. 轉換：`cmd/y2adb`

```powershell
go run ./cmd/y2adb -src 'D:\2026\xml-p5\Y' `
  -out 'build\ys-text.txt' `
  -lbs 'build\ys.lbs.tsv'
```

輸出：

- `build/ys-text.txt`：Accelon3 文字（一行一個 `<lb>` 行）
- `build/ys.lbs.tsv`：`冊<TAB>行號<TAB>lb`（除錯與人工核對用）

### 2.1 標記對照

| TEI | ys 文字 |
|---|---|
| `<body>` 開始 | `<檔 n="Y01n0001.xml">` |
| 冊標題（`title level="m"`） | `<書 l="目次" t="記" id="1">般若經講記</書>` |
| `<milestone unit="juan">` | `<卷 n="N">N</卷>` |
| `pb` 前綴類別改變 | `<頁元 n="書名" t="頁.">正文／附錄／後記</頁元>` |
| `<pb n="0065a"/>`（非 `type="old"`） | `<頁 id="22p0065a"/>`（`冊號`＋`p`＋`pb`） |
| `<lb/>`（非 `type="old"`） | 換行 |
| `<p>` | `<段/>` |
| `<head>`、`<jhead>` | `<章 u="文">…</章>`；層級看前面的 `<cb:mulu level>`：1→部、2→章、3→節、4→項、5→干、6→一、7→數、8 以後→ＡＢＣ…；層級鏈寫在每個 `<書 t="部,章,節,項,干,一,數,Ａ…">`（與舊 `yinshun.adb` 的目錄一致；標籤名只是層級，畫面顯示的是標題文字）。第一個 `<head>` 若與書名相同，只留文字、不開節點 |
| `<note>`（行內註，如「方等泥洹經〔註82〕竺法護釋法顯」） | `<註 n="37.082"/>`（行內）＋段末 `<段/><釋 n="37.082">竺法護釋法顯</釋>`；註裡的 `<ref>` 轉成 `<聯>`（2026-10-30 起與腳註相同，不再寫 `〔註n〕` 純文字行） |
| `<app>` | 只留 `<lem>` 底本文字，`<rdg>` 略過 |
| `<g ref="#CB01121">…</g>`（CB 缺字） | 讀 teiHeader 的 `<charDecl>`（`<char xml:id>` ＋ `<mapping type="unicode">`／`normal_unicode`）。`<g>` 內文 1 字就用它，否則用 charDecl 的 Unicode；都沒有就留 `&CB…;` 並記進 `build/ys-missing-gaiji.tsv`。Y 部共 696 個（486 個在 body），2026-10-31：成功 484、缺 0。 |
| `<foreign xml:lang="sa|pi|bo">`（漢詞（外語）） | `<术 sk\|pi\|tb="…">漢詞</术>`（外語進屬性、正文只留漢詞）；**可跨行**：被 `<lb>` 切成兩行時，開標籤在上一行尾端、`</术>` 收在下一行（2026-10-31；整份文件標籤平衡即可，不要求單行相等） |
| `<graphic url="../figures/Y/Y08p0052_01.svg"/>` | `<圖 n="svg/Y08/Y08p0052_01.svg"/>`（圖檔要存在，見 §2.4） |
| `<anchor n="0037056"/>`（腳註引用） | `<註 n="37.085"/>`（`<書id>.<序>`，與段末的 `<釋>` 對應） |
| `<back>` 的 `<cb:div type="yinshun-notes">` | **書末**（`</body>` 前）集中輸出 `<段/><釋 n="37.085">`注文`</釋>`；注文裡的 `<ref target="#vol:12;page:p1081a">` 轉成 `<聯 i="taisho?12p1081a">` |
| `<note>`／`<app>` 以外的元素 | 只取文字，標籤丟棄 |

`<g>`、`<cb:mulu>` 一律跳過；`<anchor>` 轉成 `<註>`，`<back>` 只讀 `type="yinshun-notes"`（印順自己的腳註，見上表與 §8）。

#### 補 `u="文"`：外部規則檔 `data/u-attrs.txt`

CBETA 的 Y 部 XML 沒有 accelon 的 `u=` 屬性，少了它節點就不會進「頻次」
（`<樹 s="*u=文">`），那幾段文字在頻次裡等於不存在。所以 `prep.ts` 每次重建
都會讀 **`data/u-attrs.txt`**（一行：標籤 → u 值）把缺的補上，已經有 `u=` 的不動：

```text
章	文	CBETA 的 <head> 一律是「文」
節	文
項	文
```

2026-10-28 實測：ys.adb 的「節」有 1,588 個、其中 75 個（附錄與後記）沒有 u，
重建後 1,588／1,588 全數帶 u，"頻次" 不再漏段。要補新的標籤或改值，只改這個檔。

### 2.2 檔頭

`header` 常數直接沿用 `yinshun.adb` 的第一個 `<檔>`：

```text
<檔 n="ys.xml"><集 l="圖集,註釋群,編目資訊">印順法師佛學著作集</集>
<隱><樹 s="*" t="書" pf1="$書.id;.">目錄</樹>
<樹 s="*u=文" pf="$頁.id;)">頻次</樹><樹 s="~00602000">西文</樹></隱>
<隱><引>《$書;》：「$_;」( Y $*頁; )</引><摘要頭>$書;($頁;)</摘要頭>
<參考資料 n="taisho,wxzj"/></隱>
<英文名>Master Yinshun Corpus of Buddhist Studies</英文名></檔>
```

### 2.3 預期數字（2026-10 版）

```text
lines=326982 卷=460 書=44 段=31882 註=16355 釋=15960 頁=17510 頁元=73；gaiji=484 missing=0（另見 §2.1）
```

頁元 **73**（CBETA 的 `pb` 前綴有 a、b、y、z），比語料／對照表的 **70** 多 3 個；
這是正常的，見 §5.1。

### 2.4 圖片：`images.ts` → `D:\2026\xml-p5\Y-images`

CBETA 的 Y 部 XML 只有圖的**檔名**（`<graphic url="../figures/Y/Y08p0052_01.svg"/>`），
沒有圖檔本身。圖檔有兩個來源，後者（CBETA 現行版）蓋掉前者：

| 來源 | 內容 |
|---|---|
| `cbeta-git/CBR2X-figures` 的 `Y/` | **現行版圖檔**，檔名已是 `Y08p0052_01.svg`（113 個檔：64 svg ＋ 49 gif，含重複的點陣版） |
| `yinshun/yinshun-corpus` 的 `svg/` | 舊 corpus（367 張），檔名 `y08.0052.svg`，要改名 |

```powershell
cd D:\2026\ys
git clone --filter=blob:none --sparse --depth 1 https://github.com/cbeta-git/CBR2X-figures.git .local\CBR2X-figures
git -C .local\CBR2X-figures sparse-checkout set Y
git clone --filter=blob:none --sparse --depth 1 https://github.com/yinshun/yinshun-corpus.git .local\yinshun-corpus
git -C .local\yinshun-corpus sparse-checkout set svg
bun run images.ts      # --figures .local\CBR2X-figures\Y --src .local\yinshun-corpus\svg
```

`images.ts` 把 corpus 的舊檔名改成 CBETA 的檔名（同頁多張時 `-1`、`-2` 變成
`_01`、`_02`），CBR2X 的檔名照用：

```text
y08.0052.svg   → Y08/Y08p0052_01.svg
y37.0503-2.svg → Y37/Y37p0503_02.svg
（CBR2X 的 Y09p0125_01.gif 原樣）
```

放到 `D:\2026\xml-p5\Y-images\svg\YNN\`。**相對於 Y-images 的路徑就是 adb 的資源名**：

```text
Y-images\svg\Y08\Y08p0052_01.svg   →   <圖 n="svg/Y08/Y08p0052_01.svg" t="圖版"/>
```

`y2adb` 只在圖檔真的存在時才寫 `<圖>`（缺圖留白，不畫佔位方塊），`pack.ts` 只打包
正文用到的圖。2026-10-29 現況：XML 引用 75 張，**75 張都收得到**（64 svg ＋ 11 gif，
gif 集中在 Y13、Y33 與 Y09p0125_01 這種只有點陣版的圖）。

> SVG 是 inline `<svg class="inlineSvg figureSvg">`（照原圖實寸、白紙底），只有點陣版
> 的走 `<img class="inlineImg figureImg">`。舊 `yinshun.adb` 是 `images/y01-18.png`
> 的點陣圖，兩者不通用。

---

## 3. 附錄與後記：`prep.ts`

```powershell
bun run prep.ts
```

輸入 `build/ys-text.txt`，輸出 `build/ys-text-full.txt`：

```text
檔頭
  ＋ 44 冊（上一步的產物）
  ＋ 附錄（書 id 99，5 頁，data/appendix.txt）
  ＋ 後記（書 id 100，頁 0p1，data/ys-epilog26.txt）
```

### 3.1 後記

「2026年AI版後記」，約 1,500 字，獨立成書、放在**全書最後**、**沒有大正鏈結**。
書名與章名不重複：書名「2026年AI版後記」，章名「後記」。
內容六節：

1. 保存歷史文獻原貌（時代背景、不作刪改、與中華書局 2009 簡體版不同、
   僅供學術研究、遵守當地法規）
2. CBETA 刪除舊版內容（五篇卷首資料、「參與人員名單」、不尊重歷史、
   本版自動補回）
3. 授權與致謝（開發者葉健欣、基金會授權、2026 年 AI 版主要測試者陳裕文、
   增值部分 CC BY）
4. 無名者的光（常明法師逐字錄入七百萬字、德志法師組織第一版光碟與招募開發者、
   厚觀法師——印順導師關門弟子，二十多年維護系統、以學術學養逐字校勘；
   2017 年他號召三十幾位福嚴佛學院師生與推廣班學員查找引文在大正藏的精確位置——名字已不可考）
5. 感謝這個時代（DeepSeek、祖國的數字基建與生活環境、文化自信）
6. 回連結是怎麼做到的（給一般讀者的白話說明：每個字編門牌號碼、
   把「這裡就是大正某頁某欄」寫在號碼旁、另做一份按「被引用的一方」排好的倒過來目錄）

第 6 節刻意不用 VA、token、TEI、索引容器等術語；要改技術細節改
`ys-steps.md`，後記只留白話。

結語：「願此資料庫有助佛法的研究與流傳，也願讀者不忘前人的辛勞。年輕人有了 AI 神器，
更要勇於實現夢想，承擔起續佛慧命的責任。」

後記原始檔是 **`data/ys-epilog26.txt`**，**已含 Accelon3 標記**（`<章>`、`<節>`、
`<段/>` 等，一行一個 adb 行）；`prep.ts` 每次重建都直接讀這個檔，改字改它即可。

### 3.2 附錄（自動補入）

CBETA Y **沒有**舊版卷首那五頁。這五頁已抽出成 `data/appendix.txt`（含標記）並進版控，
重建時直接讀檔，**不需要 `yinshun.adb`**：

| 頁 | 字數 |
|---|---|
| 00p印順導師略譜 | 10,962 |
| 00p印順導師著作出版年表 | 6,234 |
| 00p印順導師略傳 | 5,792 |
| 00p印順文教基金會簡介 | 268 |
| 00p參與人員名單 | 181 |

輸出時另立 `<檔 n="appendix.xml">`、`<書 … id="99">附錄（舊版卷首資料）</書>`，
頁 id 沿用 `00p…`。這樣 CBETA 日後補上或修改 Y 部 XML，重跑流程時
CBETA 的新文字與舊版附錄都會自動併回，不需要手動搬。

> 這五頁只存在 `yinshun.adb`（`D:\2023\yinshun-corpus` 的 `y00.xml`、
> `appendix.xml` 都沒有），所以 `yinshun.adb` 是必要輸入，請保留。

#### 分行要保留（2026-10-28 修正）

`data/appendix.txt` 現在是 **1,552 行**，一行一個 adb 行，**照 `yinshun.adb` 原樣保存**。
它由一次性工具 `extract-appendix.ts` 從 `D:\accelon3\yinshun.adb` 抽出：

```powershell
bun run extract-appendix.ts     # 只在需要重取時跑
```

先前版本把每頁壓成一整行（最大的 35,143 字），結果踩到 Accelon3 的**行長上限
`MAXLINELEN = 8192` bytes（4,096 字）**：`lines.physical` 會把超長行截在 4,096 字，
內文卻完整打包，行表與內文從此錯開——附錄尾巴的「參與人員名單」「略傳」「略譜」
「基金會簡介」定位不到，搜尋「葉健欣」第二筆就顯示「VA …：無法定位」。
現在 `pack.ts` 會先檢查行長，超過 4,096 字直接報錯，不會再默默產生壞檔。

---

## 4. 大正鏈結：`calibrate.ts`

```powershell
bun run calibrate.ts
```

輸入 `ys-text-full.txt` ＋ 對照表 ＋ 語料 XML，輸出 `build/ys-linked.txt`。

### 4.1 頁元對映（語料 70 頁元）

對照表的印順座標是 `<頁元>p<頁>.<行><字>`，頁元不是冊。規則是：
**同一檔案中 `pb.n` 首字元類別（`a`、`b`、數字）連續相同的區段，就是一個頁元**。

| 檔案類型 | 檔數 | 每檔頁元 | 小計 |
|---|---|---|---|
| `a` 段＋數字段（`a#`） | 10 | 2 | 20 |
| `a`＋`b`＋數字（`ab#`） | 8 | 3 | 24 |
| 純數字 | 26 | 1 | 26 |
| | **44** | | **70** |

`y00.xml`（目錄）不計。頁碼是「該頁元內第幾個 pb」（1 起算）。

### 4.2 定位流程

1. 從語料 XML 的 `<lb n="頁.行">` 取出該行的漢字（只用漢字，不含標點）
2. 依頁元對映把 `(頁元, 頁, 行, 字)` 換成語料文字
3. 在對應的 Y 卷（`yNN` ↔ `YNN`，兩邊標題一致）裡找該段文字：
   - 先精確比對（10 字種子＋最長連續匹配）
   - 失敗再用寬鬆比對（多個 6 字種子＋LCS，允許插字漏字，匹配 ≥10 字）
4. 同一目標且相鄰／重疊的區間先合併，再排除重疊
5. 在 `ys` 文字上用 `<聯 i="taisho?<卷>p<頁><欄>" q="<大正引文>">…</聯>` 包住
6. **位置一律用「原始字元」索引**（連標點一起算）：`locate()` 給的是漢字序號，
   換成原始位置時要走 `hanToRaw`。先前直接把漢字序號當字元位置，引文裡的「，」
   會把 `</聯>` 提前一個字——「過去未來色無常，況現在**色！**」就少塗了「色」。
7. 收尾多吃標點（作者 2026-10-28）：引文結尾通常落在標點上，`</聯>` 會往後吃
   緊接的 `，。！？；：、）」』】》〉…—·`，因為沒有人會把引文斷在標點前面。
8. **`</聯>` 不吃下一句的開括號**（2026-10-29）：結束位置是「下一個漢字之前」，
   會順手把 `「『（《〈【` 也包進來；收尾時要退回這些開括號，否則會看到
   `：「</聯>` 這種把下一句引號吃掉的樣子（跨行引文在行尾收合時也要退）。
9. **抽引文用印順座標，不是大正座標**（2026-10-29 修正）：多行引文要取
   `f[6]`（結束行）／`f[7]`（結束字）。曾經誤用大正的 `f[15]`／`f[16]`，
   把引文一路抓到第 20 行，整段討論都被包進 `<聯>`——38p0156a 那條 167 字的聯
   就是這樣來的（本該在「皆為空故」結束）。

`q` 是從大正文字按對照表 `(卷,頁,欄,行,字)` 抽出的引文（純漢字），
點 `<聯>` 時用來在大正頁面塗紅。

`j`（2026-10-28 擴充；2026-10-29 納入建置）：目標頁「可見漢字」索引 `起-訖`
（0 起算、end 不含），讓 accelon26 精準塗那一段；`i` 不變，舊 accelon3 只會忽略 `j`。
由 [`addj.ts`](../addj.ts) 在 `calibrate.ts` 之後補，見 §4.4。

### 4.3 預期數字

```text
spans 10474  exact 9723  relaxed 751  notfound 3654
accepted 9719
links 29624
```

`notfound` 多半是轉述、或兩邊文字差異過大。

`links`（`<聯>` 標籤數）比 `accepted`（引文筆數）多，是因為跨行引文會拆成
「每行一個聯」；2026-10-29 修正抽引文座標後，單條引文不再動輒跨五六行。

### 4.4 補 `j`：`addj.ts`

```powershell
bun run addj.ts
```

`build.ps1` 已排進這一步（`calibrate` 之後、`pack` 之前），就地改 `build/ys-linked.txt`。
引文若跨到標題行（`<部>/<章>/…`）一律不包（`calibrate` 的 `headingSkip`，2026-10-29：14 條），
否則 `</項>` 會被包進 `<聯>`，目錄樹的標題就被切壞。

- **定義**（與 accelon26 `renderForeign` 的 lineHtml 一致）：目標頁從 `<頁 id="…">`
  之後算起，跳過所有 `<…>` 標籤、只算漢字，遇到下一個 `<頁` 或 4,000 字為止；
  `j="from-to"` 的 `from` 是這段漢字串的 0 起算索引，`to` 不含。
- **怎麼算**（同 adb-converter 的 `retrofit-j.ts` ＋ `verify-j.ts`）：
  把 `q`（最多 160 字，`calibrate` 存的就是 160）整段放回目標頁找——精確比對用
  12→4 字種子，對不上再做寬鬆對位（允許插字漏字）；最後拿頁面原文複驗，找到才寫 `j`。
  **長引文也整段塗滿**（2026-10-30 起不再截成 80 字：37p0007a 那條 127 字的引文
  原本只塗到「呪藏」就斷）。
- **只加 `j`**：`q`、`i` 與正文一個字都不動；驗法是拿掉 `j="…"` 後與
  `calibrate` 的輸出逐字比對必須完全相同，`q` 逐條比對不得改變。
- 對不上（版本差異、轉述、跨頁）就不寫 `j`，閱讀器退回用 `q` 比對。
- 依賴 `D:\accelon3\taisho.adb`（唯讀，可用 `YS_TAISHO` 覆寫），跑一次約 6 秒。

2026-10-31 實測：`<聯>` 20,754（正文 9,310＋註 11,444）；帶 `q` 的 9,701 條；
**補上 `j` 9,178 條**（其中 469 條用寬鬆對位），沒有 `j` 的 523 條。
`38p0156a` → `taisho?8p562b` 的 `j="293-324"`。

> 與 `build/ys-j-verified-keepq.adb`（2026-10-29 的參考產物）的關係：短引文那部分
> 逐條相同；長引文不同——參考產物只塗前 80 字，本流程從 2026-10-30 起塗滿整段
> （最長 `j` 161 字，`>80` 的 2,643 條）。另外本流程多留 637 條參考產物被
> `repair-window`／`verify-j` 砍掉的範圍（例：`taisho?25p190a` 的 25 字引文，
> `j="309-334"` 正好是整段）。

### 4.5 反鏈快查表 `xrev`（2026-10-31）

`pack.ts` 會先用 `authorAdb` 產生一份「沒有 xrev」的檔，開它讀 `聯.i`／`聯.q` 與
`頁.id`，算出每一條 `<聯 i="db?page">` 的來源頁與來源 VA，再壓成 bzip2 JSON Lines
放進外層容器 `xrev`（1 個 block）：

```text
["XREV1", 20752]
["ys","36p0162a",5897587,"taisho","26p625c","原其大本頌有六千…"]
```

- 欄位：`[srcDb, srcPage, srcVa, targetDb, targetPage, quote]`，`quote` 是 `聯.q`（沒有就 `""`）。
- 目前 20,752 筆、JSON 2.15 MiB → bz2 0.53 MiB；檔案 26.5 MiB。
- 舊的 ys.adb（沒有 `xrev`）照樣可用；沒有這個容器時閱讀器會退回現算（ys 約 1.7 秒）。
- 驗證：打包後會用最終檔再算一次，`srcVa` 集合必須與寫進去的完全一致，否則自動改出不含 `xrev` 的檔。

### 4.6 為什麼不用座標直接插

實測語料「頁 65」與 CBETA `0065a` **不是同一頁**（Y22 的 `0065a` 只有 13 行，
語料同頁有 19 行以上），且語料 70 頁元、CBETA 73 頁元。
所以只能用「文字對位」，不能把 `(頁,行,字)` 直接當成 `pb`／`lb`。

---

## 5. 打包：`pack.ts`

```powershell
bun run pack.ts
```

- 讀 `build/ys-linked.txt`，以 `authorAdb`（`dbtype 0`）打包
- 資源：`logo/ys_logo.png`（資源名 `ys_logo.png`，Accelon26 會依
  `<dbname>_logo.png` 找封面）、`logo/yinshunfoundation.png`（年表頁的 `<嵌圖>`），
  以及正文 `<圖 n="…"/>` 用到的圖（資源名 `svg/YNN/…`，從
  `D:\2026\xml-p5\Y-images` 讀，只收真的有引用到的）
- 打包後把 header 的 `andtagid` 設回 0（與 `yinshun.adb` 一致）
- 打包前檢查每一行 ≤ 4,096 字（Accelon3 `MAXLINELEN`），超過直接報錯
- 打包時順便算 **反鏈快查表 `xrev`**（見下），舊讀者忽略這個容器也能開檔
- **直接寫部署位置** `D:\accelon3\ys.adb`（先寫 `.new` 再改名，避開檔案被開著時的
  `EUNKNOWN`）。工作區不留 `ys.adb`；跑 `pack.ts` 就會覆寫它（本機測試可以，上傳官網要另外說）。

預期：`dbname ys`、`linecount 328601`、`26.7 MiB`、`xrev` 20,752 筆、`resources 77`（2 張 logo ＋ 75 張圖）、
`<聯>` 17,916。

---

## 6. 驗證

```powershell
# 檢查工具在隔壁工作區（這個 repo 只有 y2adb）
cd D:\2026\adb-converter
go run ./cmd/abx inspect 'D:\accelon3\ys.adb'
bun run ts/tools/probe-adb.ts --adb 'D:\accelon3\ys.adb' --query 涅槃
```

應看到：`dbname "ys"`、`linecount 328601`、`andtagid 0`，
檢索有結果且 `consoleErrors: null`。

目錄第一項是「般若經講記」，最後一項是「2026年AI版後記」（章名「後記」）。

鏈結回歸（需同時開 `ys.adb` 與 `taisho.adb`）：

- 大正 `51p185b` → 上方「回連結」應出現 12 個按鈕
- 點按鈕 → 右欄到 ys 該頁，引文塗紅
- 在 ys 該頁點 `<聯>` → 大正 `51p185b`，引文塗紅

---

## 7. 一鍵重建

CBETA Y 更新後：

```powershell
pwsh D:\2026\ys\build.ps1
```

流程：`y2adb` → `ys-prep` → `ys-calibrate` → `addj` → `ys-pack`。

四步各自也可單獨跑（除錯時常用）。

---

## 8. 常見陷阱

1. **漢字計數**：對照表的「字」只數漢字，不含標點；抽引文、算位置都要先過濾。
2. **CJK 的 `\b`**：JS 的 `\b` 對漢字無效，`/<聯\b/` 永遠不匹配；要用 `/<聯/`。
3. **`type="old"`**：CBETA 的 `pb`／`lb` 有新旧兩套，只取沒有 `type="old"` 的。
4. **`lines.virtual` 不可靠**：它與原始字元的對應會漂移，切行會把標籤切斷；
   本流程不使用它。
5. **頁元不是冊**：70 是頁元數，不是 44 冊；`ys` 的 73 頁元是 CBETA 的 a/b/y/z 前綴。
6. **`<註>` 的兩種寫法**：`<註>內容</註>`（Accelon26 會把內容隱藏）不要用；
   要寫**自閉合**的 `<註 n="37.082"/>`，內容放段末的 `<釋 n="37.082">…</釋>`
   （行內註與 `<anchor>` 腳註都一樣；2026-10-30 起 y2adb 對 `<note>` 就是這樣輸出）。
7. **`<back>` 只收印順註**：`type="yinshun-notes"` 的 `<note>` 已收（2026-10-30；2026-10-31 起照 XML 位置集中書末，`<釋>` 15,960 條）；`apparatus`（校勘）與 `add-notes` 仍不入 `ys.adb`。
8. **`appendix` 來源是 `yinshun.adb`**：不要把 `D:\accelon3\yinshun.adb` 刪掉或覆蓋。

---

## 9. 已知限制與待辦

- 3,654 筆對照列未定位（轉述、版本差異）
- 跨行引文拆成「每行一個聯」（同一目標），視覺上是一串相鄰連結
- 寬鬆比對偶爾會落在卷中重複出現的相似句
- 反向塗紅的起點偶爾多帶一個標點
- `<back>` 校勘記未收（需要時另加開關）

---

## 10. 產物清單

| 檔案 | 說明 |
|---|---|
| `data/ys-epilog26.txt` | 後記原始檔（含 Accelon3 標記） |
| `alignments/yinshun-taisho.tsv` | 印順 → 大正藏對照表（14,128 列；2026-10-30 從 offtext 移入） |
| `build/ys-text.txt` | y2adb 輸出（44 冊，含註） |
| `build/ys.lbs.tsv` | 行號 → 冊／lb |
| `build/ys-text-full.txt` | ＋附錄＋後記 |
| `build/ys-linked.txt` | ＋`<聯>` 鏈結 |
| `D:\accelon3\ys.adb` | 打包結果＝部署位置（工作區不留 adb） |
| `D:\2026\xml-p5\Y` | CBETA Y 稀疏 clone |
| `logo/ys_logo.png` | 封面圖 |
| `cmd/y2adb/main.go` | 轉換器 |
| `data/appendix.txt` | 舊版卷首五頁（含標記） |
| `prep.ts` | 附錄＋後記 |
| `calibrate.ts` | 鏈結定位 |
| `addj.ts` | 補字元級 `j`（對回大正頁面，`calibrate` 之後跑） |
| `pack.ts` | 打包 |
| `build.ps1` | 一鍵重建 |
| `images.ts` | CBR2X-figures／corpus svg → `D:\2026\xml-p5\Y-images\svg\YNN\`（一次性） |
| `D:\2026\xml-p5\Y-images` | 正文插圖（資源名 `svg/YNN/…`；XML 引用 75 張，全部收得到） |

### 4.6 佛光大辭典對照表（`build/fgdict-terms.tsv`）

`tools/fgdict-terms.py`（Python）從 `D:\2023\fgdict-corpus\xml\fgdict*.xml` 抽
`<詞>…</詞>` 釋文裡的「（梵…）／（巴…）」，產生四欄對照表：

```text
term	lang	foreign	norm
中天竺	sk	Madhya -deśa	madhyadesa
```

- 2026-10-31 實測：7,502 組、1,885 個漢詞、4,703 個外語鍵。
- `--check` 會拿 `build/ys-linked.txt` 的 `<术>` 比對：8,344 條中有 4,765 條對到辭典。
- 用途：(1) 檢查 ys 的術語邊界是否與辭典一致；(2) 閱讀器可當術語詞表；
  (3) 之後要「以辭典優先」修邊界時的資料來源。

## 11. 發布亮點（給 ys.html／公告）

**漢梵對照是 ys.adb 的特色，CBETA 沒有提供。**

- CBETA 的 TEI 只用 `<foreign xml:lang="sa|pi|bo">` 標出外語，沒有「漢詞邊界＋外語」
  的對照表，也沒有可直接使用的詞表。
- ys 把它做成 `<术 sk|pi|tb="…">漢詞</术>`：2026-10-31 建置 8,852 條
  （梵 8,336、巴 440、藏 5；跨行 447）。
- 邊界用「外語反查」確認（同一外語對到同一漢詞），再用佛光大辭典 7,502 組對照
  （`build/fgdict-terms.tsv`）交叉檢查。
- 可給使用者：**斷詞**（詞界有外語可驗）、**詞表**（`build/term-table.tsv`，2,679 組）、
  **閱讀器 hover**（`<术>` 顯示原文，accelon26 待實作）。

每次發布 `ys.adb` 都要在 `ys.html`／公告寫這一段。

## 12. 手動術語覆寫（`data/term-overrides.tsv`）

反查、頻率規則都救不回來的個案，作者直接指定：

```text
foreign	term	note
Suhma	修摩國	作者 2026-10-31：Suhma 是修摩國
```

- `cmd/y2adb` 啟動時讀 `data/term-overrides.tsv`（可用 `YS_TERM_OVERRIDES` 換路徑）。
- 套用時只換標籤裡的漢詞；原文若有多餘前綴（「在修摩」），會用最長重疊把重複部分留在外面，
  結果是「在<术 sk="Suhma">修摩國</术>」。
- 這裡的指定優先於所有自動規則；每加一筆要重跑 `build.ps1`。

### 6.1 鏈結健檢（`tools/check-links.py`，建置時自動跑）

`build.ps1` 在 `addj.ts` 之後、`pack.ts` 之前跑這支檢查；數字不符就丟錯、
**不打包也不寫 deployment**。基準（2026-10-31）：

```text
<聯 taisho> 20,755（全部 <聯> 20,757）
有 q 9,704
有 j 9,167
```

- 這三個數字是「座標有沒有被動到」的守門員；`<术>` 插入不應該改變它們。
- 2026-10-31 補回 CB 缺字（`<g>`）後，正文字數增加，calibrate 多對到 3 條引文：
  20,752→20,755、q 9,701→9,704、j 9,164→9,167。這是變好，不是座標跑掉。
- 同一天稍早也曾出現 j 9,178→9,164；實測把 `<术>` 全拿掉再跑仍 9,164，確認與 `<术>`
  無關（來自文字調整）。改基準時要在 commit 說明原因。
- 若日後真的要改鏈結規則，先改 `tools/check-links.py` 的 `BASE` 並在 commit 說明原因。

## 13. 詞彙表 `glossary`（2026-10-31）

`glossary.ts` 從 `build/ys-linked.txt` 抽出所有 `<术>`，去重成 `(lang, foreign, han, count)`，
`pack.ts` 用 `buildContainer({ feature: 0 })` 放進外層容器 `glossary`（1 個 block、
**未壓縮** UTF-8 JSON Lines）：

```text
["GLOSSARY1", 4107]
["sk","svabhāva","自性",12]
```

- 2026-10-31 實測 4,107 筆、151 KiB；檔案因此多約 155 KB。
- 讀取端（`D:\2026\adb-converter` 的 `glossary.ts`）解析這個容器，開啟左欄「詞彙」分頁。
- 顯示時保留外語原樣（含變音符號）；**輸入**可以省略變音符號、不分大小寫。
- 舊的 `ys.adb` 沒有這個容器，閱讀器就不顯示「詞彙」分頁（向下相容）。


## 14. 術語斷詞（共用機制，2026-11-01）

`<术>` 由 y2adb 依 CBETA `<foreign>` 產生，CBETA 標到哪裡就包到哪裡，常有斷詞錯誤
（包太多：`<术>得四增上心</术>`；包太少：`<术>動作意</术>`）。**不自建詞表**，一律用
`D:\2026\adb-converter` 的共用機制：

- `ts/src/terms/terms.ts`：共用實作（`loadTerms`／`loadFixes`／`fixTerms`／`applyTerms`／`glossary`）
- `ts/tools/terms.ts`：CLI（`apply`｜`fix`｜`glossary`｜`check`）
- `data/terms.tsv`：常用詞表（3,302 條，canonical）
- `data/term-fixes.tsv`：明確修正表（63 條；`fix` 只依這張，不猜）
- `docs/adb/terms.md`：機制說明

建置步驟（`build.ps1`，插在 `addj.ts` 之後、`check-links.py` 之前）：

```powershell
# 術語斷詞修正：用 adb-converter 的共用詞表（只修 <术>，不動 <聯>/q/j）
bun run ..\adb-converter\ts\tools\terms.ts fix --text build\ys-linked.txt --write
if ($LASTEXITCODE -ne 0) { throw '術語斷詞修正失敗（tools/terms.ts）' }
```

乾跑（不寫檔，先確認數字）：

```powershell
bun run ..\adb-converter\ts\tools\terms.ts fix --text build\ys-linked.txt
```

- 只改 `<术>`：多包的字搬到標籤外、少包的字補進來；標籤數不變（一進一出）。
- 可重複執行：同一支再跑一次應該是「多包修正 0、少包修正 0」。
- 2026-11-01 首次套用：多包 55、少包 3；`<术>` 8,966 不變、`<聯>` 20,757 不變、
  `check-links` 通過、`glossary` 4,081 筆。
- 詞表雜質（例：`之為我`、`為有為`）由 adb-converter 端維護；清理後重跑即可。

## 15. 行內註（`<note place="inline">`，2026-11-02）

CBETA 的 `<note place="inline">` 是行內的大正／南傳對照（例：「大正二．九二下」），
不是腳註。y2adb 現在把它留在正文、`<ref>` 轉成 `<聯 i="taisho?…">`，不再產生
`<註>`／`<釋>`。全書約 5,291 條，修完 `<註>` 16,355→11,168、`<釋>` 15,960→10,773、
`<聯>` 20,757→23,679；`check-links.py` 基準已同步。

## 16. 紙本註號對照（2026-11-02）

舊 `D:\2023\yinshun-corpus\xml\yNN.xml` 的 `<ptr type="note" target="#noteX.YYY"/>`
標的是紙本註號（`note` 去掉就是 `X.YYY`）。`tools/build-note-map.py` 逐冊用「註標前 40
個漢字」在舊 corpus 唯一比對，產生 `data/note-numbers.tsv`（`冊\tn\t冊.紙本註號`），
y2adb 以 `-notes data/note-numbers.tsv` 讀入：`<註 n>`／`<釋 n>` 顯示 `冊.紙本註號`
（全 adb 唯一），比對不到的才用原來的 `冊.序號`。

覆蓋：11,125 條有舊 corpus 的章.註號；其餘 43 條所在的冊（Y13、Y16 等）舊 corpus 沒有 ptr，
用 `0.序號@冊` 佔位。全部 `<註>` 都是三段式，不再混用。

## 17. 圖版補齊（2026-11-02）

CBETA Y 的 TEI 只有 75 個 `<graphic>`；`Y-images/svg/YNN/` 實際有 377 個
`YNNpPPPP_SS.svg`（圖版、表格 facs）。`tools/inject-images.py` 依檔名頁碼把沒被引用的
圖插在對應 `<頁 id="NNpPPPPx"/>` 後面（`<圖 … t="圖版"/>`，不增減換行），建置順序插在
`addj.ts` 之後、`terms.ts fix` 之前。2026-11-02 首次執行：補 208 張，全書 `<圖>` 384 張（352 個頁面）。
剩下 12 個檔案在正文找不到對應頁（例 Y35p0154，CBETA 文字沒有這一頁），暫不硬塞。
`tools/import-yinshun-images.ts` 再把 yinshun.adb 的 `yNN-PPP.png` 抽到
`Y-images/png/YNN/`，只補「該頁還沒有 <圖>」的頁（2026-11-02：110 頁）。
現在頁面覆蓋 462 頁，yinshun.adb 是 444 頁；只剩 6 頁（35p0154、35p0367、36p0494、
36p0620、37p0086、37p0760）因為 CBETA 正文沒有該頁標記而放不進去。

## 18. 表格（`<表>`，2026-11-06）

CBETA 的 `<table>` 不再一律轉成頁面圖版；`y2adb` 現在會把表格緩衝成閱讀器的文字表格：

```text
<表>
| 《施設論》十空 | 《舍利弗阿毘曇論》六空 |
| --- | --- |
| <术 sk="adhyātmā-śūnyatā">內空</术> | 1 內空 |
…
</表>
```

- 每一列一行；cell 內可保留 `<術>`／`<註>`／`<聯>`／`<頁>` 等行內標籤。
- 表格中間的 `<頁 id="…"/>` 變成 `<表>` 裡的整列頁碼列（parser 會合併）。
- 只有「≥2 欄、≥2 列、欄數一致、沒有合併儲存格、沒有巢狀表」才輸出 `<表>`；
  其餘退回原圖（有圖時）或文字列（`｜` 分隔）。
- 有文字表時不再放同一頁的表格 facs；只有退回原圖的表格才保留 `<圖 t="圖版">`。
- 2026-11-06 首版：`<表>` 259 張、退回原圖 13 張、純文字列 17 張（共 289 張表格）；
  `<圖>` 從 494 降到 368。
