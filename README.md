# 🚀 go-config

**go-config** adalah library Go **production-ready** untuk membaca file konfigurasi dengan format `.ini` dan `.json` yang mendukung:

## ✨ Fitur Utama

### 🎯 **Core Features**
- ✅ **Multi-format support**: `.ini` dan `.json`
- ✅ **Thread-safe**: Aman untuk aplikasi concurrent
- ✅ **Hot reload**: Reload otomatis saat file berubah (menggunakan fsnotify)
- ✅ **Nested config**: Key dengan notasi titik (`app.database.host`)
- ✅ **Array support**: Array dan array of object
- ✅ **Type-safe**: Konversi otomatis ke tipe data yang tepat

### 🏭 **Production Features** 
- ⚡ **High Performance**: Caching dengan 6.45x speedup
- 🧠 **Smart Type Conversion**: Type-aware berdasarkan struct fields
- 📊 **Monitoring**: Built-in statistics dan monitoring API
- 🛡️ **Resource Management**: Proper cleanup dan memory management
- 🔒 **Concurrent Safe**: RWMutex untuk operasi aman
- 💾 **Memory Optimized**: Pre-allocated memory dan optimized algorithms

### 📈 **Performance Stats**
```
┌─────────────────────────────┬──────────────┬──────────────┬─────────────┐
│           OPERATION         │   TIME/OP    │   ALLOCS/OP  │   BYTES/OP  │
├─────────────────────────────┼──────────────┼──────────────┼─────────────┤
│ MapToStructNested (cold)    │    8.896 µs  │       183    │   10,113 B  │
│ MapToStructNested (cached)  │    1.337 µs  │        29    │    2,065 B  │
│ Concurrent Access           │    0.620 µs  │        27    │    2,026 B  │
│ Large Dataset (1000 items)  │ 2111.410 µs  │    47,847    │2,394,365 B  │
└─────────────────────────────┴──────────────┴──────────────┴─────────────┘
```

---

## 📦 Instalasi

```sh
go get github.com/budimanlai/go-config
```

---

## 🚀 Quick Start

### 1. Basic Usage

```go
package main

import (
    "log"
    "github.com/budimanlai/go-config"
)

func main() {
    cfg := config.Config{}
    err := cfg.Open("config.ini", "config.json")
    if err != nil {
        log.Fatal(err)
    }
    defer cfg.Close() // ⚠️ Important: Always call Close() for cleanup

    // Basic value access
    host := cfg.GetString("database.host")
    port := cfg.GetInt("database.port")
    debug := cfg.GetBool("app.debug")
    
    log.Printf("Database: %s:%d, Debug: %v", host, port, debug)
}
```

### 2. 🎯 **Type-Safe Struct Mapping** (Recommended)

```go
// Define your config structure
type AppConfig struct {
    App struct {
        Name    string `json:"name"`
        Version string `json:"version"`
        Debug   bool   `json:"debug"`
    } `json:"app"`
    Database struct {
        Host     string `json:"host"`
        Port     int    `json:"port"`
        Username string `json:"username"`
        SSL      bool   `json:"ssl"`
    } `json:"database"`
    Servers []Server `json:"servers"`
}

type Server struct {
    Name   string `json:"name"`
    Host   string `json:"host"`
    Port   int    `json:"port"`
    Active bool   `json:"active"`
}

func main() {
    cfg := config.Config{}
    err := cfg.Open("config.json")
    if err != nil {
        log.Fatal(err)
    }
    defer cfg.Close()

    var appConfig AppConfig
    err = cfg.MapToStructNested(&appConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Type-safe access
    fmt.Printf("App: %s v%s\n", appConfig.App.Name, appConfig.App.Version)
    fmt.Printf("Database: %s:%d\n", appConfig.Database.Host, appConfig.Database.Port)
    fmt.Printf("Servers: %d configured\n", len(appConfig.Servers))
}
```

### 3. 🔄 **Hot Reload & Callbacks**

```go
cfg := config.Config{}

// Set callback untuk reload
cfg.SetOnReload(func() {
    log.Println("🔄 Config reloaded automatically!")
    // Re-process your config here
})

err := cfg.Open("config.json")
if err != nil {
    log.Fatal(err)
}
defer cfg.Close()

// File akan di-monitor otomatis, callback dipanggil saat file berubah
```

### 4. 📊 **Production Monitoring**

```go
cfg := config.Config{}
err := cfg.Open("config.json")
if err != nil {
    log.Fatal(err)
}
defer cfg.Close()

// Get performance statistics
stats := cfg.GetStats()
fmt.Printf("📊 Config Stats:\n")
fmt.Printf("  Storage entries: %d\n", stats.StorageSize)
fmt.Printf("  Cache entries: %d\n", stats.CacheSize)
fmt.Printf("  Files watched: %d\n", stats.FilesWatched)
fmt.Printf("  Watching active: %v\n", stats.IsWatching)

// Clear cache when needed (for long-running processes)
cfg.ClearCache()
```

---

## 📚 Complete API Reference

### 🔍 **Value Access Methods**

```go
// String values
cfg.GetString("key")                    // Returns "" if not found
cfg.GetStringOr("key", "default")       // Returns default if not found

// Numeric values  
cfg.GetInt("key")                       // Returns 0 if not found
cfg.GetIntOr("key", 123)               // Returns default if not found
cfg.GetFloat32("key")
cfg.GetFloat64("key")

// Boolean values
cfg.GetBool("key")                      // Returns false if not found
cfg.GetBoolOr("key", true)             // Returns default if not found
```

### 📋 **Array & Object Methods**

```go
// Simple arrays
arr := cfg.GetArrayString("arr")                    // []string

// Array of objects (manual field specification)
objs := cfg.GetArrayObject("servers", []string{"host", "port"}) // []map[string]string

// Array of objects (auto-discovery)
objsAuto := cfg.GetArrayObjectAuto("servers")       // []map[string]interface{}

// Type-safe array to struct
type Server struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
var servers []Server
cfg.GetArrayToStruct("servers", &servers)
```

### 🗺️ **Struct Mapping Methods**

```go
// ⭐ RECOMMENDED: Type-aware nested mapping
err := cfg.MapToStructNested(&config)

// Flat mapping (for simple configs)
err := cfg.MapToStructFlat(&config)

// Auto-detection mapping
err := cfg.MapToStructAdvanced(&config)

// Legacy mapping (backward compatibility)
err := cfg.MapToStruct(&config)
```

### 📊 **Data Export Methods**

```go
// Get all as string map
allStrings := cfg.GetAll()              // map[string]string

// Get all with type conversion
allData := cfg.GetAllAsInterface()      // map[string]interface{}

// Get all as JSON string
jsonStr, err := cfg.GetAllAsJSON()      // Pretty-printed JSON

// Get all keys
keys := cfg.GetAllKeys()                // []string
```

### 🛠️ **Management Methods**

```go
// Manual reload
err := cfg.Reload()

// Set reload callback
cfg.SetOnReload(func() {
    // Your callback logic
})

// Production monitoring
stats := cfg.GetStats()

// Cache management
cfg.ClearCache()

// Resource cleanup (IMPORTANT!)
err := cfg.Close()
```

---

## 📄 Configuration File Examples

### 📋 INI Format

```ini
# App configuration
[app]
name = MyApp
version = 1.0.0
debug = true

[database]
host = localhost
port = 5432
username = admin
ssl = false

# Array support with indexed keys
[servers]
0.name = web1
0.host = 192.168.1.10
0.port = 8080
0.active = true

1.name = web2  
1.host = 192.168.1.11
1.port = 8081
1.active = false
```

### 🔧 JSON Format

```json
{
  "app": {
    "name": "MyApp",
    "version": "1.0.0", 
    "debug": true
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "username": "admin",
    "ssl": false
  },
  "servers": [
    {
      "name": "web1",
      "host": "192.168.1.10", 
      "port": 8080,
      "active": true
    },
    {
      "name": "web2",
      "host": "192.168.1.11",
      "port": 8081, 
      "active": false
    }
  ],
  "features": ["auth", "logging", "metrics"],
  "limits": {
    "max_connections": 1000,
    "timeout_seconds": 30
  }
}
```

---

## 🏭 Production Best Practices

### ✅ **Resource Management**

```go
func main() {
    cfg := config.Config{}
    err := cfg.Open("config.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // ⚠️ IMPORTANT: Always defer Close() for proper cleanup
    defer func() {
        if err := cfg.Close(); err != nil {
            log.Printf("Error closing config: %v", err)
        }
    }()
    
    // Your application logic here...
}
```

### 📊 **Monitoring Integration**

```go
// Production monitoring setup
func setupConfigMonitoring(cfg *config.Config) {
    go func() {
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()
        
        for range ticker.C {
            stats := cfg.GetStats()
            // Send to your monitoring system
            metrics.Gauge("config.storage_size", stats.StorageSize)
            metrics.Gauge("config.cache_size", stats.CacheSize)
            metrics.Bool("config.watching", stats.IsWatching)
        }
    }()
}
```

### ⚡ **Performance Optimization**

```go
// For long-running processes
func configMaintenance(cfg *config.Config) {
    // Clear cache periodically to prevent memory buildup
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()
        
        for range ticker.C {
            cfg.ClearCache()
            log.Println("Config cache cleared")
        }
    }()
}
```

### 🔄 **Graceful Reload Handling**

```go
type App struct {
    config *AppConfig
    mu     sync.RWMutex
}

func (app *App) setupConfigReload(cfg *config.Config) {
    cfg.SetOnReload(func() {
        var newConfig AppConfig
        if err := cfg.MapToStructNested(&newConfig); err != nil {
            log.Printf("❌ Config reload failed: %v", err)
            return
        }
        
        app.mu.Lock()
        app.config = &newConfig
        app.mu.Unlock()
        
        log.Println("✅ Config reloaded successfully")
    })
}

func (app *App) GetConfig() *AppConfig {
    app.mu.RLock()
    defer app.mu.RUnlock()
    return app.config
}
```

---

## 🎯 Advanced Features

### 🧠 **Type-Aware Conversion**

Library secara otomatis mengkonversi nilai berdasarkan tipe field di struct:

```go
type Config struct {
    // String field tetap string, meskipun JSON value berupa "123"
    UserID   string `json:"user_id"`    // "123" → "123" (string)
    
    // Integer field dikonversi ke int
    MaxUsers int    `json:"max_users"`  // "100" → 100 (int)
    
    // Boolean field dikonversi ke bool
    Enabled  bool   `json:"enabled"`    // "true" → true (bool)
}
```

### 🔄 **Multi-File Support**

```go
// Load multiple config files
cfg := config.Config{}
err := cfg.Open("base.json", "env.json", "local.json")
// Later files override earlier files
```

### 🎛️ **Environment-Specific Configs**

```go
env := os.Getenv("APP_ENV")
if env == "" {
    env = "development"
}

cfg := config.Config{}
err := cfg.Open(
    "config/base.json",
    fmt.Sprintf("config/%s.json", env),
)
```

---

## 🚀 Migration Guide

### From v1.x to v2.x

```go
// ❌ Old way (v1.x)
cfg := config.Config{}
cfg.Open("config.json")
// No cleanup

// ✅ New way (v2.x) 
cfg := config.Config{}
err := cfg.Open("config.json")
if err != nil {
    log.Fatal(err)
}
defer cfg.Close() // Proper resource cleanup

// Use MapToStructNested for better performance
var config AppConfig
err = cfg.MapToStructNested(&config)
```

---

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🏆 Acknowledgments

- Built with performance and production readiness in mind
- Uses [fsnotify](https://github.com/fsnotify/fsnotify) for file watching
- Inspired by modern configuration management best practices

**⭐ Star this repo if you find it useful!**