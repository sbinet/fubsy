// -*- mode: go; tab-width: 4; indent-tabs-mode: t -*-

// +build kyotodb

package db

import (
	"bytes"
	"testing"

	"bitbucket.org/ww/cabinet"
	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
)

func Test_KyotoDB_basics(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test1.kch", true)
	assert.NotNil(t, db.kcdb)
	if err != nil {
		t.Fatal(err)
	}

	rec1 := NewBuildRecord()
	rec1.SetTargetSignature([]byte{})
	err = db.WriteNode("node0", rec1)
	assert.Nil(t, err)
	rec2, err := db.LookupNode("node0")
	assert.Nil(t, err)
	assert.True(t, rec1.Equal(rec2))

	// Make sure it works across database close/reopen.
	rec1.AddParent("node1", []byte{34})
	rec1.AddParent("node2", []byte{54, 63})
	rec1.SetTargetSignature([]byte{200, 150, 100})

	enc, err := rec1.encode()
	assert.Nil(t, err)
	rec2 = NewBuildRecord()
	err = rec2.decode(enc)

	err = db.WriteNode("node0", rec1)
	assert.Nil(t, err)

	err = db.Close()
	assert.Nil(t, err)

	db, err = OpenKyotoDB("test1.kch", false) // open read-only
	if err != nil {
		t.Fatal(err)
	}
	rec2, err = db.LookupNode("node0")
	assert.Nil(t, err)
	assert.True(t, rec1.Equal(rec2),
		"wrote record:\n%#v\nand got back:\n%#v",
		rec1, rec2)
}

func Test_KyotoDB_checkVersion(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// brand-new empty DB with no version number in it
	db := KyotoDB{}
	db.kcdb = cabinet.New()
	filename := "test.kch"
	err := db.kcdb.Open(filename, cabinet.KCOWRITER|cabinet.KCOCREATE)
	if err != nil {
		t.Fatal(err)
	}

	// checkVersion() on an empty DB in read-only mode fails, because
	// it can neither read nor write the version number
	err = db.checkVersion(filename, false, 0, 0, 0)
	expect := "database test.kch has no version number"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error %s, but got %v", expect, err)
	}

	// in write mode it's OK, and sets the version number in the file
	err = db.checkVersion(filename, true, 513, 513, 513)
	assert.Nil(t, err)
	versionkey := makekey(PREFIX_META, "version")
	val, err := db.kcdb.Get(versionkey)
	assert.Nil(t, err)
	assert.Equal(t, "\x00\x00\x02\x01", string(val))

	// once the version number is in the file, write mode doesn't
	// matter -- now it's down to comparing with the supported range
	// of versions
	err = db.checkVersion(filename, false, 513, 134, 231)
	expect = "database test.kch is from the future (database version = 513, but max supported version = 231)"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error\n%s\nbut got\n%v", expect, err)
	}

	err = db.checkVersion(filename, false, 513, 534, 546)
	expect = "database test.kch is too old (database version = 513, but min supported version = 534)"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error\n%s\nbut got\n%v", expect, err)
	}

	err = db.checkVersion(filename, false, 513, 512, 514)
	assert.Nil(t, err)
	err = db.checkVersion(filename, false, 513, 513, 513)
	assert.Nil(t, err)
	err = db.checkVersion(filename, false, 513, 512, 513)
	assert.Nil(t, err)
	err = db.checkVersion(filename, false, 513, 513, 514)
	assert.Nil(t, err)

	// corrupt version number
	err = db.kcdb.Set(versionkey, []byte{0, 0x43, 0})
	assert.Nil(t, err)
	err = db.checkVersion(filename, false, 513, 513, 513)
	expect = "database test.kch: unable to decode version number (004300)"
	if err == nil || err.Error() != expect {
		t.Errorf("expected error\n%s\nbut got\n%v", expect, err)
	}
}

func Test_KyotoDB_key_prefix(t *testing.T) {
	// make sure we write the key exactly as expected, byte-for-byte
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test1.kch", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, db.kcdb)

	rec1 := NewBuildRecord()
	rec1.SetTargetSignature([]byte{})
	db.WriteNode("f", rec1)
	db.WriteNode("foobar", rec1)
	db.WriteNode("foo", rec1)

	// hmmm: does Kyoto Cabinet guarantee key order? it seems to
	// preserve insertion order from what I can tell, but I'm not
	// sure if that's reliable
	expect := []string{
		"\x00\x00\x00\x00version",
		"\x00\x00\x00\x01f",
		"\x00\x00\x00\x01foobar",
		"\x00\x00\x00\x01foo",
	}
	keychan := db.kcdb.Keys()
	for i, expectstr := range expect {
		expectkey := ([]byte)(expectstr)
		actualkey := <-keychan
		if !bytes.Equal(expectkey, actualkey) {
			t.Errorf("key %d: expected\n%v\nbut got\n%v", i, expectkey, actualkey)
		}
	}
}

func Test_KyotoDB_lookup_fail(t *testing.T) {
	// looking up a non-existent key is not an error
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test.kch", true)
	if err != nil {
		t.Fatal(err)
	}
	record, err := db.LookupNode("nosuchnode")
	assert.Nil(t, record)
	assert.Nil(t, err)
}

func Test_KyotoDB_error(t *testing.T) {
	// cannot create a DB in a non-existent directory
	db, err := OpenKyotoDB("no/such/directory/db.kch", true)
	assert.Nil(t, db.kcdb)
	assert.Equal(t,
		"could not open no/such/directory/db.kch: no repository", err.Error())
}
