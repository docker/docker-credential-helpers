package gitconfig

import (
	"strings"

	"github.com/gobwas/glob"
)

// globMatch implements a glob matcher that supports double-asterisk (**) patterns.
func globMatch(pattern, s string) (bool, error) {
	g, err := glob.Compile(pattern, '/')
	if err != nil {
		return false, err
	}

	return g.Match(s), nil
}

// splitKey splits a fully qualified gitconfig key into two or three parts.
// A valid key consists of either a section and a key separated by a dot
// or section, subsection and key, all separated by a dot. Note that
// the subsection might contain dots itself.
//
// Valid examples:
// - core.push
// - insteadof.git@github.com.push.
func splitKey(key string) (section, subsection, skey string) { //nolint:nonamedreturns
	n := strings.Index(key, ".")
	if n > 0 {
		section = key[:n]
	}

	if m := strings.LastIndex(key, "."); n != m && m > 0 && len(key) > m+1 {
		subsection = key[n+1 : m]
		skey = key[m+1:]

		return
	}

	skey = key[n+1:]

	return
}

func canonicalizeKey(key string) string {
	if key == "" {
		// invalid key, return empty string
		return ""
	}

	section, subsection, skey := splitKey(key)
	// "Section names are case-insensitive.""
	section = strings.ToLower(section)
	// "Subsection names are case sensitive."
	// "The variable names are case-insensitive."
	skey = strings.ToLower(skey)

	if section == "" || skey == "" {
		// invalid key, return empty string
		return ""
	}

	if subsection == "" {
		return section + "." + skey
	}

	return section + "." + subsection + "." + skey
}

func trim(s []string) {
	for i, e := range s {
		s[i] = strings.TrimSpace(e)
	}
}

// parseLineForComment separates a line into content and comment parts.
// It finds the first unquoted comment character (# or ;) to split the line.
// It trims whitespace from the content part and removes matching surrounding
// double ("") quotes from it.
// The returned comment string does NOT include the delimiter character itself
// and is also trimmed of leading/trailing whitespace.
func parseLineForComment(line string) (string, string) {
	line = strings.TrimSpace(line) // Trim whitespace from the line first
	if !strings.HasPrefix(line, `"`) {
		// no properly quoted value string, we shouldn't have ended up here.
		if value, comment, found := strings.Cut(line, "#"); found {
			return strings.TrimSpace(value), strings.TrimSpace(comment)
		}
		if value, comment, found := strings.Cut(line, ";"); found {
			return strings.TrimSpace(value), strings.TrimSpace(comment)
		}

		// no comment found, return the line as is
		return line, ""
	}
	commentStartIndex := -1 // Initialize to -1, indicating comment not found yet
	inQuotes := false
	foundComment := false // Flag to signal when to break the loop

	// Iterate through the string to find the first unquoted comment character
	for i, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case '#', ';':
			if !inQuotes {
				commentStartIndex = i
				foundComment = true
			}
		}
		if foundComment {
			break // Exit the for loop
		}
	}

	content := ""
	comment := ""

	// Determine initial content and comment parts based on index
	var initialContent string
	if commentStartIndex != -1 {
		// Comment character was found
		initialContent = line[:commentStartIndex]
		// Extract comment text *after* the delimiter and trim whitespace
		if commentStartIndex+1 <= len(line) { // Check index bounds
			comment = strings.TrimSpace(line[commentStartIndex+1:])
		} else {
			comment = "" // Delimiter was the very last character
		}
	} else {
		// No unquoted comment character found
		initialContent = line
		comment = ""
	}

	// Trim whitespace from the initial content part FIRST
	trimmedContent := strings.TrimSpace(initialContent)
	// Now, check for and remove surrounding quotes from the trimmed content
	content = strings.Trim(trimmedContent, `"`)

	// Return the processed content and the processed comment part
	return content, comment
}
