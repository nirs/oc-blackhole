// SPDX-FileCopyrightText: The oc-blackhole authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io"
	"log"
	"os"
)

type verboseWriter struct {
	writer io.Writer
}

func (w verboseWriter) Write(b []byte) (int, error) {
	if verbose {
		return w.writer.Write(b)
	}
	return len(b), nil
}

var dbglog = log.New(verboseWriter{os.Stdout}, "", 0)
var errlog = log.New(os.Stderr, "", 0)
