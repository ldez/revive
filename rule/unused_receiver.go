package rule

import (
	"fmt"
	"go/ast"
	"regexp"
	"sync"

	"github.com/mgechev/revive/lint"
)

// UnusedReceiverRule lints unused receivers in functions.
type UnusedReceiverRule struct {
	// regex to check if some name is valid for unused parameter, "^_$" by default
	allowRegex *regexp.Regexp
	failureMsg string

	configureOnce sync.Once
}

func (r *UnusedReceiverRule) configure(args lint.Arguments) {
	// while by default args is an array, i think it's good to provide structures inside it by default, not arrays or primitives
	// it's more compatible to JSON nature of configurations
	var allowRegexStr string
	if len(args) == 0 {
		allowRegexStr = "^_$"
		r.failureMsg = "method receiver '%s' is not referenced in method's body, consider removing or renaming it as _"
	} else {
		// Arguments = [{}]
		options := args[0].(map[string]any)
		// Arguments = [{allowRegex="^_"}]

		if allowRegexParam, ok := options["allowRegex"]; ok {
			allowRegexStr, ok = allowRegexParam.(string)
			if !ok {
				panic(fmt.Errorf("error configuring [unused-receiver] rule: allowRegex is not string but [%T]", allowRegexParam))
			}
		}
	}
	var err error
	r.allowRegex, err = regexp.Compile(allowRegexStr)
	if err != nil {
		panic(fmt.Errorf("error configuring [unused-receiver] rule: allowRegex is not valid regex [%s]: %v", allowRegexStr, err))
	}
	if r.failureMsg == "" {
		r.failureMsg = "method receiver '%s' is not referenced in method's body, consider removing or renaming it to match " + r.allowRegex.String()
	}
}

// Apply applies the rule to given file.
func (r *UnusedReceiverRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	r.configureOnce.Do(func() { r.configure(args) })
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	w := lintUnusedReceiverRule{
		onFailure:  onFailure,
		allowRegex: r.allowRegex,
		failureMsg: r.failureMsg,
	}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (*UnusedReceiverRule) Name() string {
	return "unused-receiver"
}

type lintUnusedReceiverRule struct {
	onFailure  func(lint.Failure)
	allowRegex *regexp.Regexp
	failureMsg string
}

func (w lintUnusedReceiverRule) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Recv == nil {
			return nil // skip this func decl, not a method
		}

		rec := n.Recv.List[0] // safe to access only the first (unique) element of the list
		if len(rec.Names) < 1 {
			return nil // the receiver is anonymous: func (aType) Foo(...) ...
		}

		recID := rec.Names[0]
		if recID.Name == "_" {
			return nil // the receiver is already named _
		}

		if w.allowRegex != nil && w.allowRegex.FindStringIndex(recID.Name) != nil {
			return nil
		}

		// inspect the func body looking for references to the receiver id
		fselect := func(n ast.Node) bool {
			ident, isAnID := n.(*ast.Ident)

			return isAnID && ident.Obj == recID.Obj
		}
		refs2recID := pick(n.Body, fselect)

		if len(refs2recID) > 0 {
			return nil // the receiver is referenced in the func body
		}

		w.onFailure(lint.Failure{
			Confidence: 1,
			Node:       recID,
			Category:   "bad practice",
			Failure:    fmt.Sprintf(w.failureMsg, recID.Name),
		})

		return nil // full method body already inspected
	}

	return w
}
