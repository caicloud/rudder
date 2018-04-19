package name

import (
	"strings"
	"testing"
)

func TestHashName(t *testing.T) {
	name := GenerateHashName("app", "deploy")
	if !strings.HasPrefix(name, "app") {
		t.Fatalf("The prefix must be app")
	}
}
