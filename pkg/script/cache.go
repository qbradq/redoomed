package script

import (
	"fmt"
	"io/fs"
	"log"
	"sync"

	"github.com/d5/tengo/v2"

	"github.com/qbradq/redoomed/pkg/data"
)

// ScriptCache caches parsed, loaded, and compiled Tengo scripts to prevent re-reading and re-parsing.
type ScriptCache struct {
	mu               sync.RWMutex
	fs               fs.FS
	scriptSources    map[string][]byte
	compiledScripts  map[string]*tengo.Compiled
	compiledSpecials map[int]*tengo.Compiled
	modules          *tengo.ModuleMap
	onError          func(error)
}

// NewScriptCache creates a new script cache initialized with the embedded filesystem and module map.
func NewScriptCache(modules *tengo.ModuleMap) *ScriptCache {
	if modules == nil {
		modules = tengo.NewModuleMap()
	}
	return &ScriptCache{
		fs:               data.FS,
		scriptSources:    make(map[string][]byte),
		compiledScripts:  make(map[string]*tengo.Compiled),
		compiledSpecials: make(map[int]*tengo.Compiled),
		modules:          modules,
	}
}

// SetFS allows overriding the backing filesystem (useful for testing).
func (c *ScriptCache) SetFS(fileSys fs.FS) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fs = fileSys
	c.scriptSources = make(map[string][]byte)
	c.compiledScripts = make(map[string]*tengo.Compiled)
	c.compiledSpecials = make(map[int]*tengo.Compiled)
}

// SetModules updates the active module map.
func (c *ScriptCache) SetModules(modMap *tengo.ModuleMap) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modules = modMap
	c.compiledScripts = make(map[string]*tengo.Compiled)
	c.compiledSpecials = make(map[int]*tengo.Compiled)
}

// SetOnError updates the error callback handler.
func (c *ScriptCache) SetOnError(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = fn
}

// GetScript retrieves a script's source bytes from the cache or reads it from the filesystem.
func (c *ScriptCache) GetScript(path string) ([]byte, error) {
	c.mu.RLock()
	if src, ok := c.scriptSources[path]; ok {
		c.mu.RUnlock()
		return src, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if src, ok := c.scriptSources[path]; ok {
		return src, nil
	}

	src, err := fs.ReadFile(c.fs, path)
	if err != nil {
		return nil, err
	}

	c.scriptSources[path] = src
	return src, nil
}

// RegisterScript adds a script source directly to the cache under a specific name.
func (c *ScriptCache) RegisterScript(name string, src []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scriptSources[name] = src
	delete(c.compiledScripts, name)
}

// HasLineSpecial checks if a line special script (e.g. line_001.tengo) exists.
func (c *ScriptCache) HasLineSpecial(special int) bool {
	c.mu.RLock()
	if _, ok := c.compiledSpecials[special]; ok {
		c.mu.RUnlock()
		return true
	}
	c.mu.RUnlock()

	path := fmt.Sprintf("scripts/lines/line_%03d.tengo", special)
	_, err := c.GetScript(path)
	return err == nil
}

// LoadLineSpecial compiles and caches the line special script corresponding to the special ID.
func (c *ScriptCache) LoadLineSpecial(special int) (*tengo.Compiled, error) {
	c.mu.RLock()
	if compiled, ok := c.compiledSpecials[special]; ok {
		c.mu.RUnlock()
		return compiled, nil
	}
	c.mu.RUnlock()

	path := fmt.Sprintf("scripts/lines/line_%03d.tengo", special)
	src, err := c.GetScript(path)
	if err != nil {
		return nil, fmt.Errorf("line special %d script not found (%s): %w", special, path, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after write lock
	if compiled, ok := c.compiledSpecials[special]; ok {
		return compiled, nil
	}

	modName := fmt.Sprintf("line_%03d", special)
	c.modules.AddSourceModule(modName, src)

	// Build caller script
	callerCode := fmt.Sprintf(`
target := import("%s")
if is_callable(target) {
	target(line_id, sec_id, thing_id, tag)
} else if is_map(target) && is_callable(target.run) {
	target.run(line_id, sec_id, thing_id, tag)
} else if is_map(target) && is_callable(target.special) {
	target.special(line_id, sec_id, thing_id, tag)
} else if is_map(target) && is_callable(target.action) {
	target.action(line_id, sec_id, thing_id, tag)
}
`, modName)

	s := tengo.NewScript([]byte(callerCode))
	s.SetImports(c.modules)
	_ = s.Add("line_id", 0)
	_ = s.Add("sec_id", 0)
	_ = s.Add("thing_id", 0)
	_ = s.Add("tag", 0)

	compiled, err := s.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile line special %d (%s): %w", special, path, err)
	}

	c.compiledSpecials[special] = compiled
	return compiled, nil
}

// ExecuteLineSpecialSync executes a line special synchronously with the given ID parameters.
func (c *ScriptCache) ExecuteLineSpecialSync(special int, lineID, secID, thingID, tag int) error {
	compiled, err := c.LoadLineSpecial(special)
	if err != nil {
		return err
	}

	cloned := compiled.Clone()
	_ = cloned.Set("line_id", lineID)
	_ = cloned.Set("sec_id", secID)
	_ = cloned.Set("thing_id", thingID)
	_ = cloned.Set("tag", tag)

	if err := cloned.Run(); err != nil {
		if c.onError != nil {
			c.onError(err)
		}
		return err
	}
	return nil
}

// ExecuteLineSpecial executes a line special asynchronously in a background goroutine.
func (c *ScriptCache) ExecuteLineSpecial(special int, lineID, secID, thingID, tag int) {
	go func() {
		if err := c.ExecuteLineSpecialSync(special, lineID, secID, thingID, tag); err != nil {
			log.Printf("Line special %d error: %v", special, err)
			c.mu.RLock()
			errHandler := c.onError
			c.mu.RUnlock()
			if errHandler != nil {
				errHandler(err)
			}
		}
	}()
}
