// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build kyotodb

package db

// Implementation of BuildDB using Kyoto Cabinet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"bitbucket.org/ww/cabinet"

	"fubsy/log"
)

type KyotoDB struct {
	kcdb *cabinet.KCDB
}

// key prefixes (to allow multiple namespaces in a single database)
const PREFIX_META = "\x00\x00\x00\x00"
const PREFIX_NODE = "\x00\x00\x00\x01"

// database version numbers: a database created by this code has
// version set to CURRENT_VERSION, and we can open databases where
// MIN_VERSION <= version <= MAX_VERSION
const CURRENT_VERSION uint32 = 0x00
const MIN_VERSION uint32 = 0x00
const MAX_VERSION uint32 = 0x00

func OpenKyotoDB(filename string, writemode bool) (KyotoDB, error) {
	db := KyotoDB{}
	db.kcdb = cabinet.New()
	mode := cabinet.KCOREADER
	if writemode {
		mode = cabinet.KCOWRITER | cabinet.KCOCREATE
	}
	err := db.kcdb.Open(filename, mode)
	if err != nil {
		db.kcdb.Del()
		db.kcdb = nil
		// errors returned by KC are pretty sparse ;-(
		return db, fmt.Errorf("could not open %s: %s", filename, err)
	}
	err = db.checkVersion(
		filename, writemode, CURRENT_VERSION, MIN_VERSION, MAX_VERSION)
	if err != nil {
		db.Close()
		return db, err
	}
	return db, nil
}

func (self KyotoDB) Close() error {
	err := self.kcdb.Close()
	if err != nil {
		return err
	}
	self.kcdb.Del()
	return nil
}

func (self KyotoDB) LookupNode(nodename string) (*BuildRecord, error) {
	log.Debug(log.DB, "loading record for node %s", nodename)
	key := makekey(PREFIX_NODE, nodename)
	val, err := self.kcdb.Get(key)

	if val == nil && (err == nil || kyotoNoRecord(err)) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := &BuildRecord{}
	err = result.decode(val)
	if err != nil {
		return nil, err
	}
	//log.DebugDump(log.DB, result)
	return result, nil
}

func (self KyotoDB) WriteNode(nodename string, record *BuildRecord) error {
	log.Debug(log.DB, "writing record for node %s", nodename)
	key := makekey(PREFIX_NODE, nodename)
	val, err := record.encode()
	if err != nil {
		return err
	}
	err = self.kcdb.Set(key, val)
	if err != nil {
		return err
	}
	return nil
}

func (self KyotoDB) Dump(writer io.Writer, indent string) {
	curs := self.kcdb.Cursor()
	defer curs.Del()
	for {
		key, value, err := curs.Get(true)
		if kyotoNoRecord(err) {
			return
		} else if err != nil {
			fmt.Fprintf(writer, "error: %s\n", err)
			return
		}

		prefix := key[0:4]
		fmt.Fprintf(writer, "%s(%s,%s):\n",
			indent, hex.EncodeToString(prefix), key[4:])
		fmt.Fprintf(writer, "%s  raw: %x\n",
			indent, value)
		switch string(prefix) {
		case PREFIX_NODE:
			record := BuildRecord{}
			err = record.decode(value)
			if err != nil {
				fmt.Fprintf(writer, "%s  decode error: %s\n", indent, err)
			} else {
				fmt.Fprintln(writer, "decoded:")
				record.Dump(writer, indent+"  ")
			}
		}
	}
}

func (self KyotoDB) checkVersion(
	filename string, writemode bool, cur, min, max uint32) error {
	key := makekey(PREFIX_META, "version")
	val, err := self.kcdb.Get(key)
	if kyotoNoRecord(err) {
		// no version number: presumably this is a brand-new empty
		// database, so we can set the version number
		if writemode {
			buf := &bytes.Buffer{}
			binary.Write(buf, binary.BigEndian, cur)
			err = self.kcdb.Set(key, buf.Bytes())
		} else {
			err = fmt.Errorf("database %s has no version number", filename)
		}
		return err
	} else if err != nil {
		return err
	}

	// successfully read the existing version number: is it compatible?
	buf := bytes.NewBuffer(val)
	var version uint32
	err = binary.Read(buf, binary.BigEndian, &version)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			err = fmt.Errorf(
				"database %s: unable to decode version number (%x)",
				filename, val)
		}
		return err
	}

	err = nil
	if version < min {
		err = fmt.Errorf("database %s is too old "+
			"(database version = %d, but min supported version = %d)",
			filename, version, min)
	} else if version > max {
		err = fmt.Errorf("database %s is from the future "+
			"(database version = %d, but max supported version = %d)",
			filename, version, max)
	}
	return err
}

func makekey(prefix, name string) []byte {
	key := make([]byte, 4+len(name))
	copy(key, prefix)
	copy(key[4:], name)
	return key
}

func kyotoNoRecord(err error) bool {
	// blechh: "no record" shouldn't be an error at all; if it must
	// be, then it should be distinguished by type
	return err != nil && err.Error() == "no record"
}
