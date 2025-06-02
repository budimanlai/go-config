package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type File struct {
	filename string
}

const (
	strRootLine = `^(?Ui)\s*([-]|)\[([a-z0-9]+)\].*$`
	strLine     = `^(?Ui)\s*([a-z0-9_.]+)\s*=\s*(.*)(\s+(?:#|/{2,}).*|)\s*$`
	strInclude  = `^include\s*(.*)\s*`
)

func NewFile(name string) File {
	return File{
		filename: name,
	}
}

func (f *File) ReadIni(c *Config) error {
	fi, e := os.Open(f.filename)
	if e != nil {
		return e
	}
	defer fi.Close()

	fmt.Println(`Read config:`, f.filename)

	scanner := bufio.NewScanner(fi)
	regexLine := regexp.MustCompile(strLine)
	regexRoot := regexp.MustCompile(strRootLine)
	regexInclude := regexp.MustCompile(strInclude)

	root := ``

	for scanner.Scan() {
		strLine := scanner.Text()

		if matches := regexLine.FindStringSubmatch(strLine); len(matches) > 0 {
			key := strings.TrimSpace(matches[1])
			val := strings.TrimSpace(matches[2])
			if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
				val = val[1 : len(val)-1]
			}
			keyPath := key
			if root != "" {
				keyPath = root + "." + key
			}
			c.storage[keyPath] = val
		} else if matches := regexRoot.FindStringSubmatch(strLine); len(matches) > 0 {
			root = matches[2]
		} else if matches := regexInclude.FindStringSubmatch(strLine); len(matches) >= 2 {
			path := matches[1]
			if !contains(c.file, path) {
				f2 := NewFile(path)
				e := f2.ReadIni(c)
				if e != nil {
					return e
				}
				// Hindari append ke c.file di sini
			} else {
				fmt.Println(`Skippp.. already read`, path)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// Membaca file JSON dan flatten ke storage
func (f *File) ReadJSON(c *Config) error {
	data, err := ioutil.ReadFile(f.filename)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	flattenJSON("", m, c.storage)
	return nil
}

func flattenJSON(prefix string, m map[string]interface{}, storage map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			flattenJSON(key, val, storage)
		case []interface{}:
			for i, item := range val {
				arrKey := fmt.Sprintf("%s.%d", key, i)
				switch itemVal := item.(type) {
				case map[string]interface{}:
					flattenJSON(arrKey, itemVal, storage)
				case []interface{}:
					// Nested array, flatten recursively
					flattenJSON(arrKey, map[string]interface{}{"": itemVal}, storage)
				default:
					storage[arrKey] = fmt.Sprintf("%v", itemVal)
				}
			}
		default:
			storage[key] = fmt.Sprintf("%v", val)
		}
	}
}
