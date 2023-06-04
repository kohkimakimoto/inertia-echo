package v8go

import (
	"fmt"
	"github.com/kohkimakimoto/inertia-echo"
	"os"
	"rogchap.com/v8go"
	"runtime"
	"sync"
)

type SsrEngine struct {
	source              string
	filename            string
	compiledScriptCache *v8go.CompilerCachedData
	pool                sync.Pool
}

func NewSsrEngine() *SsrEngine {
	e := &SsrEngine{}
	e.pool.New = func() interface{} {
		// It is inspired by https://github.com/rogchap/v8go/issues/105#issuecomment-867332376
		iso := v8go.NewIsolate()
		runtime.SetFinalizer(iso, e.finalizer)
		return iso
	}
	return e
}

func (e *SsrEngine) finalizer(iso *v8go.Isolate) {
	iso.Dispose()
	runtime.SetFinalizer(iso, nil)
}

func (e *SsrEngine) LoadFile(filename string) error {
	f, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	e.source = string(f)
	e.filename = filename

	if err := e.initCompiledScriptCache(); err != nil {
		return err
	}
	return nil
}

func (e *SsrEngine) initCompiledScriptCache() error {
	// This isolates is used for compiling script only.
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	unboundScript, err := iso.CompileUnboundScript(e.source, e.filename, v8go.CompileOptions{})
	if err != nil {
		return err
	}
	e.compiledScriptCache = unboundScript.CreateCodeCache()

	return nil
}

func (e *SsrEngine) Render(p *inertia.Page) (*inertia.SsrResponse, error) {
	iso := e.acquireIsolate()
	defer e.freeIsolate(iso)

	printfn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		fmt.Printf("%v", info.Args())
		return nil
	})
	global := v8go.NewObjectTemplate(iso)
	if err := global.Set("print", printfn); err != nil {
		return nil, err
	}

	ctx := v8go.NewContext(iso, global)
	if _, err := ctx.RunScript(e.source, e.filename); err != nil {
		return nil, err
	}

	return nil, nil
}

func (e *SsrEngine) freeIsolate(iso *v8go.Isolate) {
	e.pool.Put(iso)
}

func (e *SsrEngine) acquireIsolate() *v8go.Isolate {
	return e.pool.Get().(*v8go.Isolate)
}
