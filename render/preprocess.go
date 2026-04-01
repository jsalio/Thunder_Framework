package render

import (
	"regexp"
	"strings"
)

// Preprocess transforms Thunder directives (t-if, t-for, t-else, t-class-*, t-morph)
// into Go html/template syntax. It runs before the Go parser.
//
// Supported directives:
//   - t-if="expr"           → {{if expr}}...{{end}}
//   - t-else                → {{else}} (paired with previous t-if)
//   - t-else-if="expr"      → {{else if expr}} (paired with previous t-if)
//   - t-for="expr"          → {{range expr}}...{{end}}
//   - t-class-NAME="expr"   → conditionally adds NAME to class attribute
//   - t-morph               → adds hx-ext="morph" hx-swap="morph:innerHTML"
//   - <t-title>text</t-title> → {{define "title"}}text{{end}}
//
// When directives are used on <template> tags, the <template> tags
// are removed and only the inner content is emitted.
func Preprocess(src string) string {
	src = processClassDirectives(src)
	src = processMorphDirective(src)
	src = processBlockDirectives(src)
	return src
}

// PreprocessPage processes a page/component template.
// Transforms <t-title>...</t-title> and auto-wraps in {{define "content"}}...{{end}}.
// If the template already contains {{define }}, it leaves it as is (backward compatibility).
func PreprocessPage(src string) string {
	src = processAutoDefine(src)
	return Preprocess(src)
}

// PreprocessLayout processes a layout template.
// Removes the wrapper {{define "xxx"}}...{{end}} if it exists, since
// template.New(name).Parse(src) assigns content to the name automatically.
func PreprocessLayout(src string) string {
	src = stripLayoutDefine(src)
	return Preprocess(src)
}

var reTTitle = regexp.MustCompile(`(?s)<t-title>(.*?)</t-title>`)
var reLayoutDefine = regexp.MustCompile(`^\{\{define\s+"[^"]+"\}\}`)

// processAutoDefine transforms <t-title> and wraps the content
// in {{define "content"}}...{{end}} automatically.
func processAutoDefine(src string) string {
	// Backward compatibility: if it already uses {{define}}, don't touch.
	if strings.Contains(src, "{{define ") {
		return src
	}

	var titleBlock string
	// Extraer <t-title>...</t-title>
	if loc := reTTitle.FindStringSubmatchIndex(src); loc != nil {
		title := strings.TrimSpace(src[loc[2]:loc[3]])
		titleBlock = "{{define \"title\"}}" + title + "{{end}}\n\n"
		src = src[:loc[0]] + src[loc[1]:]
	}

	content := strings.TrimSpace(src)
	return titleBlock + "{{define \"content\"}}\n" + content + "\n{{end}}\n"
}

// stripLayoutDefine removes the wrapper {{define "xxx"}}...{{end}}
// that wraps the entire layout content.
func stripLayoutDefine(src string) string {
	trimmed := strings.TrimSpace(src)
	if !reLayoutDefine.MatchString(trimmed) {
		return src
	}
	if !strings.HasSuffix(trimmed, "{{end}}") {
		return src
	}
	loc := reLayoutDefine.FindStringIndex(trimmed)
	return trimmed[loc[1] : len(trimmed)-len("{{end}}")]
}

// ── HTML Scanning Utilities ────────────────────────────────────────────────

// findTagEnd finds the closing '>' of an open tag at pos,
// respecting quotes in attributes.
func findTagEnd(src string, pos int) int {
	inQuote := byte(0)
	for i := pos + 1; i < len(src); i++ {
		c := src[i]
		if inQuote != 0 {
			if c == inQuote {
				inQuote = 0
			}
		} else if c == '"' || c == '\'' {
			inQuote = c
		} else if c == '>' {
			return i + 1
		}
	}
	return -1
}

// extractTagName extracts the tag name from an opening "<tagname ...>".
func extractTagName(tag string) string {
	start := 1
	for start < len(tag) && isSpace(tag[start]) {
		start++
	}
	end := start
	for end < len(tag) && !isSpace(tag[end]) && tag[end] != '>' && tag[end] != '/' {
		end++
	}
	return strings.ToLower(tag[start:end])
}

// extractClosingTagName extracts the name from "</tagname>".
func extractClosingTagName(tag string) string {
	start := 2 // skip "</"
	for start < len(tag) && isSpace(tag[start]) {
		start++
	}
	end := start
	for end < len(tag) && !isSpace(tag[end]) && tag[end] != '>' {
		end++
	}
	return strings.ToLower(tag[start:end])
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// isSelfClosing checks if the tag ends with "/>".
func isSelfClosing(tag string) bool {
	for i := len(tag) - 2; i >= 0; i-- {
		if tag[i] == '/' {
			return true
		}
		if !isSpace(tag[i]) {
			return false
		}
	}
	return false
}

// findMatchingClose looks for the closing tag corresponding to tagName,
// starting from startPos. Returns (start of </tag, end of </tag>).
func findMatchingClose(src string, startPos int, tagName string) (int, int) {
	depth := 1
	pos := startPos

	for pos < len(src) {
		idx := strings.Index(src[pos:], "<")
		if idx == -1 {
			return -1, -1
		}
		pos += idx

		end := findTagEnd(src, pos)
		if end == -1 {
			return -1, -1
		}

		tag := src[pos:end]

		if len(tag) > 1 && tag[1] == '/' {
			name := extractClosingTagName(tag)
			if name == tagName {
				depth--
				if depth == 0 {
					return pos, end
				}
			}
		} else if tag[1] != '!' {
			name := extractTagName(tag)
			if name == tagName && !isSelfClosing(tag) {
				depth++
			}
		}

		pos = end
	}

	return -1, -1
}

// ── Class Directives (t-class-*) ───────────────────────────────────────────

var reClassDir = regexp.MustCompile(`\bt-class-([\w-]+)=(?:"([^"]*)"|'([^']*)')`)

func processClassDirectives(src string) string {
	var out strings.Builder
	pos := 0

	for pos < len(src) {
		idx := strings.Index(src[pos:], "<")
		if idx == -1 {
			out.WriteString(src[pos:])
			break
		}
		out.WriteString(src[pos : pos+idx])
		pos += idx

		// Skip closing tags and comments
		if pos+1 < len(src) && (src[pos+1] == '/' || src[pos+1] == '!') {
			end := findTagEnd(src, pos)
			if end == -1 {
				out.WriteString(src[pos:])
				break
			}
			out.WriteString(src[pos:end])
			pos = end
			continue
		}

		end := findTagEnd(src, pos)
		if end == -1 {
			out.WriteString(src[pos:])
			break
		}

		tag := src[pos:end]
		if reClassDir.MatchString(tag) {
			tag = transformClassTag(tag)
		}
		out.WriteString(tag)
		pos = end
	}

	return out.String()
}

var reClassAttr = regexp.MustCompile(`(class\s*=\s*")([^"]*)"`)

func transformClassTag(tag string) string {
	matches := reClassDir.FindAllStringSubmatch(tag, -1)
	tag = reClassDir.ReplaceAllString(tag, "")

	var cond strings.Builder
	for _, m := range matches {
		className := m[1]
		expr := m[2]
		if expr == "" {
			expr = m[3]
		}
		cond.WriteString("{{if ")
		cond.WriteString(expr)
		cond.WriteString("}} ")
		cond.WriteString(className)
		cond.WriteString("{{end}}")
	}

	if reClassAttr.MatchString(tag) {
		tag = reClassAttr.ReplaceAllStringFunc(tag, func(match string) string {
			sub := reClassAttr.FindStringSubmatch(match)
			return sub[1] + sub[2] + cond.String() + `"`
		})
	} else {
		// No existing class: create attribute before closing
		i := strings.LastIndex(tag, ">")
		if i > 0 && tag[i-1] == '/' {
			i--
		}
		tag = tag[:i] + ` class="` + strings.TrimPrefix(cond.String(), " ") + `"` + tag[i:]
	}

	return cleanTagSpaces(tag)
}

// cleanTagSpaces cleans extra spaces outside of quotes.
func cleanTagSpaces(tag string) string {
	var out strings.Builder
	inQuote := byte(0)
	wasSpace := false

	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if inQuote != 0 {
			out.WriteByte(c)
			if c == inQuote {
				inQuote = 0
			}
			wasSpace = false
		} else if c == '"' || c == '\'' {
			out.WriteByte(c)
			inQuote = c
			wasSpace = false
		} else if isSpace(c) {
			if !wasSpace {
				out.WriteByte(' ')
			}
			wasSpace = true
		} else {
			out.WriteByte(c)
			wasSpace = false
		}
	}

	r := out.String()
	r = strings.ReplaceAll(r, " >", ">")
	r = strings.ReplaceAll(r, " />", "/>")
	return r
}

// ── Morph Directive (t-morph) ────────────────────────────────────────────────

var reMorphDir = regexp.MustCompile(`\bt-morph\b`)
var reHxSwap = regexp.MustCompile(`hx-swap\s*=\s*"([^"]*)"`)
var reHxExt = regexp.MustCompile(`hx-ext\s*=\s*"([^"]*)"`)

// processMorphDirective replaces t-morph with hx-ext="morph" and
// hx-swap="morph:innerHTML". If hx-swap already exists, its value
// is prefixed with "morph:" (unless already prefixed).
func processMorphDirective(src string) string {
	var out strings.Builder
	pos := 0

	for pos < len(src) {
		idx := strings.Index(src[pos:], "<")
		if idx == -1 {
			out.WriteString(src[pos:])
			break
		}
		out.WriteString(src[pos : pos+idx])
		pos += idx

		// Skip closing tags and comments
		if pos+1 < len(src) && (src[pos+1] == '/' || src[pos+1] == '!') {
			end := findTagEnd(src, pos)
			if end == -1 {
				out.WriteString(src[pos:])
				break
			}
			out.WriteString(src[pos:end])
			pos = end
			continue
		}

		end := findTagEnd(src, pos)
		if end == -1 {
			out.WriteString(src[pos:])
			break
		}

		tag := src[pos:end]
		if reMorphDir.MatchString(tag) {
			tag = transformMorphTag(tag)
		}
		out.WriteString(tag)
		pos = end
	}

	return out.String()
}

func transformMorphTag(tag string) string {
	// Remove the t-morph attribute
	tag = reMorphDir.ReplaceAllString(tag, "")

	// Handle hx-swap: prefix existing value with "morph:" or add default
	if reHxSwap.MatchString(tag) {
		tag = reHxSwap.ReplaceAllStringFunc(tag, func(match string) string {
			sub := reHxSwap.FindStringSubmatch(match)
			val := sub[1]
			if !strings.HasPrefix(val, "morph:") {
				val = "morph:" + val
			}
			return `hx-swap="` + val + `"`
		})
	} else {
		// No hx-swap: insert before closing >
		i := strings.LastIndex(tag, ">")
		if i > 0 && tag[i-1] == '/' {
			i--
		}
		tag = tag[:i] + ` hx-swap="morph:innerHTML"` + tag[i:]
	}

	// Handle hx-ext: merge "morph" into existing value or add new
	if reHxExt.MatchString(tag) {
		tag = reHxExt.ReplaceAllStringFunc(tag, func(match string) string {
			sub := reHxExt.FindStringSubmatch(match)
			val := sub[1]
			if !strings.Contains(val, "morph") {
				val = val + " morph"
			}
			return `hx-ext="` + val + `"`
		})
	} else {
		i := strings.LastIndex(tag, ">")
		if i > 0 && tag[i-1] == '/' {
			i--
		}
		tag = tag[:i] + ` hx-ext="morph"` + tag[i:]
	}

	return cleanTagSpaces(tag)
}

// ── Block Directives (t-for, t-if, t-else) ─────────────────────────────────

var reBlockDir = regexp.MustCompile(`\bt-(for|if|else-if|else)\b(?:=(?:"([^"]*)"|'([^']*)'))?`)

type directiveLoc struct {
	tagStart   int
	tagEnd     int
	tagName    string
	dirType    string // "for", "if", "else-if", "else"
	dirExpr    string
	isTemplate bool
}

func processBlockDirectives(src string) string {
	for {
		loc := findFirstBlockDirective(src)
		if loc == nil {
			break
		}
		src = applyBlockDirective(src, loc)
	}
	return src
}

func findFirstBlockDirective(src string) *directiveLoc {
	pos := 0
	for pos < len(src) {
		idx := strings.Index(src[pos:], "<")
		if idx == -1 {
			return nil
		}
		pos += idx

		if pos+1 >= len(src) || src[pos+1] == '/' || src[pos+1] == '!' {
			pos++
			continue
		}

		end := findTagEnd(src, pos)
		if end == -1 {
			pos++
			continue
		}

		tag := src[pos:end]
		m := reBlockDir.FindStringSubmatch(tag)
		if m != nil {
			tagName := extractTagName(tag)
			expr := m[2]
			if expr == "" {
				expr = m[3]
			}
			return &directiveLoc{
				tagStart:   pos,
				tagEnd:     end,
				tagName:    tagName,
				dirType:    m[1],
				dirExpr:    expr,
				isTemplate: tagName == "template",
			}
		}

		pos = end
	}
	return nil
}

func removeDirectiveAttr(tag string) string {
	return cleanTagSpaces(reBlockDir.ReplaceAllString(tag, ""))
}

func applyBlockDirective(src string, loc *directiveLoc) string {
	switch loc.dirType {
	case "for":
		return applyFor(src, loc)
	case "if":
		return applyIf(src, loc)
	default:
		// Orphan t-else / t-else-if: just clean the attribute
		cleaned := removeDirectiveAttr(src[loc.tagStart:loc.tagEnd])
		return src[:loc.tagStart] + cleaned + src[loc.tagEnd:]
	}
}

func applyFor(src string, loc *directiveLoc) string {
	cleaned := removeDirectiveAttr(src[loc.tagStart:loc.tagEnd])

	if isSelfClosing(src[loc.tagStart:loc.tagEnd]) {
		return src[:loc.tagStart] +
			"{{range " + loc.dirExpr + "}}" + cleaned + "{{end}}" +
			src[loc.tagEnd:]
	}

	closeStart, closeEnd := findMatchingClose(src, loc.tagEnd, loc.tagName)
	if closeStart == -1 {
		return src
	}

	inner := src[loc.tagEnd:closeStart]
	closeTag := src[closeStart:closeEnd]

	if loc.isTemplate {
		return src[:loc.tagStart] +
			"{{range " + loc.dirExpr + "}}" + inner + "{{end}}" +
			src[closeEnd:]
	}

	return src[:loc.tagStart] +
		"{{range " + loc.dirExpr + "}}" + cleaned + inner + closeTag + "{{end}}" +
		src[closeEnd:]
}

func applyIf(src string, loc *directiveLoc) string {
	cleaned := removeDirectiveAttr(src[loc.tagStart:loc.tagEnd])

	if isSelfClosing(src[loc.tagStart:loc.tagEnd]) {
		return src[:loc.tagStart] +
			"{{if " + loc.dirExpr + "}}" + cleaned + "{{end}}" +
			src[loc.tagEnd:]
	}

	closeStart, closeEnd := findMatchingClose(src, loc.tagEnd, loc.tagName)
	if closeStart == -1 {
		return src
	}

	inner := src[loc.tagEnd:closeStart]
	closeTag := src[closeStart:closeEnd]

	var ifContent string
	if loc.isTemplate {
		ifContent = inner
	} else {
		ifContent = cleaned + inner + closeTag
	}

	// Look for sibling t-else or t-else-if after closing
	elseLoc := findElseSibling(src, closeEnd)

	if elseLoc == nil {
		return src[:loc.tagStart] +
			"{{if " + loc.dirExpr + "}}" + ifContent + "{{end}}" +
			src[closeEnd:]
	}

	// Process the else block
	elseCleaned := removeDirectiveAttr(src[elseLoc.tagStart:elseLoc.tagEnd])
	var elseContent string
	var afterElse int

	if isSelfClosing(src[elseLoc.tagStart:elseLoc.tagEnd]) {
		if elseLoc.isTemplate {
			elseContent = ""
		} else {
			elseContent = elseCleaned
		}
		afterElse = elseLoc.tagEnd
	} else {
		elseCloseStart, elseCloseEnd := findMatchingClose(src, elseLoc.tagEnd, elseLoc.tagName)
		if elseCloseStart == -1 {
			return src[:loc.tagStart] +
				"{{if " + loc.dirExpr + "}}" + ifContent + "{{end}}" +
				src[closeEnd:]
		}

		elseInner := src[elseLoc.tagEnd:elseCloseStart]
		elseCloseTag := src[elseCloseStart:elseCloseEnd]

		if elseLoc.isTemplate {
			elseContent = elseInner
		} else {
			elseContent = elseCleaned + elseInner + elseCloseTag
		}
		afterElse = elseCloseEnd
	}

	elseKeyword := "{{else}}"
	if elseLoc.dirType == "else-if" {
		elseKeyword = "{{else if " + elseLoc.dirExpr + "}}"
	}

	// Preserve whitespace between if-close and else-open
	between := src[closeEnd:elseLoc.tagStart]

	return src[:loc.tagStart] +
		"{{if " + loc.dirExpr + "}}" + ifContent + elseKeyword + between + elseContent + "{{end}}" +
		src[afterElse:]
}

// findElseSibling looks for a sibling element with t-else or t-else-if
// immediately after pos (only intermediate whitespace).
func findElseSibling(src string, pos int) *directiveLoc {
	i := pos
	for i < len(src) && isSpace(src[i]) {
		i++
	}

	if i >= len(src) || src[i] != '<' || (i+1 < len(src) && src[i+1] == '/') {
		return nil
	}

	end := findTagEnd(src, i)
	if end == -1 {
		return nil
	}

	tag := src[i:end]
	m := reBlockDir.FindStringSubmatch(tag)
	if m == nil || (m[1] != "else" && m[1] != "else-if") {
		return nil
	}

	tagName := extractTagName(tag)
	expr := m[2]
	if expr == "" {
		expr = m[3]
	}

	return &directiveLoc{
		tagStart:   i,
		tagEnd:     end,
		tagName:    tagName,
		dirType:    m[1],
		dirExpr:    expr,
		isTemplate: tagName == "template",
	}
}
