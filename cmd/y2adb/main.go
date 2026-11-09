package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ys.adb 的文字：檔頭（集/隱/樹/引/摘要頭/參考資料/英文名）與 yinshun.adb 相同，
// 內文用 <檔> <書> <卷> <頁元> <頁> <段> <章>，其餘細節不強求一致。
const header = `<檔 n="ys.xml"><集 l="圖集,註釋群,編目資訊">印順法師佛學著作集</集><隱><樹 s="*" t="書" pf1="$書.id;.">目錄</樹><樹 s="*u=文" pf="$頁.id;)">頻次</樹><樹 s="~00602000">西文</樹></隱><隱><引>《$書;》：「$_;」($URL;)</引><摘要頭>$書;($頁;)</摘要頭><參考資料 n="taisho,wxzj"/></隱><英文名>Master Yinshun Corpus of Buddhist Studies</英文名></檔>`

var titleRe = regexp.MustCompile(`<title level="m"[^>]*>([^<]+)</title>`)

// 目次層級：CBETA 的 <cb:mulu level="N"> 對到第 N 個名字（N 最多 18，實測 Y 部最深 18）。
// 前三個沿用 章／節／項；第四、五、六個用印順舊 corpus 的 干／一／數（攝大乘論講記的
// 甲、乙、丙與底下的 一、二、三），再深就 Ａ、Ｂ、Ｃ…。層級鏈寫在每個 <書 t="…">。
var tierNames = []string{
	"部", "章", "節", "項", "干", "一", "數",
	"Ａ", "Ｂ", "Ｃ", "Ｄ", "Ｅ", "Ｆ", "Ｇ", "Ｈ", "Ｉ", "Ｊ", "Ｋ",
}

var tierChain = strings.Join(tierNames, ",") + "."

// CBETA 的 <graphic url="../figures/Y/Y08p0052_01.svg"/>：檔名就是圖名，
// 前面的 Y08 是冊。圖檔放在 Y-images/svg/Y08/ 下（images.ts 從
// CBR2X-figures 與 yinshun-corpus/svg 搬過來），adb 的資源名沿用這個相對路徑。
// 少數圖只有點陣版（`.gif`），照樣收。
var figureRe = regexp.MustCompile(`^(Y\d{2})p([0-9b]+)_(\d{2})\.(svg|gif|png)$`)

type emitter struct {
	out     strings.Builder
	line    strings.Builder
	stats   map[string]int
	lineNo  int
	vol     int
	noteSeq int
	lbOut   *bufio.Writer
	// 圖檔根目錄（Y-images）：只有檔案真的存在才寫 <圖>，缺圖就只留 figDesc
	// 的文字，免得讀者看到一堆「▢」佔位方塊。
	imgDir string
	// <cb:mulu level="N"> 的層級：1＝章、2＝節、3＝項（CBETA 只用這個標記目次層級）
	muluLevel int
	// 目前開著的標題標籤（章／節／項），</章> 要對上同一個名字
	headTag string
	// 這本書的書名與「書名重複的標題」是否已經處理（第一個 <head> 常常就是書名）
	bookTitle    string
	titleDropped bool
	// <foreign xml:lang="sa|pi|bo">：術語外語，轉成 <术 sk|pi|tb="…">
	inForeign   bool
	foreignLang string
	foreignBuf  strings.Builder
	parenKind   int // 1=全形（ 2=半形(
	dropParen   bool
	// 腳註：CBETA <back> 的 yinshun-notes（印順自己的註），n（CBETA 編號）→ 已轉好的注文
	notes     map[string]string
	noteShort map[string]string // CBETA n → 短編號（<書>.<序>，顯示用）
	notePend  []string          // 這本書出現過的註，等 </body> 一起輸出 <釋>
	// 正文裡的行內註（<note place="inline2">…</note>，例如「方等泥洹經
	// 〔註82〕竺法護釋法顯」）：轉成 <註>/<釋> 時用來收集文字（2026-10-30）
	noteKey     string
	noteBuf     strings.Builder
	noteRefOpen []bool
	// <note place="inline">：只是行內的大正／南傳對照，不是腳註（作者 2026-11-02）
	inlineNote    bool
	inlineRefOpen []bool
	// 紙本註號對照（冊	n → 冊.書內註），來源 data/note-numbers.tsv（作者 2026-11-02）
	noteNum map[string]string
	// 目前的大正頁（pb n，如 "0111a"），<table> 要靠它找 Y-images 的表格圖
	page string
	// 表格緩衝（2026-11-06）：<table> 收成 <表> 文字表格
	inTable   bool
	tblBroken bool
	tblRows   [][]string
	tblRow    []string
	tblCell   *strings.Builder
	tblPage   string
	// cell 內容先寫到這裡，</cell> 再收成一格（避免表格內容直接流進正文）
	capture *strings.Builder
	inHead      bool
	// CB 缺字（<g ref="#CB…">）：每個檔案的 charDecl 先把 CB 編號對到 Unicode，
	// 轉檔時再把 <g> 換成真正的字（2026-10-31）。
	curFile      string
	curChar      string
	inMapping    bool
	mapBuf       strings.Builder
	gaiji        map[string]string
	inGaiji      bool
	gaijiRef     string
	gaijiBuf     strings.Builder
	gaijiOK      int
	gaijiMiss    int
	gaijiMissing []string
	// 上一行先留在 pending，等下一行要輸出時才寫進 out（2026-10-31：<术> 跨行
	// 時需要回頭在上一行插入開標籤，直接寫 out 就改不到了）。
	pending    string
	hasPending bool
	// 這一行是跨行術語的「收尾行」：舊行為是含 <术 的行就跳過 wrapSanskrit，
	// 為了不改變全文 <术> 的數量，收尾行也照樣跳過（2026-10-31）。
	crossClosed bool
}

// 梵語詞（作者 2026-10-31）：CBETA 的「漢詞（jātaka）」改成
//   <术 sk="jātaka">漢詞</术>
// 梵語放進屬性（閱讀器可選擇行內或 hover 顯示），正文只留漢詞。
var sktParen = regexp.MustCompile(`[（(]([\p{Latin}][\p{Latin}0-9 .,;:'’\-]*)[）)]`)

/** 詞的起點往左退到標點（或開引號）；回 -1 表示不適合包。 */
const termStops = "。！？；：，、」』）】》〉…—「『（《〈【 \t　>的是與今即而則也都就才只還更最沒及或到至"

func sanskritStart(s string, at int) int {
	const stops = termStops
	if at <= 0 {
		return -1
	}
	if strings.HasSuffix(s[:at], "」") {
		if k := strings.LastIndex(s[:at], "「"); k >= 0 {
			return k
		}
	}
	if strings.HasSuffix(s[:at], "》") {
		if k := strings.LastIndex(s[:at], "《"); k >= 0 {
			return k + len("《")
		}
	}
	rs := []rune(s[:at])
	j := len(rs) - 1
	for j >= 0 && !strings.ContainsRune(stops, rs[j]) {
		j--
	}
	start := len(string(rs[:j+1]))
	if start >= at {
		return -1
	}
	if len(rs[j+1:]) > 8 {
		return -1
	}
	return start
}

func termLang(lang string) string {
	switch {
	case lang == "sa", lang == "x-sa", lang == "x-sa-pi":
		return "sk"
	case lang == "pi", lang == "x-pi":
		return "pi"
	case lang == "bo":
		return "tb"
	}
	return ""
}

/** </foreign>：把「漢詞（外語）」換成 <术 sk|pi|tb="外語">漢詞</术>，並吃掉後面的 ）。 */
func (e *emitter) endForeign() {
	text := strings.TrimSpace(e.foreignBuf.String())
	key := e.foreignLang
	e.inForeign, e.foreignLang = false, ""
	e.foreignBuf.Reset()
	if text == "" {
		return
	}
	open := "（"
	if e.parenKind == 2 {
		open = "("
	}
	if e.parenKind == 0 {
		e.write(esc(text))
		return
	}
	s := e.line.String()
	if !strings.HasSuffix(s, open) {
		e.write(esc(text))
		e.parenKind = 0
		return
	}
	at := len(s) - len(open)
	// 書名號的詞被 <lb> 切斷（《往｜世書》（Purāṇa））：開標籤插到上一行。
	if strings.HasSuffix(s[:at], "》") && !strings.Contains(s[:at], "《") && e.hasPending {
		k := at - len("》")
		for k > 0 && unicode.Is(unicode.Han, rune(s[k-1])) {
			k--
		}
		cur := s[k : at-len("》")]
		prev := e.pending
		if cur != "" && !strings.ContainsAny(cur, "<>") {
			if j := strings.LastIndex(prev, "《"); j >= 0 {
				prevPart := prev[j+len("《"):]
				if prevPart != "" && !strings.ContainsAny(prevPart, "<>") && len([]rune(prevPart+cur)) <= 8 {
					sk := strings.ReplaceAll(text, `"`, "&quot;")
					e.pending = prev[:j+len("《")] + "<术 " + key + `="` + sk + `">` + prevPart
					e.line.Reset()
					e.line.WriteString(cur + "</术>" + s[at-len("》"):])
					e.dropParen = true
					e.parenKind = 0
					return
				}
			}
		}
	}
	// 括號在行首（詞全在上一行，例：六、到了閻浮提畔\n（Jambudvīpaśīrṣa）…）：把詞包在上一行。
	if at == 0 && e.hasPending {
		prev := e.pending
		seg, st := "", -1
		if ov, ok := termOverrides[strings.ToLower(text)]; ok && ov != "-" && strings.HasSuffix(prev, ov) {
			seg, st = ov, len(prev)-len(ov)
		}
		if st >= 0 && len([]rune(seg)) >= 2 && !strings.ContainsAny(seg, "<>") {
			sk := strings.ReplaceAll(text, `"`, "&quot;")
			e.pending = prev[:st] + "<术 " + key + `="` + sk + `">` + seg
			e.line.Reset()
			e.line.WriteString("</术>" + s[at+len(open):])
			e.dropParen = true
			e.parenKind = 0
			return
		}
	}
	if ov, ok := termOverrides[strings.ToLower(text)]; ok {
		if ov == "-" {
			e.write(esc(text))
			e.parenKind = 0
			return
		}
		if strings.HasSuffix(s[:at], ov) {
			sk := strings.ReplaceAll(text, `"`, "&quot;")
			e.line.Reset()
			e.line.WriteString(s[:at-len(ov)] + "<术 " + key + `="` + sk + `">` + ov + "</术>")
			e.dropParen = true
			e.parenKind = 0
			return
		}
	}
	start := sanskritStart(s, at)
	if start < 0 || strings.ContainsAny(s[start:at], "<>") {
		e.write(esc(text))
		e.parenKind = 0
		return
	}
	// <lb> 把漢詞切成兩行：當行只找得到行首的一兩個字（start==0），
	// 回頭看上一行尾端把詞接起來（作者 2026-10-31）。詞數不變，只是位置改變。
	if start == 0 && e.crossLineTerm(s, at, start, key, text) {
		return
	}
	if hanParenBefore(s, at) || hanParenEndingAt(s, start) {
		e.write(esc(text))
		e.parenKind = 0
		return
	}
	seg := s[start:at]
	head, tail, core := "", "", seg
	if strings.HasPrefix(seg, "「") && strings.HasSuffix(seg, "」") && len(seg) > len("「」") {
		head, tail = "「", "」"
		core = seg[len("「") : len(seg)-len("」")]
	}
	if strings.HasSuffix(seg, "》") && len(seg) > len("》") {
		tail = "》"
		core = seg[:len(seg)-len("》")]
	}
	core = hanSuffix(core)
	if core == "" || !hanOnlyRe.MatchString(core) {
		e.write(esc(text))
		e.parenKind = 0
		return
	}
	// 梵語有連字號（複合詞）時，漢詞不可能只有 1 字：往左多吃幾個漢字；
	// 若左邊被標籤擋住（八支聖<註/>道），寧可整條不包，也不要留錯的單字（2026-10-31）。
	if len([]rune(core)) == 1 && strings.Contains(text, "-") {
		rs := []rune(s[:start])
		k := len(rs)
		for k > 0 && len(rs)-k < 3 && unicode.Is(unicode.Han, rs[k-1]) {
			k--
		}
		if k < len(rs) {
			core = string(rs[k:]) + core
			start = len(string(rs[:k]))
			head = ""
		}
		if len([]rune(core)) == 1 {
			e.write(esc(text))
			e.parenKind = 0
			return
		}
	}
	if len([]rune(core)) > 8 {
		e.write(esc(text))
		e.parenKind = 0
		return
	}
	sk := strings.ReplaceAll(text, `"`, "&quot;")
	e.line.Reset()
	e.line.WriteString(s[:start] + head + "<术 " + key + `="` + sk + `">` + core + "</术>" + tail)
	e.dropParen = true
	e.parenKind = 0
}

// ---------------------------------------------------------------- 術語跨行
// 作者 2026-10-31：<lb> 會把「須彌山（Sumeru）」切成
//   …即梵語須彌
//   山（Sumeru）的音譯
// 這時當行只找得到「山」。回頭看上一行尾端的漢字（退到停止字），
// 把 <术 …> 插到上一行、</术> 收在當行，整份文件的標籤仍然平衡。

func isHanRune(r rune) bool {
	return (r >= 0x3400 && r <= 0x9fff) || (r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0x20000 && r <= 0x2fa1f)
}

/** 行尾可當詞尾的漢字起點（byte index）；沒有可用的尾巴就回 -1。 */
func hanTailRun(s string) int {
	rs := []rune(s)
	j := len(rs) - 1
	for j >= 0 && isHanRune(rs[j]) && !strings.ContainsRune(termStops, rs[j]) {
		j--
	}
	run := rs[j+1:]
	if len(run) == 0 || len(run) > 6 {
		return -1
	}
	return len(string(rs[:j+1]))
}

// 漢詞前面的語言提示不算詞的一部分（梵語須彌山 → 須彌山）
var termMetaPrefix = []string{"梵語", "巴利語", "藏語", "此言", "梵稱", "譯曰", "梵文"}

func trimTermMeta(s string) string {
	for _, p := range termMetaPrefix {
		if strings.HasPrefix(s, p) && len([]rune(s)) > len([]rune(p)) {
			s = strings.TrimPrefix(s, p)
		}
	}
	return s
}

/**
 * <lb> 切斷的跨行術語：把開標籤插到上一行（e.pending）的尾端，收在當行。
 * 只有「當行開頭就是詞」才處理；上一行尾端要是 1–6 個漢字、兩行合計 ≤ 8 字。
 */
func (e *emitter) crossLineTerm(s string, at, start int, key, text string) bool {
	if !e.hasPending || at < 0 {
		return false
	}
	prev := e.pending
	runStart := hanTailRun(prev)
	if runStart < 0 {
		return false
	}
	run := trimTermMeta(prev[runStart:])
	if run == "" || len([]rune(run)) > 6 || strings.ContainsAny(run, "<>") {
		return false
	}
	cand := ""
	if at > 0 {
		if start != 0 {
			return false
		}
		cand = s[:at]
		if cand == "" || strings.ContainsAny(cand, "<>") {
			return false
		}
		for _, r := range cand {
			if !isHanRune(r) {
				return false
			}
		}
	}
	if len([]rune(run+cand)) > 8 {
		return false
	}
	sk := strings.ReplaceAll(text, `"`, "&quot;")
	cut := len(prev) - len(run)
	e.pending = prev[:cut] + "<术 " + key + `="` + sk + `">` + run
	e.line.Reset()
	e.line.WriteString(cand + "</术>")
	e.dropParen = true
	e.parenKind = 0
	e.crossClosed = true
	return true
}

/** 把一行裡的「漢詞（skt）」改成 <term sk="skt">漢詞</term>；由右往左做。 */
func wrapSanskrit(s string) string {
	if !strings.ContainsAny(s, "（(") || strings.Contains(s, "<术 ") {
		return s
	}
	ms := sktParen.FindAllStringSubmatchIndex(s, -1)
	for i := len(ms) - 1; i >= 0; i-- {
		a, b := ms[i][0], ms[i][1]
		cs, ce := ms[i][2], ms[i][3] // 括號內的梵語
		start := sanskritStart(s, a)
		if start < 0 {
			continue
		}
		if hanParenBefore(s, a) || hanParenEndingAt(s, start) {
			continue
		}
		seg := s[start:a]
		if strings.ContainsAny(seg, "<>") {
			continue
		}
		sk := strings.ReplaceAll(s[cs:ce], `"`, "&quot;")
		if ov, ok := termOverrides[strings.ToLower(s[cs:ce])]; ok {
			if ov == "-" {
				continue
			}
			if strings.HasSuffix(s[:a], ov) {
				start = a - len(ov)
				s = s[:start] + `<术 sk="` + strings.ReplaceAll(s[cs:ce], `"`, "&quot;") + `">` + ov + "</术>" + s[b:]
				continue
			}
		}
		head, tail, core := "", "", seg
		if strings.HasPrefix(seg, "「") && strings.HasSuffix(seg, "」") && len(seg) > len("「」") {
			head, tail = "「", "」"
			core = seg[len("「") : len(seg)-len("」")]
		}
		if strings.HasSuffix(seg, "》") && len(seg) > len("》") {
			tail = "》"
			core = seg[:len(seg)-len("》")]
		}
		core = hanSuffix(core)
		if core == "" || !hanOnlyRe.MatchString(core) || len([]rune(core)) > 8 {
			continue
		}
		if len([]rune(core)) == 1 && strings.Contains(s[cs:ce], "-") {
			rs := []rune(s[:start])
			k := len(rs)
			for k > 0 && len(rs)-k < 3 && unicode.Is(unicode.Han, rs[k-1]) {
				k--
			}
			if k < len(rs) {
				core = string(rs[k:]) + core
				start = len(string(rs[:k]))
				head = ""
			}
			if len([]rune(core)) == 1 {
				continue
			}
		}
		s = s[:start] + head + `<术 sk="` + sk + `">` + core + "</术>" + tail + s[b:]
	}
	return s
}
// 術語自動校正（2026-10-31 作者）：像「如龍樹（Nāgārjuna）」這種，前面多吃了虛詞。
// 若同一個外語值在別處有「去掉首字」的版本，就以那個為準（逐步修剪，最多 2 字）。
var termTagRe = regexp.MustCompile(`<术 (sk|pi|tb)="([^"]*)">([^<]*)</术>`)
var hanOnlyRe = regexp.MustCompile(`^[\p{Han}]+$`)

// 「盧遮（舍）那（Rocana）」這種：詞中間夾了一個漢字括號注。作者 2026-10-31 決定跳過
// （不猜邊界），寧可少包一條，也不要把「那」單獨包起來。
// 詞界停在「）」時，若那個括號裡是漢字（盧遮（舍）那），就整條跳過。
func hanParenEndingAt(s string, pos int) bool {
	if pos <= 0 || !strings.HasSuffix(s[:pos], "）") {
		return false
	}
	open := strings.LastIndex(s[:pos], "（")
	if open < 0 {
		return false
	}
	inner := s[open+len("（") : pos-len("）")]
	if inner == "" || len([]rune(inner)) > 3 {
		return false
	}
	// 括號前面必須緊接漢字，才是「詞中間的注」（盧遮（舍）那）；
	// （一）（二）這種列舉前面是標點，不算。
	if open == 0 || !hanOnlyRe.MatchString(string([]rune(s[:open])[len([]rune(s[:open]))-1])) {
		return false
	}
	return hanOnlyRe.MatchString(inner)
}

func hanParenBefore(s string, at int) bool {
	if at <= 0 || !strings.HasSuffix(s[:at], "）") {
		return false
	}
	open := strings.LastIndex(s[:at], "（")
	if open < 0 || open+len("（") >= at-len("）") {
		return false
	}
	inner := s[open+len("（") : at-len("）")]
	if inner == "" || len([]rune(inner)) > 3 {
		return false
	}
	return hanOnlyRe.MatchString(inner)
}

// 術語只能是漢字（2026-10-31 作者）：把候選字串左邊的數字、箭頭、表格符號去掉，
// 只留緊貼括號的那一段連續漢字。
func hanSuffix(s string) string {
	rs := []rune(s)
	i := len(rs)
	for i > 0 && unicode.Is(unicode.Han, rs[i-1]) {
		i--
	}
	return string(rs[i:])
}
// 術語前綴黑名單（2026-10-31 作者）：朝代名與虛詞不屬於術語本體。
// 只剝前綴，不動詞身；如來／如是／由旬這種本身是詞的例外不動。
var termStripPrefixes = []string{
	"東晉", "西晉", "劉宋", "蕭齊", "蕭梁", "北魏", "東魏", "西魏", "北齊", "北周",
	"後漢", "後秦", "前秦", "姚秦", "北涼", "南涼", "西涼", "後涼", "前涼", "高齊",
	"名為", "名曰", "所謂", "即是", "就是", "所以", "因此", "因為", "由於", "於是",
	"如果", "如今", "也是", "但是", "作者", "譯者", "論師", "佛化", "有作", "或作", "亦作", "譯作",
	"唐", "宋", "隋", "元", "明", "清", "漢", "魏", "吳", "晉", "秦", "周", "齊", "梁", "陳", "燕", "趙", "楚", "越",
	"如", "從", "由", "與", "於", "但", "故", "然", "則", "乃", "雖", "亦", "復", "又", "且", "況",
}

var termKeepPrefixes = []string{
	"如來", "如是", "如實", "如理", "如幻", "如夢", "如響", "如化", "如炎", "如影", "如鏡", "如空", "如所有", "由旬",
}

func trimTermPrefix(term string) string {
	for pass := 0; pass < 3; pass++ {
		changed := false
		for _, keep := range termKeepPrefixes {
			if strings.HasPrefix(term, keep) {
				return term
			}
		}
		for _, p := range termStripPrefixes {
			if !strings.HasPrefix(term, p) {
				continue
			}
			rest := []rune(term)[len([]rune(p)):]
			min := 2
			if len([]rune(p)) >= 2 {
				min = 1
			}
			if len(rest) < min {
				continue
			}
			term = string(rest)
			changed = true
			break
		}
		if !changed {
			break
		}
	}
	return term
}
const junkRunes = "的是此與今即而則也都就才只還更最不沒及或以因於由在從對向但故然乃雖亦復又且況為有上等之唐宋隋元明清漢魏吳晉秦周齊梁陳"

func isJunkRune(r rune) bool { return strings.ContainsRune(junkRunes, r) }

// 作者手動指定的術語（data/term-overrides.tsv）：foreign → 漢詞。
// 用於反查、頻率規則都救不回來的個案。
var termOverrides = loadTermOverrides()

func loadTermOverrides() map[string]string {
	path := os.Getenv("YS_TERM_OVERRIDES")
	if path == "" {
		path = "data/term-overrides.tsv"
	}
	out := map[string]string{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "foreign") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 || f[0] == "" || f[1] == "" {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(f[0]))] = strings.TrimSpace(f[1])
	}
	return out
}// 術語前面常黏著句子成分（佛為、東南有、大夏又為、除南印度以…）。
// 規則：從詞裡找「最後一個虛詞」，切在它後面；虛詞不能在詞首（保護有為、有所說法）。
func junkLastCut(term string) string {
	junk := junkRunes
	rs := []rune(term)
	cut := -1
	for i := 1; i < len(rs); i++ {
		if strings.ContainsRune(junk, rs[i]) {
			cut = i
		}
	}
	if cut >= 0 && len(rs)-cut-1 >= 2 {
		return string(rs[cut+1:])
	}
	return term
}
// 手動補丁（作者 2026-10-31）：CBETA 把梵語夾在詞中間，例
// 「又全力爭取／薩寒若（Sahajāti）國離婆多（Revata）」——薩寒若國是國名、離婆多是
// 人名，parser 目前切錯。實例不多，直接改這兩段。
var manualTermSpans = [][2]string{
	// 作者 2026-10-31：薩寒若＝Sahajāti 即可（國留在外面），離婆多＝Revata。
	{
		"<术 sk=\"Sahajāti\">又全力爭取\n薩寒若</术><术 sk=\"Revata\">國離婆多</术>",
		"又全力爭取\n<术 sk=\"Sahajāti\">薩寒若</术>國<术 sk=\"Revata\">離婆多</术>",
	},
}

func fixManualTermSpans(s string) string {
	for _, pair := range manualTermSpans {
		s = strings.ReplaceAll(s, pair[0], pair[1])
	}
	return s
}

// 跨行的書名號術語（《往\n世書》（Purāṇa））：最後補一道，因為它跨 <lb> 又跨《》。
var crossBookRe = regexp.MustCompile("《([\\p{Han}]{1,4})\n([\\p{Han}]{1,4})》（([\\p{Latin}][^）]*)）")
var splitBookRe = regexp.MustCompile("《\n([\\p{Han}]{1,6})》（([\\p{Latin}][^）]*)）")

// 手動補丁（作者 2026-10-31）：六（內）處被 <頁> 與空白行切開，parser 不動，這裡直接補。
var chayatanRe = regexp.MustCompile(`六(<頁 [^>]*/>)\n\n（內）<术 (sk="[^"]*")>處</术>`)

func fixChayatan(s string) string {
	return chayatanRe.ReplaceAllString(s, "$1<术 $2>六（內）處</术>\n\n")
}

func fixCrossBookTitles(s string) string {
	s = crossBookRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := crossBookRe.FindStringSubmatch(m)
		return "《<术 sk=\"" + sub[3] + "\">" + sub[1] + "\n" + sub[2] + "</术>》"
	})
	return splitBookRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := splitBookRe.FindStringSubmatch(m)
		return "《\n<术 sk=\"" + sub[2] + "\">" + sub[1] + "</术>》"
	})
}

func fixTermLeads(s string) string {
	const funcs = "如此與今即而則也都就才只還更最不沒及或的是以因於由"
	funcs2 := map[string]bool{"所以": true, "如此": true, "因此": true, "因為": true, "由於": true, "於是": true, "如果": true, "如今": true, "就是": true, "也是": true, "但是": true}
	raw := termTagRe.FindAllStringSubmatchIndex(s, -1)
	ms := make([][]int, 0, len(raw))
	for _, m := range raw {
		term := s[m[6]:m[7]]
		// 跨行標籤（詞被 <lb> 切斷）不能改寫：改寫會吃掉換行、少一行
		if strings.Contains(term, "\n") || !hanOnlyRe.MatchString(term) {
			continue
		}
		ms = append(ms, m)
	}
	if len(ms) == 0 {
		return s
	}
	// 反查：同一組（語言＋外語）的漢詞應該一致；取「出現最多那個詞的最長後綴，
	// 且該後綴佔比 ≥80%」為正解，其他變體（如龍樹、唐實叉難陀）改寫成正解。
	type gcount struct {
		terms map[string]int
		total int
	}
	groups := map[string]*gcount{}
	for _, m := range ms {
		key := s[m[2]:m[3]] + "\x00" + s[m[4]:m[5]]
		g := groups[key]
		if g == nil {
			g = &gcount{terms: map[string]int{}}
			groups[key] = g
		}
		g.terms[s[m[6]:m[7]]]++
		g.total++
	}
	canon := map[string]string{}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		g := groups[key]
		best, bc := "", -1
		terms := make([]string, 0, len(g.terms))
		for t := range g.terms {
			terms = append(terms, t)
		}
		sort.Strings(terms)
		for _, t := range terms {
			c := g.terms[t]
			if c > bc || (c == bc && len([]rune(t)) > len([]rune(best))) {
				best, bc = t, c
			}
		}
		if cand := junkLastCut(best); len([]rune(cand)) >= 2 {
			canon[key] = cand
		} else if len([]rune(best)) >= 2 {
			canon[key] = best
		}
	}
	// 只出現一次的術語（反查幫不上）：用語料頻率找「去掉 1–3 字前綴」後仍常見的後綴。
	// 例：但大寺派→大寺派（21 次）、島上有大腹→大腹（4 次）。
	cand := map[string]bool{}
	type single struct{ key, term string }
	var singles []single
	for _, key := range keys {
		g := groups[key]
		terms := make([]string, 0, len(g.terms))
		for t := range g.terms {
			terms = append(terms, t)
		}
		sort.Strings(terms)
		for _, t := range terms {
			c := g.terms[t]
			if c != 1 {
				continue
			}
			rs := []rune(t)
			for L := 2; L <= 4 && L <= len(rs)-1; L++ {
				if len(rs)-L <= 3 {
					cand[string(rs[len(rs)-L:])] = true
				}
			}
			singles = append(singles, single{key, t})
		}
	}
	if len(cand) > 0 {
		rs := []rune(s)
		cnt := make(map[string]int, len(cand))
		for i := 0; i < len(rs); i++ {
			for L := 2; L <= 4 && i+L <= len(rs); L++ {
				w := string(rs[i : i+L])
				if cand[w] {
					cnt[w]++
				}
			}
		}
		for _, sg := range singles {
			rs := []rune(sg.term)
			best := ""
			for L := 2; L <= 4 && L <= len(rs)-1; L++ {
				suf := string(rs[len(rs)-L:])
				if cnt[suf] >= 3 {
					best = suf
				}
			}
			if best != "" && best != sg.term {
				canon[sg.key] = best
			}
		}
	}
	known := map[string]bool{}
	for _, m := range ms {
		known[s[m[2]:m[3]]+"\x00"+s[m[4]:m[5]]+"\x00"+s[m[6]:m[7]]] = true
	}
	// 由右往左改，索引才不會亂
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		lang, foreign := s[m[2]:m[3]], s[m[4]:m[5]]
		if ov, ok := termOverrides[strings.ToLower(foreign)]; ok && ov != "-" {
			term := s[m[6]:m[7]]
			if term != ov && strings.HasSuffix(ov, term) {
				need := ov[:len(ov)-len(term)]
				if need != "" && !strings.Contains(need, "\n") {
					j := m[0]
					for j > 0 && s[j-1] == '\n' {
						j--
					}
					// 允許中間夾 <頁/>、<註/>（例：見燃燈<頁/>佛）
					for j > 0 && s[j-1] == '>' {
						lt := strings.LastIndex(s[:j], "<")
						if lt < 0 {
							break
						}
						tg := s[lt:j]
						if !strings.HasPrefix(tg, "<頁") && !strings.HasPrefix(tg, "<註") {
							break
						}
						j = lt
					}
					if j >= len(need) && strings.HasSuffix(s[:j], need) && !strings.ContainsAny(s[j-len(need):j], "<>") {
						s = s[:j-len(need)] + s[j:m[6]] + ov + s[m[7]:]
						continue
					}
				}
			}
		}
		trimmed := junkLastCut(trimTermPrefix(s[m[6]:m[7]]))

		// 作者 2026-10-31 定案：只出現 1 次的長詞，若同組有常見的短後綴（≥3 次）就以短後綴為正解。
		// 例外：去掉的前綴是 無／不／非（梵語 a- 前綴的否定）時不縮，例「無相心三昧」。
		if g := groups[lang+"\x00"+foreign]; g != nil && g.terms[s[m[6]:m[7]]] == 1 {
			orig := s[m[6]:m[7]]
			rs := []rune(orig)
			best := ""
			for i := 1; i < len(rs); i++ {
				suf := string(rs[i:])
				if len([]rune(suf)) < 2 {
					break
				}
				if c := g.terms[suf]; c >= 3 {
					if best == "" || c > g.terms[best] || (c == g.terms[best] && len([]rune(suf)) > len([]rune(best))) {
						best = suf
					}
				}
			}
			if best != "" {
				pre := string(rs[:len(rs)-len([]rune(best))])
				if !strings.HasPrefix(pre, "無") && !strings.HasPrefix(pre, "不") && !strings.HasPrefix(pre, "非") {
					trimmed = best
				}
			}
		}
		if c := canon[lang+"\x00"+foreign]; c != "" && strings.HasSuffix(trimmed, c) && len(trimmed) > len(c) {
			// 只有「正解前面緊接著虛詞」才允許縮短，否則會把大梵天切成梵天、盧舍那切成那
			pre := []rune(trimmed[:len(trimmed)-len(c)])
			if len(pre) > 0 && isJunkRune(pre[len(pre)-1]) {
				trimmed = c
			}
		}
		rs := []rune(trimmed)
		changed := trimmed != s[m[6]:m[7]]
		// 先試兩字的虛詞（所以、如此、因為…），再試單字
		for len(rs) >= 4 {
			two := string(rs[:2])
			if !funcs2[two] || !known[lang+"\x00"+foreign+"\x00"+string(rs[2:])] {
				break
			}
			rs = rs[2:]
			changed = true
		}
		for len(rs) >= 3 && strings.ContainsRune(funcs, rs[0]) {
			if !known[lang+"\x00"+foreign+"\x00"+string(rs[1:])] {
				break
			}
			rs = rs[1:]
			changed = true
		}
		if changed {
			// 縮短術語時，被去掉的前綴要留在標籤外面（當正文），不能跟著標籤一起消失：
			// 「等五佛為五禪那佛」→ 等五佛為<术 …>五禪那佛</术>
			newTerm := string(rs)
			orig := s[m[6]:m[7]]
			prefix := ""
			if strings.HasSuffix(orig, newTerm) {
				prefix = orig[:len(orig)-len(newTerm)]
			} else {
				// 手動指定時新舊詞可能只重疊一部分（在修摩 → 修摩國）：保留不重疊的前綴
				ro, rn := []rune(orig), []rune(newTerm)
				k := 0
				for kk := 1; kk <= len(ro) && kk < len(rn); kk++ {
					if string(ro[len(ro)-kk:]) == string(rn[:kk]) {
						k = kk
					}
				}
				prefix = string(ro[:len(ro)-k])
			}
			s = s[:m[0]] + prefix + s[m[0]:m[6]] + newTerm + s[m[7]:m[1]] + s[m[1]:]
		}
	}
	s = fixCrossLineTerms(s)
	return s
}

/**
 * 跨行術語（被 <lb> 切斷）的邊界修正（作者 2026-10-31）：
 * 上一行尾端常夾帶一兩個字（「佛世摩竭／陀」的「佛世」），合起來雖然 ≤8 字，
 * 卻不是術語。用同一組（語言＋外語）的**單行**術語當正解：若正解是這個跨行詞的
 * 後綴，就把多吃的 1–3 字留在上一行（標籤往右移），換行本身不動、行數不變。
 * 例：佛世摩竭|陀（Magadha）→ 摩竭|陀；有罽|賓（Kaśpira）→ 罽|賓。
 */
func fixCrossLineTerms(s string) string {
	re := regexp.MustCompile(`<术 (sk|pi|tb)="([^"]*)">([^<]*)</术>`)
	ms := re.FindAllStringSubmatchIndex(s, -1)
	if len(ms) == 0 {
		return s
	}
	single := map[string]bool{}
	groups := map[string]map[string]int{}
	for _, m := range ms {
		t := s[m[6]:m[7]]
		if strings.Contains(t, "\n") {
			continue
		}
		k := s[m[2]:m[3]] + "\x00" + s[m[4]:m[5]]
		single[k+"\x00"+t] = true
		if groups[k] == nil {
			groups[k] = map[string]int{}
		}
		groups[k][t]++
	}
	best := map[string]string{}
	for k, g := range groups {
		bt, bc := "", -1
		for t, c := range g {
			if c > bc || (c == bc && len([]rune(t)) > len([]rune(bt))) {
				bt, bc = t, c
			}
		}
		best[k] = bt
	}
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		term := s[m[6]:m[7]]
		if !strings.Contains(term, "\n") {
			continue
		}
		lang, foreign := s[m[2]:m[3]], s[m[4]:m[5]]
		c := best[lang+"\x00"+foreign]
		if c == "" || len([]rune(c)) < 2 {
			continue
		}
		flat := strings.Replace(term, "\n", "", 1)
		if single[lang+"\x00"+foreign+"\x00"+flat] {
			continue // 這個長詞本身也在單行出現過：它可能真的是長詞，不動
		}
		cr, fr := []rune(c), []rune(flat)
		if len(cr) >= len(fr) || string(fr[len(fr)-len(cr):]) != c || len(fr)-len(cr) > 3 {
			continue
		}
		head, tail, ok := strings.Cut(term, "\n")
		if !ok {
			continue
		}
		hr, tr := []rune(head), []rune(tail)
		if len(cr) > len(tr) {
			k := len(cr) - len(tr)
			if k > len(hr) || string(hr[len(hr)-k:]) != string(cr[:k]) {
				continue
			}
			newTerm := string(cr[:k]) + "\n" + tail
			prefix := string(hr[:len(hr)-k])
			s = s[:m[0]] + prefix + s[m[0]:m[6]] + newTerm + s[m[7]:]
		} else {
			// 正解整個在下一行：開標籤移到換行之後
			s = s[:m[0]] + head + "\n" + s[m[0]:m[6]] + c + s[m[7]:]
		}
	}
	return s
}
func (e *emitter) nl() {
	if e.capture != nil {
		// 表格 cell 內不換行（硬規則：一列一行，cell 不能換行）
		e.capture.WriteString(" ")
		return
	}
	var s string
	if e.crossClosed {
		// 跨行術語的收尾行：舊行為是「含 <术 的行跳過 wrapSanskrit」，
		// 這裡也跳過，全文 <术> 的數量才不會改變（2026-10-31）
		e.crossClosed = false
		s = e.line.String()
	} else {
		s = wrapSanskrit(e.line.String())
	}
	// 第一個標題若與書名完全相同，就不要多開一個目錄節點（書已經顯示書名了）
	if !e.titleDropped && e.headTag != "" && e.bookTitle != "" {
		if s == "<"+e.headTag+` u="文">`+esc(e.bookTitle)+"</"+e.headTag+">" {
			s = esc(e.bookTitle)
			e.titleDropped = true
		}
	}
	e.line.Reset()
	// 上一行延後一行才寫出：<术> 跨行時要能回頭插入開標籤（作者 2026-10-31）
	if e.hasPending {
		e.out.WriteString(e.pending)
		e.out.WriteByte('\n')
	}
	e.pending = s
	e.hasPending = true
	e.lineNo++
}

/** 把延後的最後一行寫出（程式結束前一定要呼叫） */
func (e *emitter) flush() {
	if e.hasPending {
		e.out.WriteString(e.pending)
		e.out.WriteByte('\n')
		e.pending = ""
		e.hasPending = false
	}
}

func (e *emitter) write(s string) {
	if e.capture != nil {
		e.capture.WriteString(s)
		return
	}
	e.line.WriteString(s)
}

// ---------------------------------------------------------------- 腳註

var refTargetRe = regexp.MustCompile(`#vol:(\d+);page:p([0-9a-zA-Z]+)`)

func taishoFromTarget(target string) string {
	m := refTargetRe.FindStringSubmatch(target)
	if m == nil {
		return ""
	}
	return "taisho?" + m[1] + "p" + m[2]
}

// 注文的空白：CBETA 在標籤之間有換行與縮排，中文不該因此多出空格，
// 只有兩邊都是 ASCII（英文、數字）時才留一個空格。
func tidyNote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if c == ' ' {
			var p, n rune
			if i > 0 {
				p = r[i-1]
			}
			if i+1 < len(r) {
				n = r[i+1]
			}
			if p > 0 && p < 128 && n > 0 && n < 128 {
				b.WriteRune(c)
			}
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

// 讀紙本註號對照表：`冊\tn\t冊.書內註`（tools/build-note-map.py 產生）。
func loadNoteNumbers(path string) map[string]string {
	out := map[string]string{}
	if path == "" {
		return out
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Split(line, "\t")
		if len(p) >= 3 {
			// 表裡是「冊.章.註」，輸出改成「章.註@冊」：顯示只給章.註，@冊 用來 unique
			v := strings.TrimSpace(p[2])
			parts := strings.Split(v, ".")
			if len(parts) == 3 {
				v = parts[1] + "." + parts[2] + "@" + parts[0]
			}
			out[p[0]+"\t"+p[1]] = v
		}
	}
	return out
}

// CBETA <back> 裡 type="yinshun-notes" 的 <note>：印順自己的腳註。
// 注文轉成 Accelon3 的形式：<ref target="#vol:12;page:p1081a">…</ref>
// → <聯 i="taisho?12p1081a">…</聯>，一個引用一條聯。
func parseYinshunNotes(raw []byte) map[string]string {
	out := map[string]string{}
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	inNotes, inNote := false, false
	id := ""
	var refOpen []bool
	var cur strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "div":
				if attr(t.Attr, "type") == "yinshun-notes" {
					inNotes = true
				}
			case "note":
				if inNotes {
					id = attr(t.Attr, "n")
					inNote = id != ""
					cur.Reset()
					refOpen = refOpen[:0]
				}
			case "ref":
				if inNote {
					i := taishoFromTarget(attr(t.Attr, "target"))
					if i != "" {
						cur.WriteString(`<聯 i="` + i + `">`)
					}
					refOpen = append(refOpen, i != "")
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "div":
				inNotes = false
			case "note":
				if inNote {
					out[id] = strings.TrimSpace(cur.String())
				}
				inNote, id = false, ""
			case "ref":
				if inNote && len(refOpen) > 0 {
					if refOpen[len(refOpen)-1] {
						cur.WriteString("</聯>")
					}
					refOpen = refOpen[:len(refOpen)-1]
				}
			}
		case xml.CharData:
			if inNote {
				cur.WriteString(tidyNote(string(t)))
			}
		}
	}
	return out
}

// 正文出現 <anchor n="…"/>：寫 <註 n="…"/>，並記下這段要輸出的注文
func (e *emitter) markNote(n string) {
	short, ok := e.noteShort[n]
	if !ok {
		if m, hit := e.noteNum[fmt.Sprintf("%d\t%s", e.vol, n)]; hit {
			short = m
		} else {
			e.noteSeq++
			short = fmt.Sprintf("0.%03d@%d", e.noteSeq, e.vol)
		}
		e.noteShort[n] = short
	}
	e.write(`<註 n="` + short + `"/>`)
	e.stats["註"]++
	e.addPending(n)
}

// 同一條註在正文可能被引用多次，只排一次
func (e *emitter) addPending(n string) {
	for _, x := range e.notePend {
		if x == n {
			return
		}
	}
	e.notePend = append(e.notePend, n)
}

// 書末（</body>）：把這本書引用到的注文依序寫成 <段/><釋 n="…">…</釋>，
// 位置照 CBETA <back> 的 yinshun-notes（集中書末），不再散在每段後面。
func (e *emitter) flushNotes() {
	for _, n := range e.notePend {
		body, ok := e.notes[n]
		if !ok {
			continue
		}
		e.nl()
		e.write(`<段/><釋 n="` + e.noteShort[n] + `">` + body + `</釋>`)
		e.stats["釋"]++
		e.nl()
	}
	e.notePend = e.notePend[:0]
}

/**
 * 開一個標題。層級來自前面的 <cb:mulu level="N">，對到 tierNames：
 *   1 章、2 節、3 項、4 干、5 一、6 數、7 Ａ、8 Ｂ …
 * 目錄樹的層級鏈寫在 <書 t="章,節,項,干,一,數,Ａ.">，標題標籤本身不再帶 `t=`。
 * 2026-10-29：先前只對到「項」，level 5 以後全落到 <章>，於是「正釋」和
 * 「甲 阿賴耶教」在目錄裡同一層（攝大乘論講記）。這裡改成一路排到 18 層。
 */
func (e *emitter) openHeading() {
	tag := "章"
	if e.muluLevel >= 1 && e.muluLevel <= len(tierNames) {
		tag = tierNames[e.muluLevel-1]
	}
	e.headTag = tag
	e.write("<" + tag + ` u="文">`)
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// parseUPlus：U+35D8 → rune
func parseUPlus(v string) (rune, bool) {
	if !strings.HasPrefix(v, "U+") {
		return 0, false
	}
	n, err := strconv.ParseInt(v[2:], 16, 32)
	if err != nil || n <= 0 || n > 0x10FFFF {
		return 0, false
	}
	return rune(n), true
}

/** </g>：內文只有 1 字就用它；否則查 charDecl 的 Unicode；都沒有就留 &CODE; 並記錄。 */
func (e *emitter) endGaiji() {
	text := strings.TrimSpace(e.gaijiBuf.String())
	ref := e.gaijiRef
	e.inGaiji = false
	e.gaijiBuf.Reset()
	out := ""
	if len([]rune(text)) == 1 {
		out = text
	} else if u := e.gaiji[ref]; u != "" {
		if r, ok := parseUPlus(u); ok {
			out = string(r)
		}
	}
	if out == "" {
		out = "&" + ref + ";"
		e.gaijiMiss++
		e.gaijiMissing = append(e.gaijiMissing, e.curFile+"\t"+ref+"\t"+text)
	} else {
		e.gaijiOK++
	}
	if e.noteKey != "" {
		e.noteBuf.WriteString(out)
		return
	}
	e.write(esc(out))
}

func clean(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.Join(f, " ")
}

// ---------------------------------------------------------------- 表格

// 這一頁在 Y-images 的表格圖（沒有就空字串）。
func (e *emitter) tableImage() string {
	if e.tblPage == "" || len(e.tblPage) < 4 {
		return ""
	}
	dir := fmt.Sprintf("Y%02d", e.vol)
	pat := filepath.Join(e.imgDir, "svg", dir, fmt.Sprintf("Y%02dp%s_*.svg", e.vol, e.tblPage[:4]))
	hits, _ := filepath.Glob(pat)
	if len(hits) == 0 {
		return ""
	}
	return "svg/" + dir + "/" + filepath.Base(hits[0])
}

func pipeRow(cells []string) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = strings.ReplaceAll(c, "|", `\|`)
	}
	return "| " + strings.Join(parts, " | ") + " |"
}

func headerSep(cols int) string {
	var b strings.Builder
	for i := 0; i < cols; i++ {
		b.WriteString("| --- ")
	}
	b.WriteString("|")
	return b.String()
}

// 把緩衝的表格輸出成 <表>（每列一行）；表達不了就退回頁面圖版，沒有圖才用文字列。
func (e *emitter) emitTable() {
	rows := e.tblRows
	img := e.tableImage()
	ok := !e.tblBroken
	cols := 0
	dataRows := 0
	if ok {
		for _, r := range rows {
			if len(r) == 1 && strings.HasPrefix(r[0], "\x00page:") {
				continue
			}
			dataRows++
			if cols == 0 {
				cols = len(r)
			}
			if len(r) != cols || cols == 0 {
				ok = false
				break
			}
		}
	}
	// 閱讀器的 <表> 解析器要求 ≥2 欄、≥2 列（才能有表頭＋表身）
	if cols < 2 || dataRows < 2 {
		ok = false
	}
	writePage := func(r []string) {
		e.nl()
		e.write(`<頁 id="` + strings.TrimPrefix(r[0], "\x00page:") + `"/>`)
	}
	// 表格表達不了：留原圖（如果有），頁碼照樣輸出，其餘用文字列
	if !ok {
		if img != "" {
			e.nl()
			e.write(`<圖 n="` + img + `" t="圖版"/>`)
			e.stats["表圖"]++
		}
		for _, r := range rows {
			if len(r) == 1 && strings.HasPrefix(r[0], "\x00page:") {
				writePage(r)
				continue
			}
			e.nl()
			e.write(strings.Join(r, "｜"))
		}
		e.nl()
		e.stats["表略"]++
		return
	}
	e.nl()
	e.write("<表>")
	first := true
	for _, r := range rows {
		if len(r) == 1 && strings.HasPrefix(r[0], "\x00page:") {
			writePage(r)
			continue
		}
		e.nl()
		e.write(pipeRow(r))
		if first && len(rows) > 1 {
			e.nl()
			e.write(headerSep(cols))
		}
		first = false
	}
	e.nl()
	e.write("</表>")
	e.nl()
	e.stats["表"]++
}

// 頁 id 的數字不要前導 0（作者 2026-11-06）：0081a → 81a、a001a → a1a。
func stripPageZeros(pb string) string {
	i := strings.IndexFunc(pb, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return pb
	}
	j := i
	for j < len(pb) && pb[j] >= '0' && pb[j] <= '9' {
		j++
	}
	k := i
	for k < j-1 && pb[k] == '0' {
		k++
	}
	return pb[:i] + pb[k:]
}

func pbClass(pb string) string {
	if pb == "" {
		return ""
	}
	if pb[0] >= '0' && pb[0] <= '9' {
		return "#"
	}
	return string(pb[0])
}

func isOld(attrs []xml.Attr) bool {
	for _, a := range attrs {
		if a.Name.Local == "type" && a.Value == "old" {
			return true
		}
	}
	return false
}

func attr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func main() {
	src := flag.String("src", "", "Y 目錄")
	out := flag.String("out", "", "輸出文字檔")
	lbPath := flag.String("lbs", "", "輸出「行號到 lb」對照 TSV")
	img := flag.String("img", "D:/2026/xml-p5/Y-images", "圖檔根目錄（含 svg/YNN/）")
	gaijiOut := flag.String("gaiji", "", "缺字表輸出 TSV（可空）")
	nodesTSV := flag.String("notes", "data/note-numbers.tsv", "紙本註號對照表（可空）")
	flag.Parse()
	files, err := filepath.Glob(filepath.Join(*src, "Y*", "*.xml"))
	if err != nil || len(files) == 0 {
		panic("找不到 XML")
	}
	sort.Strings(files)
	e := &emitter{stats: map[string]int{}, imgDir: *img, gaiji: map[string]string{}}
	if *lbPath != "" {
		f, err := os.Create(*lbPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		e.lbOut = bufio.NewWriterSize(f, 1<<16)
		defer e.lbOut.Flush()
	}
	e.write(header)
	e.nl()
	vol := 0
	for _, path := range files {
		vol++
		e.vol = vol
		raw, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		title := ""
		if m := titleRe.FindSubmatch(raw); m != nil {
			title = string(m[1])
		}
		e.curFile = filepath.Base(path)
	e.write(fmt.Sprintf(`<檔 n="%s">`, filepath.Base(path)))
		e.nl()
		e.write(fmt.Sprintf(`<書 l="目次" t="%s" id="%d">%s</書>`, tierChain, vol, esc(title)))
		e.bookTitle = title
		e.titleDropped = false
		e.stats["書"]++
		e.nl()
		// 先收 <back> 的印順腳註（n → 注文），正文的 <anchor> 才能寫 <註> 並掛 <釋>
		e.notes = parseYinshunNotes(raw)
		e.noteShort = map[string]string{}
	e.noteNum = loadNoteNumbers(*nodesTSV)
		e.noteSeq = 0
		e.notePend = e.notePend[:0]
		dec := xml.NewDecoder(strings.NewReader(string(raw)))
		inBody := false
		skip := 0
		juan := 0
		pageClass := ""
		firstUnit := true
		for {
			tok, err := dec.Token()
			if err != nil {
				break
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if skip > 0 {
					skip++
					continue
				}
				if !inBody {
					switch t.Name.Local {
					case "body":
						inBody = true
					case "char":
						// <char xml:id="CB01121">：缺字宣告
						e.curChar = attr(t.Attr, "id")
					case "mapping":
						// <mapping type="unicode">U+35D8</mapping>（少數 normal_unicode）
						ty := attr(t.Attr, "type")
						e.inMapping = e.curChar != "" && (ty == "unicode" || ty == "normal_unicode")
						e.mapBuf.Reset()
					}
					continue
				}
				if e.inGaiji {
					continue
				}
				switch t.Name.Local {
				case "body":
					inBody = true
				case "note":
					// <note place="inline">：作者 2026-11-02 指正——這是行內的大正／南傳
					// 對照（例：大正二．九二下），不是腳註，直接留在正文。
					if attr(t.Attr, "place") == "inline" {
						e.inlineNote = true
						e.inlineRefOpen = e.inlineRefOpen[:0]
						break
					}
					// CBETA 的行內註（<note place="inline2">…</note>，例如
					// 「方等泥洹經<note>竺法護釋法顯</note>：右一經…」）。
					// 2026-10-30 起轉成真正的腳註：正文寫 <註 n="…"/>，
					// 內容收集起來，flushNotes() 在段末輸出 <釋 n="…">…</釋>，
					// 這樣 hover／點擊／釋文都跟其他註一樣（以前是獨立一行的
					// 純文字「〔註82〕…」，句子會被切斷，也沒有釋文）。
					e.noteSeq++
					key := attr(t.Attr, "n")
					if key == "" {
						key = fmt.Sprintf("inline%d", e.noteSeq)
					}
					short := e.noteShort[key]
					if short == "" {
						short = fmt.Sprintf("0.%03d@%d", e.noteSeq, e.vol)
						e.noteShort[key] = short
					}
					e.write(`<註 n="` + short + `"/>`)
					e.stats["註"]++
					e.addPending(key)
					e.noteKey = key
					e.noteBuf.Reset()
					e.noteRefOpen = e.noteRefOpen[:0]
				case "ref":
					// 行內對照（place="inline"）裡的 <ref>：直接寫 <聯>
					if e.inlineNote {
						i := taishoFromTarget(attr(t.Attr, "target"))
						if i != "" {
							e.write(`<聯 i="` + i + `">`)
						}
						e.inlineRefOpen = append(e.inlineRefOpen, i != "")
						break
					}
					// 行內註裡的 <ref target="#vol:12;page:p1081a">：收成 <聯>
					if e.noteKey != "" {
						i := taishoFromTarget(attr(t.Attr, "target"))
						if i != "" {
							e.noteBuf.WriteString(`<聯 i="` + i + `">`)
						}
						e.noteRefOpen = append(e.noteRefOpen, i != "")
					}
				case "quote":
					// CBETA 的引文（<quote>「…」</quote>）→ <引>：閱讀器用楷體顯示，
					// 和作者自己的敘述分開（2026-10-30）
					if e.noteKey == "" {
						e.write("<引>")
					}
				case "app":
					// 校勘層：只留底本 <lem>，異文 <rdg> 略過
					e.stats["app"]++
				case "rdg":
					skip = 1
				case "anchor":
					// CBETA 的腳註錨點：<anchor n="0037056"/> → <註 n="5.056"/>
					if !e.inHead {
						if n := attr(t.Attr, "n"); n != "" {
							if _, ok := e.notes[n]; ok {
								e.markNote(n)
							}
						}
					}
					skip = 1
				case "foreign":
					// <foreign xml:lang="sa">svabhāva</foreign> → <术 sk="svabhāva">漢詞</术>
					if k := termLang(attr(t.Attr, "lang")); k != "" {
						e.inForeign = true
						e.foreignLang = k
						e.foreignBuf.Reset()
						e.parenKind = 0
						if strings.HasSuffix(e.line.String(), "（") {
							e.parenKind = 1
						} else if strings.HasSuffix(e.line.String(), "(") {
							e.parenKind = 2
						}
					}
				case "g":
					// <g ref="#CB01121">：CBETA 缺字。先記住編號與內文，</g> 再決定輸出。
					e.inGaiji = true
					e.gaijiRef = strings.TrimPrefix(attr(t.Attr, "ref"), "#")
					e.gaijiBuf.Reset()
				case "mulu":
					// 目次層級：1＝章、2＝節、3＝項。文字另有 <head>，這裡只記層級。
					if n := attr(t.Attr, "level"); n != "" {
						e.muluLevel = int(n[0] - '0')
					}
					skip = 1
				case "milestone":
					if attr(t.Attr, "unit") == "juan" && inBody {
						juan++
						e.nl()
						e.write(fmt.Sprintf(`<卷 n="%d">%d</卷>`, juan, juan))
						e.stats["卷"]++
						e.nl()
					}
				case "cell":
					// 表格 cell：內容先收進緩衝，</cell> 再放進這一列（2026-11-06）
					if e.inTable {
						if n := attr(t.Attr, "rows"); n != "" && n != "1" {
							e.tblBroken = true // 合併儲存格 <表> 表達不了 → 改用原圖
						}
						if n := attr(t.Attr, "cols"); n != "" && n != "1" {
							e.tblBroken = true
						}
						e.tblCell = &strings.Builder{}
						e.capture = e.tblCell
					}
				case "table":
					// CBETA 的 <table>：收成 <表> 文字表格（作者 2026-11-06）。
					// 巢狀表格 <表> 表達不了，標記 tblBroken，退回原本的頁面圖版。
					if e.inTable {
						e.tblBroken = true
						break
					}
					e.inTable = true
					e.tblBroken = false
					e.tblRows = nil
					e.tblRow = nil
					e.tblPage = e.page
				case "graphic":
					// <figure><graphic url="../figures/Y/Y08p0052_01.svg"/><figDesc>說明</figDesc></figure>
					// 圖檔放 Y-images/svg/YNN/；沒有檔案就只留 figDesc 的文字。
					base := filepath.Base(attr(t.Attr, "url"))
					m := figureRe.FindStringSubmatch(base)
					if m == nil {
						e.stats["圖略"]++
						break
					}
					if _, err := os.Stat(filepath.Join(e.imgDir, "svg", m[1], base)); err != nil {
						e.stats["圖缺"]++
						break
					}
					e.write(`<圖 n="svg/` + m[1] + `/` + base + `" t="圖版"/>`)
					e.stats["圖"]++
				case "pb":
					if !inBody || isOld(t.Attr) {
						continue
					}
					pb := attr(t.Attr, "n")
					if pb == "" {
						continue
					}
					e.page = pb
					if e.inTable {
						// 表格中間的頁碼：收成 <表> 裡的「原書頁」列（parser 會整列合併）
						if e.capture != nil {
							e.tblBroken = true
						} else {
							e.tblRows = append(e.tblRows, []string{"\x00page:" + fmt.Sprintf("%dp%s", e.vol, stripPageZeros(pb))})
						}
						continue
					}
					cls := pbClass(pb)
					if cls != pageClass {
						e.nl()
						label := "正文"
						if cls != "#" {
							if firstUnit {
								label = "序分頁"
							} else {
								label = "附錄"
							}
						}
						firstUnit = false
						e.write(fmt.Sprintf(`<頁元 n="%s" t="頁.">%s</頁元>`, esc(title), label))
						e.stats["頁元"]++
						e.nl()
						pageClass = cls
					}
					e.write(fmt.Sprintf(`<頁 id="%dp%s"/>`, vol, stripPageZeros(pb)))
					e.stats["頁"]++
					e.nl()
				case "lb":
					if isOld(t.Attr) {
						continue
					}
					if n := attr(t.Attr, "n"); n != "" && e.lbOut != nil {
						fmt.Fprintf(e.lbOut, "%d\t%d\t%s\n", e.vol, e.lineNo, n)
					}
					e.nl()
				case "p":
					e.nl()
					e.write("<段/>")
					e.stats["段"]++
				case "head":
					e.nl()
					e.inHead = true
					e.openHeading()
				case "jhead":
					e.nl()
					e.inHead = true
					e.openHeading()
				}
			case xml.EndElement:
				if t.Name.Local == "mapping" && e.inMapping {
					if v := strings.TrimSpace(e.mapBuf.String()); e.curChar != "" && strings.HasPrefix(v, "U+") {
						e.gaiji[e.curChar] = v
					}
					e.inMapping = false
					continue
				}
				if t.Name.Local == "char" {
					e.curChar = ""
					continue
				}
				if t.Name.Local == "g" && e.inGaiji {
					e.endGaiji()
					continue
				}
				if skip > 0 {
					skip--
					continue
				}
				if !inBody {
					continue
				}
				switch t.Name.Local {
				case "head", "jhead":
					e.inHead = false
					tag := e.headTag
					if tag == "" {
						tag = "章"
					}
					e.write("</" + tag + ">")
					// 第一個標題若就是書名，改成純文字，不要多一個目錄節點
					if !e.titleDropped && e.bookTitle != "" &&
						e.line.String() == "<"+tag+` u="文">`+esc(e.bookTitle)+"</"+tag+">" {
						e.line.Reset()
						e.write(esc(e.bookTitle))
						e.titleDropped = true
					}
					e.headTag = ""
					e.muluLevel = 0
				case "cell":
					if e.inTable && e.capture != nil {
						txt := strings.Join(strings.Fields(e.tblCell.String()), " ")
						e.tblRow = append(e.tblRow, txt)
						e.capture = nil
						e.tblCell = nil
					}
				case "row":
					if e.inTable {
						if len(e.tblRow) > 0 {
							e.tblRows = append(e.tblRows, e.tblRow)
						}
						e.tblRow = nil
						break
					}
					e.nl()
				case "table":
					if e.inTable {
						e.capture = nil
						e.inTable = false
						e.emitTable()
					}
				case "note":
					if e.inlineNote {
						e.inlineNote = false
						e.inlineRefOpen = e.inlineRefOpen[:0]
						break
					}
					if e.noteKey != "" {
						e.notes[e.noteKey] = tidyNote(e.noteBuf.String())
						e.noteKey = ""
						e.noteBuf.Reset()
						e.noteRefOpen = e.noteRefOpen[:0]
					}
				case "ref":
					if e.inlineNote && len(e.inlineRefOpen) > 0 {
						if e.inlineRefOpen[len(e.inlineRefOpen)-1] {
							e.write("</聯>")
						}
						e.inlineRefOpen = e.inlineRefOpen[:len(e.inlineRefOpen)-1]
						break
					}
					if e.noteKey != "" && len(e.noteRefOpen) > 0 {
						if e.noteRefOpen[len(e.noteRefOpen)-1] {
							e.noteBuf.WriteString("</聯>")
						}
						e.noteRefOpen = e.noteRefOpen[:len(e.noteRefOpen)-1]
					}
				case "foreign":
					if e.inForeign {
						e.endForeign()
					}
				case "quote":
					if e.noteKey == "" {
						e.write("</引>")
					}
				case "body":
					e.flushNotes()
					inBody = false
				}
			case xml.CharData:
				if e.inMapping {
					e.mapBuf.WriteString(string(t))
					continue
				}
				if e.inGaiji {
					e.gaijiBuf.WriteString(string(t))
					continue
				}
				if skip > 0 || !inBody {
					continue
				}
				raw := string(t)
				if e.dropParen {
					if strings.HasPrefix(raw, "）") {
						raw = strings.TrimPrefix(raw, "）")
						e.dropParen = false
					} else if strings.HasPrefix(raw, ")") {
						raw = strings.TrimPrefix(raw, ")")
						e.dropParen = false
					}
				}
				s := clean(raw)
				if s == "" {
					continue
				}
				if e.inForeign {
					e.foreignBuf.WriteString(s)
					continue
				}
				if e.noteKey != "" {
					e.noteBuf.WriteString(s)
					continue
				}
				e.write(esc(s))
			}
		}
	}
	if e.line.Len() > 0 {
		e.nl()
	}
	e.flush()
	if err := os.WriteFile(*out, []byte(fixManualTermSpans(fixChayatan(fixCrossBookTitles(fixTermLeads(e.out.String()))))), 0o644); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "lines=%d %v\n", strings.Count(e.out.String(), "\n"), e.stats)
	fmt.Fprintf(os.Stderr, "gaiji=%d missing=%d\n", e.gaijiOK+e.gaijiMiss, e.gaijiMiss)
	if *gaijiOut != "" {
		var b strings.Builder
		b.WriteString("file\tcode\ttext\n")
		for _, m := range e.gaijiMissing {
			b.WriteString(m + "\n")
		}
		if err := os.WriteFile(*gaijiOut, []byte(b.String()), 0o644); err != nil {
			panic(err)
		}
	}
}

