// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"fubsy/db"
)

// Interfaces for the Fubsy build database.

type BuildDB interface {
	// lookup the source signatures of the specified node as recorded
	// in the database when that node was last successfully built
	LookupParents(name string) (*db.SourceRecord, error)

	// record the source signatures of the specified node for use by
	// future builds (should only be called after successfully
	// building that node)
	WriteParents(name string, record *db.SourceRecord) error
}
