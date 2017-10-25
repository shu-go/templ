package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	home := homePath()
	if home == "" {
		fmt.Fprintf(os.Stderr, "set env $TEMPL_HOME first.\n")
		return
	}
	os.MkdirAll(home, os.ModeDir|0600)

	app := cli.NewApp()
	app.Name = "templ"
	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Aliases:   []string{"gen"},
			Usage:     "generate a template",
			ArgsUsage: "{name}",
			Flags:     []cli.Flag{
			//cli.StringFlag{Name: "name, n", Usage: "name of the template"},
			},
			Action: func(c *cli.Context) error {
				//name := c.String("name")
				name := c.Args().First()

				if name == "" {
					fmt.Fprintf(os.Stderr, "missing name\n")
					return nil
				}

				if err := runGenerate(home, name); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					return nil
				}
				return nil
			},
		},
		{
			Name:      "check",
			Aliases:   []string{"chk", "test"},
			Usage:     "check a template",
			ArgsUsage: "{name}",
			Flags:     []cli.Flag{
			//cli.StringFlag{Name: "name, n", Usage: "name of the template"},
			},
			Action: func(c *cli.Context) error {
				//name := c.String("name")
				name := c.Args().First()

				if name == "" {
					fmt.Fprintf(os.Stderr, "missing name\n")
					return nil
				}

				if err := runCheck(home, name); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					return nil
				}
				return nil
			},
		},
		{
			Name: "apply",
			//Aliases: []string{"app"},
			Usage:     "apply a template",
			ArgsUsage: "{name} {target}",
			Flags:     []cli.Flag{
			//cli.StringFlag{Name: "name", Usage: "name of the template"},
			//cli.StringFlag{Name: "target", Usage: "path of applied dir"},
			},
			Action: func(c *cli.Context) error {
				//name := c.String("name")
				//target := c.String("target")

				name := c.Args().First()
				target := c.Args().Get(1)

				if name == "" {
					fmt.Fprintf(os.Stderr, "missing name\n")
					return nil
				}
				if target == "" {
					target = "."
				}
				target, _ = filepath.Abs(target)

				if err := runApply(home, name, target); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					return nil
				}
				return nil
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list templates",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "verbose, v", Usage: "verbose output"},
			},
			Action: func(c *cli.Context) error {
				verbose := c.Bool("verbose")
				return runList(home, verbose)
			},
		},
	}
	app.Run(os.Args)
}

func runList(home string, verbose bool) error {
	templs := []string{}

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
		if verbose && len(t.Def.Vars) > 0 {
			fmt.Println("  [Variables]")
			for k, v := range t.Def.Vars {
				fmt.Printf("  - %v:\t%T\n", k, v)
			}
		}
	}

	return nil
}

func runCheck(home, name string) error {
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
func runApply(home, name, target string) error {
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
			if _, err := fmt.Scanln(&val); err != nil {
				val = ""
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

func runGenerate(home, name string) error {
	templ := NewTemplate(filepath.Join(home, name))
	fmt.Printf("generate `%s`...\n", templ.DefPath())

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
