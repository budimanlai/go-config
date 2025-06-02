package config

import (
	"errors"
	"log"
	"path/filepath"
	"strconv"
	"sync"

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
	defer c.mu.Unlock()
	c.storage = make(map[string]string)
	c.file = file

	for _, obj := range c.file {
		ff := NewFile(obj)
		e := ff.Read(c)
		if e != nil {
			return e
		}
	}

	c.WatchAndReload()

	return nil
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
		e := ff.Read(c)
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

	// Watch setiap file config
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
