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
	"sync"

	"github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

/*
Rules:
 - Flatten source folder to destination with by file(s), by certain file(s).
 - Uncompress target file(s), Uncompress directory of archives recursively.
 - Generate md5 of folder recursively.
*/

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

	// 2.) Uncompress zip files.
	unzipAll()

	// 3.) Flatten files targeted by extension.
	flattenAssets()
}

func setupLogFile() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	logrus.SetOutput(os.Stdout)
	logrus.Infof("Starting goranize on source folder:%s, dest folder:%s", *sourceFolderPtr, *destFolderPtr)
	logrus.Info("Looking for the following files: ", allSet.String())
}

func unzipAll() {
	err := filepath.Walk(*sourceFolderPtr, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".zip" {
			return unzip(path, filepath.Join(*sourceFolderPtr, "C"))
		}
		return nil
	})
	if err != nil {
		fmt.Printf("walk error [%v]\n", err)
	}
}

func flattenAssets() {
	// 2.) Begin walking filesystem.
	err := filepath.Walk(
		filepath.Join(*sourceFolderPtr, "C"),
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				allSet.Each(func(item interface{}) bool {
					ext := item.(string)
					pathLowerCase := strings.ToLower(path)
					if strings.ToLower(filepath.Ext(pathLowerCase)) == ext {
						sourceFile := pathLowerCase
						destFile := filepath.Join(*destFolderPtr, filepath.Base(pathLowerCase))

						err := copyFile(sourceFile, destFile)
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

func createDirIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			logrus.Fatalf("Couldn't create destination dir:%s with err: %s", path, err)
		}
	}
}

// copyFile the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func copyFile(src, dst string) error {
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
		hash, md5SumError := md5Sum(src)
		if err != nil {
			sourceHashError = errors.Wrap(md5SumError, "couldn't md5sum sourceHash")
		}
		sourceHash = hash
	}()

	var destHash string
	var destHashError error

	go func() {
		defer wg.Done()
		hash, md5SumError := md5Sum(dst)
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

func md5Sum(file string) (string, error) {
	existFile, err := os.Open(file)
	if err != nil {
		return "", errors.Wrap(err, "md5Sum couldn't open file")
	}
	defer existFile.Close()

	h := md5.New()
	if _, err := io.Copy(h, existFile); err != nil {
		return "", errors.Wrap(err, "md5Sum couldn't io.Copy file")
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return errors.Wrapf(err, "Failed to zip.OpenReader of archive: %s", archive)
	}

	if err := os.MkdirAll(target, 0777); err != nil {
		return errors.Wrapf(err, "Failed to MkdirAll of target: %s", target)
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return errors.Wrap(err, "Failed to open source file during uncompress")
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return errors.Wrap(err, "Failed to open target file during uncompress")
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return errors.Wrap(err, "Failed to io.Copy file during uncompress")
		}
	}

	return nil
}
