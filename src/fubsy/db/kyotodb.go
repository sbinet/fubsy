package db

// Implementation of BuildDB using Kyoto Cabinet

import (
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

func OpenKyotoDB(basename string, writemode bool) (KyotoDB, error) {
	db := KyotoDB{}
	db.kcdb = cabinet.New()
	filename := basename + ".kch"
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
	key := nodekey(nodename)
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
	key := nodekey(nodename)
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
		// fmt.Fprintf(writer, "%s%x: %x\n", indent, key, value)

		prefix := hex.EncodeToString(key[0:4])
		key = key[4:]
		fmt.Fprintf(writer, "%s(%s,%s): %x\n", indent, prefix, key, value)
	}
}

func nodekey(name string) []byte {
	key := make([]byte, 4+len(name))
	copy(key, PREFIX_NODE)
	copy(key[4:], name)
	return key
}

func kyotoNoRecord(err error) bool {
	// blechh: "no record" shouldn't be an error at all; if it must
	// be, then it should be distinguished by type
	return err != nil && err.Error() == "no record"
}
