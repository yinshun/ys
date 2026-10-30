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
	"strings"
)

// ys.adb 的文字：檔頭（集/隱/樹/引/摘要頭/參考資料/英文名）與 yinshun.adb 相同，
// 內文用 <檔> <書> <卷> <頁元> <頁> <段> <章>，其餘細節不強求一致。
const header = `<檔 n="ys.xml"><集 l="圖集,註釋群,編目資訊">印順法師佛學著作集</集><隱><樹 s="*" t="書" pf1="$書.id;.">目錄</樹><樹 s="*u=文" pf="$頁.id;)">頻次</樹><樹 s="~00602000">西文</樹></隱><隱><引>《$書;》：「$_;」( Y $*頁; )</引><摘要頭>$書;($頁;)</摘要頭><參考資料 n="taisho,wxzj"/></隱><英文名>Master Yinshun Corpus of Buddhist Studies</英文名></檔>`

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
	// 腳註：CBETA <back> 的 yinshun-notes（印順自己的註），n（CBETA 編號）→ 已轉好的注文
	notes     map[string]string
	noteShort map[string]string // CBETA n → 短編號（<書>.<序>，顯示用）
	notePend  []string          // 這一段出現過的註，等 </p> 一起輸出 <釋>
	inHead    bool
}

func (e *emitter) nl() {
	s := e.line.String()
	// 第一個標題若與書名完全相同，就不要多開一個目錄節點（書已經顯示書名了）
	if !e.titleDropped && e.headTag != "" && e.bookTitle != "" {
		if s == "<"+e.headTag+` u="文">`+esc(e.bookTitle)+"</"+e.headTag+">" {
			s = esc(e.bookTitle)
			e.titleDropped = true
		}
	}
	e.line.Reset()
	e.out.WriteString(s)
	e.out.WriteByte('\n')
	e.lineNo++
}

func (e *emitter) write(s string) { e.line.WriteString(s) }

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
		e.noteSeq++
		short = fmt.Sprintf("%d.%03d", e.vol, e.noteSeq)
		e.noteShort[n] = short
	}
	e.write(`<註 n="` + short + `"/>`)
	e.stats["註"]++
	e.notePend = append(e.notePend, n)
}

// 一段結束：把這段引用到的注文依序寫成 <段/><釋 n="…">…</釋>
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

func clean(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.Join(f, " ")
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
	flag.Parse()
	files, err := filepath.Glob(filepath.Join(*src, "Y*", "*.xml"))
	if err != nil || len(files) == 0 {
		panic("找不到 XML")
	}
	sort.Strings(files)
	e := &emitter{stats: map[string]int{}, imgDir: *img}
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
					if t.Name.Local == "body" {
						inBody = true
					}
					continue
				}
				switch t.Name.Local {
				case "body":
					inBody = true
				case "note":
					// CBETA 的校勘／註記：保留內容，獨立成行（不用 <註> 標籤，避免被當成腳註標記隱藏）
					e.noteSeq++
					n := attr(t.Attr, "n")
					if n == "" {
						n = fmt.Sprintf("%d", e.noteSeq)
					}
					e.nl()
					e.write("〔註" + n + "〕")
					e.stats["註"]++
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
				case "g":
					skip = 1
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
					e.write(fmt.Sprintf(`<頁 id="%dp%s"/>`, vol, pb))
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
				case "note":
					e.nl()
				case "p":
					e.flushNotes()
				case "body":
					e.flushNotes()
					inBody = false
				}
			case xml.CharData:
				if skip > 0 || !inBody {
					continue
				}
				s := clean(string(t))
				if s == "" {
					continue
				}
				e.write(esc(s))
			}
		}
	}
	if e.line.Len() > 0 {
		e.nl()
	}
	if err := os.WriteFile(*out, []byte(e.out.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "lines=%d %v\n", strings.Count(e.out.String(), "\n"), e.stats)
}

