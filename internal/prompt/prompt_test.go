package prompt

import (
	"strings"
	"testing"

	"captchagpt/internal/api"
)

func TestBuild(t *testing.T) {
	value := Build(api.CaptchaHints{
		Length:  4,
		Charset: "numeric",
	})

	for _, expected := range []string{
		"exactly 4 characters",
		"digits only",
	} {
		if !strings.Contains(value, expected) {
			t.Fatalf("missing %q in prompt", expected)
		}
	}
}

func TestBuildMathPrompt(t *testing.T) {
	value := Build(api.CaptchaHints{
		Task: "math",
	})

	for _, expected := range []string{
		"solve the arithmetic captcha",
		"Chinese numerals",
		"return only the final answer as Arabic numerals",
	} {
		if !strings.Contains(value, expected) {
			t.Fatalf("missing %q in math prompt", expected)
		}
	}
}
