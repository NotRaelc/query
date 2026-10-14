package query

import "strings"

func stripMinecraftColors(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	skip := false
	for _, r := range s {
		if skip {
			skip = false
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func htmlFromMinecraft(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	open := false
	skip := false
	for _, r := range s {
		if skip {
			skip = false
			if open {
				b.WriteString("</span>")
				open = false
			}
			if color := colorFromCode(r); color != "" {
				b.WriteString(`<span style="color: `)
				b.WriteString(color)
				b.WriteString(`">`)
				open = true
			}
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	if open {
		b.WriteString("</span>")
	}
	return b.String()
}

func colorFromCode(r rune) string {
	switch r {
	case '0':
		return "#000000"
	case '1':
		return "#0000AA"
	case '2':
		return "#00AA00"
	case '3':
		return "#00AAAA"
	case '4':
		return "#AA0000"
	case '5':
		return "#AA00AA"
	case '6':
		return "#FFAA00"
	case '7':
		return "#AAAAAA"
	case '8':
		return "#555555"
	case '9':
		return "#5555FF"
	case 'a', 'A':
		return "#55FF55"
	case 'b', 'B':
		return "#55FFFF"
	case 'c', 'C':
		return "#FF5555"
	case 'd', 'D':
		return "#FF55FF"
	case 'e', 'E':
		return "#FFFF55"
	case 'f', 'F':
		return "#FFFFFF"
	case 'g', 'G':
		return "#DDD605"
	default:
		return ""
	}
}
