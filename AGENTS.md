# 工作區規則（D:\2026\ys）

## 1. 只管 ys.adb 與 tx.adb

這個工作區只處理 `ys.adb`（印順法師佛學著作集）與 `tx.adb`（太虛大師全集）。
其他 adb（taisho、wxzj、goryeo、swjzz…）或通用 adb 工具，屬於隔壁
`D:\2026\adb-converter`，不要在這裡改。

## 2. 本機可以部署，官網不能

- **可以**直接覆寫 `D:\accelon3\ys.adb`（本機測試位置），不必先問。
- **不可以**發布到官網／鏡像（`shutonggui.cn`、`nissaya.cn`）：`ys.adb`、`ys.html`
  上傳都算發布，要等使用者說「發布新 adb」。
- **發布交給 adb-converter（作者 2026-11-06 定案）**：本工作區只負責**產出 adb**
  （`build.ps1` → `D:\accelon3\ys.adb`）並更新 `deploy/adb-notes.json` 的改版說明；
  **不要自己上傳**。部署由 `D:\2026\adb-converter` 統一處理（程序＋需要更新的 adb
  一起上，`deploy.ts` 會重生 index／manifest）。`publish.ps1` 已停用，只留著當緊急紀錄。
- `build.ps1` 最後一步（`pack.ts`）就是寫 `D:\accelon3\ys.adb`，那是允許的。

## 3. 不動 accelon26

在本工作區看到「修 accelon26」的需求時，**不要動手**，提醒使用者回
`d:\2026\adb-converter` 工作區處理。

## 4. 文件跟著改

產物數字、流程或輸出路徑有變動時，**直接**同步更新 `README.md` 與
`docs/ys-steps.md`，不必先問。

## 現況（2026-10-29 23:01）

- `D:\accelon3\ys.adb` ＝最新建置，**已發布**（2026-11-06 09:15，兩站 sha256 驗證一致；引用格式 `$URL;`）：31,163,498 bytes、
  SHA256 `39E6CDC1A7F8609D`、`linecount 329456`（maxVA 8,235,531）、`<聯>` 23,679（含 `place="inline"` 的行內大正／南傳對照）、`j` 9,213、`<引>` 49,103、`<註>` 11,168（`<釋>` 10,773；`<note place="inline">` 不再算腳註）。`<聯>` 改成引文開頭一個「大正」晶片；`<釋>` 集中在書末；新增 `xrev` 反鏈快查表（20,752 筆）與 `<术 sk|pi|tb="…">` 術語標記（8,976 條，跨行 459；`glossary` 詞彙表 4,097 筆（每列含 line／page／off／hanline，閱讀器用行號＋行內漢字精準跳）；CB 缺字 gaiji=480 missing=0；<圖> 368（t="圖版"：CBETA graphic 75＋SVG 補圖 202＋yinshun.adb PNG 78＋表格原圖 13）＋文字表格 <表> 259 張；見 build/term-table.tsv、佛光辭典對照 build/fgdict-terms.tsv）。
- 目次層級改成 18 層鏈（`<書 t="部,章,節,項,干,一,數,Ａ…">`）：`<cb:mulu level>`
  1→部、2→章、3→節、4→項、5→干、6→一、7→數、8 以後→ＡＢＣ…，與 `yinshun.adb` 的目錄一致；第一個 `<head>` 等於書名時不開節點；引文不跨標題行（`calibrate` 的 `headingSkip`）。
- **術語斷詞（2026-11-01）**：用 `D:\2026\adb-converter` 的共用機制——`bun run ..\adb-converter\ts\tools\terms.ts fix --text build\ys-linked.txt --write`（詞表 `data/terms.tsv` 3,302 條、修正表 `data/term-fixes.tsv`）；只修 `<术>`，不動 `<聯>`/q/j；可重複執行（第二輪 0、0）。本 repo 不再自帶詞表／修補程式。
- **紙本註號（2026-11-02）**：`tools/build-note-map.py` 從舊 `D:\2023\yinshun-corpus\xml\yNN.xml` 的 `<ptr type="note" target="#noteX.YYY"/>` 建 `data/note-numbers.tsv`（冊\tn\t冊.書內註）；y2adb 用 `-notes` 讀它，`<註 n>`／`<釋 n>` 改顯示紙本註號（格式 `章.註@冊`，例 `11.001@38`；顯示只看 `@` 前面，`@冊` 保證全 adb unique；舊 corpus 沒資料的 43 條用章 0），對不上的才用原本序號。
- `pack.ts` 只寫部署位置，工作區不留 `ys.adb`。
- **發布亮點（下次發布要提）**：漢梵對照是 ys.adb 的特色，CBETA 沒提供——`<术>` 8,966 條＋`build/term-table.tsv` 詞表，可給使用者斷詞與術語查詢；`ys.html`／公告要寫這段（見 README「特色：漢梵對照」）。
- `D:\accelon3\tx.adb` 目前與 `D:\2026\adb-converter\build\tx.adb` 一致；
  本 repo 尚無 tx 的建置流程。

---

## 語義分句與軌號（canonical）

指導性原則見 `D:\2026\agent-norms\shutonggui.md`（書同軌）：句級地址、最小切分、
軌號（`track_id`）與 crosswalk。本區只放實作與資料，不另存一份原則再改。
