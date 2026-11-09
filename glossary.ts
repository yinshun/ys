/**
 * 多語詞彙對照表（`glossary` 容器）：把 ys-text 裡的 `<术 sk|pi|tb="外語">漢詞</术>`
 * 整理成去重的清單，供閱讀器開「詞彙」分頁查詢與「直接跳到某次出現」。
 *
 * 容器格式（讀取端 adb-converter 的 glossary.ts 依此解析）：
 *   - 外層容器 `glossary`，只有 1 個 block；
 *   - block 內容是**未壓縮**的 UTF-8 JSON Lines（純文字，reader 直接 decode）；
 *   - 第一行 `["GLOSSARY1", <筆數>]`；
 *   - 之後每行 `["<lang>","<foreign>","<han>",<count>,<line>,<page>,<off>]`
 *     line＝1 起的行號、page＝該行所屬的 `<頁 id>`、off＝該頁第幾個漢字（0 起）。
 *     舊版沒有這三欄，閱讀器會退回「點了搜尋」。
 *
 * 排序：語言（sk、pi、tb）→ 漢詞 → 外語；同（語言＋外語＋漢詞）只留一筆並累計次數，
 * 位置取第一次出現。
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { BUILD } from './paths';

const TAG_SRC = '<术 (sk|pi|tb)="([^"]*)">([^<]*)<\\/术>';
const PAGE_RE = /<頁 id="([^"]+)"/g;
const HAN = /\p{Script=Han}/u;

export interface GlossaryRow {
  lang: string;
  foreign: string;
  han: string;
  count: number;
  /** 1 起的行號（build/ys-linked.txt） */
  line: number;
  /** 該行所屬的 `<頁 id>`（沒有就空字串） */
  page: string;
  /** 該頁第幾個漢字（0 起；閱讀器用它精準跳頁） */
  off: number;
  /** 該行第幾個漢字（0 起；頁碼對不上時退回行內定位） */
  hanline: number;
}

// 與閱讀器 resolveCharPosition() 的 hanCountOf 一致：先剝掉標籤再數漢字，
// 否則 <术>／<註> 這些標籤本身的漢字會被算進去，位置就越跳越遠（2026-10-31）。
function hanLen(s: string): number {
  let n = 0;
  for (const ch of s.replace(/<[^>]*>/g, '')) if (HAN.test(ch)) n++;
  return n;
}

export function buildGlossary(text: string): GlossaryRow[] {
  const map = new Map<string, GlossaryRow>();
  const lines = text.split('\n');
  const starts: number[] = new Array(lines.length);
  let acc = 0;
  for (let i = 0; i < lines.length; i++) { starts[i] = acc; acc += lines[i].length + 1; }
  const lineAt = (pos: number): number => {
    let lo = 0, hi = starts.length - 1;
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1;
      if (starts[mid] <= pos) lo = mid; else hi = mid - 1;
    }
    return lo;
  };

  // 頁標記與 <术> 依位置合併掃描：一邊累計「這一頁第幾個漢字」，一邊記下每個詞的位置。
  type Ev = { pos: number; page?: string; tag?: RegExpExecArray };
  const evs: Ev[] = [];
  PAGE_RE.lastIndex = 0;
  let pm: RegExpExecArray | null;
  while ((pm = PAGE_RE.exec(text))) evs.push({ pos: pm.index, page: pm[1] });
  const tagRe = new RegExp(TAG_SRC, 'g');
  let tm: RegExpExecArray | null;
  while ((tm = tagRe.exec(text))) evs.push({ pos: tm.index, tag: tm });
  evs.sort((a, b) => a.pos - b.pos);

  let page = '';
  let hanSincePage = 0;
  let last = 0;
  for (const ev of evs) {
    hanSincePage += hanLen(text.slice(last, ev.pos));
    last = ev.pos;
    if (ev.page) { page = ev.page; hanSincePage = 0; continue; }
    const m = ev.tag!;
    const lang = m[1];
    const foreign = m[2].replace(/&quot;/g, '"').replace(/\s+/g, ' ').trim();
    const han = m[3].replace(/\s+/g, '');
    if (!foreign || !han) continue;
    const key = `${lang}\u0000${foreign}\u0000${han}`;
    const hit = map.get(key);
    if (hit) { hit.count++; continue; }
    const li = lineAt(ev.pos);
    const lineStart = starts[li];
    const hanline = hanLen(text.slice(lineStart, ev.pos));
    map.set(key, { lang, foreign, han, count: 1, line: li + 1, page, off: hanSincePage, hanline });
  }

  const order: Record<string, number> = { sk: 0, pi: 1, tb: 2 };
  return [...map.values()].sort((a, b) =>
    (order[a.lang] ?? 9) - (order[b.lang] ?? 9)
    || a.han.localeCompare(b.han)
    || a.foreign.localeCompare(b.foreign));
}

export function glossaryLines(rows: GlossaryRow[]): string[] {
  const lines = [JSON.stringify(['GLOSSARY1', rows.length])];
  for (const r of rows) lines.push(JSON.stringify([r.lang, r.foreign, r.han, r.count, r.line, r.page, r.off, r.hanline]));
  return lines;
}

/** 由 build/ys-linked.txt 產生 glossary 的 JSON Lines（pack.ts 用）。 */
export function buildGlossaryFromBuild(): string[] {
  const text = readFileSync(join(BUILD, 'ys-linked.txt'), 'utf8');
  return glossaryLines(buildGlossary(text));
}
