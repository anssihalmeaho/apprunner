package extensions

import (
	"embed"
	"strings"

	"github.com/anssihalmeaho/funl/funl"
)

// CallMe is just for taking this into build
func CallMe() {}

//go:embed *.fnl
var content embed.FS

func init() {
	funl.AddExtensionInitializer(addFunlModules)
}

func addFunlModules(interpreter *funl.Interpreter) error {
	entries, err := content.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		filename := entry.Name()
		source, err := content.ReadFile(filename)
		if err != nil {
			return err
		}
		modName := strings.Split(filename, ".")[0]
		if err = funl.AddFunModToNamespace(modName, source, interpreter); err != nil {
			return err
		}
	}
	return nil
}
