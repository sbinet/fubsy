// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"fmt"
)

// for use by fake BuildDB implementations
type NotAvailableError struct {
	filename string
	libname  string
}

func (err NotAvailableError) Error() string {
	return fmt.Sprintf(
		"cannot open database in %s: %s library not available",
		err.filename, err.libname)
}
