// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package plugins

import (
	"errors"
	"fmt"

	"fubsy/types"
)

// Finding and using meta-plugins, i.e. plugins that interface with
// other languages.

// ordered collection of builtin Fubsy functions that can be called
// from another language
type BuiltinList interface {
	NumBuiltins() int
	Builtin(idx int) (name string, code types.FuCode)
}

type MetaPlugin interface {
	InstallBuiltins(builtins BuiltinList) error

	// Execute the code in content, making the values in builtins
	// available to it in a language-specific way. Return a map of
	// stuff defined by the code, e.g. functions the user can call
	// from Fubsy code.
	Run(content string) (types.ValueMap, error)

	// Release any resources held by this metaplugin
	Close()
}

// for use by dummy MetaPlugin implementations
type NotAvailableError struct {
	lang string
}

type factoryFunc func() (MetaPlugin, error)

var metaFactory map[string]factoryFunc
var metaCache map[string]MetaPlugin

func init() {
	// this just declares which languages we support -- don't actually
	// create the required metaplugins until we know they are needed
	metaFactory = make(map[string]factoryFunc)
	metaFactory["python2"] = NewPythonPlugin

	metaCache = make(map[string]MetaPlugin)
}

func LoadMetaPlugin(language string, builtins BuiltinList) (MetaPlugin, error) {
	meta, ok := metaCache[language]
	if ok && meta != nil {
		return meta, nil
	}

	factory := metaFactory[language]
	if factory == nil {
		return nil, errors.New("unsupported language for inline plugins: " + language)
	}

	meta, err := factory()
	if err != nil {
		return nil, err
	}
	err = meta.InstallBuiltins(builtins)
	if err != nil {
		return nil, err
	}

	metaCache[language] = meta
	return meta, nil
}

// Close() all metaplugins that have been created in this process and
// empty the cache of metaplugins.
func CloseAll() {
	for lang, meta := range metaCache {
		meta.Close()
		delete(metaCache, lang)
	}
}

func (err NotAvailableError) Error() string {
	return fmt.Sprintf(
		"cannot run plugin: support for %s not available",
		err.lang)
}
