package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	storage     map[string]string
	file        []string
	mu          sync.RWMutex                       // Thread safety untuk storage
	onReload    func()                             // Callback yang akan dipanggil setelah reload
	watcherOnce sync.Once                          // Proteksi agar WatchAndReload hanya bisa dipanggil sekali
	watcher     *fsnotify.Watcher                  // Keep reference untuk cleanup
	typeCache   map[string]map[string]reflect.Type // Cache untuk reflection results
	typeCacheMu sync.RWMutex                       // Mutex untuk type cache
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
		return errors.New("config file path cannot be empty")
	}

	c.mu.Lock()
	if c.storage == nil {
		c.storage = make(map[string]string)
	} else {
		// Clear existing storage
		for k := range c.storage {
			delete(c.storage, k)
		}
	}
	c.file = make([]string, len(file))
	copy(c.file, file)
	c.mu.Unlock()

	// Initialize type cache if not exists
	c.typeCacheMu.Lock()
	if c.typeCache == nil {
		c.typeCache = make(map[string]map[string]reflect.Type)
	}
	c.typeCacheMu.Unlock()

	for _, obj := range c.file {
		ff := NewFile(obj)
		ext := filepath.Ext(obj)

		var err error
		if ext == ".json" {
			err = ff.ReadJSON(c)
		} else {
			err = ff.ReadIni(c)
		}
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", obj, err)
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
	var err error
	c.watcherOnce.Do(func() {
		c.mu.RLock()
		files := make([]string, len(c.file))
		copy(files, c.file) // Safe copy
		c.mu.RUnlock()

		if len(files) == 0 {
			err = errors.New("no config file to watch")
			return
		}

		watcher, e := fsnotify.NewWatcher()
		if e != nil {
			err = fmt.Errorf("failed to create file watcher: %w", e)
			return
		}

		// Store watcher reference for cleanup
		c.mu.Lock()
		c.watcher = watcher
		c.mu.Unlock()

		for _, f := range files {
			absPath, e := filepath.Abs(f)
			if e != nil {
				watcher.Close()
				err = fmt.Errorf("failed to get absolute path for %s: %w", f, e)
				return
			}
			e = watcher.Add(absPath)
			if e != nil {
				watcher.Close()
				err = fmt.Errorf("failed to watch file %s: %w", absPath, e)
				return
			}
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Config watcher panic recovered: %v", r)
				}
				watcher.Close()
			}()

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
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
	})
	return err
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

// GetAll mengembalikan copy dari semua setting yang sudah di-load.
// Mengembalikan map[string]string yang berisi semua key-value pairs.
func (c *Config) GetAll() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Buat copy dari storage untuk mencegah modifikasi dari luar
	result := make(map[string]string, len(c.storage))
	for key, value := range c.storage {
		result[key] = value
	}
	return result
}

// GetAllAsInterface mengembalikan copy dari semua setting yang sudah di-load
// dengan mencoba mengkonversi nilai ke tipe data yang sesuai (int, float64, bool, string).
// Mengembalikan map[string]interface{} yang berisi semua key-value pairs.
func (c *Config) GetAllAsInterface() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Buat copy dari storage dengan konversi tipe data
	result := make(map[string]interface{}, len(c.storage))
	for key, value := range c.storage {
		// Coba konversi ke tipe data yang sesuai
		if ival, err := strconv.Atoi(value); err == nil {
			result[key] = ival
		} else if fval, err := strconv.ParseFloat(value, 64); err == nil {
			result[key] = fval
		} else if bval, err := strconv.ParseBool(value); err == nil {
			result[key] = bval
		} else {
			result[key] = value
		}
	}
	return result
}

// GetAllAsJSON mengembalikan semua setting dalam format JSON string.
// Nilai akan dikonversi ke tipe data yang sesuai sebelum di-marshal ke JSON.
func (c *Config) GetAllAsJSON() (string, error) {
	data := c.GetAllAsInterface()
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// GetAllKeys mengembalikan slice yang berisi semua key yang tersedia di config.
func (c *Config) GetAllKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.storage))
	for key := range c.storage {
		keys = append(keys, key)
	}
	return keys
}

// MapToStruct memetakan semua setting ke struct yang diberikan.
// Fungsi ini menggunakan tag JSON untuk mapping field struct dengan key config.
// Contoh penggunaan:
//
//	type AppConfig struct {
//	    Database struct {
//	        Host string `json:"database.host"`
//	        Port int    `json:"database.port"`
//	    }
//	    App struct {
//	        Name  string `json:"app.name"`
//	        Debug bool   `json:"app.debug"`
//	    }
//	}
//	var config AppConfig
//	err := cfg.MapToStruct(&config)
func (c *Config) MapToStruct(out interface{}) error {
	data := c.GetAllAsInterface()

	// Marshal ke JSON dulu
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling config data: %w", err)
	}

	// Unmarshal ke struct
	if err := json.Unmarshal(jsonBytes, out); err != nil {
		return fmt.Errorf("error unmarshaling to struct: %w", err)
	}

	return nil
}

// MapToStructFlat memetakan semua setting ke struct dengan struktur flat.
// Fungsi ini cocok untuk struct yang field-nya langsung menggunakan key config sebagai tag JSON.
// Contoh penggunaan:
//
//	type FlatConfig struct {
//	    DatabaseHost string `json:"database.host"`
//	    DatabasePort int    `json:"database.port"`
//	    AppName      string `json:"app.name"`
//	    AppDebug     bool   `json:"app.debug"`
//	}
//	var config FlatConfig
//	err := cfg.MapToStructFlat(&config)
func (c *Config) MapToStructFlat(out interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Gunakan reflection untuk mengisi struct
	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("output must be a pointer to struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()

	for i := 0; i < structVal.NumField(); i++ {
		field := structVal.Field(i)
		fieldType := structType.Field(i)

		// Skip field yang tidak bisa di-set
		if !field.CanSet() {
			continue
		}

		// Ambil tag JSON untuk mendapatkan key config
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse tag JSON (ambil bagian pertama sebelum koma)
		configKey := strings.Split(jsonTag, ",")[0]

		// Ambil nilai dari storage
		configValue, exists := c.storage[configKey]
		if !exists {
			continue
		}

		// Set nilai ke field berdasarkan tipe data
		if err := setFieldValue(field, configValue); err != nil {
			return fmt.Errorf("error setting field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// Helper function untuk set nilai field berdasarkan tipe data
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(intVal)
		} else {
			return fmt.Errorf("cannot convert %s to int", value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			field.SetUint(uintVal)
		} else {
			return fmt.Errorf("cannot convert %s to uint", value)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(floatVal)
		} else {
			return fmt.Errorf("cannot convert %s to float", value)
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			field.SetBool(boolVal)
		} else {
			return fmt.Errorf("cannot convert %s to bool", value)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// MapToStructNested memetakan semua setting ke struct dengan struktur nested/bertingkat.
// Menggunakan approach type-aware dengan caching untuk performance
func (c *Config) MapToStructNested(out interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Step 1: Get field types dengan caching
	fieldTypes := c.extractFieldTypesWithCache(out)

	// Step 2: Konversi flat storage ke nested map dengan type-aware conversion
	nestedData := c.flatToNestedWithTypeAwareness(fieldTypes)

	// Step 3: Marshal ke JSON bytes (pre-allocate buffer untuk performance)
	jsonBytes, err := json.Marshal(nestedData)
	if err != nil {
		return fmt.Errorf("error marshaling nested data: %w", err)
	}

	// Step 4: Unmarshal ke struct - json.Unmarshal built-in akan handle konversi
	if err := json.Unmarshal(jsonBytes, out); err != nil {
		return fmt.Errorf("error unmarshaling to nested struct: %w", err)
	}

	return nil
}

// extractFieldTypesWithCache menggunakan cache untuk reflection results
func (c *Config) extractFieldTypesWithCache(out interface{}) map[string]reflect.Type {
	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr {
		return make(map[string]reflect.Type)
	}

	structVal := val.Elem()
	if structVal.Kind() != reflect.Struct {
		return make(map[string]reflect.Type)
	}

	structType := structVal.Type()
	typeName := structType.String()

	// Check cache first
	c.typeCacheMu.RLock()
	if cached, exists := c.typeCache[typeName]; exists {
		c.typeCacheMu.RUnlock()
		return cached
	}
	c.typeCacheMu.RUnlock()

	// Not in cache, compute and store
	fieldTypes := make(map[string]reflect.Type)
	c.extractFieldTypesRecursive("", structType, fieldTypes)

	c.typeCacheMu.Lock()
	if c.typeCache == nil {
		c.typeCache = make(map[string]map[string]reflect.Type)
	}
	c.typeCache[typeName] = fieldTypes
	c.typeCacheMu.Unlock()

	return fieldTypes
}

// extractFieldTypesRecursive melakukan recursive extraction untuk nested struct
func (c *Config) extractFieldTypesRecursive(prefix string, structType reflect.Type, fieldTypes map[string]reflect.Type) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip field yang tidak exported
		if !field.IsExported() {
			continue
		}

		// Ambil JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag (ambil nama sebelum koma)
		fieldName := strings.Split(jsonTag, ",")[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		// Buat full path
		fullPath := fieldName
		if prefix != "" {
			fullPath = prefix + "." + fieldName
		}

		fieldType := field.Type

		// Handle pointer types
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Jika field adalah struct, recursive
		if fieldType.Kind() == reflect.Struct {
			c.extractFieldTypesRecursive(fullPath, fieldType, fieldTypes)
		} else if fieldType.Kind() == reflect.Slice {
			// Handle slice/array
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Struct {
				// Slice of struct - map dengan pattern "path.id", "path.name", etc (tanpa index)
				// Ini akan memungkinkan type-aware conversion untuk semua element di slice
				c.extractFieldTypesRecursive(fullPath, elemType, fieldTypes)
			} else {
				// Slice of primitive - map langsung ke element type
				fieldTypes[fullPath] = elemType
			}
		} else {
			// Primitive type - map langsung
			fieldTypes[fullPath] = fieldType
		}
	}
}

// flatToNestedWithTypeAwareness mengkonversi flat keys menjadi nested structure
// dengan menggunakan informasi tipe dari struct target untuk konversi yang tepat
func (c *Config) flatToNestedWithTypeAwareness(fieldTypes map[string]reflect.Type) map[string]interface{} {
	// Pre-allocate dengan estimated size untuk mengurangi reallocations
	result := make(map[string]interface{}, len(c.storage)/3) // Estimate nested depth

	// Use string builder untuk optimasi string operations
	var keyBuilder strings.Builder
	keyBuilder.Grow(64) // Pre-allocate reasonable size

	for key, value := range c.storage {
		parts := strings.Split(key, ".")

		// Konversi value berdasarkan tipe yang dibutuhkan
		convertedValue := c.convertValueByTargetType(key, value, fieldTypes)

		// Buat nested structure dengan optimized path
		current := result
		for i, part := range parts {
			if i == len(parts)-1 {
				// Ini adalah leaf node, set value
				current[part] = convertedValue
			} else {
				// Ini adalah intermediate node, buat map jika belum ada
				if _, exists := current[part]; !exists {
					// Pre-allocate dengan estimated size
					current[part] = make(map[string]interface{}, 4)
				}
				// Pastikan tipe data adalah map[string]interface{}
				if nextMap, ok := current[part].(map[string]interface{}); ok {
					current = nextMap
				} else {
					// Jika sudah ada value di key ini, skip
					break
				}
			}
		}
	}

	// Post-process untuk mengkonversi map dengan key numerik menjadi array
	processed := c.convertMapToArrayRecursive(result)
	if processedMap, ok := processed.(map[string]interface{}); ok {
		return processedMap
	}
	return result
}

// convertValueByTargetType mengkonversi string value berdasarkan tipe target yang dibutuhkan
func (c *Config) convertValueByTargetType(key, value string, fieldTypes map[string]reflect.Type) interface{} {
	// Cari tipe target untuk key ini
	targetType, exists := fieldTypes[key]
	if !exists {
		// Coba cari dengan pattern array (hilangkan index)
		keyWithoutIndex := c.removeArrayIndex(key)
		targetType, exists = fieldTypes[keyWithoutIndex]
		if !exists {
			// Fallback ke auto-detection
			return c.convertStringToJSONType(value)
		}
	}

	// Konversi berdasarkan tipe target
	switch targetType.Kind() {
	case reflect.String:
		return value // Tetap string - ini yang menyelesaikan masalah!
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
		return value // Fallback ke string jika gagal parse
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uintVal
		}
		return value
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
		return value
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		return value
	default:
		return value
	}
}

// removeArrayIndex menghilangkan index array dari key untuk mencari tipe element
// Contoh: "servers.0.port" -> "servers.port", "numbers.1" -> "numbers"
// Optimized version menggunakan string builder untuk mengurangi allocations
func (c *Config) removeArrayIndex(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) <= 1 {
		return key
	}

	// Pre-allocate builder dengan estimated size
	var builder strings.Builder
	builder.Grow(len(key)) // Worst case sama dengan input

	first := true
	for _, part := range parts {
		// Skip jika part adalah angka (index array)
		if _, err := strconv.Atoi(part); err != nil {
			if !first {
				builder.WriteByte('.')
			}
			builder.WriteString(part)
			first = false
		}
	}

	return builder.String()
}

// convertMapToArrayRecursive mengkonversi map dengan key numerik berurutan menjadi array
func (c *Config) convertMapToArrayRecursive(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Cek apakah ini map dengan key numerik berurutan yang bisa jadi array
		if c.isNumericArrayMap(v) {
			return c.convertToArray(v)
		}

		// Recursive untuk nested map
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = c.convertMapToArrayRecursive(value)
		}
		return result
	case []interface{}:
		// Recursive untuk array
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = c.convertMapToArrayRecursive(value)
		}
		return result
	default:
		return data
	}
}

// isNumericArrayMap mengecek apakah map memiliki key berupa angka berurutan mulai dari 0
func (c *Config) isNumericArrayMap(m map[string]interface{}) bool {
	if len(m) == 0 {
		return false
	}

	// Cek apakah semua key adalah angka berurutan mulai dari 0
	for i := 0; i < len(m); i++ {
		if _, exists := m[strconv.Itoa(i)]; !exists {
			return false
		}
	}
	return true
}

// convertToArray mengkonversi map dengan key numerik menjadi array
func (c *Config) convertToArray(m map[string]interface{}) []interface{} {
	result := make([]interface{}, len(m))
	for i := 0; i < len(m); i++ {
		result[i] = c.convertMapToArrayRecursive(m[strconv.Itoa(i)])
	}
	return result
}

// convertStringToJSONType mengkonversi string value ke proper JSON type (fallback function)
func (c *Config) convertStringToJSONType(value string) interface{} {
	// Coba bool dulu
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal
	}

	// Coba int
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Coba float64
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Default: tetap sebagai string
	return value
}

// MapToStructAdvanced memetakan setting ke struct dengan dukungan untuk:
// 1. Nested struct dengan tag JSON
// 2. Flat mapping dengan dot notation
// 3. Array/slice mapping
// Fungsi ini otomatis mendeteksi apakah struct menggunakan nested atau flat mapping.
func (c *Config) MapToStructAdvanced(out interface{}) error {
	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("output must be a pointer to struct")
	}

	structVal := val.Elem()
	structType := structVal.Type()

	// Deteksi apakah menggunakan nested struktur
	useNested := c.detectNestedStructure(structType)

	if useNested {
		// Gunakan nested mapping
		return c.MapToStructNested(out)
	} else {
		// Gunakan flat mapping
		return c.MapToStructFlat(out)
	}
}

// detectNestedStructure mendeteksi apakah struct menggunakan struktur nested
func (c *Config) detectNestedStructure(structType reflect.Type) bool {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Jika ada field yang berupa struct (bukan pointer) dan memiliki tag JSON tanpa dot
		if field.Type.Kind() == reflect.Struct {
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" && !strings.Contains(jsonTag, ".") {
				return true
			}
		}
	}
	return false
}

// Stats returns usage statistics for monitoring in production
type ConfigStats struct {
	StorageSize  int  `json:"storage_size"`
	FilesWatched int  `json:"files_watched"`
	CacheSize    int  `json:"cache_size"`
	IsWatching   bool `json:"is_watching"`
}

// GetStats returns current configuration statistics for monitoring
func (c *Config) GetStats() ConfigStats {
	c.mu.RLock()
	storageSize := len(c.storage)
	filesWatched := len(c.file)
	isWatching := c.watcher != nil
	c.mu.RUnlock()

	c.typeCacheMu.RLock()
	cacheSize := len(c.typeCache)
	c.typeCacheMu.RUnlock()

	return ConfigStats{
		StorageSize:  storageSize,
		FilesWatched: filesWatched,
		CacheSize:    cacheSize,
		IsWatching:   isWatching,
	}
}

// ClearCache clears the type reflection cache (useful for long-running processes)
func (c *Config) ClearCache() {
	c.typeCacheMu.Lock()
	defer c.typeCacheMu.Unlock()

	// Clear but don't nil - reuse the map
	for k := range c.typeCache {
		delete(c.typeCache, k)
	}
}

// Close cleans up resources used by the config instance
// Should be called when the config is no longer needed (production best practice)
func (c *Config) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.watcher != nil {
		err := c.watcher.Close()
		c.watcher = nil
		return err
	}

	// Clear type cache
	c.typeCacheMu.Lock()
	c.typeCache = nil
	c.typeCacheMu.Unlock()

	return nil
}
