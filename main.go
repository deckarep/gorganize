package main

import (
	"archive/zip"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/deckarep/golang-set"
	"github.com/sirupsen/logrus"
)

var (
	sourceFolderPtr = flag.String("source", "/Users/deckarep/Desktop/test-folder/", "starting source folder")
	destFolderPtr   = flag.String("dest", "/Users/deckarep/Desktop/dest-folder/", "destination source folder")
)

var (
	// imageSet is all image file types.
	imageSet = mapset.NewThreadUnsafeSetFromSlice([]interface{}{
		".psd", ".pdf", ".png", ".gif", ".jpg", ".jpeg", ".tiff", ".nef", ".raw"})

	// videoSet is video file types.
	videoSet = mapset.NewThreadUnsafeSetFromSlice([]interface{}{
		".mov", ".avi"})

	// allSet is the entire kitchen sink.
	allSet = imageSet.Union(videoSet)
)

func init() {
	flag.Parse()
}

func main() {
	// 0.) Init log file
	setupLogFile()

	// 1.) Ensure the destination directory exists.
	createDirIfNotExists(*destFolderPtr)

	// 2.) Begin walking filesystem.
	err := filepath.Walk(
		*sourceFolderPtr,
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				imageSet.Each(func(item interface{}) bool {
					ext := item.(string)
					pathLowerCase := strings.ToLower(path)
					if strings.ToLower(filepath.Ext(pathLowerCase)) == ext {
						sourceFile := pathLowerCase
						destFile := filepath.Join(*destFolderPtr, filepath.Base(pathLowerCase))

						err := copyFile(sourceFile, destFile)
						if err != nil {
							logrus.Fatalf("Failed to copy file: %s to dest %s with err: %s", sourceFile, destFile, err.Error())
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

func setupLogFile() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	logrus.SetOutput(os.Stdout)
	logrus.Infof("Starting goranize on source folder:%s, dest folder:%s", *sourceFolderPtr, *destFolderPtr)
}

func createDirIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			logrus.Fatal("Couldn't create dir with err: ", err)
		}
	}
}

// copyFile the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return writeDestFile(in, src, dst)
	}

	sourceHash := md5Sum(src)
	destHash := md5Sum(dst)

	if sourceHash != destHash {
		logrus.Printf("Similar file found:%s, diff hash:%s", dst, destHash)
		name, ext := filenameAndExt(dst)
		writeDestFile(in, src, fmt.Sprintf("%s-%s%s", name, destHash[0:5], ext))
	} else {
		logrus.Printf("Exact match found:%s, skipping...", filepath.Base(dst))
	}

	return nil
}

func filenameAndExt(path string) (string, string) {
	justExt := filepath.Ext(path)
	justName := path[0 : len(path)-len(justExt)]
	return justName, justExt
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

func md5Sum(file string) string {
	existFile, err := os.Open(file)
	if err != nil {
		logrus.Fatal(err)
	}
	defer existFile.Close()

	h := md5.New()
	if _, err := io.Copy(h, existFile); err != nil {
		logrus.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func uncompress(folder string) {
	// Open a zip archive for reading.
	r, err := zip.OpenReader(folder)
	if err != nil {
		logrus.Fatal(err)
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		logrus.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			logrus.Fatal(err)
		}
		_, err = io.CopyN(os.Stdout, rc, 68)
		if err != nil {
			logrus.Fatal(err)
		}
		rc.Close()
	}
}
