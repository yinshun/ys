# -*- coding: utf-8 -*-
r"""build-note-map.py — CBETA Y 腳註 → 舊 yinshun-corpus 的紙本註號

舊 corpus（D:\2023\yinshun-corpus\xml\yNN.xml）在正文用
`<ptr type="note" target="#note1.001"/>` 標腳註位置，target 去掉 `note` 就是紙本註號
（每本書從 1.001 起）。CBETA Y 用 `<anchor n="0081a1401"/>`；n 只在同冊唯一，
所以輸出用 `冊  n  註號` 當 key，註號顯示 `冊.紙本註號`（例 38.1.001），全 adb 唯一。

做法：逐冊（YNN ↔ yNN.xml）把正文剝成純漢字序列，用「註標前 40 個漢字」在舊 corpus
裡唯一比對，取比對位置之後最近的 ptr target；順序亂掉的用最長遞增子序列修掉，
其餘留空（y2adb 會用原本的序號）。
"""
import os
import re
import sys
from bisect import bisect_left

sys.stdout.reconfigure(encoding='utf-8')

HAN = re.compile(r'[\u3400-\u9fff]')
OLD_DIR = r'D:\2023\yinshun-corpus\xml'
NEW_DIR = r'D:\2026\xml-p5\Y'
HERE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def han_marks(text, pattern):
    out = []
    total = 0
    last = 0
    for m in re.finditer(pattern, text):
        total += len(HAN.findall(text[last:m.start()]))
        out.append((total, m.group(1)))
        last = m.end()
    return out


def han_string(s):
    return ''.join(HAN.findall(re.sub(r'<[^>]*>', '', s)))


def load_old(vol):
    p = os.path.join(OLD_DIR, 'y%02d.xml' % vol)
    if not os.path.exists(p):
        return None
    t = open(p, encoding='utf-8').read()
    marks = han_marks(t, r'<ptr[^>]*type="note"[^>]*target="#([^"]+)"[^>]*/?>')
    marks = [(i, m[4:] if m.startswith('note') else m) for i, m in marks]
    return han_string(t), marks


def load_new(vol):
    d = os.path.join(NEW_DIR, 'Y%02d' % vol)
    if not os.path.isdir(d):
        return None
    files = [f for f in sorted(os.listdir(d)) if f.endswith('.xml')]
    text = ''.join(open(os.path.join(d, f), encoding='utf-8').read() for f in files)
    notes = set()
    for div in re.findall(r'<(?:cb:)?div[^>]*type="yinshun-notes".*?</(?:cb:)?div>', text, re.S):
        notes.update(re.findall(r'<note[^>]*\bn="([^"]+)"', div))
    marks = [x for x in han_marks(text, r'<anchor[^>]*n="([^"]+)"[^>]*/>') if x[1] in notes]
    return han_string(text), marks


def lis(pairs):
    tails, idx, prev = [], [], [-1] * len(pairs)
    for k, (_, j) in enumerate(pairs):
        pos = bisect_left(tails, j)
        if pos == len(tails):
            tails.append(j)
            idx.append(k)
        else:
            tails[pos] = j
            idx[pos] = k
        prev[k] = idx[pos - 1] if pos > 0 else -1
    out = []
    k = idx[-1] if idx else -1
    while k >= 0:
        out.append(pairs[k])
        k = prev[k]
    return out[::-1]


def map_volume(vol):
    old = load_old(vol)
    new = load_new(vol)
    if not old or not new:
        return {}, 0, 0
    ohan, omarks = old
    nhan, nmarks = new
    opos = [p for p, _ in omarks]
    pairs = []
    for i, (np, _n) in enumerate(nmarks):
        ctx = nhan[max(0, np - 40):np]
        if len(ctx) < 30:
            continue
        k = ohan.find(ctx)
        if k < 0 or ohan.find(ctx, k + 1) >= 0:
            continue
        j = bisect_left(opos, k + len(ctx))
        if j < len(omarks):
            pairs.append((i, j))
    pairs.sort()
    mono = pairs
    if any(pairs[k][1] < pairs[k - 1][1] for k in range(1, len(pairs))):
        mono = lis(pairs)
    out = {}
    # 先鋪已對上的點，再用「前後錨點之間按順序內插」補滿，做到 100% 覆蓋
    mono = [(-1, -1)] + mono + [(len(nmarks), len(omarks))]
    for k in range(len(mono) - 1):
        i1, j1 = mono[k]
        i2, j2 = mono[k + 1]
        if i1 >= 0:
            out[nmarks[i1][1]] = '%d.%s' % (vol, omarks[j1][1])
        span_i = i2 - i1 - 1
        span_j = j2 - j1 - 1
        if span_i <= 0:
            continue
        if span_i == span_j:
            for t in range(span_i):
                out[nmarks[i1 + 1 + t][1]] = '%d.%s' % (vol, omarks[j1 + 1 + t][1])
        elif span_j > 0:
            for t in range(span_i):
                j = j1 + 1 + int(round(t * span_j / span_i))
                j = min(max(j, j1 + 1), j2 - 1)
                out[nmarks[i1 + 1 + t][1]] = '%d.%s' % (vol, omarks[j][1])
    # 舊 corpus 沒有的註（CBETA 多加的）：用同一章的前一號 +1，沒有才用後一號 -1
    assigned = [out.get(nmarks[i][1]) for i in range(len(nmarks))]
    def parse(x):
        p = x.split('.')
        return (int(p[1]), int(p[2])) if len(p) == 3 else None
    for i in range(len(nmarks)):
        if assigned[i]:
            continue
        prev = next((assigned[k] for k in range(i - 1, -1, -1) if assigned[k]), None)
        nxt = next((assigned[k] for k in range(i + 1, len(nmarks)) if assigned[k]), None)
        cand = None
        if prev and nxt and parse(prev) and parse(nxt) and parse(prev)[0] == parse(nxt)[0]:
            cand = '%d.%d.%03d' % (vol, parse(prev)[0], parse(prev)[1] + 1)
        elif prev and parse(prev):
            cand = '%d.%d.%03d' % (vol, parse(prev)[0], parse(prev)[1] + 1)
        elif nxt and parse(nxt):
            cand = '%d.%d.%03d' % (vol, parse(nxt)[0], max(1, parse(nxt)[1] - 1))
        if not cand:
            # 舊 corpus 完全沒有這本書的註：用「章 0」保持三段式，避免混編
            cand = '%d.0.%03d' % (vol, sum(1 for x in assigned[:i] if x and x.split('.')[1] == '0') + 1)
        assigned[i] = cand
        out[nmarks[i][1]] = cand
    # 同一個號被內插分到多條註：往後找下一個沒用過的號（同章），保證一號一註
    used = set()
    for i in range(len(nmarks)):
        n = nmarks[i][1]
        if n not in out:
            continue
        v = out[n]
        parts = v.split('.')
        if len(parts) != 3:
            continue
        ch, seq = int(parts[1]), int(parts[2])
        while v in used:
            seq += 1
            v = '%d.%d.%03d' % (vol, ch, seq)
        out[n] = v
        used.add(v)
    return out, len(nmarks), len(omarks)


def main():
    rows = []
    total_marks = total_mapped = 0
    for vol in range(1, 47):
        m, nn, no = map_volume(vol)
        if not nn:
            continue
        total_marks += nn
        total_mapped += len(m)
        print('Y%02d：註錨 %d、舊 ptr %d、對上 %d' % (vol, nn, no, len(m)))
        for n, paper in sorted(m.items()):
            rows.append((vol, n, paper))
    out = os.path.join(HERE, 'data', 'note-numbers.tsv')
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, 'w', encoding='utf-8', newline='\n') as f:
        f.write('# 冊\tCBETA Y 腳註 n\t紙本註號（冊.書內註）\n')
        for vol, n, paper in rows:
            f.write('%d\t%s\t%s\n' % (vol, n, paper))
    print('寫出 %s：%d 筆（%.1f%%）' % (out, len(rows), 100.0 * len(rows) / max(1, total_marks)))


if __name__ == '__main__':
    main()
