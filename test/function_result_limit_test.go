package test

import (
	"testing"

	"github.com/mgechev/revive/lint"
	"github.com/mgechev/revive/rule"
)

func TestFunctionResultsLimit(t *testing.T) {
	testRule(t, "function_result_limit", &rule.FunctionResultsLimitRule{}, &lint.RuleConfig{
		Arguments: []any{int64(3)},
	})
}
