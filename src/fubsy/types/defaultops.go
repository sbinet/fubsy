// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"fmt"
)

// Tiny embeddable types that provide default implementations for
// various FuObject methods. "Null" means a harmless no-op
// implementation; "Unsupported" means it always returns an error.

// Provides a default implementation of FuObject.Lookup() for use by
// types that have no attributes.
type NullLookupT struct {
}

func (self NullLookupT) Lookup(name string) (FuObject, bool) {
	return nil, false
}

// Make it easy for implementations FuObject.Add() to explain they
// cannot be added to the other object.
func UnsupportedAdd(o1, o2 FuObject, hint string) (FuObject, error) {
	var err error
	if hint == "" {
		err = fmt.Errorf("unsupported operation: cannot add %s to %s",
			o2.Typename(), o1.Typename())
	} else {
		err = fmt.Errorf("unsupported operation: cannot add %s to %s (%s)",
			o2.Typename(), o1.Typename(), hint)
	}
	return nil, err
}
