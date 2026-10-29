/**
 * 圖檔 → `D:\2026\xml-p5\Y-images`（CBETA Y 部 XML 的 `<graphic>` 指到的地方）
 *
 * 目的是讓 `y2adb` 能按 `<graphic url="../figures/Y/Y08p0052_01.svg">` 直接找到圖。
 * `Y-images` 底下的相對路徑就是 adb 的資源名：
 *
 *   D:\2026\xml-p5\Y-images\svg\Y08\Y08p0052_01.svg
 *   → `<圖 n="svg/Y08/Y08p0052_01.svg" t="圖版"/>`
 *
 * 兩個來源（後者蓋掉前者，因為它是 CBETA 現行版本）：
 *
 *   --figures  cbeta-git/CBR2X-figures 的 `Y/`（檔名已是 Y08p0052_01.svg，原樣複製）
 *   --src      yinshun/yinshun-corpus 的 `svg/`（檔名 y08.0052.svg，改名後複製）
 *
 * 用法（在 D:\2026\ys 下）：
 *   bun run images.ts                       # 兩個都用（有的話）
 *   bun run images.ts --src ''              # 只取 CBR2X
 */
import { copyFileSync, existsSync, mkdirSync, readdirSync, statSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { CORPUS_SVG, IMAGES, ROOT } from './paths';

const arg = (name: string, fallback: string): string => {
  const i = process.argv.indexOf(`--${name}`);
  return i >= 0 && process.argv[i + 1] !== undefined ? process.argv[i + 1] : fallback;
};

const SRC = arg('src', existsSync(CORPUS_SVG) ? CORPUS_SVG : join(ROOT, '.local/yinshun-corpus/svg'));
const FIG = arg('figures', join(ROOT, '.local/CBR2X-figures/Y'));
const DEST = arg('dest', IMAGES);

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    return statSync(full).isDirectory() ? walk(full) : [full];
  });

const copyTo = (file: string, rel: string): void => {
  const out = join(DEST, rel);
  mkdirSync(dirname(out), { recursive: true });
  copyFileSync(file, out);
};

/* ---------------------------------------------- 1) corpus：y08.0052.svg → Y08p0052_01.svg */
function corpus(): number {
  if (!SRC || !existsSync(SRC)) return 0;
  let copied = 0;
  for (const file of walk(SRC).filter((f) => f.toLowerCase().endsWith('.svg'))) {
    const m = /^(y\d{2})\.(\d{4})(?:-(\d+))?\.svg$/i.exec(file.split(/[\\/]/).pop()!);
    if (!m) continue;
    const vol = m[1].toUpperCase();
    const name = `${vol}p${m[2]}_${String(m[3] ? Number(m[3]) : 1).padStart(2, '0')}.svg`;
    copyTo(file, `svg/${vol}/${name}`);
    copied++;
  }
  return copied;
}

/* ---------------------------------------------- 2) CBR2X：檔名照用（Y08p0052_01.svg / .gif） */
function figures(): number {
  if (!FIG || !existsSync(FIG)) return 0;
  let copied = 0;
  for (const file of walk(FIG)) {
    const name = file.split(/[\\/]/).pop()!;
    const m = /^(Y\d{2})/i.exec(name);
    if (!m) continue;
    const vol = m[1].toUpperCase();
    copyTo(file, `svg/${vol}/${name}`);
    copied++;
  }
  return copied;
}

const a = corpus();
const b = figures();
console.log(`corpus ${SRC}：${a} 張；CBR2X ${FIG}：${b} 張 → ${DEST}`);
if (a + b === 0) {
  console.error('兩個來源都沒有檔案');
  process.exit(1);
}
