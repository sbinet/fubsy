// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build !kyotodb

package db

// Dummy version of KyotoDB -- used when the build host does not
// have Kyoto Cabinet installed.

import (
	"io"
)

type KyotoDB struct {
}

func OpenKyotoDB(filename string, writemode bool) (KyotoDB, error) {
	err := NotAvailableError{filename, "Kyoto Cabinet"}
	return KyotoDB{}, err
}

func (self KyotoDB) Close() error {
	panic("fake implementation")
}

func (self KyotoDB) LookupNode(nodename string) (*BuildRecord, error) {
	panic("fake implementation")
}

func (self KyotoDB) WriteNode(nodename string, record *BuildRecord) error {
	panic("fake implementation")
}

func (self KyotoDB) Dump(writer io.Writer, indent string) {
	panic("fake implementation")
}
