package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"bitbucket.org/shu/templ/templfunc"
)

type TemplateDef struct {
	Description string                 `json:"desc", toml:"desc"`
	Author      string                 `json:"author", toml:"author"`
	Vars        map[string]interface{} `json:"vars", toml:"vars"`
}
type Template struct {
	Path string
	Def  TemplateDef
}

var (
	templDefFileName string = "template.json"
)

func NewTemplate(path string) *Template {
	return &Template{
		Path: path,
		Def: TemplateDef{
			Vars: make(map[string]interface{}),
		},
	}
}

func (t *Template) DefPath() string {
	return filepath.Join(t.Path, templDefFileName)
}

func (t *Template) LoadDef() error {
	if !fileExists(t.DefPath()) {
		// no definition file
		return nil
	}

	data, err := ioutil.ReadFile(t.DefPath())
	if err != nil {
		return fmt.Errorf("read config: %v", err)
	}

	var def TemplateDef
	err = json.Unmarshal(data, &def)
	if err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}

	t.Def = def

	return nil
}

func (t *Template) Save() error {
	if err := os.MkdirAll(t.Path, os.ModeDir|0600); err != nil {
		return fmt.Errorf("mkdir: %v", err)
	}

	data, err := json.MarshalIndent(t.Def, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}

	if err := ioutil.WriteFile(t.DefPath(), data, 0600); err != nil {
		return fmt.Errorf("write: %v", err)
	}

	return nil
}

func (t *Template) ApplyTo(destPath string) error {
	if !fileExists(t.Path) {
		return fmt.Errorf("template path does not exist: %v", t.Path)
	}
	if !fileExists(destPath) {
		return fmt.Errorf("dest path does not exist: %v", destPath)
	}

	t.Def.Vars["DEST_PATH"] = destPath

	if err := filepath.Walk(t.Path, func(e string, _ os.FileInfo, _ error) error {
		if e == t.Path || strings.ToUpper(filepath.Base(e)) == strings.ToUpper(templDefFileName) {
			return nil
		}

		fmt.Fprintf(os.Stdout, "  %v\n", strings.Replace(e, t.Path+string(filepath.Separator), "", -1))

		exrelpath, excontent, fileInfo, err := t.ExpandEachTempl(e)
		if err != nil {
			return fmt.Errorf("apply template: %v", err)
		}

		if fileInfo.IsDir() {
			err = os.MkdirAll(filepath.Join(destPath, exrelpath), fileInfo.Mode())
			if err != nil {
				return fmt.Errorf("apply template: %v", err)
			}
		} else {
			err = ioutil.WriteFile(filepath.Join(destPath, exrelpath), excontent, fileInfo.Mode())
			if err != nil {
				return fmt.Errorf("apply template: %v", err)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type TemplatingErrorElem struct {
	FileName string
	Error    string
}

func (t *Template) ExpandingErrors() []TemplatingErrorElem {
	var errors []TemplatingErrorElem

	destPath := "."

	if !fileExists(t.Path) {
		fmt.Println("template path does not exist")
	}

	t.Def.Vars["DEST_PATH"] = destPath

	//entries, _ := filepath.Glob(filepath.Join(t.Path, "*"))
	filepath.Walk(t.Path, func(e string, _ os.FileInfo, _ error) error {
		if strings.ToUpper(filepath.Base(e)) == strings.ToUpper(templDefFileName) {
			return nil
		}

		exrelpath, excontent, _, err := t.ExpandEachTempl(e)
		if err != nil {
			errors = append(errors, TemplatingErrorElem{
				FileName: e,
				Error:    fmt.Sprintf("error %v\n", err),
			})
			return nil
		}

		if strings.Contains(exrelpath, "<no value>") {
			errors = append(errors, TemplatingErrorElem{
				FileName: e,
				Error:    "name contains <no value>  => " + exrelpath,
			})
		}
		strcontent := string(excontent)
		if pos := strings.Index(strcontent, "<no value>"); pos != -1 {
			lineNum := strings.Count(strcontent[:pos], "\n") + 1
			errors = append(errors, TemplatingErrorElem{
				FileName: e,
				Error:    fmt.Sprintf("contains <no value> at line %d\n", lineNum),
			})
		}

		return nil
	})

	return errors
}

func (t *Template) ExpandEachTempl(path string) (string, []byte, os.FileInfo, error) {
	relpath := path[len(t.Path):] // => /relpath/to/fileordir

	funcMap := template.FuncMap{
		//"time": strings.Title,
		"time": templfunc.Time,
	}

	// name
	nameTempl, err := template.New("name").Funcs(funcMap).Parse(relpath)
	if err != nil {
		return "", nil, nil, fmt.Errorf("parse name `%v`: %v", relpath, err)
	}
	buf := &bytes.Buffer{}
	err = nameTempl.Execute(buf, t.Def.Vars)
	if err != nil {
		return "", nil, nil, fmt.Errorf("parse name `%v`: %v", relpath, err)
	}
	name := buf.String()

	// stat
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read content `%v`: %v", relpath, err)
	}

	if info.IsDir() {
		return name, nil, info, nil
	}

	// content
	// read content
	rawcontent, err := ioutil.ReadFile(path)
	if err != nil {
		return "", nil, nil, fmt.Errorf("read content `%v`: %v", relpath, err)
	}
	contentTempl, err := template.New("content").Funcs(funcMap).Parse(string(rawcontent))
	if err != nil {
		return "", nil, nil, fmt.Errorf("parse content `%v`: %v", relpath, err)
	}
	buf = &bytes.Buffer{}
	err = contentTempl.Execute(buf, t.Def.Vars)
	if err != nil {
		return "", nil, nil, fmt.Errorf("parse content `%v`: %v", relpath, err)
	}
	content := buf.Bytes()

	return name, content, info, nil
}
