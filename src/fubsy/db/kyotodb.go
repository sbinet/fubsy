package db

// Implementation of BuildDB using Kyoto Cabinet

import (
	"fmt"

	"bitbucket.org/ww/cabinet"

	"fubsy/log"
)

type KyotoDB struct {
	kcdb *cabinet.KCDB
}

// key prefixes (to allow multiple namespaces in a single database)
const PREFIX_META = "\x00\x00\x00\x00"
const PREFIX_NODE = "\x00\x00\x00\x01"

func OpenKyotoDB(basename string) (KyotoDB, error) {
	db := KyotoDB{}
	db.kcdb = cabinet.New()
	filename := basename + ".kch"
	err := db.kcdb.Open(filename, cabinet.KCOWRITER|cabinet.KCOCREATE)
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

	// blechh: "no record" shouldn't be an error at all; if it must
	// be, then it should be distinguished by type
	if val == nil && (err == nil || err.Error() == "no record") {
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

func nodekey(name string) []byte {
	key := make([]byte, 4+len(name))
	copy(key, PREFIX_NODE)
	copy(key, name)
	return key
}
