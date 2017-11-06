package md5

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Sum computes the MD5 for a given file and returns the hex encoded string.
func Sum(file string) (string, error) {
	existFile, err := os.Open(file)
	if err != nil {
		return "", errors.Wrap(err, "md5.Sum couldn't open file")
	}
	defer existFile.Close()

	h := md5.New()
	if _, err := io.Copy(h, existFile); err != nil {
		return "", errors.Wrap(err, "md5.Sum couldn't io.Copy file")
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// SumValue is the tuple of Name/Hash results returned from the PSum operation.
type SumValue struct {
	Name string
	Hash string
}

// PSum executes MD5 sum in parallel based on a worker count.
// If workers is 0, runtime.NumCPU() is utilized for the worker count.
func PSum(workers int) (chan<- string, <-chan SumValue) {
	if workers == 0 {
		workers = runtime.NumCPU()
		logrus.Debugf("Running PSum with a max parallelism of %d", workers)
	}

	incomingChan := make(chan string, workers)
	outgoingChan := make(chan SumValue, workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for item := range incomingChan {
				result, err := Sum(item)
				if err != nil {
					logrus.Error("Error calculating md5 sum: ", err.Error())
					continue
				}
				outgoingChan <- SumValue{
					Name: item,
					Hash: result,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		// Once incomingChan is closed, all goroutines finish, we close outgoing.
		close(outgoingChan)
	}()

	return incomingChan, outgoingChan
}
