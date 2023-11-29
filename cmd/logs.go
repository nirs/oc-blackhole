// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log"
	"os"
)

type verboseWriter struct{}

func (w verboseWriter) Write(b []byte) (int, error) {
	if verbose {
		return os.Stdout.Write(b)
	}
	return len(b), nil
}

var dbglog = log.New(verboseWriter{}, "", 0)
var errlog = log.New(os.Stderr, "", 0)
