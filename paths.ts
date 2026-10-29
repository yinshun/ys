/**
 * 建置 ys.adb 用到的所有路徑，集中這裡（環境變數可覆寫）。
 *
 * 這個 repo 只放「印順」獨有的東西：來源資料（data/）、圖檔來源、轉換與打包腳本。
 * 通用的 adb 讀寫工具在隔壁的 `adb-converter` 工作區（第一步要一起存在）。
 */
import { join } from 'node:path';

/** 這個 repo 的根目錄（腳本都放在根目錄，所以 cwd 沒差）。 */
export const ROOT = import.meta.dir;
/** 中間產物：ys-text.txt、ys-linked.txt、ys.adb …（不進版控）。 */
export const BUILD = join(ROOT, 'build');

const env = (name: string, fallback: string): string => {
  const v = process.env[name];
  return v && v.trim() ? v.replace(/\\/g, '/') : fallback;
};

/** CBETA Y 部 XML 的 sparse clone（`Y/Y01/Y01n0001.xml` …），只用 Y 一個目錄。 */
export const XML_DIR = env('YS_XML', 'D:/2026/xml-p5/Y');

/** 正文插圖：`svg/YNN/YNNpNNNN_KK.svg`（資源名就是相對路徑）。 */
export const IMAGES = env('YS_IMAGES', 'D:/2026/xml-p5/Y-images');

/** 舊 corpus 的 XML（頁元與引文定位用；不是 CBETA 那份）。 */
export const CORPUS_XML = env('YS_CORPUS_XML', 'D:/2023/yinshun-corpus/xml');

/** 舊 corpus 的 svg（只有 `images.ts` 會用到）。 */
export const CORPUS_SVG = env('YS_CORPUS_SVG', 'D:/2023/yinshun-corpus/svg');

/** 印順 → 大正藏的引文對照表。 */
export const ALIGN = env('YS_ALIGN', 'D:/2026/offtext/alignments/yinshun-taisho.tsv');

/** 大正藏 adb（`addj.ts` 用它算 `j` 的頁面字元索引；唯讀）。 */
export const TAISHO = env('YS_TAISHO', 'D:/accelon3/taisho.adb');

/** 打包後要覆寫的部署位置（Accelon26／accelon3 讀的那一份）。 */
export const DEPLOY = env('YS_DEPLOY', 'D:/accelon3/ys.adb');

