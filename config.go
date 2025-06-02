package config

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"encoding/json"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	storage  map[string]string
	file     []string
	mu       sync.RWMutex // Tambahkan mutex untuk thread safety
	onReload func()       // Callback yang akan dipanggil setelah reload
}

// SetOnReload untuk mendaftarkan callback yang akan dipanggil setelah reload
func (c *Config) SetOnReload(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onReload = fn
}

// Read config file
func (c *Config) Open(file ...string) error {
	if len(file) == 0 {
		return errors.New(`File config blank`)
	}

	c.mu.Lock()
	c.storage = make(map[string]string)
	c.file = file
	c.mu.Unlock()

	for _, obj := range c.file {
		ff := NewFile(obj)
		ext := filepath.Ext(obj)

		var e error
		if ext == ".json" {
			e = ff.ReadJSON(c)
		} else {
			e = ff.ReadIni(c)
		}
		if e != nil {
			return e
		}
	}

	return c.WatchAndReload()
}

// GetString retrieves a string property from the configuration.
// If the property does not exist or is empty, it returns an empty string.
func (c *Config) GetString(name string) string {
	return c.GetStringOr(name, "")
}

// Read string property or retun defValue if property is not exists or empty
func (c *Config) GetStringOr(name string, defValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.storage[name]; ok && val != "" {
		return val
	}
	return defValue
}

// Read integer property. If property is not exists or empty will return 0
func (c *Config) GetInt(name string) int {
	return c.GetIntOr(name, 0)
}

// Read integer property or return defValue if property is not exists or empty
func (c *Config) GetIntOr(name string, defValue int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.storage[name]; ok {
		r, e := strconv.Atoi(val)
		if e != nil {
			return defValue
		}

		return r
	}
	return defValue
}

// Read boolean property. If property is not exists or empty will return false
func (c *Config) GetBool(name string) bool {
	return c.GetBoolOr(name, false)
}

// Read boolean property or return defValue if property is not exists or empty
func (c *Config) GetBoolOr(name string, defValue bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.storage[name]; ok && val != "" {
		r, err := strconv.ParseBool(val)
		if err != nil {
			return defValue
		}
		return r
	}
	return defValue
}

// Read float32 property. If property is not exists or empty will return 0
func (c *Config) GetFloat32(name string) float32 {
	return c.GetFloat32Or(name, 0)
}

// Read float32 property or return defValue if property is not exists or empty
func (c *Config) GetFloat32Or(name string, defValue float32) float32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.storage[name]; ok && val != "" {
		r, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return defValue
		}
		return float32(r)
	}
	return defValue
}

// Read float64 property. If property is not exists or empty will return 0
func (c *Config) GetFloat64(name string) float64 {
	return c.GetFloat64Or(name, 0)
}

// Read float64 property or return defValue if property is not exists or empty
func (c *Config) GetFloat64Or(name string, defValue float64) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.storage[name]; ok && val != "" {
		r, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return defValue
		}
		return r
	}
	return defValue
}

// Reload config file(s) using the last loaded file(s)
func (c *Config) Reload() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.file) == 0 {
		return errors.New("no config file to reload")
	}

	c.storage = make(map[string]string)
	for _, obj := range c.file {
		ff := NewFile(obj)
		ext := filepath.Ext(obj)

		var e error
		if ext == ".json" {
			e = ff.ReadJSON(c)
		} else {
			e = ff.ReadIni(c)
		}
		if e != nil {
			return e
		}
	}

	// Panggil callback jika ada
	if c.onReload != nil {
		go c.onReload()
	}
	return nil
}

// WatchAndReload akan memonitor file config dan otomatis reload jika file berubah.
// Fungsi ini berjalan di goroutine, pastikan dipanggil sekali saja.
func (c *Config) WatchAndReload() error {
	c.mu.RLock()
	files := append([]string{}, c.file...) // copy slice agar aman
	c.mu.RUnlock()

	if len(files) == 0 {
		return errors.New("no config file to watch")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, f := range files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			watcher.Close()
			return err
		}
		err = watcher.Add(absPath)
		if err != nil {
			watcher.Close()
			return err
		}
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Jika file diubah (write/rename/remove), reload config
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Rename == fsnotify.Rename {
					if err := c.Reload(); err != nil {
						log.Printf("Config reload error: %v", err)
					} else {
						log.Printf("Config reloaded: %s", event.Name)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	return nil
}

func (c *Config) GetArrayString(prefix string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var arr []string
	for i := 0; ; i++ {
		key := fmt.Sprintf("%s.%d", prefix, i)
		val, ok := c.storage[key]
		if !ok {
			break
		}
		arr = append(arr, val)
	}
	return arr
}

// GetArrayToStruct mengisi slice struct dari array object yang sudah di-flatten.
// Contoh: GetArrayToStruct("servers", &[]Server{})
func (c *Config) GetArrayToStruct(prefix string, out interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var arr []map[string]interface{}
	for i := 0; ; i++ {
		obj := make(map[string]interface{})
		found := false
		prefixIdx := fmt.Sprintf("%s.%d.", prefix, i)
		for key, val := range c.storage {
			if strings.HasPrefix(key, prefixIdx) {
				field := strings.TrimPrefix(key, prefixIdx)
				// Coba konversi ke int, float64, bool, jika gagal tetap string
				if ival, err := strconv.Atoi(val); err == nil {
					obj[field] = ival
				} else if fval, err := strconv.ParseFloat(val, 64); err == nil {
					obj[field] = fval
				} else if bval, err := strconv.ParseBool(val); err == nil {
					obj[field] = bval
				} else {
					obj[field] = val
				}
				found = true
			}
		}
		if !found {
			break
		}
		arr = append(arr, obj)
	}
	// Marshal ke JSON, lalu unmarshal ke struct slice
	b, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// GetArrayObject mengambil array of object dari config yang sudah di-flatten.
// Contoh: jika JSON berisi "servers": [{"host":"a"},{"host":"b"}],
// maka GetArrayObject("servers", []string{"host"}) akan mengembalikan:
// []map[string]string{ {"host":"a"}, {"host":"b"} }
// Contoh penggunaan:
// servers := cfg.GetArrayObject("servers", []string{"host", "port"})
// servers[0]["host"], servers[0]["port"], dst
func (c *Config) GetArrayObject(prefix string, fields []string) []map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var arr []map[string]string
	for i := 0; ; i++ {
		obj := make(map[string]string)
		found := false
		for _, field := range fields {
			key := fmt.Sprintf("%s.%d.%s", prefix, i, field)
			if val, ok := c.storage[key]; ok {
				obj[field] = val
				found = true
			}
		}
		if !found {
			break
		}
		arr = append(arr, obj)
	}
	return arr
}

// GetArrayObjectAuto mengambil array of object dari config yang sudah di-flatten
// tanpa perlu menentukan field-nya. Fungsi ini akan otomatis mencari semua field
// yang ada pada setiap objek berdasarkan prefix yang diberikan.
// Contoh: jika JSON berisi "servers": [{"host":"a"},{"host":"b"}],
// maka GetArrayObjectAuto("servers") akan mengembalikan:
// []map[string]interface{}{{"host":"a"}, {"host":"b"}}
func (c *Config) GetArrayObjectAuto(prefix string) []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var arr []map[string]interface{}
	for i := 0; ; i++ {
		obj := make(map[string]interface{})
		found := false
		prefixIdx := fmt.Sprintf("%s.%d.", prefix, i)
		for key, val := range c.storage {
			if strings.HasPrefix(key, prefixIdx) {
				field := strings.TrimPrefix(key, prefixIdx)
				obj[field] = val
				found = true
			}
		}
		if !found {
			break
		}
		arr = append(arr, obj)
	}
	return arr
}
