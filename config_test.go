package config_test

import (
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

	called := false
	cfg.SetOnReload(func() { called = true })

	// Ubah file, lalu reload
	writeFile("testdata/reload2.conf", `
[main]
val = 2
`)
	_ = cfg.Reload()
	assert.True(t, called)
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
