package slice

import (
	"strings"
	"testing"
)

func TestExcludeWhitespaces(t *testing.T) {
	arr := []string{
		"lorem", "ipsum", " ", "dolor", "", "sit", "amet",
	}
	trimmed := ExcludeWhitespaces(arr)
	for _, v := range trimmed {
		if v == "" {
			t.Error("found unstripped whice space")
		}
	}

	expected := "lorem,ipsum,dolor,sit,amet"
	str := strings.Join(trimmed, ",")
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}
