package render

import (
	"regexp"
	"strings"
)

// Preprocess transforma directivas Thunder (t-if, t-for, t-else, t-class-*)
// en sintaxis de Go html/template. Se ejecuta antes del parser de Go.
//
// Directivas soportadas:
//   - t-if="expr"           → {{if expr}}...{{end}}
//   - t-else                → {{else}} (emparejado con t-if anterior)
//   - t-else-if="expr"      → {{else if expr}} (emparejado con t-if anterior)
//   - t-for="expr"          → {{range expr}}...{{end}}
//   - t-class-NAME="expr"   → agrega NAME al class condicionalmente
//   - <t-title>text</t-title> → {{define "title"}}text{{end}}
//
// Cuando las directivas se usan en <template>, las etiquetas <template>
// se eliminan y solo se emite el contenido interior.
func Preprocess(src string) string {
	src = processClassDirectives(src)
	src = processBlockDirectives(src)
	return src
}

// PreprocessPage procesa un template de página/componente.
// Transforma <t-title>...</t-title> y auto-envuelve en {{define "content"}}...{{end}}.
// Si el template ya contiene {{define }}, lo deja tal cual (retrocompatibilidad).
func PreprocessPage(src string) string {
	src = processAutoDefine(src)
	return Preprocess(src)
}

// PreprocessLayout procesa un template de layout.
// Elimina el wrapper {{define "xxx"}}...{{end}} si existe, ya que
// template.New(name).Parse(src) asigna el contenido al nombre automáticamente.
func PreprocessLayout(src string) string {
	src = stripLayoutDefine(src)
	return Preprocess(src)
}

var reTTitle = regexp.MustCompile(`(?s)<t-title>(.*?)</t-title>`)
var reLayoutDefine = regexp.MustCompile(`^\{\{define\s+"[^"]+"\}\}`)

// processAutoDefine transforma <t-title> y envuelve el contenido
// en {{define "content"}}...{{end}} automáticamente.
func processAutoDefine(src string) string {
	// Retrocompatibilidad: si ya usa {{define}}, no tocar.
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

// stripLayoutDefine elimina el wrapper {{define "xxx"}}...{{end}}
// que envuelve todo el contenido del layout.
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

// ── Utilidades de escaneo HTML ────────────────────────────────────────────

// findTagEnd encuentra el cierre '>' de una etiqueta abierta en pos,
// respetando comillas en atributos.
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

// extractTagName extrae el nombre de etiqueta de una apertura "<tagname ...>".
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

// extractClosingTagName extrae el nombre de "</tagname>".
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

// isSelfClosing verifica si la etiqueta termina con "/>".
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

// findMatchingClose busca la etiqueta de cierre que corresponde a tagName,
// comenzando desde startPos. Retorna (inicio de </tag, fin de </tag>).
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

// ── Directivas de clase (t-class-*) ──────────────────────────────────────

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

		// Saltar cierres y comentarios
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
		// Sin class existente: crear atributo antes del cierre
		i := strings.LastIndex(tag, ">")
		if i > 0 && tag[i-1] == '/' {
			i--
		}
		tag = tag[:i] + ` class="` + strings.TrimPrefix(cond.String(), " ") + `"` + tag[i:]
	}

	return cleanTagSpaces(tag)
}

// cleanTagSpaces limpia espacios extra fuera de comillas.
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

// ── Directivas de bloque (t-if, t-for, t-else) ──────────────────────────

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
		// t-else / t-else-if huérfano: solo limpiar el atributo
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

	// Buscar t-else o t-else-if hermano después del cierre
	elseLoc := findElseSibling(src, closeEnd)

	if elseLoc == nil {
		return src[:loc.tagStart] +
			"{{if " + loc.dirExpr + "}}" + ifContent + "{{end}}" +
			src[closeEnd:]
	}

	// Procesar el bloque else
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

	// Preservar whitespace entre el if-close y el else-open
	between := src[closeEnd:elseLoc.tagStart]

	return src[:loc.tagStart] +
		"{{if " + loc.dirExpr + "}}" + ifContent + elseKeyword + between + elseContent + "{{end}}" +
		src[afterElse:]
}

// findElseSibling busca un elemento hermano con t-else o t-else-if
// inmediatamente después de pos (solo whitespace intermedio).
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
