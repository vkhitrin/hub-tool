package e2e

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"
)

func TestCompletionDoesNotRequireLogin(t *testing.T) {
	cmd, cleanup := hubToolCmd(t, "completion", "bash")
	cleanup()

	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Check(t, strings.Contains(output, "# bash completion"))
	assert.Check(t, strings.Contains(output, "__complete"))
}
