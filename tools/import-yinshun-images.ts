/**
 * import-yinshun-images.ts — 從 yinshun.adb 抽出頁圖 PNG，補進 ys 的缺頁
 *
 *   bun run tools/import-yinshun-images.ts
 *
 * yinshun.adb 的 `resources` 裡有一批 `yNN-PPP.png`（舊版點陣頁圖），CBETA Y-images
 * 沒有對應的 SVG。這裡：
 *   1. 把 PNG 寫到 `D:\2026\xml-p5\Y-images\png\YNN\`（pack.ts 的圖檔根目錄）；
 *   2. 在 `build/ys-linked.txt` 的對應 `<頁 id="NNpPPPPx"/>` 後面插
 *      `<圖 n="png/YNN/yNN-PPP.png" t="圖版"/>`；
 *   3. 該頁已經有 `<圖>`（SVG 或表格圖）就跳過，不重複；
 *   4. 不動換行（行號不變）。
 */
import { AdbReader, SyncIO } from '../../adb-converter/ts/src/accelon26/adb';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const YINSHUN = 'D:/accelon3/yinshun.adb';
const IMG_ROOT = 'D:/2026/xml-p5/Y-images';
const TEXT = 'D:/2026/ys/build/ys-linked.txt';

const r = await AdbReader.open(new SyncIO(YINSHUN));
const c = r.containers.get('resources')!;
const pages: { vol: string; page: string; name: string; bytes: Uint8Array }[] = [];
for (let i = 0; i < c.count; i++) {
  const { name, bytes } = await r.resourceBytes(i);
  const m = /(?:^|\/)(?:y|Y)(\d+)-(\d{2,4})\.png$/i.exec(name);
  if (!m) continue;
  pages.push({ vol: String(Number(m[1])), page: String(Number(m[2])), name: name.split('/').pop()!, bytes });
}
(r as any).close?.();
console.log('yinshun 頁圖 PNG', pages.length);

let text = readFileSync(TEXT, 'utf8');
const pageRe = /<頁 id="(\d+)p[a-z]?(\d{1,4})[a-z]"\s*\/>/g;
const markerList = [...text.matchAll(pageRe)];
const markers = new Map<string, number>(); // key vol|page → 插入位置（第一個頁標記之後）
const markerIndex = new Map<string, number>();
markerList.forEach((m, i) => {
  const key = `${Number(m[1])}|${Number(m[2])}`;
  if (!markers.has(key)) { markers.set(key, m.index! + m[0].length); markerIndex.set(key, i); }
});
// 每頁是否已經有 <圖>
const hasImg = new Set<string>();
{
  const parts = text.split(/(<頁 id="(\d+)p[a-z]?(\d{1,4})[a-z]"\s*\/>)/);
  let cur = '';
  for (const part of parts) {
    const m = /^<頁 id="(\d+)p[a-z]?(\d{1,4})[a-z]"\s*\/>$/.exec(part);
    if (m) { cur = `${Number(m[1])}|${Number(m[2])}`; continue; }
    if (cur && part.includes('<圖 ')) hasImg.add(cur);
  }
}

// <表> 區塊裡不要插圖（頁碼列必須單獨一行）
const tableRanges: [number, number][] = [];
for (const m of text.matchAll(/<表>[\s\S]*?<\/表>/g)) tableRanges.push([m.index!, m.index! + m[0].length]);
const inTable = (pos: number) => tableRanges.some(([a, b]) => a <= pos && pos < b);

const inserts = new Map<number, string[]>();
let added = 0, skippedHas = 0, skippedNoPage = 0;
const dir = join(IMG_ROOT, 'png');
for (const p of pages) {
  const key = `${p.vol}|${p.page}`;
  const at = markers.get(key);
  if (at == null || inTable(at)) { skippedNoPage++; continue; }
  const mi = markerIndex.get(key)!;
  const segEnd = mi + 1 < markerList.length ? markerList[mi + 1].index! : text.length;
  if (text.slice(at, segEnd).includes('<表>')) { skippedNoPage++; continue; }
  if (hasImg.has(key)) { skippedHas++; continue; }
  const volDir = 'Y' + p.vol.padStart(2, '0');
  const outDir = join(dir, volDir);
  if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });
  writeFileSync(join(outDir, p.name), p.bytes);
  inserts.set(at, [...(inserts.get(at) ?? []), `<圖 n="png/${volDir}/${p.name}" t="圖版"/>`]);
  added++;
}

const out: string[] = [];
let last = 0;
for (const at of [...inserts.keys()].sort((a, b) => a - b)) {
  out.push(text.slice(last, at));
  out.push(inserts.get(at)!.join(''));
  last = at;
}
out.push(text.slice(last));
const next = out.join('');
if (next.split('\n').length !== text.split('\n').length) {
  console.error('警告：換行數改變，放棄寫入');
  process.exit(1);
}
writeFileSync(TEXT, next, 'utf8');
console.log(`補 PNG ${added} 頁（已有圖 ${skippedHas}、無頁標記 ${skippedNoPage}）`);
