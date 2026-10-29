# 工作區規則（D:\2026\ys）

## 1. 只管 ys.adb 與 tx.adb

這個工作區只處理 `ys.adb`（印順法師佛學著作集）與 `tx.adb`（太虛大師全集）。
其他 adb（taisho、wxzj、goryeo、swjzz…）或通用 adb 工具，屬於隔壁
`D:\2026\adb-converter`，不要在這裡改。

## 2. 本機可以部署，官網不能

- **可以**直接覆寫 `D:\accelon3\ys.adb`（本機測試位置），不必先問。
- **不可以**發布到官網／鏡像（`shutonggui.cn`、`nissaya.cn`）：`ys.adb`、`ys.html`
  上傳都算發布，要等使用者說「發布新 adb」。
- `build.ps1` 最後一步（`pack.ts`）就是寫 `D:\accelon3\ys.adb`，那是允許的。

## 3. 不動 accelon26

在本工作區看到「修 accelon26」的需求時，**不要動手**，提醒使用者回
`d:\2026\adb-converter` 工作區處理。

## 4. 文件跟著改

產物數字、流程或輸出路徑有變動時，**直接**同步更新 `README.md` 與
`docs/ys-steps.md`，不必先問。

## 現況（2026-10-29 23:01）

- `D:\accelon3\ys.adb`（本機測試）＝最新建置：26,109,364 bytes、
  SHA256 `0C91406A…`、`linecount 307141`、`<聯>` 17,870、`j` 16,683。
- 目次層級改成 18 層鏈（`<書 t="部,章,節,項,干,一,數,Ａ…">`）：`<cb:mulu level>`
  1→部、2→章、3→節、4→項、5→干、6→一、7→數、8 以後→ＡＢＣ…，與 `yinshun.adb` 的目錄一致；第一個 `<head>` 等於書名時不開節點；引文不跨標題行（`calibrate` 的 `headingSkip`）。
- `pack.ts` 只寫部署位置，工作區不留 `ys.adb`。
- `D:\accelon3\tx.adb` 目前與 `D:\2026\adb-converter\build\tx.adb` 一致；
  本 repo 尚無 tx 的建置流程。

## 待辦：官網 ys 頁面與 adb 一起發布

線上 `https://shutonggui.cn/accelon26/adb/ys.adb` 仍是舊版（27,205,727 bytes），
官網說明頁也就還是舊數字。等 `adb-converter` 跑完 `retrofit-j` 後，一次做完：

1. 改 `D:\2026\adb-converter\docs\adb\ys.md`：大正鏈結 25,918→17,916、
   `j` 的條數、檔案大小 25.9→約 24.5 MiB、§五 的版本與下載。
2. `bun run ts/tools/md-page.ts --md docs/adb/ys.md --out www/adb/ys.html --title "ys.adb 與 yinshun.adb 有什麼不同"`。
3. 連 `ys.adb` 一起上傳主站＋鏡像（等使用者說「發布新 adb」）。
