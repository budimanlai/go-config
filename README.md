# go-config

**go-config** adalah library Go sederhana untuk membaca file konfigurasi dengan format `.ini` dan `.json` yang mendukung:
- Thread safety (aman untuk aplikasi concurrent)
- Reload otomatis saat file berubah (hot reload, menggunakan fsnotify)
- Nested config (key dengan notasi titik)
- Array dan array of object (akses dengan notasi index)
- Callback saat reload
- API sederhana untuk akses berbagai tipe data

---

## Instalasi

```sh
go get github.com/budimanlai/go-config
```

---

## Fitur Utama

- Baca file `.ini` dan `.json`
- Mendukung section, include, dan komentar pada `.ini`
- Mendukung nested object dan array pada `.json`
- Reload otomatis saat file berubah (hot reload)
- Callback saat reload
- Thread-safe

---

## Contoh Penggunaan

### 1. Membaca File Config

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

    // Akses value
    host := cfg.GetString("database.host")
    port := cfg.GetInt("database.port")
    debug := cfg.GetBool("app.debug")
}
```

### 2. Reload Otomatis & Callback

```go
cfg.SetOnReload(func() {
    log.Println("Config di-reload otomatis!")
})
```

---

## API

### Akses Value

```go
cfg.GetString("key")
cfg.GetStringOr("key", "default")
cfg.GetInt("key")
cfg.GetIntOr("key", 123)
cfg.GetBool("key")
cfg.GetBoolOr("key", true)
cfg.GetFloat32("key")
cfg.GetFloat64("key")
```

### Array & Object

```go
arr := cfg.GetArrayString("arr") // []string
objs := cfg.GetArrayObject("servers", []string{"host", "port"}) // []map[string]string
objsAuto := cfg.GetArrayObjectAuto("servers") // []map[string]interface{}

// Isi slice struct dari array object
type Server struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
var servers []Server
cfg.GetArrayToStruct("servers", &servers)
```

---

## Format File

### Contoh `.ini`

```ini
[database]
host = localhost
port = 3306
enabled = true
```

### Contoh `.json`

```json
{
  "app": {
    "debug": true
  },
  "servers": [
    {"host": "a", "port": 1},
    {"host": "b", "port": 2}
  ],
  "arr": ["x", "y", "z"]
}
```

---

## Reload Manual

```go
cfg.Reload()
```

---

## Lisensi

MIT