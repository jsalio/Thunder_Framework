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
	return src
}

// componentTagLoc describes the location of a <t-NAME> tag in the source.
type componentTagLoc struct {
	tagStart    int
	tagEnd      int
	tagName     string // full tag name, e.g. "t-counter"
	compName    string // component name, e.g. "counter"
	selfClosing bool
}

// reComponentTag matches <t-NAME where NAME starts with a letter (not reserved directives).
var reComponentTag = regexp.MustCompile(`<(t-([\w][\w-]*))\b`)

// reservedTags are t-* tags handled by the directive preprocessor, not components.
var reservedTags = map[string]bool{
	"t-title": true,
}

// reservedAttrs are t-* attribute prefixes that indicate a directive, not a component.
var reservedAttrs = []string{"t-if", "t-for", "t-else", "t-else-if", "t-class-", "t-morph"}

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

		return &componentTagLoc{
			tagStart:    matchStart,
			tagEnd:      end,
			tagName:     fullTagName,
			compName:    compName,
			selfClosing: isSelfClosing(tag),
		}
	}
	return nil
}

func applyComponentTag(src string, loc *componentTagLoc) string {
	replacement := `{{component "` + loc.compName + `"}}`

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
