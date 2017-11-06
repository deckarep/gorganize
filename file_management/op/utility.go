package op

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func filenameAndExt(path string) (string, string) {
	justExt := filepath.Ext(path)
	justName := path[0 : len(path)-len(justExt)]
	return justName, justExt
}

func createDirIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			logrus.Fatalf("Couldn't create destination dir:%s with err: %s", path, err)
		}
	}
}
