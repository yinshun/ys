# -*- coding: utf-8 -*-
"""fix-image-refs.py — 修正書稿（build/ys-linked.txt）的圖檔引用

陳裕文 2026-11-15 10:20：「印順法師佛學著作集圖檔請直接用我提供的SVG．
請注意圖檔位置．」

三件事：

1. 移除舊 PNG 引用（54 個 `<圖 n="png/YNN/yNN-PP.png">`）。
   那批 PNG 是舊版 `yinshun.adb` 的「紙本表格掃描」，內容與正文文字重複
   （例：`y37-33.png` 的 1–35 經名列，正文 37p34 一字不差都有），
   而且是按檔名頁碼插在頁首，內容其實屬於後面段落 —— 位置也是錯的。

2. GIF → SVG。CBETA 有些圖只給 `_01.gif`，裕文這次給了同名的 `.svg`，
   換過去；沒有 SVG 版的（Y09p0125_01、Y13 七張）保留 GIF。

3. 補插圖集有、書稿還沒有的圖：
   - `Y37p1123_01.svg`（37p1123 頁首）
   - `Y40p0415_07~10.svg`（40p415 頁首，接在既有 _06 之後）

用法：
  python tools/fix-image-refs.py --dry-run
  python tools/fix-image-refs.py --write
"""
import argparse
import os
import re
import sys

sys.stdout.reconfigure(encoding='utf-8')

TEXT = r'D:\2026\ys\build\ys-linked.txt'
IMG_ROOT = r'D:\2026\xml-p5\Y-images'

# 要補插的圖：資源名 → 插在哪個既有標記之後（None = 插在指定頁標記之後）
EXTRA = {
    'svg/Y37/Y37p1123_01.svg': ('after_page', '37p1123'),
    'svg/Y40/Y40p0415_07.svg': ('after_img', 'svg/Y40/Y40p0415_06.svg'),
    'svg/Y40/Y40p0415_08.svg': ('after_img', 'svg/Y40/Y40p0415_07.svg'),
    'svg/Y40/Y40p0415_09.svg': ('after_img', 'svg/Y40/Y40p0415_08.svg'),
    'svg/Y40/Y40p0415_10.svg': ('after_img', 'svg/Y40/Y40p0415_09.svg'),
}


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument('--write', action='store_true')
    ap.add_argument('--dry-run', action='store_true')
    args = ap.parse_args()

    text = open(TEXT, encoding='utf-8').read()
    orig = text
    log = []

    # 1) 移除 PNG 引用
    png_re = re.compile(r'<圖 n="png/[^"]+" t="[^"]*"/>')
    pngs = png_re.findall(text)
    text = png_re.sub('', text)
    log.append(f'移除 PNG 引用 {len(pngs)} 個')

    # 2) GIF → SVG（同名 .svg 存在才換）
    def gif_sub(m):
        rel = m.group(1)
        svg_rel = re.sub(r'\.gif$', '.svg', rel)
        if os.path.exists(os.path.join(IMG_ROOT, svg_rel.replace('/', os.sep))):
            log.append(f'  {rel} → {svg_rel}')
            return m.group(0).replace(rel, svg_rel)
        return m.group(0)

    gif_re = re.compile(r'<圖 n="(svg/[^"]+\.gif)" t="[^"]*"/>')
    text = gif_re.sub(gif_sub, text)
    log.append(f'GIF 引用 {len(gif_re.findall(orig))} 個，換成 SVG {sum(1 for x in log if x.startswith("  "))} 個')

    # 3) 補插
    for rel, (how, anchor) in EXTRA.items():
        if rel in text:
            log.append(f'已在書稿，跳過 {rel}')
            continue
        if how == 'after_page':
            pat = re.compile(r'(<頁 id="%s[a-z]"\s*/>)' % re.escape(anchor))
            m = pat.search(text)
            if not m:
                log.append(f'找不到頁標記 {anchor}，跳過 {rel}')
                continue
            ins = '<圖 n="%s" t="圖版"/>' % rel
            text = text[:m.end()] + ins + text[m.end():]
            log.append(f'插 {rel} 於 {anchor} 頁首')
        else:
            pat = re.compile(r'(<圖 n="%s" t="[^"]*"/>|<圖 n="%s"/>)' % (re.escape(anchor), re.escape(anchor)))
            m = pat.search(text)
            if not m:
                log.append(f'找不到錨點 {anchor}，跳過 {rel}')
                continue
            ins = '<圖 n="%s" t="圖版"/>' % rel
            text = text[:m.end()] + ins + text[m.end():]
            log.append(f'插 {rel} 於 {anchor} 之後')

    print('\n'.join(log))
    if text.count('\n') != orig.count('\n'):
        print('警告：換行數改變，放棄')
        return 1
    if not args.write:
        print('（dry-run，未寫入）')
        return 0
    open(TEXT, 'w', encoding='utf-8', newline='\n').write(text)
    print('已寫入', TEXT)
    return 0


if __name__ == '__main__':
    sys.exit(main())
