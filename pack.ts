import { existsSync, readFileSync, renameSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
// 通用的 adb 打包工具在隔壁的 adb-converter 工作區（第一步要一起存在）
import { authorAdb } from '../adb-converter/ts/src/adbauthor';
import { MAXLINELEN } from '../adb-converter/ts/src/index-builder';
import { BUILD, DEPLOY, IMAGES, ROOT } from './paths';

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

const out = await authorAdb({
  dbname: 'ys',
  dbcname: '印順法師佛學著作集',
  dbtype: 0,
  lines,
  resources: [
    { name: 'ys_logo.png', bytes: logo },
    { name: 'yinshunfoundation.png', bytes: found },
    ...images,
  ],
  log: (m) => console.error(m),
});
const buf = Buffer.from(out.file); buf.writeUInt32LE(0, 100);
// 產物只寫部署位置（D:\accelon3\ys.adb）；工作區不留 ys.adb。
// 直接覆寫偶爾會拿到 EUNKNOWN（檔案被 Accelon26 開著、或 Defender 正在掃 26 MB 的新檔）。
// 先寫 .new 再改名，是同一顆磁碟上的原子替換。
const deployed = DEPLOY;
writeFileSync(deployed + '.new', buf);
renameSync(deployed + '.new', deployed);
console.log('wrote', deployed, (buf.length / 1048576).toFixed(1), 'MiB, lines', lines.length);
