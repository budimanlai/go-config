package config_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/budimanlai/go-config"
)

// BenchmarkMapToStructNested mengukur performance MapToStructNested
func BenchmarkMapToStructNested(b *testing.B) {
	// Setup test data
	writeFile("testdata/benchmark.json", `{
		"app": {"name": "benchmark-app", "version": "2.0", "debug": true},
		"database": {"host": "localhost", "port": 5432, "ssl": true},
		"cache": {"ttl": 3600, "size": 1000},
		"servers": [
			{"name": "web1", "port": 8080, "active": true},
			{"name": "web2", "port": 8081, "active": false},
			{"name": "api1", "port": 9090, "active": true}
		]
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	type Server struct {
		Name   string `json:"name"`
		Port   int    `json:"port"`
		Active bool   `json:"active"`
	}

	type AppConfig struct {
		App struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Debug   bool   `json:"debug"`
		} `json:"app"`
		Database struct {
			Host string `json:"host"`
			Port int    `json:"port"`
			SSL  bool   `json:"ssl"`
		} `json:"database"`
		Cache struct {
			TTL  int `json:"ttl"`
			Size int `json:"size"`
		} `json:"cache"`
		Servers []Server `json:"servers"`
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var config AppConfig
		err := cfg.MapToStructNested(&config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMapToStructNestedCached mengukur performance dengan caching
func BenchmarkMapToStructNestedCached(b *testing.B) {
	writeFile("testdata/benchmark_cached.json", `{
		"data": {"value": 42, "name": "cached-test"}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark_cached.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	type CachedConfig struct {
		Data struct {
			Value int    `json:"value"`
			Name  string `json:"name"`
		} `json:"data"`
	}

	// Warm up cache dengan satu call
	var warmup CachedConfig
	cfg.MapToStructNested(&warmup)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var config CachedConfig
		err := cfg.MapToStructNested(&config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetAllAsInterface mengukur performance konversi data
func BenchmarkGetAllAsInterface(b *testing.B) {
	writeFile("testdata/benchmark_interface.json", `{
		"string_field": "test",
		"int_field": 123,
		"float_field": 45.67,
		"bool_field": true,
		"nested": {"key": "value", "number": 999}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark_interface.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = cfg.GetAllAsInterface()
	}
}

// BenchmarkConcurrentAccess mengukur performance concurrent access
func BenchmarkConcurrentAccess(b *testing.B) {
	writeFile("testdata/benchmark_concurrent.json", `{
		"shared": {"counter": 100, "active": true}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark_concurrent.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	type SharedConfig struct {
		Shared struct {
			Counter int  `json:"counter"`
			Active  bool `json:"active"`
		} `json:"shared"`
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var config SharedConfig
			err := cfg.MapToStructNested(&config)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLargeDataset mengukur performance dengan dataset besar
func BenchmarkLargeDataset(b *testing.B) {
	// Generate large JSON data
	largeData := `{
		"metadata": {"version": "1.0", "timestamp": "2025-07-15T18:00:00Z"},
		"items": [`

	for i := 0; i < 1000; i++ {
		if i > 0 {
			largeData += ","
		}
		largeData += fmt.Sprintf(`{"id": "%d", "name": "item_%d", "value": %d, "active": %t}`,
			i, i, i*10, i%2 == 0)
	}
	largeData += `]}`

	writeFile("testdata/benchmark_large.json", largeData)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark_large.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	type Item struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Value  int    `json:"value"`
		Active bool   `json:"active"`
	}

	type LargeConfig struct {
		Metadata struct {
			Version   string `json:"version"`
			Timestamp string `json:"timestamp"`
		} `json:"metadata"`
		Items []Item `json:"items"`
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var config LargeConfig
		err := cfg.MapToStructNested(&config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryUsage mengukur memory usage
func BenchmarkMemoryUsage(b *testing.B) {
	writeFile("testdata/benchmark_memory.json", `{
		"config": {"key1": "value1", "key2": 42, "key3": true}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/benchmark_memory.json")
	if err != nil {
		b.Fatal(err)
	}
	defer cfg.Close()

	type MemoryConfig struct {
		Config struct {
			Key1 string `json:"key1"`
			Key2 int    `json:"key2"`
			Key3 bool   `json:"key3"`
		} `json:"config"`
	}

	// Measure baseline memory
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var config MemoryConfig
		err := cfg.MapToStructNested(&config)
		if err != nil {
			b.Fatal(err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.StopTimer()
	b.Logf("Memory allocated per operation: %d bytes", (m2.TotalAlloc-m1.TotalAlloc)/uint64(b.N))
}

// TestPerformanceComparison membandingkan performance sebelum dan sesudah optimasi
func TestPerformanceComparison(t *testing.T) {
	writeFile("testdata/perf_test.json", `{
		"app": {"name": "perf-test", "version": "1.0"},
		"items": [
			{"id": "1", "value": 100},
			{"id": "2", "value": 200},
			{"id": "3", "value": 300}
		]
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/perf_test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()

	type Item struct {
		ID    string `json:"id"`
		Value int    `json:"value"`
	}

	type PerfConfig struct {
		App struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"app"`
		Items []Item `json:"items"`
	}

	// Test first call (cache miss)
	start := time.Now()
	var config1 PerfConfig
	err = cfg.MapToStructNested(&config1)
	firstCallDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	// Test second call (cache hit)
	start = time.Now()
	var config2 PerfConfig
	err = cfg.MapToStructNested(&config2)
	secondCallDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	// Performance metrics
	stats := cfg.GetStats()

	t.Logf("Performance Metrics:")
	t.Logf("  First call (cache miss): %v", firstCallDuration)
	t.Logf("  Second call (cache hit): %v", secondCallDuration)
	t.Logf("  Cache hit speedup: %.2fx", float64(firstCallDuration)/float64(secondCallDuration))
	t.Logf("  Storage size: %d entries", stats.StorageSize)
	t.Logf("  Cache size: %d entries", stats.CacheSize)
	t.Logf("  Files watched: %d", stats.FilesWatched)
	t.Logf("  Watching active: %v", stats.IsWatching)

	// Assertions
	if secondCallDuration >= firstCallDuration {
		t.Logf("Warning: Second call not faster than first (caching may not be effective)")
	}

	if stats.CacheSize == 0 {
		t.Error("Cache should contain entries after struct mapping")
	}
}

// TestMemoryEfficiencyCheck mengukur memory efficiency
func TestMemoryEfficiencyCheck(t *testing.T) {
	writeFile("testdata/memory_test.json", `{
		"data": {"items": [{"id": "1"}, {"id": "2"}, {"id": "3"}]}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/memory_test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()

	type Item struct {
		ID string `json:"id"`
	}
	type MemTestConfig struct {
		Data struct {
			Items []Item `json:"items"`
		} `json:"data"`
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform multiple operations
	for i := 0; i < 100; i++ {
		var config MemTestConfig
		err := cfg.MapToStructNested(&config)
		if err != nil {
			t.Fatal(err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedBytes := m2.TotalAlloc - m1.TotalAlloc
	t.Logf("Memory efficiency metrics:")
	t.Logf("  Total allocated: %d bytes", allocatedBytes)
	t.Logf("  Per operation: %d bytes", allocatedBytes/100)
	t.Logf("  Current heap: %d bytes", m2.HeapAlloc)

	// Test cache effectiveness
	stats := cfg.GetStats()
	t.Logf("  Cache entries: %d", stats.CacheSize)

	// Memory should be reasonable (less than 1KB per operation for this simple case)
	bytesPerOp := allocatedBytes / 100
	if bytesPerOp > 1024 {
		t.Logf("Warning: High memory usage per operation: %d bytes", bytesPerOp)
	}
}
