/**
 * 併稿：y2adb 的 44 冊 ＋ 附錄（data/appendix.txt）＋ 後記（data/ys-epilog26.txt）。
 *
 * 輸出 `build/ys-text-full.txt`，並替 `<章>`／`<節>`／`<項>` 補 `u="文"`
 * （規則在 `data/u-attrs.txt`，少了它節點不進「頻次」）。
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { BUILD, ROOT } from './paths';

const DATA = join(ROOT, 'data');

const readBlock = (p: string) =>
  readFileSync(p, 'utf8')
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);

// 附錄的表格靠行首空格對齊，所以只去行尾空白，不 trim 行首
const readAppendix = (p: string) =>
  readFileSync(p, 'utf8')
    .split(/\r?\n/)
    .map((s) => s.replace(/\s+$/, ''))
    .filter((s) => s.trim().length > 0);

const readUAttrs = (p: string): Map<string, string> => {
  const map = new Map<string, string>();
  for (const line of readFileSync(p, 'utf8').split(/\r?\n/)) {
    if (!line.trim() || line.trim().startsWith('#')) continue;
    const [tag, value] = line.split('\t');
    if (tag && value) map.set(tag.trim(), value.trim());
  }
  return map;
};

/** 對每一行的開標籤補 u=（已經有 u= 的不動） */
const applyUAttrs = (lines: string[], rules: Map<string, string>): number => {
  let fixed = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line.indexOf('<') < 0) continue;
    lines[i] = line.replace(/<([^\s<>/]+)((?:[^<>]*?))(\/?)>/g, (m, tag: string, attrs: string, selfClose: string) => {
      const u = rules.get(tag);
      if (!u) return m;
      if (/(^|\s)u=/.test(attrs)) return m;
      fixed++;
      return '<' + tag + attrs + ' u="' + u + '"' + (selfClose ? '/' : '') + '>';
    });
  }
  return fixed;
};

const ep = readBlock(join(DATA, 'ys-epilog26.txt'));
const app = readAppendix(join(DATA, 'appendix.txt'));

const text = readFileSync(join(BUILD, 'ys-text.txt'), 'utf8');
const lines = text.split('\n');
if (lines[lines.length - 1] === '') lines.pop();
const header = lines[0];
const rest = lines.slice(1);
const all = [header, ...rest, ...app, ...ep];
const fixed = applyUAttrs(all, readUAttrs(join(DATA, 'u-attrs.txt')));
const out = all.join('\n') + '\n';
writeFileSync(join(BUILD, 'ys-text-full.txt'), out);
console.log('appendix lines', app.length, 'epilogue lines', ep.length,
  'u= 補上', fixed, 'total', out.split('\n').length - 1);
