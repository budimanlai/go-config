package config_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/budimanlai/go-config"
	"github.com/stretchr/testify/assert"
)

func writeFile(path, content string) {
	_ = os.MkdirAll("testdata", 0755)
	f, _ := os.Create(path)
	defer f.Close()
	f.WriteString(content)
}

func TestINIConfig(t *testing.T) {
	writeFile("testdata/test.conf", `
[database]
hostname = localhost
port = 3306
enabled = true
`)

	cfg := config.Config{}
	err := cfg.Open("testdata/test.conf")
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.GetString("database.hostname"))
	assert.Equal(t, 3306, cfg.GetInt("database.port"))
	assert.Equal(t, true, cfg.GetBool("database.enabled"))
	assert.Equal(t, "default", cfg.GetStringOr("database.unknown", "default"))
}

func TestJSONConfig(t *testing.T) {
	writeFile("testdata/test.json", `
{
  "app": {
    "name": "myapp",
    "debug": true
  },
  "numbers": [1,2,3],
  "servers": [
    {"host": "a", "port": 1},
    {"host": "b", "port": 2}
  ]
}
`)
	cfg := config.Config{}
	err := cfg.Open("testdata/test.json")
	assert.NoError(t, err)
	assert.Equal(t, "myapp", cfg.GetString("app.name"))
	assert.Equal(t, true, cfg.GetBool("app.debug"))
	assert.Equal(t, 1, cfg.GetInt("numbers.0"))
	assert.Equal(t, 2, cfg.GetInt("numbers.1"))
	assert.Equal(t, "a", cfg.GetString("servers.0.host"))
	assert.Equal(t, 2, cfg.GetInt("servers.1.port"))
}

func TestGetArrayString(t *testing.T) {
	writeFile("testdata/arr.json", `{"arr":["x","y","z"]}`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/arr.json")
	arr := cfg.GetArrayString("arr")
	assert.Equal(t, []string{"x", "y", "z"}, arr)
}

func TestGetArrayObjectAuto(t *testing.T) {
	writeFile("testdata/obj.json", `
{
  "servers": [
    {"host": "a", "port": 1},
    {"host": "b", "port": 2}
  ]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/obj.json")
	arr := cfg.GetArrayObjectAuto("servers")
	assert.Len(t, arr, 2)
	assert.Equal(t, "a", arr[0]["host"])
	assert.Equal(t, "2", arr[1]["port"])
}

func TestGetArrayToStruct(t *testing.T) {
	type Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	writeFile("testdata/obj2.json", `
{
  "servers": [
    {"host": "a", "port": 1},
    {"host": "b", "port": 2}
  ]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/obj2.json")
	var servers []Server
	err := cfg.GetArrayToStruct("servers", &servers)
	assert.NoError(t, err)
	assert.Equal(t, "a", servers[0].Host)
	assert.Equal(t, 2, servers[1].Port)
}

func TestReloadAndCallback(t *testing.T) {
	writeFile("testdata/reload2.conf", `
[main]
val = 1
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/reload2.conf")

	called := make(chan bool, 1)
	cfg.SetOnReload(func() { called <- true })

	// Ubah file, lalu reload
	writeFile("testdata/reload2.conf", `
[main]
val = 2
`)
	_ = cfg.Reload()

	// Tunggu callback dipanggil (dengan timeout)
	select {
	case <-called:
		// OK, callback dipanggil
	case <-time.After(1 * time.Second):
		t.Error("reload callback not called")
	}
	assert.Equal(t, 2, cfg.GetInt("main.val"))
}

func TestWatchAndReload(t *testing.T) {
	writeFile("testdata/watch.conf", `
[main]
val = 1
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/watch.conf")

	called := make(chan bool, 1)
	cfg.SetOnReload(func() { called <- true })

	// Ubah file setelah delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		writeFile("testdata/watch.conf", `
[main]
val = 99
`)
	}()

	select {
	case <-called:
		assert.Equal(t, 99, cfg.GetInt("main.val"))
	case <-time.After(2 * time.Second):
		t.Error("reload callback not called")
	}
}

func TestGetAll(t *testing.T) {
	writeFile("testdata/getall.json", `
{
  "app": {
    "name": "testapp",
    "port": 8080,
    "debug": true
  },
  "database": {
    "host": "localhost",
    "timeout": 30.5
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/getall.json")

	// Test GetAll() - returns map[string]string
	allSettings := cfg.GetAll()
	assert.Equal(t, "testapp", allSettings["app.name"])
	assert.Equal(t, "8080", allSettings["app.port"])
	assert.Equal(t, "true", allSettings["app.debug"])
	assert.Equal(t, "localhost", allSettings["database.host"])
	assert.Equal(t, "30.5", allSettings["database.timeout"])

	// Pastikan ini adalah copy, bukan reference
	allSettings["app.name"] = "modified"
	assert.Equal(t, "testapp", cfg.GetString("app.name")) // original tidak berubah
}

func TestGetAllKeys(t *testing.T) {
	writeFile("testdata/getkeys.json", `
{
  "app": {
    "name": "testapp",
    "port": 8080
  },
  "database": {
    "host": "localhost"
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/getkeys.json")

	keys := cfg.GetAllKeys()
	assert.Contains(t, keys, "app.name")
	assert.Contains(t, keys, "app.port")
	assert.Contains(t, keys, "database.host")
	assert.Len(t, keys, 3)
}

func TestGetAllAsInterface(t *testing.T) {
	writeFile("testdata/getinterface.json", `
{
  "app": {
    "name": "testapp",
    "port": 8080,
    "debug": true,
    "version": 1.5
  },
  "numbers": [1, 2, 3]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/getinterface.json")

	// Test GetAllAsInterface() - returns map[string]interface{} with proper types
	allSettings := cfg.GetAllAsInterface()

	// Test string value
	assert.Equal(t, "testapp", allSettings["app.name"])
	assert.IsType(t, "", allSettings["app.name"])

	// Test int value
	assert.Equal(t, 8080, allSettings["app.port"])
	assert.IsType(t, 0, allSettings["app.port"])

	// Test bool value
	assert.Equal(t, true, allSettings["app.debug"])
	assert.IsType(t, true, allSettings["app.debug"])

	// Test float value
	assert.Equal(t, 1.5, allSettings["app.version"])
	assert.IsType(t, 0.0, allSettings["app.version"])

	// Test array elements
	assert.Equal(t, 1, allSettings["numbers.0"])
	assert.Equal(t, 2, allSettings["numbers.1"])
	assert.Equal(t, 3, allSettings["numbers.2"])
}

func TestGetAllAsJSON(t *testing.T) {
	writeFile("testdata/getjson.json", `
{
  "app": {
    "name": "testapp",
    "port": 8080,
    "debug": true
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/getjson.json")

	// Test GetAllAsJSON()
	jsonStr, err := cfg.GetAllAsJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	// Verify JSON contains expected values
	assert.Contains(t, jsonStr, `"app.name": "testapp"`)
	assert.Contains(t, jsonStr, `"app.port": 8080`)
	assert.Contains(t, jsonStr, `"app.debug": true`)

	// Verify it's valid JSON by parsing it back
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "testapp", parsed["app.name"])
	assert.Equal(t, float64(8080), parsed["app.port"]) // JSON numbers are float64
	assert.Equal(t, true, parsed["app.debug"])
}

func TestMapToStruct(t *testing.T) {
	writeFile("testdata/mapstruct.json", `
{
  "database": {
    "host": "localhost",
    "port": 3306,
    "enabled": true
  },
  "app": {
    "name": "myapp",
    "version": 1.5,
    "debug": false
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/mapstruct.json")

	// Test MapToStruct with nested structure
	type AppConfig struct {
		DatabaseHost    string  `json:"database.host"`
		DatabasePort    int     `json:"database.port"`
		DatabaseEnabled bool    `json:"database.enabled"`
		AppName         string  `json:"app.name"`
		AppVersion      float64 `json:"app.version"`
		AppDebug        bool    `json:"app.debug"`
	}

	var appConfig AppConfig
	err := cfg.MapToStruct(&appConfig)
	assert.NoError(t, err)
	assert.Equal(t, "localhost", appConfig.DatabaseHost)
	assert.Equal(t, 3306, appConfig.DatabasePort)
	assert.Equal(t, true, appConfig.DatabaseEnabled)
	assert.Equal(t, "myapp", appConfig.AppName)
	assert.Equal(t, 1.5, appConfig.AppVersion)
	assert.Equal(t, false, appConfig.AppDebug)
}

func TestMapToStructFlat(t *testing.T) {
	writeFile("testdata/mapflat.json", `
{
  "server": {
    "host": "127.0.0.1",
    "port": 8080,
    "timeout": 30.5,
    "ssl": true
  },
  "app": {
    "name": "testapp",
    "workers": 4
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/mapflat.json")

	// Test MapToStructFlat with flat structure
	type FlatConfig struct {
		ServerHost    string  `json:"server.host"`
		ServerPort    int     `json:"server.port"`
		ServerTimeout float64 `json:"server.timeout"`
		ServerSSL     bool    `json:"server.ssl"`
		AppName       string  `json:"app.name"`
		AppWorkers    int     `json:"app.workers"`
		NotExists     string  `json:"not.exists"` // This should remain empty
	}

	var flatConfig FlatConfig
	err := cfg.MapToStructFlat(&flatConfig)
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", flatConfig.ServerHost)
	assert.Equal(t, 8080, flatConfig.ServerPort)
	assert.Equal(t, 30.5, flatConfig.ServerTimeout)
	assert.Equal(t, true, flatConfig.ServerSSL)
	assert.Equal(t, "testapp", flatConfig.AppName)
	assert.Equal(t, 4, flatConfig.AppWorkers)
	assert.Equal(t, "", flatConfig.NotExists) // Should be empty since key doesn't exist
}

func TestMapToStructError(t *testing.T) {
	cfg := config.Config{}
	_ = cfg.Open("testdata/test.conf")

	// Test error cases
	var notAPointer string
	err := cfg.MapToStructFlat(notAPointer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output must be a pointer to struct")

	var notAStruct *string
	err = cfg.MapToStructFlat(notAStruct)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output must be a pointer to struct")
}

func TestMapToStructNested(t *testing.T) {
	writeFile("testdata/nested.json", `
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "enabled": true,
    "timeout": 30.5
  },
  "app": {
    "name": "myapp",
    "version": "1.0.0",
    "debug": false,
    "workers": 4
  },
  "cache": {
    "redis": {
      "host": "redis-server",
      "port": 6379
    }
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/nested.json")

	// Test nested structure mapping
	type DatabaseConfig struct {
		Host    string  `json:"host"`
		Port    int     `json:"port"`
		Enabled bool    `json:"enabled"`
		Timeout float64 `json:"timeout"`
	}

	type AppConfig struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Debug   bool   `json:"debug"`
		Workers int    `json:"workers"`
	}

	type RedisConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	type CacheConfig struct {
		Redis RedisConfig `json:"redis"`
	}

	type Config struct {
		Database DatabaseConfig `json:"database"`
		App      AppConfig      `json:"app"`
		Cache    CacheConfig    `json:"cache"`
	}

	var nestedConfig Config
	err := cfg.MapToStructNested(&nestedConfig)
	assert.NoError(t, err)

	// Verify database config
	assert.Equal(t, "localhost", nestedConfig.Database.Host)
	assert.Equal(t, 5432, nestedConfig.Database.Port)
	assert.Equal(t, true, nestedConfig.Database.Enabled)
	assert.Equal(t, 30.5, nestedConfig.Database.Timeout)

	// Verify app config
	assert.Equal(t, "myapp", nestedConfig.App.Name)
	assert.Equal(t, "1.0.0", nestedConfig.App.Version)
	assert.Equal(t, false, nestedConfig.App.Debug)
	assert.Equal(t, 4, nestedConfig.App.Workers)

	// Verify nested cache config
	assert.Equal(t, "redis-server", nestedConfig.Cache.Redis.Host)
	assert.Equal(t, 6379, nestedConfig.Cache.Redis.Port)
}

func TestMapToStructAdvanced(t *testing.T) {
	writeFile("testdata/advanced.json", `
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "ssl": true
  },
  "database": {
    "host": "db-server",
    "port": 3306
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/advanced.json")

	// Test 1: Nested structure (should use nested mapping)
	type ServerConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		SSL  bool   `json:"ssl"`
	}

	type DatabaseConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	type NestedConfig struct {
		Server   ServerConfig   `json:"server"`
		Database DatabaseConfig `json:"database"`
	}

	var nestedConfig NestedConfig
	err := cfg.MapToStructAdvanced(&nestedConfig)
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0", nestedConfig.Server.Host)
	assert.Equal(t, 8080, nestedConfig.Server.Port)
	assert.Equal(t, true, nestedConfig.Server.SSL)
	assert.Equal(t, "db-server", nestedConfig.Database.Host)
	assert.Equal(t, 3306, nestedConfig.Database.Port)

	// Test 2: Flat structure (should use flat mapping)
	type FlatConfig struct {
		ServerHost   string `json:"server.host"`
		ServerPort   int    `json:"server.port"`
		ServerSSL    bool   `json:"server.ssl"`
		DatabaseHost string `json:"database.host"`
		DatabasePort int    `json:"database.port"`
	}

	var flatConfig FlatConfig
	err = cfg.MapToStructAdvanced(&flatConfig)
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0", flatConfig.ServerHost)
	assert.Equal(t, 8080, flatConfig.ServerPort)
	assert.Equal(t, true, flatConfig.ServerSSL)
	assert.Equal(t, "db-server", flatConfig.DatabaseHost)
	assert.Equal(t, 3306, flatConfig.DatabasePort)
}

func TestFlatToNested(t *testing.T) {
	writeFile("testdata/flatnested.json", `
{
  "level1": {
    "level2": {
      "level3": {
        "value": "deep"
      }
    }
  },
  "simple": "value"
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/flatnested.json")

	// Test konversi flat ke nested secara internal
	nestedData := cfg.GetAllAsInterface()

	// Verify struktur nested terbentuk dengan benar
	assert.Contains(t, nestedData, "level1.level2.level3.value")
	assert.Contains(t, nestedData, "simple")
	assert.Equal(t, "deep", nestedData["level1.level2.level3.value"])
	assert.Equal(t, "value", nestedData["simple"])
}

func TestMapToStructWithArrays(t *testing.T) {
	writeFile("testdata/arrays.json", `
{
  "app": {
    "name": "testapp",
    "tags": ["web", "api", "service"]
  },
  "servers": [
    {"host": "server1", "port": 8080},
    {"host": "server2", "port": 8081}
  ],
  "numbers": [1, 2, 3, 4, 5]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/arrays.json")

	// Test 1: Nested structure dengan array
	type Server struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	type AppConfig struct {
		App struct {
			Name string   `json:"name"`
			Tags []string `json:"tags"`
		} `json:"app"`
		Servers []Server `json:"servers"`
		Numbers []int    `json:"numbers"`
	}

	var config AppConfig
	err := cfg.MapToStructNested(&config)
	assert.NoError(t, err)

	// Verify app config
	assert.Equal(t, "testapp", config.App.Name)
	assert.Equal(t, []string{"web", "api", "service"}, config.App.Tags)

	// Verify servers array
	assert.Len(t, config.Servers, 2)
	assert.Equal(t, "server1", config.Servers[0].Host)
	assert.Equal(t, 8080, config.Servers[0].Port)
	assert.Equal(t, "server2", config.Servers[1].Host)
	assert.Equal(t, 8081, config.Servers[1].Port)

	// Verify numbers array
	assert.Equal(t, []int{1, 2, 3, 4, 5}, config.Numbers)
}

func TestMapToStructWithFlatArrays(t *testing.T) {
	writeFile("testdata/flatarray.json", `
{
  "app": {
    "name": "testapp"
  },
  "servers": [
    {"host": "server1", "port": 8080},
    {"host": "server2", "port": 8081}
  ]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/flatarray.json")

	// Test flat mapping dengan array elements
	type FlatArrayConfig struct {
		AppName     string `json:"app.name"`
		Server0Host string `json:"servers.0.host"`
		Server0Port int    `json:"servers.0.port"`
		Server1Host string `json:"servers.1.host"`
		Server1Port int    `json:"servers.1.port"`
	}

	var config FlatArrayConfig
	err := cfg.MapToStructFlat(&config)
	assert.NoError(t, err)
	assert.Equal(t, "testapp", config.AppName)
	assert.Equal(t, "server1", config.Server0Host)
	assert.Equal(t, 8080, config.Server0Port)
	assert.Equal(t, "server2", config.Server1Host)
	assert.Equal(t, 8081, config.Server1Port)
}

func TestMapToStructWithComplexObjects(t *testing.T) {
	writeFile("testdata/complex.json", `
{
  "database": {
    "primary": {
      "host": "primary-db",
      "port": 5432,
      "config": {
        "max_connections": 100,
        "timeout": 30.5,
        "ssl": true
      }
    },
    "replicas": [
      {
        "host": "replica1",
        "port": 5432,
        "weight": 1
      },
      {
        "host": "replica2", 
        "port": 5432,
        "weight": 2
      }
    ]
  },
  "cache": {
    "redis": {
      "clusters": [
        {"host": "redis1", "port": 6379},
        {"host": "redis2", "port": 6379}
      ]
    }
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/complex.json")

	// Test complex nested structure
	type DBConfig struct {
		MaxConnections int     `json:"max_connections"`
		Timeout        float64 `json:"timeout"`
		SSL            bool    `json:"ssl"`
	}

	type DatabaseServer struct {
		Host   string   `json:"host"`
		Port   int      `json:"port"`
		Config DBConfig `json:"config,omitempty"`
		Weight int      `json:"weight,omitempty"`
	}

	type RedisCluster struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	type ComplexConfig struct {
		Database struct {
			Primary  DatabaseServer   `json:"primary"`
			Replicas []DatabaseServer `json:"replicas"`
		} `json:"database"`
		Cache struct {
			Redis struct {
				Clusters []RedisCluster `json:"clusters"`
			} `json:"redis"`
		} `json:"cache"`
	}

	var config ComplexConfig
	err := cfg.MapToStructNested(&config)
	assert.NoError(t, err)

	// Verify primary database config
	assert.Equal(t, "primary-db", config.Database.Primary.Host)
	assert.Equal(t, 5432, config.Database.Primary.Port)
	assert.Equal(t, 100, config.Database.Primary.Config.MaxConnections)
	assert.Equal(t, 30.5, config.Database.Primary.Config.Timeout)
	assert.Equal(t, true, config.Database.Primary.Config.SSL)

	// Verify replicas
	assert.Len(t, config.Database.Replicas, 2)
	assert.Equal(t, "replica1", config.Database.Replicas[0].Host)
	assert.Equal(t, 1, config.Database.Replicas[0].Weight)
	assert.Equal(t, "replica2", config.Database.Replicas[1].Host)
	assert.Equal(t, 2, config.Database.Replicas[1].Weight)

	// Verify redis clusters
	assert.Len(t, config.Cache.Redis.Clusters, 2)
	assert.Equal(t, "redis1", config.Cache.Redis.Clusters[0].Host)
	assert.Equal(t, 6379, config.Cache.Redis.Clusters[0].Port)
	assert.Equal(t, "redis2", config.Cache.Redis.Clusters[1].Host)
	assert.Equal(t, 6379, config.Cache.Redis.Clusters[1].Port)
}

func TestDebugArrayStructure(t *testing.T) {
	writeFile("testdata/debug.json", `
{
  "simple": ["a", "b", "c"],
  "numbers": [1, 2, 3]
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/debug.json")

	// Debug: lihat struktur flat storage
	allData := cfg.GetAll()
	t.Logf("Flat storage: %+v", allData)

	// Debug: lihat hasil nested conversion
	nestedData := cfg.GetAllAsInterface() // Ini masih flat
	t.Logf("GetAllAsInterface: %+v", nestedData)

	// Test simple nested structure
	type SimpleConfig struct {
		Simple  []string `json:"simple"`
		Numbers []int    `json:"numbers"`
	}

	var config SimpleConfig
	err := cfg.MapToStructNested(&config)
	t.Logf("MapToStructNested error: %v", err)
	t.Logf("Result: %+v", config)
}

func TestMapToStructNestedStringNumbers(t *testing.T) {
	writeFile("testdata/stringnumbers.json", `
{
  "user": {
    "id": "1234567",
    "phone": "08123456789", 
    "age": 25,
    "name": "John Doe"
  },
  "product": {
    "sku": "PROD001",
    "barcode": "1234567890123",
    "price": 99.99,
    "stock": 100
  }
}
`)
	cfg := config.Config{}
	_ = cfg.Open("testdata/stringnumbers.json")

	// Test struct dengan string field yang berisi angka
	type UserConfig struct {
		ID    string `json:"id"`    // String yang berisi angka
		Phone string `json:"phone"` // String yang berisi angka
		Age   int    `json:"age"`   // Integer asli
		Name  string `json:"name"`  // String biasa
	}

	type ProductConfig struct {
		SKU     string  `json:"sku"`     // String biasa
		Barcode string  `json:"barcode"` // String yang berisi angka panjang
		Price   float64 `json:"price"`   // Float asli
		Stock   int     `json:"stock"`   // Integer asli
	}

	type Config struct {
		User    UserConfig    `json:"user"`
		Product ProductConfig `json:"product"`
	}

	var config Config
	err := cfg.MapToStructNested(&config)
	assert.NoError(t, err)

	// Verify string fields yang berisi angka tetap string
	assert.Equal(t, "1234567", config.User.ID)        // Harus string, bukan int
	assert.Equal(t, "08123456789", config.User.Phone) // Harus string, bukan int
	assert.Equal(t, 25, config.User.Age)              // Boleh int
	assert.Equal(t, "John Doe", config.User.Name)     // String biasa

	assert.Equal(t, "PROD001", config.Product.SKU)           // String biasa
	assert.Equal(t, "1234567890123", config.Product.Barcode) // Harus string, bukan int
	assert.Equal(t, 99.99, config.Product.Price)             // Boleh float
	assert.Equal(t, 100, config.Product.Stock)               // Boleh int
}
