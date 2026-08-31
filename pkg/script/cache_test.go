package script

import (
	"testing"
	"testing/fstest"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

func TestScriptCacheBasic(t *testing.T) {
	mockFS := fstest.MapFS{
		"scripts/test.tengo": &fstest.MapFile{
			Data: []byte(`
val := 42
export val
`),
		},
		"scripts/lines/line_999.tengo": &fstest.MapFile{
			Data: []byte(`
special_fn := func(line_id, sec_id, thing_id, tag) {
	return line_id * 100 + sec_id
}
export special_fn
`),
		},
	}

	modMap := stdlib.GetModuleMap(stdlib.AllModuleNames()...)
	cache := NewScriptCache(modMap)
	cache.SetFS(mockFS)

	// Test GetScript
	src, err := cache.GetScript("scripts/test.tengo")
	if err != nil {
		t.Fatalf("GetScript failed: %v", err)
	}
	if len(src) == 0 {
		t.Fatal("expected non-empty script content")
	}

	// Verify cached (modify mockFS and verify same content returned)
	delete(mockFS, "scripts/test.tengo")
	cachedSrc, err := cache.GetScript("scripts/test.tengo")
	if err != nil {
		t.Fatalf("GetScript from cache failed: %v", err)
	}
	if string(cachedSrc) != string(src) {
		t.Errorf("expected cached script content to match, got %s", string(cachedSrc))
	}

	// Test HasLineSpecial
	if !cache.HasLineSpecial(999) {
		t.Error("expected HasLineSpecial(999) to be true")
	}
	if cache.HasLineSpecial(998) {
		t.Error("expected HasLineSpecial(998) to be false")
	}

	// Test ExecuteLineSpecialSync
	err = cache.ExecuteLineSpecialSync(999, 5, 12, 0, 0)
	if err != nil {
		t.Fatalf("ExecuteLineSpecialSync failed: %v", err)
	}

	// Second execution should hit compiled cache
	err = cache.ExecuteLineSpecialSync(999, 10, 20, 0, 0)
	if err != nil {
		t.Fatalf("ExecuteLineSpecialSync (cached) failed: %v", err)
	}
}

func TestScriptCacheErrorHandling(t *testing.T) {
	mockFS := fstest.MapFS{
		"scripts/lines/line_888.tengo": &fstest.MapFile{
			Data: []byte(`
invalid := syntax +++ error
`),
		},
	}

	modMap := tengo.NewModuleMap()
	cache := NewScriptCache(modMap)
	cache.SetFS(mockFS)

	var reportedErr error
	cache.SetOnError(func(err error) {
		reportedErr = err
	})

	err := cache.ExecuteLineSpecialSync(888, 1, 2, 0, 0)
	if err == nil {
		t.Fatal("expected compile error for invalid syntax")
	}

	// Execute missing special
	err = cache.ExecuteLineSpecialSync(777, 1, 2, 0, 0)
	if err == nil {
		t.Fatal("expected error for missing line special script")
	}
	_ = reportedErr
}
