"""產生兩份檢查清單（只讀 build/ys-text.txt，不改任何規則）。

1. build/remaining-one-char-terms.tsv：<术> 內容只有 1 個漢字的條目，
   附上一行尾／下一行頭與原因分類。
2. build/crossline-prefix-issues.tsv：跨行術語攤平後仍多吃前綴的條目，
   附同組單行詞與建議正解。
"""

import re
from collections import Counter, defaultdict

SRC = r'D:\2026\ys\build\ys-text.txt'
OUT1 = r'D:\2026\ys\build\remaining-one-char-terms.tsv'
OUT2 = r'D:\2026\ys\build\crossline-prefix-issues.tsv'

text = open(SRC, encoding='utf-8').read()
lines = text.split('\n')
TAG = re.compile(r'<术 (sk|pi|tb)="([^"]*)">([^<]*)</术>')
STOP = '。！？；：，、」』）】》〉…—－─-–—「『（《〈【> \t　'


def classify(prev_char: str, at_line_start: bool, prev_tail: str = '') -> str:
    if at_line_start:
        # 上一行尾是漢字 → 詞其實跨行，只是 fixCrossLineTerms 沒接回來
        if prev_tail and '\u3400' <= prev_tail[-1] <= '\u9fff':
            return '行首（跨行未接回）'
        return '行首（本行就是新段）'
    if prev_char == '>':
        return '前一字是標籤結尾（標籤把候選切壞）'
    if prev_char in STOP:
        return '前一字是標點／停止字（被切斷）'
    if '\u3400' <= prev_char <= '\u9fff':
        return '前一字是漢字（可能是正解過短或詞本身如此）'
    return '其他'


rows1 = []
cross = []          # (lang, foreign, flat, line, prev, next)
single_by = defaultdict(Counter)
for m in TAG.finditer(text):
    term = m.group(3)
    lang, foreign = m.group(1), m.group(2)
    start_line = text.count('\n', 0, m.start()) + 1
    end_line = text.count('\n', 0, m.end()) + 1
    prev_tail = lines[start_line - 2][-20:] if start_line >= 2 else ''
    prev2_tail = lines[start_line - 3][-20:] if start_line >= 3 else ''
    prev_han = re.sub(r'<[^>]*>', '', lines[start_line - 2] if start_line >= 2 else '')[-20:]
    before = text[m.start() - 1] if m.start() > 0 else '\n'
    at_start = before == '\n'
    # 給作者判斷用的較長上下文（標籤剝掉，保留 40 字）
    line_now = lines[start_line - 1]
    at_pos = text.find('<术', sum(len(x) + 1 for x in lines[:start_line - 1]))
    prev40 = re.sub(r'<[^>]*>', '', line_now[:max(0, line_now.find(m.group(0)))])[-40:] if m else ''
    if at_start:
        src = lines[start_line - 2] if start_line >= 2 else ''
        prev40 = re.sub(r'<[^>]*>', '', src)[-40:]
        if not prev40.strip() and start_line >= 3:
            prev40 = re.sub(r'<[^>]*>', '', lines[start_line - 3])[-40:]
    tail40 = re.sub(r'<[^>]*>', '', line_now[line_now.find(m.group(0)) + len(m.group(0)):])[:40] if m else ''
    prev2_han = re.sub(r'<[^>]*>', '', lines[start_line - 3] if start_line >= 3 else '')[-20:]
    next_head = lines[end_line][:20] if end_line - 1 < len(lines) - 1 else ''
    if '\n' in term:
        cross.append((lang, foreign, term.replace('\n', ''), start_line, prev_tail, next_head))
        continue
    single_by[(lang, foreign)][term] += 1
    if len(term) == 1:
        before = text[m.start() - 1] if m.start() > 0 else '\n'
        at_start = before == '\n'
        rows1.append((start_line, term, lang, foreign, prev_tail, next_head,
                      classify(before, at_start, prev_han if prev_han.strip() else prev2_han), prev40, tail40))

with open(OUT1, 'w', encoding='utf-8', newline='\n') as fh:
    fh.write('line\tterm\tlang\tforeign\tprev_tail20\tnext_head20\tcause\tprev40\tnext40\n')
    for r in rows1:
        fh.write('\t'.join(str(x).replace('\t', ' ') for x in r) + '\n')

rows2 = []
for lang, foreign, flat, line, prev, nxt in cross:
    cands = single_by.get((lang, foreign))
    same = '、'.join(f'{t}({c})' for t, c in cands.most_common()) if cands else ''
    shorter = [(t, c) for t, c in (cands or {}).items() if len(t) >= 2 and flat.endswith(t) and t != flat]
    suggestion = ''
    prefix = ''
    if shorter:
        shorter.sort(key=lambda x: (-x[1], -len(x[0])))
        suggestion = f'{shorter[0][0]}({shorter[0][1]})'
        prefix = flat[:-len(shorter[0][0])]
    if suggestion:
        rows2.append((line, flat, foreign, same, suggestion, prefix, prev, nxt))

with open(OUT2, 'w', encoding='utf-8', newline='\n') as fh:
    fh.write('line\tcrossline_term\tforeign\tsame_group_single\tsuggested\tprefix\tprev_tail20\tnext_head20\n')
    for r in rows2:
        fh.write('\t'.join(str(x).replace('\t', ' ') for x in r) + '\n')

print('one-char rows', len(rows1))
print('cross-line rows with prefix issue', len(rows2))
cnt = Counter(r[6] for r in rows1)
for k, v in cnt.most_common():
    print(f'{v:5d}  {k}')
