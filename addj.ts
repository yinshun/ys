/**
 * 補字元級 `j="起-訖"`：把每個 `<聯 i="taisho?<頁>" q="…">` 的 `q`
 * 放回大正藏該頁的文字裡找位置，寫進標籤（只加 `j`；`q` 與正文都不動）。
 *
 *   bun run addj.ts            # 就地改 build/ys-linked.txt
 *
 * 演算法是 adb-converter 的 `retrofit-j.ts` ＋ `verify-j.ts` 的移植：
 *   1. 在目標頁的漢字串（跳過標籤、遇下一個 `<頁` 停、取 4,000 字）裡，
 *      用 12→4 字種子精確比對；對不上再寬鬆對位（允許插字漏字）。
 *   2. 用「頁面原文」複驗一次：把對到的片段拿回同頁原文找，找到才寫 `j`。
 * 差別只在：`q` 一律保留原樣，不換成頁面原文（作者 2026-10-29 指定）。
 *
 * `j` 的定義與 accelon26 `renderForeign` 的 lineHtml 一致：目標頁從
 * `<頁 id="…">` 之後算起，跳過所有 `<…>` 標籤、只算漢字，`j` 是這段漢字串的
 * 0 起算索引，`to` 不含（`j="293-324"` 表示第 293～323 個漢字）。
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { AdbReader, SyncIO } from '../adb-converter/ts/src/accelon26/adb';
import { expandToken } from '../adb-converter/ts/src/accelon26/variants';
import { BUILD, TAISHO } from './paths';

const isHan = (c: string): boolean => /[\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff]/.test(c);
function hanOf(s: string): string {
  let o = '';
  for (let i = 0; i < s.length; i++) {
    if (s[i] === '<') { const gt = s.indexOf('>', i); if (gt < 0) break; i = gt; continue; }
    if (isHan(s[i])) o += s[i];
  }
  return o;
}
/** 異體字展開後取排序第一個，讓「說／説」這類版本差異也能對上（同 retrofit-j） */
const canonCache = new Map<string, string>();
function canon(c: string): string {
  let v = canonCache.get(c);
  if (v) return v;
  const g = expandToken(c);
  v = g.length > 1 ? [...g].sort()[0] : c;
  canonCache.set(c, v);
  return v;
}
const canonAll = (h: string): string => [...h].map(canon).join('');
const attr = (attrs: string, name: string): string =>
  new RegExp('\\b' + name + '\\s*=\\s*"([^"]*)"').exec(attrs)?.[1] ?? '';

/** 寬鬆對位：在頁面字串裡找與 q 最接近的一段（允許插字、漏字） */
function fuzzySpan(q: string, hay: string): { first: number; last: number; matched: number } | null {
  if (q.length < 8 || !hay) return null;
  const seeds = new Set<string>();
  for (let s = 0; s + 6 <= q.length && seeds.size < 8; s += 3) seeds.add(q.slice(s, s + 6));
  let best: { first: number; last: number; matched: number } | null = null;
  const greedy = (at: number) => {
    let i = 0, j = at, first = -1, last = -1, matched = 0;
    const stop = Math.min(hay.length, at + Math.min(240, q.length * 2 + 40));
    while (i < q.length && j < stop) {
      if (q[i] === hay[j]) {
        if (first < 0) first = j;
        last = j; i++; j++; matched++;
      } else {
        let k = 1;
        while (k <= 4 && j + k < stop && hay[j + k] !== q[i]) k++;
        if (k <= 4 && j + k < stop && hay[j + k] === q[i]) j += k;
        else i++;
      }
    }
    return { first, last, matched };
  };
  for (const seed of seeds) {
    let at = hay.indexOf(seed), n = 0;
    while (at >= 0 && n++ < 6) {
      const g = greedy(Math.max(0, at - 3));
      if (g.first >= 0 && (!best || g.matched > best.matched)) best = g;
      at = hay.indexOf(seed, at + 1);
    }
  }
  if (!best || best.first < 0) return null;
  if (best.matched < Math.max(8, Math.ceil(q.length * 0.55))) return null;
  if (best.last - best.first + 1 > 220) return null;
  return best;
}

const txtPath = join(BUILD, 'ys-linked.txt');
const lines = readFileSync(txtPath, 'utf8').split('\n');
if (lines[lines.length - 1] === '') lines.pop();

console.log('開啟', TAISHO);
const tr = await AdbReader.open(new SyncIO(TAISHO));
const tagC = tr.containers.get('tags')!;
const pv = Int32Array.from(await tr.decodeBlockList('tags', tagC.names.indexOf('頁')));
const ids = (await tr.attributeValues('頁.id'))!.values;
const pageIdx = new Map(ids.map((v, k) => [v, k]));
const bv = await tr.blocksVirtual();

const cache = new Map<string, { raw: string; canon: string }>();
/** 目標頁的可見漢字（跳標籤、遇到下一個 <頁 就停；取前 4,000 字） */
async function pageText(id: string): Promise<{ raw: string; canon: string }> {
  const hit = cache.get(id);
  if (hit) return hit;
  const pi = pageIdx.get(id);
  const empty = { raw: '', canon: '' };
  if (pi === undefined) { cache.set(id, empty); return empty; }
  let blk = 0;
  for (let k = 0; k < bv.length; k++) { if (bv[k] <= pv[pi]) blk = k; else break; }
  let t = '';
  for (const off of [0, -1, 1, -2, 2, -3, 3, -4, 4]) {
    const b = blk + off;
    if (b < 0 || b >= bv.length) continue;
    const s = await tr.sourceText(b);
    const at = s.indexOf('<頁 id="' + id + '"');
    if (at >= 0) { t = s.slice(at); blk = b; break; }
  }
  if (!t) { cache.set(id, empty); return empty; }
  const gt = t.indexOf('>');
  if (gt >= 0) t = t.slice(gt + 1);
  for (let b = blk + 1; t.indexOf('<頁 ') < 0 && t.length < 4500 && b < bv.length; b++) t += await tr.sourceText(b);
  const cut = t.indexOf('<頁 ');
  if (cut > 0) t = t.slice(0, cut);
  const raw = hanOf(t).slice(0, 4000);
  const out = { raw, canon: canonAll(raw) };
  cache.set(id, out);
  return out;
}

let links = 0, added = 0, skipped = 0, fuzzy = 0;
const noJ: { i: string; q: string }[] = [];

for (let li = 0; li < lines.length; li++) {
  const line = lines[li];
  if (!line.includes('<聯')) continue;
  const reps: { at: number; len: number; text: string }[] = [];
  for (const m of line.matchAll(/<聯([^>]*)>/g)) {
    const whole = m[0], attrs = m[1];
    const i = attr(attrs, 'i'), q = attr(attrs, 'q');
    if (!i.startsWith('taisho?')) continue;
    if (!q) { skipped++; noJ.push({ i, q: '' }); continue; }
    links++;
    // 取 q（最多 160 字）整段比對，對不上再寬鬆對位；不再截成 80 字。
    // 長引文也要整段塗滿（作者 2026-10-30：37p0007a 那條到「呪藏」就斷了）。
    const qh = canonAll(hanOf(q)).slice(0, 160); // q 本身最多 160 字（calibrate 截的），這裡等於不截斷
    if (qh.length < 4) { skipped++; noJ.push({ i, q }); continue; }
    const { raw, canon: ph } = await pageText(i.slice(7));
    if (!ph) { skipped++; noJ.push({ i, q }); continue; }
    // 1) 精確：12→4 字種子，整段（前 80 字）對上才算
    let pos = -1;
    for (const L of [12, 10, 8, 6, 4]) {
      if (L > qh.length) continue;
      const at = ph.indexOf(qh.slice(0, L));
      if (at < 0) continue;
      if (ph.slice(at, at + qh.length) === qh) { pos = at; break; }
    }
    let end: number;
    if (pos < 0) {
      const fz = fuzzySpan(qh, ph);
      if (!fz) { skipped++; noJ.push({ i, q }); continue; }
      pos = fz.first; end = fz.last + 1; fuzzy++;
    } else {
      end = Math.min(ph.length, pos + qh.length);
    }
    if (end - pos < 4) { skipped++; noJ.push({ i, q }); continue; }
    // 2) 用頁面原文複驗（verify-j 的規則），只取位置，不改 q
    const q2 = raw.slice(pos, end);
    const at2 = raw.indexOf(q2);
    if (at2 < 0) { skipped++; noJ.push({ i, q }); continue; }
    const tag = `<聯${attrs.replace(/\s*$/, '')} j="${at2}-${at2 + q2.length}">`;
    reps.push({ at: m.index, len: whole.length, text: tag });
    added++;
  }
  if (reps.length) {
    let out = line;
    for (const r of reps.sort((a, b) => b.at - a.at)) out = out.slice(0, r.at) + r.text + out.slice(r.at + r.len);
    lines[li] = out;
  }
}

writeFileSync(txtPath, lines.join('\n') + '\n');
const tags = lines.reduce((n, l) => n + (l.match(/<聯 /g) || []).length, 0);
console.log(`聯 ${tags}；有 q 的 ${links}；補 j ${added}（寬鬆 ${fuzzy}）；沒補 ${skipped}`);
if (noJ.length) console.log('沒 j 的前 10 條:', JSON.stringify(noJ.slice(0, 10)));
tr.close?.();
process.exit(0);
