package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bitbucket.org/shu/gli"
)

type globalCmd struct {
	Generate generateCmd `cli:"generate, gen"  help:"generate a template"  usage:"templ generate {TEMPL_NAME}"`
	Check    checkCmd    `cli:"check, chk, test"  help:"check a template validity"  usage:"templ check {TEMPL_NAME}"`
	Apply    applyCmd    `cli:"apply"  help:"apply a template"  usage:"templ apply {TEMPL_NAME} {DEST_DIR(default: .)}"`
	List     listCmd     `cli:"list, ls"  help:"list templates"  usage:"templ list [-v]}"`
}

type generateCmd struct{}

func (c generateCmd) Run(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		fmt.Fprintf(os.Stderr, "missing name\n")
		return nil
	}

	home := homePath()

	templ := NewTemplate(filepath.Join(home, name))
	fmt.Printf("generate `%s`...\n", templ.Path)

	templ.Def.Author = "author"
	templ.Def.Description = "description of this template."

	// define sample variables
	templ.Def.Vars["_SampleStrPrompted"] = ""
	templ.Def.Vars["_SampleNumPrompted"] = 0
	templ.Def.Vars["SampleStr"] = "value1"
	templ.Def.Vars["SampleNum"] = 100
	templ.Def.Vars["SampleList"] = []interface{}{"a", "b", 100}
	templ.Def.Vars["SampleMap"] = map[string]interface{}{"key1": "a", "key2": 100}

	if err := templ.Save(); err != nil {
		return fmt.Errorf("save: %v", err)
	}

	// extras
	os.MkdirAll(filepath.Join(home, name, "this_is_based_on"), os.ModeDir|0600)
	ioutil.WriteFile(filepath.Join(home, name, "this_is_based_on", "file_{{ .SampleStr }}_{{ .SampleNum }}.txt"), []byte(`SampleList:
{{ range .SampleList }}
{{- "" }}  - {{ . }}
{{ end }}

Map:
{{ range $key, $value := .SampleMap }}
{{- "" }}  {{ $key }} : {{ $value }}
{{ end }}

Prompted:
  - _SampleStrPrompted : {{ ._SampleStrPrompted }}
  - _SampleNumPrompted : {{ ._SampleNumPrompted }}
`), 0600)

	return nil
}

type checkCmd struct{}

func (c checkCmd) Run(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		fmt.Fprintf(os.Stderr, "missing name\n")
		return nil
	}

	home := homePath()

	templ := NewTemplate(filepath.Join(home, name))
	fmt.Printf("check `%s`...\n", templ.Path)

	if err := templ.LoadDef(); err != nil {
		return fmt.Errorf("load def: %v", err)
	}

	errors := templ.ExpandingErrors()
	if len(errors) == 0 {
		fmt.Println("OK")
	} else {
		var fileName string
		for _, e := range errors {
			if fileName != e.FileName {
				fileName = e.FileName

				fmt.Println()
				fmt.Println(strings.Replace(fileName, templ.Path+string(filepath.Separator), "", -1))
			}
			fmt.Println("  - " + e.Error)
		}
	}

	return nil
}

type applyCmd struct{}

func (c applyCmd) Run(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		fmt.Fprintf(os.Stderr, "missing name\n")
		return nil
	}

	var target string
	if len(args) > 1 {
		target = args[1]
	} else {
		target = "."
	}
	target, _ = filepath.Abs(target)

	home := homePath()

	templ := NewTemplate(filepath.Join(home, name))
	fmt.Printf("apply `%s`...\n", templ.Path)

	if err := templ.LoadDef(); err != nil {
		return fmt.Errorf("load def: %v", err)
	}

	os.MkdirAll(target, os.ModeDir|0600)

	fmt.Println()
	for k, v := range templ.Def.Vars {
		if strings.HasPrefix(k, "_") {
			fmt.Printf("%s (%T): ", k, v)

			val := ""
			reader := bufio.NewReader(os.Stdin)
			if scanval, err := reader.ReadString('\n'); err != nil {
				fmt.Fprintf(os.Stderr, "scan: %v", err)
				val = ""
			} else {
				val = strings.TrimSpace(scanval)
			}

			if _, ok := v.(float64); ok {
				if val == "" {
					val = "0"
				}
				fval, err := strconv.ParseFloat(val, 10)
				if err != nil {
					return fmt.Errorf("re-defining var %q to %v: %v", k, val, err)
				}
				templ.Def.Vars[k] = fval
			} else {
				templ.Def.Vars[k] = val
			}
		}
	}
	fmt.Println()

	if err := templ.ApplyTo(target); err != nil {
		return fmt.Errorf("apply: %v", err)
	}

	return nil
}

type listCmd struct {
	Verbose bool `cli:"verbose, v"  help:"verbose output"`
}

func (c listCmd) Run(args []string) error {
	templs := []string{}

	home := homePath()

	dirs, err := filepath.Glob(filepath.Join(home, "*"))
	if err != nil {
		return err
	}
	for _, v := range dirs {
		if stat, err := os.Lstat(v); err != nil {
			continue
		} else if stat.IsDir() {
			templs = append(templs, v)
		}
	}

	for _, v := range templs {
		fmt.Println("* " + filepath.Base(v))

		t := NewTemplate(v)
		if err := t.LoadDef(); err != nil {
			fmt.Printf("  failed to load definitions: %s\n", err)
			continue
		}

		if t.Def.Description != "" {
			fmt.Println("  [Description]")
			fmt.Printf("  %v\n", t.Def.Description)
		}

		// do something?
		if c.Verbose && len(t.Def.Vars) > 0 {
			fmt.Println("  [Variables]")
			for k, v := range t.Def.Vars {
				fmt.Printf("  - %v:\t%T\n", k, v)
			}
		}
	}

	return nil
}

func main() {
	home := homePath()
	if home == "" {
		fmt.Fprintf(os.Stderr, "set env TEMPL_HOME first.\nthen `templ help`\n")
		return
	}
	os.MkdirAll(home, os.ModeDir|0600)

	app := gli.New(&globalCmd{})
	app.Name = "templ"
	app.Desc = "file templater"
	app.Version = "0.2.0"
	app.Usage = `set env $TEMPL_HOME as a template repository(storage).
then, templ gen {your template name here}
then, cd to where to apply the template
then, templ apply {the template name}`
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
