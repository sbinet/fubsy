package db

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
)

func Test_KyotoDB_basics(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	db, err := OpenKyotoDB("test1")
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

	db, err = OpenKyotoDB("test1")
	assert.Nil(t, err)
	rec2, err = db.LookupNode("node0")
	assert.Nil(t, err)
	assert.True(t, rec1.Equal(rec2),
		"wrote record:\n%#v\nand got back:\n%#v",
		rec1, rec2)
}

func Test_KyotoDB_error(t *testing.T) {
	db, err := OpenKyotoDB("no/such/directory")
	assert.Nil(t, db.kcdb)
	assert.Equal(t, "no repository", err.Error())
}
