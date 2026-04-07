package render

import (
	"regexp"
	"strings"
)

// PreprocessComponents scans HTML for <t-NAME /> and <t-NAME>...</t-NAME> tags
// and transforms them into {{component "NAME"}} template calls.
//
// Only tags whose name exists in the knownComponents set are transformed;
// other t-* tags (t-if, t-for, t-title, etc.) are left untouched.
//
// If a component tag has a t-sse="event-name" attribute, the output is
// wrapped in a <div> with data-t-sse and data-t-sse-url attributes
// so the SSE client JS can auto-refresh it when that event fires.
//
// This preprocessor runs BEFORE the directive preprocessor so that
// component output containing t-if, t-for, etc. is processed correctly.
func PreprocessComponents(src string, knownComponents map[string]bool) string {
	if len(knownComponents) == 0 {
		return src
	}
	for {
		loc := findFirstComponentTag(src, knownComponents)
		if loc == nil {
			break
		}
		src = applyComponentTag(src, loc)
	}
	// Also process t-sse on regular HTML elements
	src = processSSEDirective(src)
	return src
}

// componentTagLoc describes the location of a <t-NAME> tag in the source.
type componentTagLoc struct {
	tagStart    int
	tagEnd      int
	tagName     string // full tag name, e.g. "t-counter"
	compName    string // component name, e.g. "counter"
	selfClosing bool
	sseEvent    string // value of t-sse attribute, if any
}

// reComponentTag matches <t-NAME where NAME starts with a letter (not reserved directives).
var reComponentTag = regexp.MustCompile(`<(t-([\w][\w-]*))\b`)

// reSSEAttr matches the t-sse="event-name" attribute on a tag.
// It requires a leading space to avoid matching data-t-sse.
var reSSEAttr = regexp.MustCompile(`\s+t-sse=(?:"([^"]*)"|'([^']*)')`)

// reservedTags are t-* tags handled by the directive preprocessor, not components.
var reservedTags = map[string]bool{
	"t-title": true,
}

// reservedAttrs are t-* attribute prefixes that indicate a directive, not a component.
var reservedAttrs = []string{"t-if", "t-for", "t-else", "t-else-if", "t-class-", "t-morph", "t-sse"}

func findFirstComponentTag(src string, known map[string]bool) *componentTagLoc {
	pos := 0
	for pos < len(src) {
		loc := reComponentTag.FindStringIndex(src[pos:])
		if loc == nil {
			return nil
		}
		matchStart := pos + loc[0]

		// Skip if this is inside a closing tag
		if matchStart > 0 && src[matchStart-1] == '/' {
			pos = pos + loc[1]
			continue
		}

		end := findTagEnd(src, matchStart)
		if end == -1 {
			pos = pos + loc[1]
			continue
		}

		tag := src[matchStart:end]
		m := reComponentTag.FindStringSubmatch(tag)
		if m == nil {
			pos = pos + loc[1]
			continue
		}

		fullTagName := m[1] // e.g. "t-counter"
		compName := m[2]    // e.g. "counter"

		// Skip reserved tags
		if reservedTags[fullTagName] {
			pos = end
			continue
		}

		// Skip tags that are actually directives used as attributes on other elements
		isDirective := false
		for _, attr := range reservedAttrs {
			if fullTagName == attr || strings.HasPrefix(fullTagName, attr) {
				isDirective = true
				break
			}
		}
		if isDirective {
			pos = end
			continue
		}

		// Only transform known components
		if !known[compName] {
			pos = end
			continue
		}

		// Extract t-sse attribute if present
		var sseEvent string
		if sm := reSSEAttr.FindStringSubmatch(tag); sm != nil {
			sseEvent = sm[1]
			if sseEvent == "" {
				sseEvent = sm[2]
			}
		}

		return &componentTagLoc{
			tagStart:    matchStart,
			tagEnd:      end,
			tagName:     fullTagName,
			compName:    compName,
			selfClosing: isSelfClosing(tag),
			sseEvent:    sseEvent,
		}
	}
	return nil
}

func applyComponentTag(src string, loc *componentTagLoc) string {
	componentCall := `{{component "` + loc.compName + `"}}`

	// If t-sse is set, wrap in a div with SSE data attributes
	var replacement string
	if loc.sseEvent != "" {
		replacement = `<div data-t-sse="` + loc.sseEvent + `" data-t-sse-url="/__thunder/component/` + loc.compName + `">` +
			componentCall + `</div>`
	} else {
		replacement = componentCall
	}

	if loc.selfClosing {
		return src[:loc.tagStart] + replacement + src[loc.tagEnd:]
	}

	// Find the matching closing tag </t-NAME>
	closeStart, closeEnd := findMatchingClose(src, loc.tagEnd, loc.tagName)
	if closeStart == -1 {
		// No closing tag found — treat as self-closing
		return src[:loc.tagStart] + replacement + src[loc.tagEnd:]
	}

	// Replace the entire <t-NAME>...</t-NAME> block
	return src[:loc.tagStart] + replacement + src[closeEnd:]
}

// processSSEDirective transforms t-sse="event-name" attributes on regular
// HTML elements into data-t-sse="event-name" data-t-sse-url="..." attributes.
// The data-t-sse-url must be provided explicitly on non-component elements
// since the framework can't infer a render endpoint for arbitrary HTML.
func processSSEDirective(src string) string {
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
		if reSSEAttr.MatchString(tag) {
			tag = transformSSETag(tag)
		}
		out.WriteString(tag)
		pos = end
	}

	return out.String()
}

// transformSSETag converts t-sse="event-name" to data-t-sse="event-name"
// on a regular HTML tag. If data-t-sse-url is not present, it preserves
// the tag as-is (component tags handle URL injection automatically).
func transformSSETag(tag string) string {
	m := reSSEAttr.FindStringSubmatch(tag)
	if m == nil {
		return tag
	}
	eventName := m[1]
	if eventName == "" {
		eventName = m[2]
	}

	// Remove the t-sse attribute
	tag = reSSEAttr.ReplaceAllString(tag, "")

	// Add data-t-sse attribute (before closing >)
	i := strings.LastIndex(tag, ">")
	if i > 0 && tag[i-1] == '/' {
		i--
	}
	tag = tag[:i] + ` data-t-sse="` + eventName + `"` + tag[i:]

	return cleanTagSpaces(tag)
}
