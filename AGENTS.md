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

- `D:\accelon3\ys.adb` ＝最新建置，**本機測試版**（2026-10-30 20:13；線上仍是 10-29 版）：28,470,736 bytes、
  SHA256 `5A903925…`、`linecount 329391`、`<聯>` 29,396（正文 17,952＋註 11,444）、`j` 16,763、`<註>`／`<釋>` 11,125 條。
- 目次層級改成 18 層鏈（`<書 t="部,章,節,項,干,一,數,Ａ…">`）：`<cb:mulu level>`
  1→部、2→章、3→節、4→項、5→干、6→一、7→數、8 以後→ＡＢＣ…，與 `yinshun.adb` 的目錄一致；第一個 `<head>` 等於書名時不開節點；引文不跨標題行（`calibrate` 的 `headingSkip`）。
- `pack.ts` 只寫部署位置，工作區不留 `ys.adb`。
- `D:\accelon3\tx.adb` 目前與 `D:\2026\adb-converter\build\tx.adb` 一致；
  本 repo 尚無 tx 的建置流程。
