import { existsSync, readFileSync, renameSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
// 通用的 adb 打包工具在隔壁的 adb-converter 工作區（第一步要一起存在）
import { authorAdb } from '../adb-converter/ts/src/adbauthor';
import { MAXLINELEN } from '../adb-converter/ts/src/index-builder';
import { buildContainer } from '../adb-converter/ts/src/accelon26/adb-bytes';
import { createEncoder } from '../adb-converter/ts/src/bz2';
import { BUILD, DEPLOY, IMAGES, ROOT } from './paths';
import { buildXrev } from './xrev';
import { buildGlossaryFromBuild } from './glossary';

/** 圖檔根目錄：資源名就是相對於這裡的路徑（svg/Y08/Y08p0052_01.svg）。 */
const IMG_ROOT = IMAGES;

const text = readFileSync(join(BUILD, 'ys-linked.txt'), 'utf8');
const lines = text.split('\n');
if (lines[lines.length - 1] === '') lines.pop();

// Accelon3 的行長上限（8192 bytes = 4096 chars）：lines.physical 會把每行截到這裡，
// 超過的行會讓行表與內文錯開（搜尋結果定位錯誤）。寧可擋下來，也不要默默產生壞檔。
const over = lines.map((l, i) => [i + 1, l.length]).filter(([, n]) => (n as number) * 2 > MAXLINELEN);
if (over.length) {
  throw new Error(
    `${over.length} 行的長度超過 ${MAXLINELEN / 2} 字（Accelon3 行長上限），例如第 ${over
      .slice(0, 5)
      .map(([i, n]) => `${i} 行 ${n} 字`)
      .join('、')}`,
  );
}

const logo = readFileSync(join(ROOT, 'logo/ys_logo.png'));
const found = readFileSync(join(ROOT, 'logo/yinshunfoundation.png'));

// 正文的 <圖 n="svg/Y08/Y08p0052_01.svg"/> 指向 resources 裡同名的資源。
// 只打包正文真的用到的圖（Y-images 有幾百張，ys 只用到 75 張）。
const imageNames = [...new Set(
  [...text.matchAll(/<圖[^>]*?\bn\s*=\s*"([^"]*)"/g)].map((m) => m[1]),
)].sort();
const images: { name: string; bytes: Uint8Array }[] = [];
const missing: string[] = [];
for (const name of imageNames) {
  const path = `${IMG_ROOT}/${name}`;
  if (!existsSync(path)) { missing.push(name); continue; }
  images.push({ name, bytes: readFileSync(path) });
}
const imageBytes = images.reduce((n, r) => n + r.bytes.length, 0);
console.log(`圖 ${images.length} 張（${(imageBytes / 1048576).toFixed(2)} MiB）` +
  (missing.length ? `，缺 ${missing.length} 張：${missing.slice(0, 3).join('、')}…` : ''));

const authorInput = {
  dbname: 'ys',
  dbcname: '印順法師佛學著作集',
  dbtype: 0,
  lines,
  resources: [
    { name: 'ys_logo.png', bytes: logo },
    { name: 'yinshunfoundation.png', bytes: found },
    ...images,
  ],
  log: (m: string) => console.error(m),
};

// 第一段：先產生「沒有 xrev」的檔，反鏈表要從它的 VA／屬性算。
const first = await authorAdb(authorInput);
const tmpAdb = join(BUILD, 'ys-noxrev.adb');
writeFileSync(tmpAdb, Buffer.from(first.file));

// 第二段：算反鏈表 → xrev 容器（bzip2 JSON Lines）→ 重新打包。
let out = first;
try {
  const xr = await buildXrev(tmpAdb);
  if (xr.entries > 0) {
    const { encoder, notes } = createEncoder();
    if (!encoder) throw new Error('no bzip2 encoder: ' + notes.join('; '));
    const payload = Buffer.from(xr.lines.join('\n') + '\n', 'utf8');
    const packed = encoder.encode(payload);
    const xrev = buildContainer({ feature: 0, blocks: [packed] });
    // 多語詞彙表（2026-10-31）：未壓縮 UTF-8 JSON Lines，供閱讀器「詞彙」分頁。
    const glines = buildGlossaryFromBuild();
    const gbytes = new TextEncoder().encode(glines.join('\n') + '\n');
    const glossary = buildContainer({ feature: 0, blocks: [gbytes] });
    console.log(`glossary ${glines.length - 1} 筆（${(gbytes.length / 1024).toFixed(0)} KiB）`);
    out = await authorAdb({ ...authorInput, extraContainers: { xrev, glossary } });
    console.log(`xrev ${xr.entries} 筆（JSON ${(payload.length / 1048576).toFixed(2)} MiB → bz2 ${(packed.length / 1048576).toFixed(2)} MiB）`);
    const vas = (r: { lines: string[] }) => r.lines.slice(1).map((l) => JSON.parse(l)[2]).sort((a: number, b: number) => a - b).join(',');
    const want = vas(xr);
    let ok = false;
    try {
      const finalTmp = join(BUILD, 'ys-withxrev.adb');
      writeFileSync(finalTmp, Buffer.from(out.file));
      const again = await buildXrev(finalTmp);
      ok = again.entries === xr.entries && vas(again) === want;
      rmSync(finalTmp, { force: true });
    } catch { ok = false; }
    if (!ok) {
      console.warn('xrev 驗證失敗（加容器後 VA 位移），改出不含 xrev 的檔');
      out = first;
    }
  } else {
    console.warn('xrev：沒有可用的 <聯 i="db?page">，略過');
  }
} catch (e) {
  console.warn('xrev 產生失敗，改出不含 xrev 的檔：', e instanceof Error ? e.message : e);
} finally {
  rmSync(tmpAdb, { force: true });
}

const buf = Buffer.from(out.file); buf.writeUInt32LE(0, 100);
// 產物只寫部署位置（D:\accelon3\ys.adb）；工作區不留 ys.adb。
// 直接覆寫偶爾會拿到 EUNKNOWN（檔案被 Accelon26 開著、或 Defender 正在掃 26 MB 的新檔）。
// 先寫 .new 再改名，是同一顆磁碟上的原子替換。
const deployed = DEPLOY;
writeFileSync(deployed + '.new', buf);
renameSync(deployed + '.new', deployed);
console.log('wrote', deployed, (buf.length / 1048576).toFixed(1), 'MiB, lines', lines.length);
