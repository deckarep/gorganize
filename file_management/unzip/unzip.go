package unzip

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func All(sourceFolder string) {
	err := filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".zip" {
			err := unzip(path, filepath.Join(sourceFolder, "C"))
			if err != nil {
				logrus.Error(err.Error())
			}
		}
		return nil
	})
	if err != nil {
		logrus.Error("Error on filepath.Walk during unzipAll:", err.Error())
	}
}

func unzip(archive, dest string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return errors.Wrapf(err, "Failed to zip.OpenReader of archive: %s", archive)
	}

	if err := os.MkdirAll(dest, 0777); err != nil {
		return errors.Wrapf(err, "Failed to MkdirAll of dest: %s", dest)
	}

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return errors.Wrap(err, "Failed to open source file during uncompress")
		}
		defer fileReader.Close()

		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return errors.Wrap(err, "Failed to open dest file during uncompress")
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, fileReader); err != nil {
			return errors.Wrap(err, "Failed to io.Copy file during uncompress")
		}
	}

	return nil
}
