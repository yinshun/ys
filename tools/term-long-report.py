"""列出「可能被拉長」的術語：長度 ≥5 且同組（語言＋外語）有更短的後綴詞可當正解。

輸出 build/long-terms.tsv：line / term / lang / foreign / len / suggested / count / prev40 / next40
只讀 build/ys-text.txt，不改任何產物。
"""

import re
from collections import Counter, defaultdict

SRC = r'D:\2026\ys\build\ys-text.txt'
OUT = r'D:\2026\ys\build\long-terms.tsv'
TAG = re.compile(r'<术 (sk|pi|tb)="([^"]*)">([^<]*)</术>')

text = open(SRC, encoding='utf-8').read()
lines = text.split('\n')
groups = defaultdict(Counter)
for m in TAG.finditer(text):
    term = m.group(3)
    if '\n' in term:
        continue
    groups[(m.group(1), m.group(2))][term] += 1

rows = []
for m in TAG.finditer(text):
    term = m.group(3)
    if '\n' in term or len(term) < 5:
        continue
    lang, foreign = m.group(1), m.group(2)
    cands = groups[(lang, foreign)]
    shorter = [(t, c) for t, c in cands.items() if len(t) >= 2 and t != term and term.endswith(t)]
    if not shorter:
        continue
    shorter.sort(key=lambda x: (-x[1], -len(x[0])))
    best, n = shorter[0]
    line = text.count('\n', 0, m.start()) + 1
    line_text = lines[line - 1]
    at = line_text.find(m.group(0))
    prev = re.sub(r'<[^>]*>', '', line_text[:max(0, at)])[-40:]
    nxt = re.sub(r'<[^>]*>', '', line_text[at + len(m.group(0)):])[:40]
    rows.append((line, term, lang, foreign, len(term), best, n, cands[term], prev, nxt))

with open(OUT, 'w', encoding='utf-8', newline='\n') as fh:
    fh.write('line\tterm\tlang\tforeign\tlen\tsuggested\tcount\ttermcount\tprev40\tnext40\n')
    for r in rows:
        fh.write('\t'.join(str(x) for x in r) + '\n')
print('long rows', len(rows))
strict = [r for r in rows if r[7] == 1 and r[6] >= 3]
with open(r'D:\2026\ys\build\long-terms-strict.tsv', 'w', encoding='utf-8', newline='\n') as fh:
    fh.write('line\tterm\tlang\tforeign\tlen\tsuggested\tcount\ttermcount\tprev40\tnext40\n')
    for r in strict:
        fh.write('\t'.join(str(x) for x in r) + '\n')
print('strict rows', len(strict))
