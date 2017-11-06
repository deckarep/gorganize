/*
Open Source Initiative OSI - The MIT License (MIT):Licensing
The MIT License (MIT)
Copyright (c) 2017 Ralph Caraveo (deckarep@gmail.com)
Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package cmd

import (
	md5 "github.com/deckarep/gorganize/file_management/md5"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(md5Cmd)
}

var md5Cmd = &cobra.Command{
	Use:   "md5 [file(s) ...]",
	Short: "calculates hashes against one or more files",
	Long:  "md5 [file(s) ...] will calculate md5 operations against one or more files.",
	Run: func(cmd *cobra.Command, args []string) {
		producerChan, receiverChan := md5.PSum(0)

		go func() {
			for _, file := range args {
				producerChan <- file
			}
			close(producerChan)
		}()

		for result := range receiverChan {
			logrus.Infof("%s %s", result.Hash, result.Name)
		}
	},
}
