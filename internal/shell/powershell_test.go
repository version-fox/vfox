package shell

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"text/template"

	"github.com/version-fox/vfox/internal/env"
)

// powerShellValue evaluates an expression the way PowerShell reads it, for the
// two forms PowerShellEscape returns: a single-quoted literal, whose only escape
// is a doubled quote, and the decoding of base64 UTF-8. Anything else fails the
// test.
func powerShellValue(t *testing.T, expr string) string {
	t.Helper()
	const decodePrefix = "([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String('"
	const decodeSuffix = "')))"
	switch {
	case strings.HasPrefix(expr, decodePrefix) && strings.HasSuffix(expr, decodeSuffix):
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(expr, decodePrefix), decodeSuffix))
		if err != nil {
			t.Fatalf("%q does not decode: %v", expr, err)
		}
		return string(raw)
	case len(expr) >= 2 && strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'"):
		body := expr[1 : len(expr)-1]
		if strings.Contains(strings.ReplaceAll(body, "''", ""), "'") {
			t.Fatalf("%q has a quote that does not close the literal", expr)
		}
		return strings.ReplaceAll(body, "''", "'")
	}
	t.Fatalf("%q is neither a single-quoted literal nor the decoding of base64", expr)
	return ""
}

// isASCIIText reports whether s holds only ASCII bytes.
func isASCIIText(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// PowerShell reads the output of a native command with the console's code page,
// so what vfox prints for it has to be ASCII, and it has to mean the value.
func TestPowerShellEscapeIsASCIIAndMeansTheValue(t *testing.T) {
	for _, value := range []string{
		"",
		`C:\bin`,
		"C:/Program Files (x86)/vfox",
		"it's a 'quoted' value",
		"a$b`c\"d",
		"{{ not an action }}",
		"tab\tand\nnewline\rand control \x01",
		"C:\\用户\\bin",
		"emoji 😀 and a typographic quote’s",
	} {
		expr := PowerShellEscape(value)
		if !isASCIIText(expr) {
			t.Errorf("PowerShellEscape(%q) = %q is not ASCII", value, expr)
		}
		if got := powerShellValue(t, expr); got != value {
			t.Errorf("PowerShellEscape(%q) = %q, which PowerShell reads as %q", value, expr, got)
		}
	}
}

func TestPwshExportIsASCII(t *testing.T) {
	path := "C:\\用户\\bin;C:\\Windows"
	version := "1.0.0"
	out := Pwsh.Export(env.Vars{
		"PATH":          &path,
		"ALPHA_VERSION": &version,
		"WITH.DOT":      &version,
		"GONE":          nil,
	})
	if !isASCIIText(out) {
		t.Fatalf("Export returned text that is not ASCII:\n%s", out)
	}
	for _, want := range []string{
		"$env:ALPHA_VERSION='1.0.0';",
		"$env:PATH=" + PowerShellEscape(path) + ";",
		// A name that $env: cannot spell goes through the .NET API.
		"[System.Environment]::SetEnvironmentVariable('WITH.DOT','1.0.0');",
		"Remove-Item -Path 'env:/GONE';",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Export is missing %q:\n%s", want, out)
		}
	}
	if got := Pwsh.Export(env.Vars{"WITH.DOT": nil}); got != "[System.Environment]::SetEnvironmentVariable('WITH.DOT',$null);" {
		t.Errorf("unset of a name that $env: cannot spell = %q", got)
	}
}

func TestPwshActivateHookIsASCIIAndStillATemplate(t *testing.T) {
	for _, self := range []string{
		"C:/Users/张三/scoop/apps/vfox/current/vfox.exe", // a user name that is not ASCII
		"C:/Users/O'Brien/vfox.exe",
		"C:/{{odd}}/vfox.exe", // text/template would read {{ as the start of an action
	} {
		script, err := Pwsh.Activate(ActivateConfig{SelfPath: self})
		if err != nil {
			t.Fatalf("Activate(%q): %v", self, err)
		}
		if !isASCIIText(script) {
			t.Errorf("Activate(%q) is not ASCII:\n%s", self, script)
		}
		// renderActivateScript executes the script as a template, with the export as EnvContent.
		tmpl, err := template.New("hook").Parse(script)
		if err != nil {
			t.Fatalf("Activate(%q) is not a template: %v", self, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, struct{ SelfPath, EnvContent string }{self, "$env:ALPHA_VERSION='1.0.0';"}); err != nil {
			t.Fatalf("executing the hook for %q: %v", self, err)
		}
		hook := buf.String()
		if !strings.Contains(hook, "$env:ALPHA_VERSION='1.0.0';") {
			t.Errorf("the hook for %q does not hold the export:\n%s", self, hook)
		}
		// The first line of the hook runs vfox by the path it was given.
		end := strings.Index(hook, " env --cleanup 2>$null")
		if end < 0 {
			t.Fatalf("the hook for %q has no cleanup call:\n%s", self, hook)
		}
		head := hook[:end]
		start := strings.LastIndex(head, "& ")
		if start < 0 {
			t.Fatalf("the cleanup call of the hook for %q is not a call operator:\n%s", self, hook)
		}
		if got := powerShellValue(t, head[start+len("& "):]); got != self {
			t.Errorf("the hook runs %q for %q", got, self)
		}
	}
}
