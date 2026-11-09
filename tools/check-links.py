"""建置後的鏈結健檢：確認 <术> 的插入沒有動到 <聯 i="taisho?…"> 的數量與 j 覆蓋。

用法：python tools/check-links.py
離開碼 1 表示數字與基準不符（可能座標被改動）。
"""

import re
import sys

SRC = r'D:\2026\ys\build\ys-linked.txt'
BASE = {'links': 23677, 'with_q': 9761, 'with_j': 9213}  # 2026-11-06：表格改 <表> 文字表，cell 內的 <聯> 回來了  # 2026-11-02：表格改放 <圖>（內容不再以文字重複）  # 2026-11-02：place="inline" 的 <note> 改回行內 <聯>（不再算腳註），<聯> +2,922  # j 依 taisho.adb 而變（作者 2026-11-01 重建 taisho 後 9167→9149）  # <聯> 全部 20,754，其中 20,752 條是 taisho 鏈結；j 基準 2026-10-31 19:xx（補回 CB 缺字後多 3 條鏈結，見 docs §6.1）

txt = open(SRC, encoding='utf-8').read()
links = re.findall(r'<聯 i="taisho\?[^"]+"[^>]*>', txt)
with_q = [x for x in links if ' q="' in x]
with_j = [x for x in links if ' j="' in x]
print(f'<聯 taisho> {len(links)}（基準 {BASE["links"]}）')
print(f'有 q {len(with_q)}（基準 {BASE["with_q"]}）')
print(f'有 j {len(with_j)}（基準 {BASE["with_j"]}）')

ok = True
for name, got, base in (('links', len(links), BASE['links']),
                        ('with_q', len(with_q), BASE['with_q']),
                        ('with_j', len(with_j), BASE['with_j'])):
    if got != base:
        print(f'  差異：{name} {base} → {got}')
        ok = False
if not ok:
    print('FAIL：鏈結數與基準不同，請檢查 <术> 插入或 calibrate/addj 是否改了座標。')
    sys.exit(1)
print('OK：鏈結數與基準一致。')
