package db

// Implementation of BuildDB using Kyoto Cabinet

import (
	"bitbucket.org/ww/cabinet"
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
	err := db.kcdb.Open(
		basename+".kch", cabinet.KCOWRITER|cabinet.KCOCREATE)
	if err != nil {
		db.kcdb.Del()
		db.kcdb = nil
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
	key := nodekey(nodename)
	val, err := self.kcdb.Get(key)
	if err != nil {
		return nil, err
	}
	result := &BuildRecord{}
	err = result.decode(val)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (self KyotoDB) WriteNode(nodename string, record *BuildRecord) error {
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
