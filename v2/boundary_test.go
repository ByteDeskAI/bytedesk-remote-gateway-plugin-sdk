package pluginsdk_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
	"github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2/transport/natsconn"
)

// The operator's first rule, enforced on this side of the contract: all NATS
// functionality is abstracted behind the SDK. github.com/nats-io/* is imported
// by exactly one package — v2/transport/natsconn — and no nats type appears in
// any exported signature anywhere, including natsconn's own.
//
// The two halves fail for different reasons and neither alone is the fence: the
// source scan catches an import, the reflect walk catches a type that arrived
// through one. An import can exist with no exported type, and an exported type
// can arrive through a re-export.
//
// Both halves are therefore accompanied by a test that breaks them on purpose:
// TestImportScanDetectsAForbiddenImport and TestReflectWalkDetectsAForeignType.
// A fence that cannot detect its own vacuity is not a fence.

const (
	brokerImport = "github.com/nats-io/"
	// transportPkg is the ONLY directory in this module that may import it.
	transportPkg = "transport/natsconn"
)

// TestOnlyNatsconnImportsBroker scans every Go file in the module, tests
// included, and fails on a broker import outside the transport.
//
// Tests are not exempt. A test that imports nats.go to build a fixture is how
// the dependency arrives in the next non-test file: the author sees the import
// already present in the package and stops thinking about it.
func TestOnlyNatsconnImportsBroker(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()
	scanned, inTransport := 0, 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		allowed := strings.HasPrefix(filepath.ToSlash(rel), transportPkg+"/")
		scanned++
		for _, imp := range fileImports(t, fset, path) {
			if !strings.HasPrefix(imp, brokerImport) {
				continue
			}
			if !allowed {
				t.Errorf("%s imports %q: only %s may reach the broker", rel, imp, transportPkg)
				continue
			}
			inTransport++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 10 {
		t.Fatalf("scanned only %d Go files; the walk is not reaching the module", scanned)
	}
	// The transport MUST import the broker. If it stopped, this test would go
	// on passing while the rule it enforces had become meaningless.
	if inTransport == 0 {
		t.Fatalf("no file under %s imports %s: the scan is passing because there is nothing to find", transportPkg, brokerImport)
	}
	t.Logf("scanned %d Go files; %d broker imports, all under %s", scanned, inTransport, transportPkg)
}

// TestImportScanDetectsAForbiddenImport breaks the scanner on purpose: it runs
// the same parse over a file written outside the transport that imports the
// broker, and requires the import to be seen.
func TestImportScanDetectsAForbiddenImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "violation.go")
	src := "package p\n\nimport _ \"github.com/nats-io/nats.go\"\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	imports := fileImports(t, token.NewFileSet(), path)
	found := false
	for _, imp := range imports {
		if strings.HasPrefix(imp, brokerImport) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the import scan missed %s in %v: TestOnlyNatsconnImportsBroker proves nothing", brokerImport, imports)
	}
}

// TestGoModRequiresBrokerOnce records the other half of the dependency story:
// this module DOES depend on nats.go, deliberately and in one place. The
// gateway's SD module must not (its own TestGoModRequiresNoBroker holds that
// line); this one is the adapter, so the dependency is expected and the test
// says so rather than leaving a reader to guess.
func TestGoModRequiresBrokerOnce(t *testing.T) {
	mod, err := os.ReadFile(filepath.Join(moduleRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), brokerImport) {
		t.Fatalf("go.mod does not require %s; the transport cannot be built", brokerImport)
	}
	if strings.Contains(string(mod), "\nreplace ") || strings.HasPrefix(string(mod), "replace ") {
		t.Fatal("go.mod must not carry a replace directive")
	}
}

// TestNoBrokerTypeInPublicSignatures walks every exported type of this module —
// struct fields, method parameters and results, interface methods, and
// everything they point at — and fails if any of them was declared under
// nats-io.
//
// Unexported struct fields are NOT walked, and that is the whole point of the
// design rather than a gap in the test: natsconn.Conn holds a *nats.Conn
// privately, which is exactly how the abstraction is supposed to work. The rule
// is about signatures, so the walk is about signatures.
func TestNoBrokerTypeInPublicSignatures(t *testing.T) {
	w := newWalker(isBrokerPkg)
	for _, rt := range publicTypes() {
		w.walk(rt, rt.String())
	}
	for _, f := range w.findings {
		t.Errorf("%s reaches %s, declared in %s", f.where, f.typ, f.pkg)
	}
	if w.visited() < 40 {
		t.Fatalf("walked only %d types; the walk is not reaching the API", w.visited())
	}
	t.Logf("walked %d distinct types across %d roots", w.visited(), len(publicTypes()))
}

// TestReflectWalkDetectsAForeignType breaks the walk on purpose, by treating
// the SDK's own bus package as foreign over types that reach one by four
// different routes. A walker that handles only struct fields is caught here.
func TestReflectWalkDetectsAForeignType(t *testing.T) {
	foreign := func(pkg string) bool { return strings.HasSuffix(pkg, "/v2/bus") }

	for _, tc := range []struct {
		name string
		rt   reflect.Type
	}{
		{"through an exported struct field", reflect.TypeOf(probeField{})},
		{"through a method result", reflect.TypeOf(probeMethod{})},
		{"through an interface method parameter", reflect.TypeOf((*probeIface)(nil)).Elem()},
		{"through a slice of a map value", reflect.TypeOf(probeNested{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWalker(foreign)
			w.walk(tc.rt, tc.rt.String())
			if len(w.findings) == 0 {
				t.Fatalf("walker missed a foreign type reachable %s: the real gate is vacuous", tc.name)
			}
		})
	}

	// A type that reaches nothing foreign must produce nothing, or the walker
	// is simply flagging everything it sees.
	w := newWalker(foreign)
	w.walk(reflect.TypeOf(probeClean{}), "probeClean")
	if len(w.findings) != 0 {
		t.Fatalf("walker reported %d findings for a clean type: it flags indiscriminately", len(w.findings))
	}

	// And an UNEXPORTED field holding a foreign type must NOT be flagged: that
	// is how natsconn.Conn holds its connection, and a walk that failed on it
	// would make the abstraction impossible to implement.
	w = newWalker(foreign)
	w.walk(reflect.TypeOf(probePrivate{}), "probePrivate")
	if len(w.findings) != 0 {
		t.Fatalf("walker flagged an unexported field: the rule is about signatures, not storage")
	}
}

type probeField struct{ S pluginsdk.Subject }

type probeMethod struct{}

func (probeMethod) Subject() pluginsdk.Subject { return "" }

type probeIface interface{ Take(pluginsdk.Subject) }

type probeNested struct {
	M map[string][]pluginsdk.Subject
}

type probeClean struct{ N int }

type probePrivate struct{ s pluginsdk.Subject }

func (p probePrivate) N() int { return len(p.s) }

// isBrokerPkg is the real predicate.
func isBrokerPkg(pkg string) bool { return strings.Contains(pkg, "nats-io") }

type walker struct {
	foreign  func(pkg string) bool
	seen     map[reflect.Type]bool
	findings []finding
}

type finding struct {
	where string
	typ   string
	pkg   string
}

func newWalker(foreign func(string) bool) *walker {
	return &walker{foreign: foreign, seen: map[reflect.Type]bool{}}
}

func (w *walker) visited() int { return len(w.seen) }

func (w *walker) walk(t reflect.Type, where string) {
	if t == nil || w.seen[t] {
		return
	}
	w.seen[t] = true

	if p := t.PkgPath(); p != "" && w.foreign(p) {
		w.findings = append(w.findings, finding{where: where, typ: t.String(), pkg: p})
		return
	}

	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan:
		w.walk(t.Elem(), where+" -> elem")
	case reflect.Map:
		w.walk(t.Key(), where+" -> key")
		w.walk(t.Elem(), where+" -> value")
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			w.walk(f.Type, where+"."+f.Name)
		}
	case reflect.Func:
		w.walkFunc(t, where)
	case reflect.Interface:
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			w.walkFunc(m.Type, where+"."+m.Name)
		}
	}

	w.walkMethods(t, where)
	if t.Kind() != reflect.Ptr && t.Kind() != reflect.Interface {
		w.walkMethods(reflect.PointerTo(t), where)
	}
}

// walkMethods visits the exported method set. reflect reports only exported
// methods for a non-interface type, which is the set the rule is about.
func (w *walker) walkMethods(t reflect.Type, where string) {
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		w.walkFunc(m.Type, where+"."+m.Name)
	}
}

func (w *walker) walkFunc(ft reflect.Type, where string) {
	if ft.Kind() != reflect.Func {
		return
	}
	for i := 0; i < ft.NumIn(); i++ {
		w.walk(ft.In(i), where+" arg"+strconv.Itoa(i))
	}
	for i := 0; i < ft.NumOut(); i++ {
		w.walk(ft.Out(i), where+" ret"+strconv.Itoa(i))
	}
}

// publicPackages are the directories whose exported types form this module's
// public signature.
var publicPackages = []string{".", "transport/natsconn", "v1compat"}

// publicTypes is the registry the walk starts from.
//
// Every alias is registered through THIS module's own name for it, so the
// registry cannot drift from the alias list: `reflect.TypeOf((*Bus)(nil))`
// resolves to whatever Bus is an alias for today.
//
// TestEveryExportedTypeIsWalked is what makes a hand-written list trustworthy:
// it reads the same packages with go/doc and fails on any exported type the
// registry does not carry.
func publicTypes() []reflect.Type {
	iface := func(p any) reflect.Type { return reflect.TypeOf(p).Elem() }
	return []reflect.Type{
		// Addressing and messages.
		reflect.TypeOf(pluginsdk.Subject("")),
		reflect.TypeOf(pluginsdk.Pattern("")),
		reflect.TypeOf(pluginsdk.Headers{}),
		reflect.TypeOf(pluginsdk.Msg{}),
		reflect.TypeOf(pluginsdk.Caller{}),
		reflect.TypeOf(pluginsdk.Fault{}),
		reflect.TypeOf(pluginsdk.Handler(nil)),
		iface((*pluginsdk.Bus)(nil)),
		iface((*pluginsdk.Subscription)(nil)),
		// Identity and capability.
		reflect.TypeOf(pluginsdk.Capabilities{}),
		reflect.TypeOf(pluginsdk.Identity{}),
		reflect.TypeOf(pluginsdk.Grants{}),
		reflect.TypeOf(pluginsdk.GrantKind("")),
		reflect.TypeOf(pluginsdk.Role("")),
		reflect.TypeOf(pluginsdk.Lease{}),
		// Services and durable surfaces.
		iface((*pluginsdk.Service)(nil)),
		reflect.TypeOf(pluginsdk.ServiceSpec{}),
		reflect.TypeOf(pluginsdk.EndpointSpec{}),
		reflect.TypeOf(pluginsdk.ServiceInfo{}),
		iface((*pluginsdk.Streams)(nil)),
		iface((*pluginsdk.KV)(nil)),
		iface((*pluginsdk.Bucket)(nil)),
		iface((*pluginsdk.Objects)(nil)),
		iface((*pluginsdk.Scheduler)(nil)),
		iface((*pluginsdk.Trace)(nil)),
		// The plugin contract.
		iface((*pluginsdk.Plugin)(nil)),
		iface((*pluginsdk.Bound)(nil)),
		reflect.TypeOf(pluginsdk.Base{}),
		reflect.TypeOf(pluginsdk.Binding{}),
		reflect.TypeOf(pluginsdk.Manifest{}),
		reflect.TypeOf(pluginsdk.ConsentCapability{}),
		iface((*pluginsdk.Logger)(nil)),
		iface((*pluginsdk.Profiler)(nil)),
		iface((*pluginsdk.HTTPPlugin)(nil)),
		iface((*pluginsdk.HealthContributor)(nil)),
		iface((*pluginsdk.Validator)(nil)),
		iface((*pluginsdk.Readier)(nil)),
		iface((*pluginsdk.ActivationChecker)(nil)),
		iface((*pluginsdk.Draining)(nil)),
		iface((*pluginsdk.DataVersioned)(nil)),
		reflect.TypeOf(pluginsdk.DataVersion("")),
		// Manifest sub-contracts.
		reflect.TypeOf(pluginsdk.ProtocolRequirements{}),
		reflect.TypeOf(pluginsdk.Permissions{}),
		reflect.TypeOf(pluginsdk.ServiceDecl{}),
		reflect.TypeOf(pluginsdk.EndpointDecl{}),
		reflect.TypeOf(pluginsdk.StreamDecl{}),
		reflect.TypeOf(pluginsdk.KVDecl{}),
		reflect.TypeOf(pluginsdk.ObjectDecl{}),
		reflect.TypeOf(pluginsdk.ManifestIdentity{}),
		reflect.TypeOf(pluginsdk.Images{}),
		reflect.TypeOf(pluginsdk.Support{}),
		reflect.TypeOf(pluginsdk.Publisher{}),
		reflect.TypeOf(pluginsdk.StaticMount{}),
		reflect.TypeOf(pluginsdk.Point("")),
		reflect.TypeOf(pluginsdk.Registrar{}),
		reflect.TypeOf(pluginsdk.Descriptor{}),
		reflect.TypeOf(pluginsdk.ProjectViewContribution{}),
		reflect.TypeOf(pluginsdk.DirectoryContextActionContribution{}),
		reflect.TypeOf(pluginsdk.ProjectDirectoryContext{}),
		reflect.TypeOf(pluginsdk.DirectoryContextActionEligibilityRequest{}),
		reflect.TypeOf(pluginsdk.DirectoryContextActionEligibilityResult{}),
		reflect.TypeOf(pluginsdk.DirectoryContextActionWizardContext{}),
		// This module's own declarations.
		reflect.TypeOf(pluginsdk.Config{}),
		reflect.TypeOf(pluginsdk.PluginConfig{}),
		reflect.TypeOf(pluginsdk.NegotiateRequest{}),
		reflect.TypeOf(pluginsdk.HostCapabilities{}),
		reflect.TypeOf(pluginsdk.PackResult{}),
		// The transport. These are the types the rule is really about.
		reflect.TypeOf(natsconn.Options{}),
		reflect.TypeOf(natsconn.Conn{}),
	}
}

// TestEveryExportedTypeIsWalked reads every public package with go/doc and
// fails on any exported type name the registry does not carry.
//
// Without it the walk would be exactly as complete as whoever last remembered
// to update a list.
func TestEveryExportedTypeIsWalked(t *testing.T) {
	registered := map[string]bool{}
	for _, rt := range publicTypes() {
		registered[typeKey(rt)] = true
	}
	declared, aliases := 0, 0
	var missing []string
	for _, pkgDir := range publicPackages {
		for _, d := range exportedTypes(t, pkgDir) {
			declared++
			want := shortPkg(pkgDir) + "." + d.name
			if d.alias != "" {
				aliases++
				want = d.alias
			}
			if !registered[want] {
				missing = append(missing, pkgDir+"."+d.name+" (walk needs "+want+")")
			}
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("%s is exported but the boundary walk does not reach it; add it to publicTypes()", m)
	}
	if declared < 40 {
		t.Fatalf("go/doc found only %d exported types across %v; the inventory is not reading the packages", declared, publicPackages)
	}
	t.Logf("go/doc resolved %d exported types (%d of them aliases); %d registered", declared, aliases, len(registered))
}

type typeDecl struct {
	name  string
	alias string
}

func exportedTypes(t *testing.T, pkgDir string) []typeDecl {
	t.Helper()
	dir := filepath.Join(moduleRoot(t), pkgDir)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("public package %q is missing: %v", pkgDir, err)
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var out []typeDecl
	for _, p := range pkgs {
		for _, f := range p.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ast.IsExported(ts.Name.Name) {
						continue
					}
					d := typeDecl{name: ts.Name.Name}
					if ts.Assign.IsValid() {
						d.alias = qualified(ts.Type)
						if d.alias == "" {
							t.Errorf("%s.%s is an alias this check cannot resolve; keep aliases simple", pkgDir, ts.Name.Name)
						}
					}
					out = append(out, d)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func qualified(e ast.Expr) string {
	if t, ok := e.(*ast.SelectorExpr); ok {
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name + "." + t.Sel.Name
		}
	}
	return ""
}

func typeKey(rt reflect.Type) string {
	p := rt.PkgPath()
	// A /vN module root's last path element is the major-version suffix, not
	// the package name: this module is package pluginsdk at .../plugin-sdk/v2.
	// go/doc reports the name, reflect reports the path, and the two have to
	// meet somewhere.
	if strings.HasSuffix(p, "-plugin-sdk/v2") {
		return "pluginsdk." + typeName(rt)
	}
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	return p + "." + typeName(rt)
}

// typeName drops a generic instantiation's arguments: the contract is the
// generic type, not any one instantiation of it.
func typeName(rt reflect.Type) string {
	n := rt.Name()
	if i := strings.IndexByte(n, '['); i >= 0 {
		n = n[:i]
	}
	return n
}

// shortPkg maps a directory to the package name go/doc and reflect agree on.
func shortPkg(pkgDir string) string {
	switch pkgDir {
	case ".":
		return "pluginsdk"
	default:
		return filepath.Base(pkgDir)
	}
}

func fileImports(t *testing.T, fset *token.FileSet, path string) []string {
	t.Helper()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	out := make([]string, 0, len(f.Imports))
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
