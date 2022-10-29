package config

import (
	"bufio"
	"fmt"
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

func (f *File) Read(c *Config) error {
	fi, e := os.Open(f.filename)
	if e != nil {
		return e
	}
	defer fi.Close()

	fmt.Println(`Read config:`, f.filename)
	c.file = append(c.file, f.filename)

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
			keyPath := root + "." + key
			c.storage[keyPath] = val
		} else if matches := regexRoot.FindStringSubmatch(strLine); len(matches) > 0 {
			root = matches[2]
		} else if matches := regexInclude.FindStringSubmatch(strLine); len(matches) >= 2 {
			path := matches[1]

			if !contains(c.file, path) {
				f2 := NewFile(path)
				e := f2.Read(c)
				if e != nil {
					return e
				}
				c.file = append(c.file, path)
			} else {
				fmt.Println(`Skippp.. already read`, path)
			}
		}
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
