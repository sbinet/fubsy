// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"fubsy/db"
	"fubsy/log"
)

// Interfaces for the Fubsy build database.

type BuildDB interface {
	// release all resources associated with this object
	Close() error

	// Lookup everything we know about the specified name from the
	// last time it was successfully built: the signature of the built
	// node, the list of parents it was built from, and their
	// signatures. Returns nil record if node not found. Non-nil error
	// is only for real errors, like the database disappeared, is
	// unreadable, etc.
	LookupNode(nodename string) (*db.BuildRecord, error)

	// Record the source signatures of the specified node for use by
	// future builds (should only be called after successfully
	// building that node). Again, non-nil errors is only for serious
	// database I/O problems.
	WriteNode(nodename string, record *db.BuildRecord) error

	log.Dumper
}
