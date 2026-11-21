// The Go half of Vector's optional observability audit. This tool only needs the standard library.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

type finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type source struct {
	fset      *token.FileSet
	file      *ast.File
	imports   map[string]string
	factories map[string]bool
	findings  []finding
	reported  map[token.Pos]bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run() error {
	var paths []string
	if err := json.NewDecoder(os.Stdin).Decode(&paths); err != nil {
		return err
	}
	report := []finding{}
	for _, path := range paths {
		findings, err := scan(path, nil)
		if err != nil {
			return err
		}
		report = append(report, findings...)
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}

func scan(path string, code any) ([]finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, code, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if ast.IsGenerated(file) {
		return nil, nil
	}
	s := source{
		fset: fset, file: file, imports: make(map[string]string),
		factories: make(map[string]bool), reported: make(map[token.Pos]bool),
	}
	for _, spec := range file.Imports {
		importPath, _ := strconv.Unquote(spec.Path.Value)
		alias := importPath[strings.LastIndex(importPath, "/")+1:]
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		s.imports[alias] = importPath
	}
	s.collectFactories()
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			check, ok := node.(*ast.IfStmt)
			if ok {
				if cause := checkedError(check.Cond); cause != "" && !s.isCancellation(check.Cond, cause) {
					s.scanBlock(fn, check.Body.List, cause, none)
				}
			}
			return true
		})
	}
	return s.findings, nil
}

func ident(expr ast.Expr) string {
	if node, ok := expr.(*ast.Ident); ok {
		return node.Name
	}
	return ""
}

func checkedError(expr ast.Expr) string {
	condition, ok := ast.Unparen(expr).(*ast.BinaryExpr)
	if !ok {
		return ""
	}
	if condition.Op == token.LAND {
		left, right := checkedError(condition.X), checkedError(condition.Y)
		if left == "" {
			return right
		}
		if right == "" || left == right {
			return left
		}
		return ""
	}
	if condition.Op != token.NEQ {
		return ""
	}
	left, right := ident(condition.X), ident(condition.Y)
	if left == "nil" {
		left, right = right, left
	}
	if right == "nil" && (strings.HasSuffix(strings.ToLower(left), "err") ||
		strings.HasSuffix(left, "Error") || left == "cause") {
		return left
	}
	return ""
}

func receiver(fn *ast.FuncDecl) (variable, typeName string) {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return "", ""
	}
	field := fn.Recv.List[0]
	variable = ""
	if len(field.Names) == 1 {
		variable = field.Names[0].Name
	}
	switch expr := field.Type.(type) {
	case *ast.StarExpr:
		return variable, ident(expr.X)
	default:
		return variable, ident(expr)
	}
}

func (s *source) collectFactories() {
	// These are one-statement, same-file constructors that emit generic typed errors.
	for _, declaration := range s.file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil || len(fn.Body.List) != 1 || fn.Type.Results == nil ||
			len(fn.Type.Results.List) != 1 || ident(fn.Type.Results.List[0].Type) != "error" {
			continue
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if ok && s.serverReturn(fn, ret, false) {
			_, typeName := receiver(fn)
			s.factories[typeName+"."+fn.Name.Name] = true
		}
	}
}

type evidence int

const (
	none evidence = iota
	weak
	contextual
)

func (s *source) scanBlock(fn *ast.FuncDecl, statements []ast.Stmt, cause string, observed evidence) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.ReturnStmt:
			if s.serverReturn(fn, stmt, true) {
				s.record(stmt.Pos(), observed)
			}
			return
		case *ast.ExprStmt:
			call, ok := stmt.X.(*ast.CallExpr)
			if !ok {
				continue
			}
			observed = max(observed, s.signal(call, cause))
			if s.serverWrite(call) {
				s.record(stmt.Pos(), observed)
			}
		case *ast.IfStmt:
			if checkedError(stmt.Cond) == "" && !s.isCancellation(stmt.Cond, cause) {
				s.scanBlock(fn, stmt.Body.List, cause, observed)
			}
			if stmt.Else != nil {
				s.scanBlock(fn, []ast.Stmt{stmt.Else}, cause, observed)
			}
		case *ast.BlockStmt:
			s.scanBlock(fn, stmt.List, cause, observed)
		}
	}
}

func (s *source) isCancellation(expr ast.Expr, cause string) bool {
	expr = ast.Unparen(expr)
	if pair, ok := expr.(*ast.BinaryExpr); ok {
		switch pair.Op {
		case token.LAND:
			return s.isCancellation(pair.X, cause) || s.isCancellation(pair.Y, cause)
		case token.EQL:
			return (ident(pair.X) == cause && s.isContextCanceled(pair.Y)) ||
				(ident(pair.Y) == cause && s.isContextCanceled(pair.X))
		}
		return false
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || ident(call.Args[0]) != cause {
		return false
	}
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || s.imports[ident(fn.X)] != "errors" || fn.Sel.Name != "Is" {
		return false
	}
	return s.isContextCanceled(call.Args[1])
}

func (s *source) isContextCanceled(expr ast.Expr) bool {
	target, ok := ast.Unparen(expr).(*ast.SelectorExpr)
	return ok && s.imports[ident(target.X)] == "context" && target.Sel.Name == "Canceled"
}

func (s *source) record(pos token.Pos, observed evidence) {
	if observed == contextual || s.reported[pos] {
		return
	}
	s.reported[pos] = true
	position := s.fset.Position(pos)
	rule := "unclassified-server-failure"
	message := "This branch maps a checked error to a generic 5xx without operation-level diagnostics. Use a safe operation, stage, and error class."
	if observed == weak {
		rule = "weak-server-failure-diagnostic"
		message = "This branch emits a diagnostic before a generic 5xx but does not connect the diagnostic to the checked error."
	}
	s.findings = append(s.findings, finding{position.Filename, position.Line, position.Column, rule, message})
}

func (s *source) isServerStatus(expr ast.Expr) bool {
	switch value := ast.Unparen(expr).(type) {
	case *ast.BasicLit:
		status, err := strconv.Atoi(value.Value)
		return value.Kind == token.INT && err == nil && status >= 500 && status < 600
	case *ast.SelectorExpr:
		if s.imports[ident(value.X)] != "net/http" {
			return false
		}
		switch value.Sel.Name {
		case "StatusInternalServerError", "StatusNotImplemented", "StatusBadGateway", "StatusServiceUnavailable",
			"StatusGatewayTimeout", "StatusHTTPVersionNotSupported", "StatusVariantAlsoNegotiates",
			"StatusInsufficientStorage", "StatusLoopDetected", "StatusNotExtended", "StatusNetworkAuthenticationRequired":
			return true
		}
	}
	return false
}

func (s *source) serverReturn(fn *ast.FuncDecl, ret *ast.ReturnStmt, useFactories bool) bool {
	if len(ret.Results) > 1 && s.isServerStatus(ret.Results[0]) {
		return true
	}
	for _, result := range ret.Results {
		switch value := ast.Unparen(result).(type) {
		case *ast.CallExpr:
			if s.serverConstructor(fn, value, useFactories) {
				return true
			}
		case *ast.UnaryExpr:
			if literal, ok := value.X.(*ast.CompositeLit); ok && value.Op == token.AND && s.isGeneratedFailure(literal) {
				return true
			}
		}
	}
	return false
}

func (s *source) serverConstructor(fn *ast.FuncDecl, value *ast.CallExpr, useFactories bool) bool {
	name := ident(value.Fun)
	factoryName := "." + name
	if sel, ok := value.Fun.(*ast.SelectorExpr); ok {
		name = sel.Sel.Name
		variable, receiverType := receiver(fn)
		if variable != "" && ident(sel.X) == variable {
			factoryName = receiverType + "." + name
		}
	}
	lower := strings.ToLower(name)
	return (useFactories && len(value.Args) == 0 && s.factories[factoryName]) ||
		(len(value.Args) > 0 && s.isServerStatus(value.Args[0]) &&
			(strings.HasSuffix(lower, "error") || strings.HasSuffix(lower, "failure")))
}

func (s *source) isGeneratedFailure(literal *ast.CompositeLit) bool {
	sel, ok := literal.Type.(*ast.SelectorExpr)
	if !ok || !strings.Contains(s.imports[ident(sel.X)], "/gen/") {
		return false
	}
	if strings.HasSuffix(sel.Sel.Name, "InternalError") || strings.HasSuffix(sel.Sel.Name, "InternalServerError") {
		return true
	}
	for _, entry := range literal.Elts {
		field, ok := entry.(*ast.KeyValueExpr)
		if !ok || ident(field.Key) != "Name" {
			continue
		}
		if value, ok := field.Value.(*ast.BasicLit); ok && value.Kind == token.STRING {
			name, _ := strconv.Unquote(value.Value)
			status, err := strconv.Atoi(strings.TrimPrefix(name, "status_"))
			return name == "internal_error" || name == "internal_server_error" ||
				(strings.HasPrefix(name, "status_") && err == nil && status >= 500 && status < 600)
		}
	}
	return false
}

func (s *source) serverWrite(call *ast.CallExpr) bool {
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return (fn.Sel.Name == "WriteHeader" && len(call.Args) == 1 && s.isServerStatus(call.Args[0])) ||
		(s.imports[ident(fn.X)] == "net/http" && fn.Sel.Name == "Error" &&
			len(call.Args) == 3 && s.isServerStatus(call.Args[2]))
}

func (s *source) signal(call *ast.CallExpr, cause string) evidence {
	if name := ident(call.Fun); name == "logOperationError" || name == "workItemLogError" || name == "cycleLifecycleLogError" ||
		name == "moduleLifecycleLogError" || name == "logPublicProjectMetadataCompatibilityError" {
		if len(call.Args) > 0 && mentions(call.Args[len(call.Args)-1], cause) {
			return contextual
		}
		return weak
	}
	fn, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return none
	}
	pkg := s.imports[ident(fn.X)]
	method := fn.Sel.Name
	if (pkg == "log/slog" && (method == "Error" || method == "ErrorContext" || method == "Warn" || method == "WarnContext")) ||
		(pkg == "github.com/CaliLuke/loom/clue/log" && (method == "Error" || method == "Warn" || method == "Print")) ||
		method == "RecordError" || method == "RecordException" {
		for _, arg := range call.Args {
			if mentions(arg, cause) {
				return contextual
			}
		}
		return weak
	}
	return none
}

func mentions(expr ast.Expr, cause string) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		switch candidate := node.(type) {
		case *ast.Ident:
			found = found || candidate.Name == cause
		case *ast.SelectorExpr:
			// A field or method name may match the checked variable by coincidence.
			found = found || mentions(candidate.X, cause)
			return false
		case *ast.FuncLit:
			return false
		}
		return !found
	})
	return found
}
