# -*- coding: utf-8 -*-
"""sync-svg.py — 用陳裕文提供的 SVG 全集，重整 `D:\\2026\\xml-p5\\Y-images\\svg`

問題（陳裕文 2026-11-15 10:20 回報）：
  「為什麼會有 png 和 svg 二個同時存在？又，有些本已不用的圖檔，為什麼還出現？」

  `Y-images/svg/YNN/` 裡混了三種東西：
    1. 舊 corpus（`yNN.NNNN.svg`）與新 CBETA（`YNNpPPPP_SS.svg`）兩代混在一起，
       同名卻不同內容（例：`Y37p0503` 舊版 6 張、CBETA 現在只用 2 張）。
    2. 同一張圖同時有 `.gif` 與 `.svg`（42 處）。
    3. 只有舊版才有的圖（CBETA 已改用文字排，例：`Y37p0503_03~06.svg`）。

做法：
  以裕文給的 `ys.svg.rar`（367 個 `yNN.NNNN[-K].svg`）為**唯一**圖版來源，
  改名成 `YNNpPPPP_SS.svg` 覆蓋寫入；不在這批、也不是 CBETA XML 仍在引用的
  檔案，一律移到 `Y-images/_obsolete/svg/`（不刪，可回溯）。

用法：
  python tools/sync-svg.py --src D:/2026/fuyan/raw/ys-svg-20261115/yssvg --dry-run
  python tools/sync-svg.py --src D:/2026/fuyan/raw/ys-svg-20261115/yssvg --write
"""
import argparse
import hashlib
import os
import re
import shutil
import sys

sys.stdout.reconfigure(encoding='utf-8')

IMG_ROOT = r'D:\2026\xml-p5\Y-images'
SVG_DIR = os.path.join(IMG_ROOT, 'svg')
OBSOLETE = os.path.join(IMG_ROOT, '_obsolete')
XML_LIST = r'D:\2026\ys\build\xml-graphics.txt'
TEXT = r'D:\2026\ys\build\ys-linked.txt'


def target_name(fn: str) -> str | None:
    """`y37.503-1.svg` → `Y37p0503_01.svg`"""
    m = re.match(r'^y(\d+)\.(\d+)(?:-(\d+))?\.svg$', fn, re.I)
    if not m:
        return None
    vol = str(int(m.group(1))).zfill(2)
    page = m.group(2).zfill(4)
    seq = (m.group(3) or '1').zfill(2)
    return f'Y{vol}p{page}_{seq}.svg'


def sha(path: str) -> str:
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for b in iter(lambda: f.read(65536), b''):
            h.update(b)
    return h.hexdigest()[:12]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument('--src', required=True)
    ap.add_argument('--write', action='store_true')
    ap.add_argument('--dry-run', action='store_true')
    a = ap.parse_args()
    write = a.write

    # 1) 新圖集：來源 → 目標相對路徑
    src_map: dict[str, str] = {}
    bad = []
    for d in sorted(os.listdir(a.src)):
        p = os.path.join(a.src, d)
        if not os.path.isdir(p):
            continue
        for fn in sorted(os.listdir(p)):
            tgt = target_name(fn)
            if not tgt:
                bad.append(fn)
                continue
            rel = f'svg/Y{str(int(d[1:])).zfill(2)}/{tgt}'
            if rel in src_map:
                bad.append(f'重複目標 {rel}（{os.path.basename(src_map[rel])} / {fn}）')
            src_map[rel] = os.path.join(p, fn)
    print(f'來源 {len(src_map)} 個 svg' + (f'，命名例外 {len(bad)}：{bad[:5]}' if bad else ''))

    # 2) CBETA XML 仍在引用的（沒有替代品就不能移走）
    xml_refs = set()
    if os.path.exists(XML_LIST):
        for line in open(XML_LIST, encoding='utf-8'):
            line = line.strip()
            m = re.match(r'^Y(\d+)p', line, re.I)
            if m:
                xml_refs.add(f'svg/Y{str(int(m.group(1))).zfill(2)}/{line}')
    print(f'CBETA XML 引用 {len(xml_refs)} 個')

    # 2b) 書稿（ys-linked.txt）目前引用的資源：還在用的絕不移走
    txt_refs = set()
    if os.path.exists(TEXT):
        for m in re.finditer(r'<圖 n="([^"]+)"', open(TEXT, encoding='utf-8').read()):
            txt_refs.add(m.group(1))
    print(f'書稿引用 {len(txt_refs)} 個')

    # 3) 現有檔案
    cur: dict[str, str] = {}
    for d in sorted(os.listdir(SVG_DIR)):
        p = os.path.join(SVG_DIR, d)
        if os.path.isdir(p):
            for fn in os.listdir(p):
                cur[f'svg/{d}/{fn}'] = os.path.join(p, fn)
    print(f'現有 {len(cur)} 個')

    # 4) 分類
    replace, same, add, drop = [], [], [], []
    for rel, src in src_map.items():
        if rel not in cur:
            add.append(rel)
        elif sha(src) != sha(cur[rel]):
            replace.append(rel)
        else:
            same.append(rel)
    for rel in cur:
        if rel in src_map:
            continue
        stem, ext = os.path.splitext(rel)
        # 書稿還在用的先留（等書稿改完再 prune 一次）
        if rel in txt_refs and not (ext.lower() == '.gif' and (stem + '.svg') in src_map):
            continue
        # 同名有 .svg 的 .gif → 舊格式，移走
        if ext.lower() == '.gif' and (stem + '.svg') in src_map:
            drop.append(rel)
            continue
        if rel in xml_refs:
            continue  # CBETA 仍在引用（Y13 等只有 gif 的圖）
        drop.append(rel)

    print(f'新增 {len(add)}、覆蓋 {len(replace)}、相同 {len(same)}、移走 {len(drop)}')
    for rel in drop[:50]:
        print(f'  移走 {rel}')

    if not write:
        print('（dry-run，未寫入）')
        return 0

    # 5) 寫入
    for rel in add + replace:
        dst = os.path.join(IMG_ROOT, rel.replace('/', os.sep))
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.copyfile(src_map[rel], dst)
    for rel in drop:
        src_path = cur[rel]
        dst = os.path.join(OBSOLETE, rel.replace('/', os.sep))
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        shutil.move(src_path, dst)
    # 清掉空目錄
    for d in sorted(os.listdir(SVG_DIR)):
        p = os.path.join(SVG_DIR, d)
        if os.path.isdir(p) and not os.listdir(p):
            os.rmdir(p)
    print(f'完成：新增 {len(add)}、覆蓋 {len(replace)}、移走 {len(drop)}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
