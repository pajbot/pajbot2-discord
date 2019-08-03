package roles

import (
	"testing"

	"github.com/pajbot/testhelper"
)

func TestResolve(t *testing.T) {
	testhelper.AssertStringSlicesEqual(t, []string{"admin"}, resolve("admin"))
	testhelper.AssertStringSlicesEqual(t, []string{"mod", "admin"}, resolve("mod"))
	testhelper.AssertStringSlicesEqual(t, []string{"minimod", "mod", "admin"}, resolve("minimod"))
	testhelper.AssertStringSlicesEqual(t, []string{"muted"}, resolve("muted"))
	testhelper.AssertStringSlicesEqual(t, []string{"nitrobooster"}, resolve("nitrobooster"))
}
