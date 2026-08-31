package script

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/d5/tengo/v2/stdlib"
)

const resultVarName = "__repl_res__"

// REPL manages an interactive Tengo scripting environment with persistent state.
type REPL struct {
	mu          sync.Mutex
	modules     *tengo.ModuleMap
	symbolTable *tengo.SymbolTable
	globals     []tengo.Object
	resIndex          int
	onExit            func()
	onStartMap        func(string)
	onPlayMusic       func(string)
	onStopMusic       func()
	onSetMusicVolume  func(float64)
	onGetMusicTrack   func() string
	printFunc         func(string)
}

// NewREPL creates a new Tengo REPL environment.
// It registers standard library modules, replaces "fmt" with one that redirects to printFunc,
// and registers the custom "game" module.
func NewREPL(onExit func(), printFunc func(string)) *REPL {
	r := &REPL{
		symbolTable: tengo.NewSymbolTable(),
		globals:     make([]tengo.Object, tengo.GlobalsSize),
		onExit:      onExit,
		printFunc:   printFunc,
	}

	// Reserve a slot in symbol table for capturing expression results
	sym := r.symbolTable.Define(resultVarName)
	r.resIndex = sym.Index

	startMapFunc := &tengo.UserFunction{
		Name: "StartMap",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("StartMap: missing map_name argument")
			}
			mapNameStr, ok := args[0].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("StartMap: expected string for map_name, found %s", args[0].TypeName())
			}
			if r.onStartMap != nil {
				r.onStartMap(mapNameStr.Value)
			}
			return tengo.UndefinedValue, nil
		},
	}

	playMusicFunc := &tengo.UserFunction{
		Name: "PlayMusic",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("PlayMusic: missing music_name argument")
			}
			trackStr, ok := args[0].(*tengo.String)
			if !ok {
				return nil, fmt.Errorf("PlayMusic: expected string for music_name, found %s", args[0].TypeName())
			}
			if r.onPlayMusic != nil {
				r.onPlayMusic(trackStr.Value)
			}
			return tengo.UndefinedValue, nil
		},
	}

	stopMusicFunc := &tengo.UserFunction{
		Name: "StopMusic",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if r.onStopMusic != nil {
				r.onStopMusic()
			}
			return tengo.UndefinedValue, nil
		},
	}

	setVolumeFunc := &tengo.UserFunction{
		Name: "SetMusicVolume",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) == 0 {
				return nil, fmt.Errorf("SetMusicVolume: missing volume argument")
			}
			var vol float64
			if fVal, ok := args[0].(*tengo.Float); ok {
				vol = fVal.Value
			} else if iVal, ok := args[0].(*tengo.Int); ok {
				vol = float64(iVal.Value)
			} else {
				return nil, fmt.Errorf("SetMusicVolume: expected number for volume, found %s", args[0].TypeName())
			}
			if r.onSetMusicVolume != nil {
				r.onSetMusicVolume(vol)
			}
			return tengo.UndefinedValue, nil
		},
	}

	getMusicFunc := &tengo.UserFunction{
		Name: "GetMusicTrack",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			track := ""
			if r.onGetMusicTrack != nil {
				track = r.onGetMusicTrack()
			}
			return &tengo.String{Value: track}, nil
		},
	}

	gameModule := map[string]tengo.Object{
		"exit": &tengo.UserFunction{
			Name: "exit",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if r.onExit != nil {
					r.onExit()
				}
				return tengo.UndefinedValue, nil
			},
		},
		"StartMap":         startMapFunc,
		"start_map":        startMapFunc,
		"PlayMusic":        playMusicFunc,
		"play_music":       playMusicFunc,
		"StopMusic":        stopMusicFunc,
		"stop_music":       stopMusicFunc,
		"SetMusicVolume":   setVolumeFunc,
		"set_music_volume": setVolumeFunc,
		"GetMusicTrack":    getMusicFunc,
		"get_music_track":  getMusicFunc,
		"GetMusic":         getMusicFunc,
		"get_music":        getMusicFunc,
	}

	r.modules = stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	r.modules.AddBuiltinModule("game", gameModule)
	r.modules.AddBuiltinModule("fmt", r.createFmtModule())

	return r
}

// SetPlayMusicFunc updates the music playback callback.
func (r *REPL) SetPlayMusicFunc(fn func(string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onPlayMusic = fn
}

// SetStopMusicFunc updates the music stop callback.
func (r *REPL) SetStopMusicFunc(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStopMusic = fn
}

// SetSetMusicVolumeFunc updates the music volume callback.
func (r *REPL) SetSetMusicVolumeFunc(fn func(float64)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onSetMusicVolume = fn
}

// SetGetMusicTrackFunc updates the get current music track callback.
func (r *REPL) SetGetMusicTrackFunc(fn func() string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onGetMusicTrack = fn
}

// SetStartMapFunc updates the map start callback.
func (r *REPL) SetStartMapFunc(fn func(string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStartMap = fn
}

// SetPrintFunc updates the output print callback.
func (r *REPL) SetPrintFunc(fn func(string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.printFunc = fn
}

// createFmtModule creates a Tengo module that redirects all console printing to r.printFunc.
func (r *REPL) createFmtModule() map[string]tengo.Object {
	output := func(s string) {
		if r.printFunc != nil {
			r.printFunc(s)
		}
	}

	formatArgs := func(args []tengo.Object) []any {
		res := make([]any, len(args))
		for i, a := range args {
			res[i] = tengo.ToInterface(a)
		}
		return res
	}

	return map[string]tengo.Object{
		"print": &tengo.UserFunction{
			Name: "print",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var buf bytes.Buffer
				for i, a := range args {
					if i > 0 {
						buf.WriteString(" ")
					}
					if str, ok := a.(*tengo.String); ok {
						buf.WriteString(str.Value)
					} else {
						buf.WriteString(a.String())
					}
				}
				output(buf.String())
				return tengo.UndefinedValue, nil
			},
		},
		"println": &tengo.UserFunction{
			Name: "println",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var buf bytes.Buffer
				for i, a := range args {
					if i > 0 {
						buf.WriteString(" ")
					}
					if str, ok := a.(*tengo.String); ok {
						buf.WriteString(str.Value)
					} else {
						buf.WriteString(a.String())
					}
				}
				output(buf.String())
				return tengo.UndefinedValue, nil
			},
		},
		"printf": &tengo.UserFunction{
			Name: "printf",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, nil
				}
				formatStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("printf: expected string for first argument, found %s", args[0].TypeName())
				}
				goArgs := formatArgs(args[1:])
				formatted := fmt.Sprintf(formatStr.Value, goArgs...)
				output(strings.TrimRight(formatted, "\n"))
				return tengo.UndefinedValue, nil
			},
		},
		"sprintf": &tengo.UserFunction{
			Name: "sprintf",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return &tengo.String{Value: ""}, nil
				}
				formatStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("sprintf: expected string for first argument, found %s", args[0].TypeName())
				}
				goArgs := formatArgs(args[1:])
				formatted := fmt.Sprintf(formatStr.Value, goArgs...)
				return &tengo.String{Value: formatted}, nil
			},
		},
		"sprint": &tengo.UserFunction{
			Name: "sprint",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var buf bytes.Buffer
				for _, a := range args {
					if str, ok := a.(*tengo.String); ok {
						buf.WriteString(str.Value)
					} else {
						buf.WriteString(a.String())
					}
				}
				return &tengo.String{Value: buf.String()}, nil
			},
		},
		"sprintln": &tengo.UserFunction{
			Name: "sprintln",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var buf bytes.Buffer
				for i, a := range args {
					if i > 0 {
						buf.WriteString(" ")
					}
					if str, ok := a.(*tengo.String); ok {
						buf.WriteString(str.Value)
					} else {
						buf.WriteString(a.String())
					}
				}
				buf.WriteString("\n")
				return &tengo.String{Value: buf.String()}, nil
			},
		},
	}
}

// Eval executes a line of Tengo code in the REPL.
// If the line is an expression, it returns the string representation of its value.
// If an error occurs (syntax or runtime), it returns the error.
func (r *REPL) Eval(input string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("repl", -1, len(trimmed))

	p := parser.NewParser(srcFile, []byte(trimmed), nil)
	file, err := p.ParseFile()
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	// Check if the input is a single expression statement (e.g. `1 + 2`, `a`, `game.exit()`)
	if len(file.Stmts) == 1 {
		if _, ok := file.Stmts[0].(*parser.ExprStmt); ok {
			// Try evaluating as an assignment to the reserved result variable
			res, ok, evalErr := r.evalExpression(trimmed)
			if ok {
				return res, evalErr
			}
		}
	}

	return r.evalStatements(trimmed)
}

// evalExpression evaluates input as an expression assigned to resultVarName.
func (r *REPL) evalExpression(expr string) (string, bool, error) {
	exprCode := fmt.Sprintf("%s = (%s)", resultVarName, expr)
	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile("repl_expr", -1, len(exprCode))

	p := parser.NewParser(srcFile, []byte(exprCode), nil)
	file, err := p.ParseFile()
	if err != nil {
		return "", false, nil
	}

	c := tengo.NewCompiler(srcFile, r.symbolTable, nil, r.modules, nil)
	if err := c.Compile(file); err != nil {
		return "", false, nil
	}

	bytecode := c.Bytecode()
	vm := tengo.NewVM(bytecode, r.globals, -1)
	if err := vm.Run(); err != nil {
		return "", true, fmt.Errorf("runtime error: %w", err)
	}

	if r.resIndex < len(r.globals) {
		val := r.globals[r.resIndex]
		r.globals[r.resIndex] = tengo.UndefinedValue
		if val != nil && val != tengo.UndefinedValue {
			return val.String(), true, nil
		}
	}

	return "", true, nil
}

// evalStatements compiles and executes general statements, updating globals.
func (r *REPL) evalStatements(code string) (string, error) {
	return r.evalNamedStatements("repl_stmt", code)
}

// EvalScript compiles and executes a named script (e.g. "autoexec.tengo"), preserving globals.
// Errors include the script filename and line numbers for precise error reporting.
func (r *REPL) EvalScript(scriptName, code string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.evalNamedStatements(scriptName, code)
}

func (r *REPL) evalNamedStatements(name, code string) (string, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return "", nil
	}

	fileSet := parser.NewFileSet()
	srcFile := fileSet.AddFile(name, -1, len(trimmed))

	p := parser.NewParser(srcFile, []byte(trimmed), nil)
	file, err := p.ParseFile()
	if err != nil {
		return "", fmt.Errorf("syntax error: %w", err)
	}

	c := tengo.NewCompiler(srcFile, r.symbolTable, nil, r.modules, nil)
	if err := c.Compile(file); err != nil {
		return "", fmt.Errorf("compile error: %w", err)
	}

	bytecode := c.Bytecode()
	vm := tengo.NewVM(bytecode, r.globals, -1)
	if err := vm.Run(); err != nil {
		return "", fmt.Errorf("runtime error: %w", err)
	}

	return "", nil
}
