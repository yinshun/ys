# -*- coding: utf-8 -*-
"""inject-images.py — 把 Y-images 裡沒被 CBETA <graphic> 引用的圖版補進正文

CBETA 的 TEI 只標了 75 個 `<graphic>`，但 `D:\\2026\\xml-p5\\Y-images\\svg\\YNN\\`
有 377 個 `YNNpPPPP_SS.svg`（圖版、表格 facs）。這支依檔名的頁碼
（Y38p0111_01.svg → 頁 id `38p0111a`）把還沒出現過的圖插在該頁標記之後：

    <頁 id="38p0111a"/><圖 n="svg/Y38/Y38p0111_01.svg" t="圖版"/>

只插在既有的 `<頁 id="…"/>` 後面，不新增/刪除換行（行號不變）；檔名已在正文裡
（例如 y2adb 已放的表格圖）就跳過。
"""
import glob
import os
import re
import sys

sys.stdout.reconfigure(encoding='utf-8')

IMG_DIR = r'D:\2026\xml-p5\Y-images\svg'
TEXT = r'D:\2026\ys\build\ys-linked.txt'


def main(path=TEXT):
    text = open(path, encoding='utf-8').read()
    files = sorted(glob.glob(os.path.join(IMG_DIR, 'Y*', '*.svg')))
    # 頁碼 → 第一個頁標記（用 (vol, page) 當 key，頁 id 例：38p0111a）
    page_re = re.compile(r'<頁 id="(\d+)p[a-z]?(\d{1,4})[a-z]"/>')
    marks = list(page_re.finditer(text))
    first_at = {}
    for idx, m in enumerate(marks):
        key = (str(int(m.group(1))), str(int(m.group(2))))
        first_at.setdefault(key, (m.end(), idx))
    # 這一頁有沒有文字表（<表>）：有的話就不用再插同一頁的表格 facs（會與文字表重複）
    def page_has_table(idx: int) -> bool:
        start = marks[idx].end()
        end = marks[idx + 1].start() if idx + 1 < len(marks) else len(text)
        return '<表>' in text[start:end]

    # <表> 區塊裡不要插圖：頁碼列必須是單獨一行（一列一行）
    table_ranges = [(m.start(), m.end()) for m in re.finditer(r'<表>.*?</表>', text, re.S)]

    def in_table(pos: int) -> bool:
        return any(a <= pos < b for a, b in table_ranges)

    inserts = {}
    skipped_dup = skipped_nopage = skipped_table = 0
    for f in files:
        base = os.path.basename(f)
        m = re.match(r'Y(\d+)p(\d{1,4})_\d+\.svg$', base)
        if not m:
            skipped_nopage += 1
            continue
        if base in text:
            skipped_dup += 1
            continue
        key = (str(int(m.group(1))), str(int(m.group(2))))
        hit = first_at.get(key)
        if hit is None:
            skipped_nopage += 1
            continue
        at, idx = hit
        if in_table(at) or page_has_table(idx):
            skipped_nopage += 1
            continue
        inserts.setdefault(at, []).append(base)

    out = []
    last = 0
    added = 0
    for at in sorted(inserts):
        out.append(text[last:at])
        for base in inserts[at]:
            vol = re.match(r'Y(\d+)p', base).group(1)
            out.append('<圖 n="svg/Y%s/%s" t="圖版"/>' % (vol, base))
            added += 1
        last = at
    out.append(text[last:])
    new = ''.join(out)
    if new.count('\n') != text.count('\n'):
        print('警告：換行數改變，放棄寫入')
        return 1
    open(path, 'w', encoding='utf-8', newline='\n').write(new)
    print('補圖 %d 張（已存在 %d、無對應頁／在表格內 %d）' % (added, skipped_dup, skipped_nopage))
    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else TEXT))
