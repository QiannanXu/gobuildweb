package assets

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"sort"

	"github.com/mijia/gobuildweb/loggers"
)

type AssetsMappingItem struct {
	Src    string
	Target string
}

type AssetsMapping struct {
	PkgName  string
	Mappings []AssetsMappingItem
}

func (am *AssetsMapping) AddItem(src, target string) {
	am.Mappings = append(am.Mappings, AssetsMappingItem{src, target})
}

func (am *AssetsMapping) Len() int {
	return len(am.Mappings)
}

func (am *AssetsMapping) Swap(i, j int) {
	am.Mappings[i], am.Mappings[j] = am.Mappings[j], am.Mappings[i]
}

func (am *AssetsMapping) Less(i, j int) bool {
	return am.Mappings[i].Src < am.Mappings[j].Src
}

type _Mappings struct {
	_Asset
}

func Mappings(config Config) _Mappings {
	return _Mappings{
		_Asset: _Asset{config, ""},
	}
}

func (m _Mappings) Build(isProduction bool) error {
	pkgName := m.config.AssetsMappingPkg
	targetPath := "assets_gen.go"
	if pkgName == "" || pkgName == "." || pkgName == "main" {
		pkgName = "main"
	} else {
		goPath := os.Getenv("GOPATH")
		targetPath = path.Join(goPath, "src", pkgName, "assets_gen.go")
		pkgName = path.Base(pkgName)
	}
	mapping := &AssetsMapping{
		PkgName:  pkgName,
		Mappings: make([]AssetsMappingItem, 0),
	}

	err := filepath.Walk("public", func(name string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			name = name[len("public/"):]
			parts := strings.Split(name, "/")
			filename := info.Name()
			if len(parts) > 0 &&
				(parts[0] == "images" || parts[0] == "javascripts" || parts[0] == "stylesheets") &&
				strings.HasPrefix(filename, "fp") {
				target := name
				parts[len(parts)-1] = parts[len(parts)-1][35:]
				src := path.Join(parts...)
				mapping.AddItem(src, target)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Sort(mapping)

	if file, err := os.Create(targetPath); err != nil {
		return fmt.Errorf("Cannot create the assets mapping go file, %+v", err)
	} else {
		defer file.Close()
		if err := tmAssetsMapping.Execute(file, mapping); err != nil {
			return fmt.Errorf("Cannot generate assets mapping file, %+v", err)
		}
	}

	var out bytes.Buffer
	cmd := exec.Command("gofmt", "-w", targetPath)
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		loggers.Error("[AssetMapping] failed to gofmt source code, %v", out.String())
	}
	loggers.Succ("[AssetMappings] Saved asssets mapping go file: %q", targetPath)
	return nil
}

var tmplAssetsMapping = `// This file is generated by GoBuildWeb 
// Containing all the assets mapping data for your router reverse lookup
// Better not to edit this.

package {{.PkgName}}

var allAssetsMapping = map[string]string{
    {{range .Mappings}}"{{.Src}}": "{{.Target}}",
    {{ end }}
}
`
var tmAssetsMapping *template.Template

func init() {
	tmAssetsMapping = template.Must(template.New("assets_mapping").Parse(tmplAssetsMapping))
}