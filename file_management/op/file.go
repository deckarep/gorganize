package op

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	mapset "github.com/deckarep/golang-set"
	md5 "github.com/deckarep/gorganize/file_management/md5"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// FlattenFolderByExtension will take a source folder, find all files by the extensions
// argument and flatten the found files into a single destination folder.
func FlattenFolderByExtension(sourceFolder, destFolder string, extensions mapset.Set) {
	// 2.) Begin walking filesystem.
	err := filepath.Walk(
		filepath.Join(sourceFolder, "C"),
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				extensions.Each(func(item interface{}) bool {
					ext := item.(string)
					pathLowerCase := strings.ToLower(path)
					if strings.ToLower(filepath.Ext(pathLowerCase)) == ext {
						sourceFile := pathLowerCase
						destFile := filepath.Join(destFolder, filepath.Base(pathLowerCase))

						err := CopyFile(sourceFile, destFile)
						if err != nil {
							logrus.Errorf("Failed to copy file: %s to dest %s with err: %s", sourceFile, destFile, err.Error())
						}
					}
					return false
				})
			}
			return nil
		})

	if err != nil {
		logrus.Fatal("Couldn't walk the root folder with err: ", err.Error())
	}
}

// CopyFile the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "couldn't open src file during copyFile")
	}
	defer in.Close()

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return writeDestFile(in, src, dst)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var sourceHash string
	var sourceHashError error

	go func() {
		defer wg.Done()
		hash, md5SumError := md5.Sum(src)
		if err != nil {
			sourceHashError = errors.Wrap(md5SumError, "couldn't md5sum sourceHash")
		}
		sourceHash = hash
	}()

	var destHash string
	var destHashError error

	go func() {
		defer wg.Done()
		hash, md5SumError := md5.Sum(dst)
		if err != nil {
			destHashError = errors.Wrap(md5SumError, "couldn't md5Sum destHash")
		}
		destHash = hash
	}()

	wg.Wait()

	if sourceHashError != nil {
		return sourceHashError
	}

	if destHashError != nil {
		return destHashError
	}

	if sourceHash != destHash {
		logrus.Printf("Similar file found:%s, diff hash:%s", dst, destHash)
		name, ext := filenameAndExt(dst)
		return writeDestFile(in, src, fmt.Sprintf("%s-%s%s", name, destHash[0:5], ext))
	}

	logrus.Printf("Exact match found:%s, skipping...", filepath.Base(dst))

	return nil
}

func writeDestFile(srcReader io.Reader, src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, srcReader)
	if err != nil {
		return err
	}

	logrus.Printf("Copied file: %s -> %s", src, dst)
	return nil
}
