"""從佛光大辭典（D:\\2023\\fgdict-corpus\\xml）抽「漢詞—梵/巴」對照表。

用法：
    python tools/fgdict-terms.py            # 產生 build/fgdict-terms.tsv
    python tools/fgdict-terms.py --check    # 順便比對 build/ys-linked.txt 的 <术> 標記

輸出欄位：term（漢詞）、lang（sk 梵／pi 巴）、foreign（原樣）、norm（去變音符號、
去空白與連字號的小寫鍵，用來比對）。
"""

import glob
import re
import sys
import unicodedata

XML_GLOB = r'D:\2023\fgdict-corpus\xml\fgdict*.xml'
OUT = r'D:\2026\ys\build\fgdict-terms.tsv'
YS = r'D:\2026\ys\build\ys-linked.txt'
LATIN = re.compile(r"^[\sA-Za-z\u00C0-\u024F\u1E00-\u1EFF.\-'’]+")


def read(path: str) -> str:
    raw = open(path, 'rb').read()
    for enc in ('utf-8-sig', 'utf-16', 'gb18030'):
        try:
            return raw.decode(enc)
        except UnicodeDecodeError:
            continue
    return raw.decode('utf-8', 'replace')


def clean_foreign(s: str) -> str:
    m = LATIN.match(s.strip())
    if not m:
        return ''
    return m.group(0).strip(" .-'’")


def norm(s: str) -> str:
    s = unicodedata.normalize('NFD', s)
    s = ''.join(c for c in s if not unicodedata.combining(c))
    s = unicodedata.normalize('NFC', s).lower()
    return re.sub(r"[\s.\-'’()（）]", '', s)


def extract() -> list[tuple[str, str, str, str]]:
    rows: list[tuple[str, str, str, str]] = []
    seen = set()
    for f in sorted(glob.glob(XML_GLOB)):
        s = read(f)
        for m in re.finditer(r'<詞>(.*?)</詞>(.*?)(?=<詞>|$)', s, re.S):
            term = m.group(1).strip()
            if not term:
                continue
            for pm in re.finditer(r'（(梵|巴)([^）]{1,60})）', m.group(2)):
                lang = 'sk' if pm.group(1) == '梵' else 'pi'
                foreign = clean_foreign(pm.group(2))
                if not foreign or len(foreign) < 2:
                    continue
                key = (term, lang, norm(foreign))
                if key in seen:
                    continue
                seen.add(key)
                rows.append((term, lang, foreign, norm(foreign)))
    return rows


def check(rows: list[tuple[str, str, str, str]]) -> None:
    dic: dict[str, set[str]] = {}
    for term, _lang, _foreign, n in rows:
        dic.setdefault(n, set()).add(term)
    pat = re.compile(r'<术 (sk|pi|tb)="([^"]*)">([^<]*)</术>')
    hit = miss = 0
    for line in open(YS, encoding='utf-8'):
        for m in pat.finditer(line):
            if '\n' in m.group(3):
                continue
            if norm(m.group(2)) in dic:
                hit += 1
            else:
                miss += 1
    print(f'ys <术> 對到辭典的：{hit}，沒對到：{miss}')


def main() -> None:
    rows = extract()
    with open(OUT, 'w', encoding='utf-8', newline='\n') as fh:
        fh.write('term\tlang\tforeign\tnorm\n')
        for term, lang, foreign, n in rows:
            fh.write(f'{term}\t{lang}\t{foreign}\t{n}\n')
    print(f'寫入 {OUT}：{len(rows)} 組、{len({r[0] for r in rows})} 個漢詞、{len({r[3] for r in rows})} 個外語鍵')
    if '--check' in sys.argv:
        check(rows)


if __name__ == '__main__':
    main()
