package config_test

import (
	"testing"
	"time"

	"github.com/budimanlai/go-config"
	"github.com/stretchr/testify/assert"
)

func TestProductionFeatures(t *testing.T) {
	writeFile("testdata/prod.json", `{
		"app": {"name": "prod-app", "version": "1.0"},
		"database": {"host": "localhost", "port": 5432}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/prod.json")
	assert.NoError(t, err)

	// Test stats
	stats := cfg.GetStats()
	assert.Greater(t, stats.StorageSize, 0)
	assert.Equal(t, 1, stats.FilesWatched)
	assert.True(t, stats.IsWatching)

	// Test caching with struct mapping
	type AppConfig struct {
		App struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"app"`
		Database struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"database"`
	}

	var config1 AppConfig
	var config2 AppConfig

	// First call - should populate cache
	err = cfg.MapToStructNested(&config1)
	assert.NoError(t, err)

	statsAfterFirst := cfg.GetStats()
	assert.Greater(t, statsAfterFirst.CacheSize, 0)
	// Second call - should use cache (measure performance)
	start := time.Now()
	err = cfg.MapToStructNested(&config2)
	duration := time.Since(start)
	assert.NoError(t, err)

	// Cache should still be same size
	statsAfterSecond := cfg.GetStats()
	assert.Equal(t, statsAfterFirst.CacheSize, statsAfterSecond.CacheSize)

	// Second call should be reasonably fast (cached)
	assert.Less(t, int64(duration), int64(10*time.Millisecond))

	// Test results are identical
	assert.Equal(t, config1, config2)

	// Test cache clearing
	cfg.ClearCache()
	statsAfterClear := cfg.GetStats()
	assert.Equal(t, 0, statsAfterClear.CacheSize)

	// Test resource cleanup
	err = cfg.Close()
	assert.NoError(t, err)
}

func TestMemoryEfficiency(t *testing.T) {
	writeFile("testdata/large.json", `{
		"data": {
			"items": [
				{"id": "1", "name": "item1", "value": 100},
				{"id": "2", "name": "item2", "value": 200},
				{"id": "3", "name": "item3", "value": 300}
			]
		}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/large.json")
	assert.NoError(t, err)
	defer cfg.Close()

	type Item struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	type DataConfig struct {
		Data struct {
			Items []Item `json:"items"`
		} `json:"data"`
	}
	// Multiple calls to test memory stability
	for i := 0; i < 100; i++ {
		var config DataConfig
		err = cfg.MapToStructNested(&config)
		assert.NoError(t, err)
		assert.Len(t, config.Data.Items, 3)
	}

	// Cache should be stable
	stats := cfg.GetStats()
	assert.Equal(t, 1, stats.CacheSize) // Only one struct type cached
}

func TestConcurrentAccess(t *testing.T) {
	writeFile("testdata/concurrent.json", `{
		"shared": {"counter": 42, "name": "test"}
	}`)

	cfg := config.Config{}
	err := cfg.Open("testdata/concurrent.json")
	assert.NoError(t, err)
	defer cfg.Close()

	type SharedConfig struct {
		Shared struct {
			Counter int    `json:"counter"`
			Name    string `json:"name"`
		} `json:"shared"`
	}

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				var config SharedConfig
				err := cfg.MapToStructNested(&config)
				assert.NoError(t, err)
				assert.Equal(t, 42, config.Shared.Counter)
				assert.Equal(t, "test", config.Shared.Name)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Stats should be consistent
	stats := cfg.GetStats()
	assert.Greater(t, stats.StorageSize, 0)
	assert.Greater(t, stats.CacheSize, 0)
}
