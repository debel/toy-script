package main

import (
	"fmt"
	"os"
	"strings"
)

type (
	inode = any

	frame struct {
		vars   map[string]inode
		parent *frame
	}

	toyInterpreter struct {
		globals *frame
	}

	funcType = func(a ...any) any
)

// HARD-CODED LIBS

func newFrame(p *frame) *frame {
	return &frame{
		vars:   map[string]inode{},
		parent: p,
	}
}

func (f *frame) set(k string, v inode) {
	f.vars[k] = v
}

func (f *frame) get(k string) (inode, bool) {
	n, ok := f.vars[k]
	if ok {
		return n, true
	}

	if f.parent == nil {
		return nil, false
	}

	return f.parent.get(k)
}

func NewInterpreter(globals map[string]inode) *toyInterpreter {
	topFrame := newFrame(nil)
	injectBuiltins(topFrame)

	for k, v := range globals {
		topFrame.set(k, v)
	}

	return &toyInterpreter{
		globals: topFrame,
	}
}

func (i *toyInterpreter) Exec(p *ProgramStatement) {
	for _, s := range p.Body {
		i.execNode(s, i.globals)
	}
}

func (i *toyInterpreter) execNode(n Node, f *frame) any {
	switch n.Type() {
	case "ImportStatement":
		i.execImport(n.(*ImportStatement))
		return nil
	case "VarStatement":
		i.execVar(n.(*VarStatement), f)
		return nil
	case "CallExpression":
		return i.execFuncCall(n.(*CallExpression), f)
	case "ReferenceExpression":
		return i.resolveRef(n.(*ReferenceExpression), f)
	case "StringLiteral":
		return n.(*StringLiteral).Value
	case "NumberLiteral":
		return n.(*NumberLiteral).Value
	case "BooleanLiteral":
		return n.(*BooleanLiteral).Value
	case "ListLiteral":
		return i.evalList(n.(*ListLiteral), f)
	case "HashLiteral":
		return i.evalHash(n.(*HashLiteral), f)
	case "FuncLiteral":
		return i.defineFunc(n.(*FuncLiteral), f)
	case "ExportStatement":
		// NOTE: we don't care about the exports of the currently executing program
		return nil
	case "SeqExpression":
		return i.execSeq(n.(*SeqExpression), f)
	case "MatchExpression":
		return i.evalMatch(n.(*MatchExpression), f)
	case "ChainExpression":
		return i.defineChain(n.(*ChainExpression), f)
	case "AsyncExpression":
		return i.execAsync(n.(*AsyncExpression), f)
	}

	panic(fmt.Sprintf("failed to execute: unexpected node %v", n))
}

func (i *toyInterpreter) evalList(list *ListLiteral, f *frame) any {
	results := []any{}
	for _, el := range list.Elements {
		results = append(results, i.execNode(el, f))
	}

	return results
}

func (i *toyInterpreter) evalHash(hash *HashLiteral, f *frame) any {
	results := map[string]any{}
	for key, el := range hash.Elements {
		results[key] = i.execNode(el, f)
	}

	return results
}

func (i *toyInterpreter) execModule(alias string, p *ProgramStatement, f *frame) *frame {
	var exports *frame
	for _, s := range p.Body {
		switch s.Type() {
		case "ExportStatement":
			exports = i.execExport(s.(*ExportStatement), f)
		default:
			i.execNode(s, f)
		}
	}

	if exports == nil {
		panic(fmt.Sprintf("module has not exports %s", alias))
	}

	return exports
}

func (i *toyInterpreter) execVar(v *VarStatement, f *frame) {
	if f == nil {
		panic("unexpected nil stackframe")
	}

	for k, v := range v.Vars {
		resolvedValue := i.execNode(v, f)
		f.set(k, resolvedValue)
	}
}

func (i *toyInterpreter) execImport(imprt *ImportStatement) {
	// NOTE: imports are always in the global scope
	// TODO: fetch the lib, parse it, execute it,
	// extract the exports and import them here

	// HACK: inject hard coded deps
	for alias, path := range imprt.Imports {
		f := newFrame(i.globals)
		switch alias {
		case "http":
			f.set("get", httpGet)
			i.globals.set("http", f)
		case "json":
			f.set("parse", jsonParse)
			i.globals.set("json", f)
		case "stdio":
			f.set("print", stdioPrint)
			f.set("read", stdioRead)
			f.set("string", stdioBuildStr)

			i.globals.set("stdio", f)
		default:
			fileBytes, err := os.ReadFile(path)
			if err != nil {
				panic(fmt.Sprintf("failed to read import %s", path))
			}
			scanner := NewScanner(string(fileBytes))
			tokens, err := scanner.ScanTokens()
			if err != nil {
				panic(fmt.Sprintf("failed to scan import %s", path))
			}

			parser := NewParser(tokens)
			ast, hasError := parser.Parse()
			if hasError {
				panic(fmt.Sprintf("failed to parse import %s\n\n%s", path, ast.String()))
			}

			f = i.execModule(alias, &ast, i.globals)
			i.globals.set(alias, f)
		}
	}
}

func (i *toyInterpreter) execExport(export *ExportStatement, f *frame) *frame {
	output := newFrame(nil)
	for _, e := range export.Exports {
		expRef := e.(*ReferenceExpression)
		expVal := i.execNode(e, f)
		output.set(expRef.RefName, expVal)
	}

	return output
}

func (i *toyInterpreter) defineFunc(fn *FuncLiteral, f *frame) funcType {
	return func(a ...any) any {
		innerFrame := newFrame(f)
		for idx, paramName := range fn.Params {
			if len(a) > idx {
				innerFrame.set(paramName, a[idx])
			}
		}

		var lastResult any
		for _, funcExpr := range fn.Body {
			lastResult = i.execNode(funcExpr, innerFrame)
		}
		return lastResult
	}
}

func (i *toyInterpreter) execFuncCall(c *CallExpression, f *frame) any {
	callee := i.execNode(c.Callee, f)

	args := []inode{}
	for _, arg := range c.Args {
		args = append(args, i.execNode(arg, f))
	}

	fn, ok := callee.(funcType)
	if !ok {
		panic("failed to cast function in call expression")
	}

	return fn(args...)
}

func (i toyInterpreter) resolveRef(r *ReferenceExpression, f *frame) any {
	var (
		v  inode
		ok bool
	)

	switch r.RefType {
	case REF_TYPE_BUILTIN:
		v, ok = i.globals.get(r.RefName)
	case REF_TYPE_DECLARED:
		v, ok = f.get(r.RefName)
	case REF_TYPE_IMPORTED:
		impRef := strings.Split(r.RefName, ".")
		if len(impRef) != 2 {
			panic(fmt.Sprintf("malformed imported ref: %s", r.RefName))
		}
		v, ok := i.globals.get(impRef[0])
		if ok {
			v, ok = v.(*frame).get(impRef[1])
			if ok {
				return v
			}
		}
	}

	if !ok {
		panic(fmt.Sprintf("failed to resolve ref %s (%s)", r.RefName, r.RefType))
	}
	vN, ok := v.(Node)
	if ok {
		return i.execNode(vN, f)
	}

	return v
}

func (i *toyInterpreter) execSeq(s *SeqExpression, f *frame) any {
	var lastValue any
	for _, e := range s.Expressions {
		lastValue = i.execNode(e, f)
	}

	return lastValue
}

func (i *toyInterpreter) defineChain(c *ChainExpression, f *frame) any {
	return func(a ...any) any {
		var lastResult any
		switch c.Expressions[0].Type() {
		case "FuncLiteral":
			fn1 := c.Expressions[0].(*FuncLiteral)

			innerFrame := newFrame(f)
			for idx, paramName := range fn1.Params {
				if len(a) > idx {
					innerFrame.set(paramName, a[idx])
				}
			}

			lastResult = i.execFuncCall(&CallExpression{fn1, []Node{}}, innerFrame)
		case "ReferenceExpression":
			ref := c.Expressions[0].(*ReferenceExpression)
			refVal := i.resolveRef(ref, f)

			fn1, ok := refVal.(funcType)
			if !ok {
				panic(fmt.Sprintf("expected func in chain: %s", refVal))
			}

			lastResult = fn1(a...)
		default:
			panic(fmt.Sprintf("unexpected expression in chain: %s ", c.Expressions[0]))
		}

		for _, e := range c.Expressions[1:] {
			switch e.Type() {
			case "FuncLiteral":
				fn1 := e.(*FuncLiteral)

				innerFrame := newFrame(f)
				if len(fn1.Params) > 0 {
					innerFrame.set(fn1.Params[0], lastResult)
				}

				lastResult = i.execFuncCall(&CallExpression{fn1, []Node{}}, innerFrame)
			case "ReferenceExpression":
				ref := e.(*ReferenceExpression)
				refVal := i.resolveRef(ref, f)
				fn1, ok := refVal.(funcType)
				if !ok {
					panic(fmt.Sprintf("expected func in chain: %s", refVal))
				}

				lastResult = fn1(lastResult)
			default:
				panic(fmt.Sprintf("unexpected expression in chain: %s ", e))
			}
		}

		return lastResult
	}
}

func (i *toyInterpreter) evalMatch(m *MatchExpression, f *frame) any {
	mf := newFrame(f)

	expected := i.execNode(m.Cond, f)

	mf.set("value", expected)

	cases := map[any]any{}
	for cc, ca := range m.Cases {
		cr := i.execNode(cc, mf)
		if res, ok := cr.(bool); cc.Type() != "BooleanLiteral" && ok && res {
			return i.execNode(ca, mf)
		}
		cases[cr] = i.execNode(ca, mf)
	}

	return cases[expected]
}

func (i *toyInterpreter) execAsync(a *AsyncExpression, f *frame) any {
	ch := make(chan any)

	go func() {
		for _, e := range a.Expressions {
			ch <- i.execNode(e, f)
		}
		close(ch)
	}()

	return ch
}
