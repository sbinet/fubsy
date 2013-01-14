package db

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
)

func Test_KyotoDB_basics(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test1", true)
	assert.NotNil(t, db.kcdb)
	assert.Nil(t, err)

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

	db, err = OpenKyotoDB("test1", false) // open read-only
	assert.Nil(t, err)
	rec2, err = db.LookupNode("node0")
	assert.Nil(t, err)
	assert.True(t, rec1.Equal(rec2),
		"wrote record:\n%#v\nand got back:\n%#v",
		rec1, rec2)
}

func Test_KyotoDB_key_prefix(t *testing.T) {
	// make sure we write the key exactly as expected, byte-for-byte
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test1", true)
	assert.NotNil(t, db.kcdb)
	assert.Nil(t, err)

	rec1 := NewBuildRecord()
	rec1.SetTargetSignature([]byte{})
	db.WriteNode("f", rec1)
	db.WriteNode("foobar", rec1)
	db.WriteNode("foo", rec1)

	keychan := db.kcdb.Keys()
	keys := [][]byte{
		<-keychan,
		<-keychan,
		<-keychan,
	}
	// hmmm: does Kyoto Cabinet guarantee key order? it seems to
	// preserve insertion order from what I can tell, but I'm not
	// sure if that's reliable
	assert.Equal(t, []byte{0, 0, 0, 1, 'f'}, keys[0])
	assert.Equal(t, []byte{0, 0, 0, 1, 'f', 'o', 'o', 'b', 'a', 'r'}, keys[1])
	assert.Equal(t, []byte{0, 0, 0, 1, 'f', 'o', 'o'}, keys[2])
}

func Test_KyotoDB_lookup_fail(t *testing.T) {
	// looking up a non-existent key is not an error
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test", true)
	assert.Nil(t, err)
	record, err := db.LookupNode("nosuchnode")
	assert.Nil(t, record)
	assert.Nil(t, err)
}

func Test_KyotoDB_error(t *testing.T) {
	// cannot create a DB in a non-existent directory
	db, err := OpenKyotoDB("no/such/directory/db", true)
	assert.Nil(t, db.kcdb)
	assert.Equal(t,
		"could not open no/such/directory/db.kch: no repository", err.Error())
}
