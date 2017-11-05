package main

import (
	"archive/zip"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/deckarep/golang-set"
)

var (
	logFile         = flag.String("log", "gorganize.log", "location of log file, default is working directory")
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
	setupLogFile(*logFile)

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
							log.Fatalf("Failed to copy file: %s to dest %s with err: %s", sourceFile, destFile, err.Error())
						}

					}
					return false
				})
			}
			return nil
		})

	if err != nil {
		log.Fatal("Couldn't walk the root folder with err: ", err.Error())
	}
}

func setupLogFile(path string) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Printf("Starting goranize on source folder:%s, dest folder:%s", *sourceFolderPtr, *destFolderPtr)
	log.Println("Hi")
}

func createDirIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			log.Fatal("Couldn't create dir with err: ", err)
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
		fmt.Printf("Similar file found:%s, diff hash:%s\n", dst, destHash)
		name, ext := filenameAndExt(dst)
		writeDestFile(in, src, fmt.Sprintf("%s-%s%s", name, destHash[0:5], ext))
	} else {
		fmt.Printf("Exact match found:%s, skipping...\n", filepath.Base(dst))
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
	fmt.Printf("Copied file: %s -> %s\n", src, dst)
	return nil
}

func md5Sum(file string) string {
	existFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer existFile.Close()

	h := md5.New()
	if _, err := io.Copy(h, existFile); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func uncompress(folder string) {
	// Open a zip archive for reading.
	r, err := zip.OpenReader(folder)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.CopyN(os.Stdout, rc, 68)
		if err != nil {
			log.Fatal(err)
		}
		rc.Close()
		fmt.Println()
	}
}
