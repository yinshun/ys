/**
 * 反鏈快查表（`xrev` 容器）：把每一個 `<聯 i="db?page">` 的來源頁與來源 VA 先算好，
 * 閱讀器（accelon26）開檔時直接讀，不必每次用 `聯.i`／`聯.q` 現算整張表。
 *
 * 容器格式（讀取端 adb-converter 已照這個解析）：
 *   - 外層容器 `xrev`，只有 1 個 block；
 *   - block 內容是 bzip2 壓縮的 UTF-8 JSON Lines；
 *   - 第一行 `["XREV1", <筆數>]`；
 *   - 之後每行 `["<srcDb>","<srcPage>",<srcVa>,"<targetDb>","<targetPage>","<quote>"]`。
 *
 * 沒有這個容器時，舊的 ys.adb 照舊可用（只是慢）。
 */
import { AdbReader, SyncIO } from '../adb-converter/ts/src/accelon26/adb';

interface AttrTable {
  values: string[];
  pointers: Int32Array | null;
  tags: string[];
}

/** 由 pointer 的高位元組取 tag 名（tags.names 的索引就是這個 byte）。 */
function tagOf(t: AttrTable, k: number, names: string[]): string {
  if (!t.pointers) return '聯';
  return names[(t.pointers[k] >>> 24) & 0xff] ?? '';
}
function instOf(t: AttrTable, k: number): number {
  return t.pointers ? (t.pointers[k] & 0xffffff) - 1 : k;
}

function lastLE(arr: number[], v: number): number {
  let lo = 0, hi = arr.length - 1, ans = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (arr[mid] <= v) { ans = mid; lo = mid + 1; } else hi = mid - 1;
  }
  return ans;
}

export interface XrevResult {
  lines: string[];
  entries: number;
  skipped: number;
}

export async function buildXrev(adbPath: string): Promise<XrevResult> {
  const r = await AdbReader.open(new SyncIO(adbPath));
  try {
    const tags = r.containers.get('tags');
    if (!tags) return { lines: ['["XREV1",0]'], entries: 0, skipped: 0 };
    const lianIdx = tags.names.indexOf('聯');
    const pageIdx = tags.names.indexOf('頁');
    if (lianIdx < 0) return { lines: ['["XREV1",0]'], entries: 0, skipped: 0 };

    const names = tags.names;
    const lianVas = await r.decodeBlockList('tags', lianIdx);
    const av = (await r.attributeValues('聯.i')) as AttrTable | null;
    if (!av) return { lines: ['["XREV1",0]'], entries: 0, skipped: 0 };
    const avq = (await r.attributeValues('聯.q')) as AttrTable | null;

    // 頁 VA → 頁 id
    const pageByVa = new Map<number, string>();
    if (pageIdx >= 0) {
      const pageVas = await r.decodeBlockList('tags', pageIdx);
      const pid = (await r.attributeValues('頁.id')) as AttrTable | null;
      if (pid) {
        for (let k = 0; k < pid.values.length; k++) {
          if (pid.pointers && (names[(pid.pointers[k] >>> 24) & 0xff] ?? '') !== '頁') continue;
          const inst = pid.pointers ? (pid.pointers[k] & 0xffffff) - 1 : k;
          const va = pageVas[inst];
          if (va != null) pageByVa.set(va, pid.values[k]);
        }
      }
    }
    const pageVaSorted = [...pageByVa.keys()].sort((a, b) => a - b);

    // 每個 <聯> instance 的 q（大正原文）
    const qByInst = new Map<number, string>();
    if (avq) for (let k = 0; k < avq.values.length; k++) {
      if (tagOf(avq, k, names) !== '聯') continue;
      qByInst.set(instOf(avq, k), avq.values[k]);
    }

    const srcDb = r.header.dbname;
    const out: string[] = [];
    let skipped = 0;
    for (let k = 0; k < av.values.length; k++) {
      if (tagOf(av, k, names) !== '聯') { skipped++; continue; }
      const val = av.values[k];
      const q = val.indexOf('?');
      if (q <= 0 || q === val.length - 1) { skipped++; continue; }
      const db = val.slice(0, q);
      const anchor = val.slice(q + 1);
      const va = lianVas[instOf(av, k)];
      if (va == null) { skipped++; continue; }
      const pi = lastLE(pageVaSorted, va);
      const srcPage = pi >= 0 ? pageByVa.get(pageVaSorted[pi]) ?? '' : '';
      const quote = qByInst.get(instOf(av, k)) ?? '';
      out.push(JSON.stringify([srcDb, srcPage, va, db, anchor, quote]));
    }
    return { lines: [JSON.stringify(['XREV1', out.length]), ...out], entries: out.length, skipped };
  } finally {
    (r as any).close?.();
  }
}
