package config

import (
	"errors"
	"strconv"
)

type Config struct {
	storage map[string]string
	file    []string
}

// Read config file
func (c *Config) Open(file ...string) error {
	if len(file) == 0 {
		return errors.New(`File config blank`)
	}

	c.storage = make(map[string]string)

	for _, obj := range file {
		ff := NewFile(obj)
		e := ff.Read(c)
		if e != nil {
			return e
		}
	}
	return nil
}

func (c *Config) GetString(name string) string {
	return c.GetStringOr(name, "")
}

// Read string property or retun defValue if property is not exists or empty
func (c *Config) GetStringOr(name string, defValue string) string {
	if val, ok := c.storage[name]; ok {
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
	if val, ok := c.storage[name]; ok {
		r, e := strconv.Atoi(val)
		if e != nil {
			return defValue
		}

		return r
	}
	return defValue
}
