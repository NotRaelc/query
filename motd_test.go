package query

import "testing"

func TestStripMinecraftColors(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"§ahello", "hello"},
		{"§a§lWorld§r!", "World!"},
		{"§cRed §7Gray", "Red Gray"},
		{"§0§1§2§3§4§5§6§7§8§9§a§b§c§d§e§f", ""},
		{"§zunknown", "unknown"},
		{"multi\nline", "multi\nline"},
		{"Unicode §aПривет", "Unicode Привет"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := stripMinecraftColors(tt.in); got != tt.want {
				t.Fatalf("stripMinecraftColors(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHTMLFromMinecraft(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{"§ahello", `<span style="color: #55FF55">hello</span>`},
		{"§ahello§r world", `<span style="color: #55FF55">hello</span> world`},
		{"§cRed §7Gray", `<span style="color: #FF5555">Red </span><span style="color: #AAAAAA">Gray</span>`},
		{"§zunknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := htmlFromMinecraft(tt.in); got != tt.want {
				t.Fatalf("htmlFromMinecraft(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestColorFromCode(t *testing.T) {
	tests := []struct {
		in   rune
		want string
	}{
		{'0', "#000000"},
		{'a', "#55FF55"},
		{'A', "#55FF55"},
		{'f', "#FFFFFF"},
		{'g', "#DDD605"},
		{'r', ""},
		{'k', ""},
		{'z', ""},
	}
	for _, tt := range tests {
		if got := colorFromCode(tt.in); got != tt.want {
			t.Fatalf("colorFromCode(%c) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
