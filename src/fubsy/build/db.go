// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"fubsy/db"
)

// Interfaces for the Fubsy build database.

type BuildDB interface {
	// lookup everything we know about the specified name from the
	// last time it was successfully built: the signature of the built
	// node, the list of parents it was built from, and their
	// signatures
	LookupNode(nodename string) (*db.BuildRecord, error)

	// record the source signatures of the specified node for use by
	// future builds (should only be called after successfully
	// building that node)
	WriteNode(nodename string, record *db.BuildRecord) error
}
