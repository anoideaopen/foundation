package cachestub_test

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/stretchr/testify/assert"
)

func TestBatchStub(t *testing.T) {
	stub := newMockStub()
	_ = stub.PutState("KEY1", []byte("key1_value_1"))
	_ = stub.PutState("KEY2", []byte("key2_value_1"))
	_ = stub.PutState("KEY3", []byte("key3_value_1"))

	btchStub := cachestub.NewBatchCacheStub(stub)

	_ = btchStub.PutState("KEY1", []byte("key1_value_2"))
	_ = btchStub.DelState("KEY2")
	_ = btchStub.Commit()

	val1, _ := btchStub.GetState("KEY2")
	assert.Equal(t, "", string(val1))

	val2, _ := btchStub.GetState("KEY1")
	assert.Equal(t, "key1_value_2", string(val2))

	_ = btchStub.PutState("KEY2", []byte("key2_value_2"))
	_ = btchStub.DelState("KEY3")
	_ = btchStub.Commit()

	assert.Equal(t, 2, len(stub.state))
	assert.Equal(t, "key1_value_2", string(stub.state["KEY1"]))
	assert.Equal(t, "key2_value_2", string(stub.state["KEY2"]))

	_ = btchStub.PutState("KEY4", []byte("key4_value_1"))
	_ = btchStub.DelState("KEY4")

	_ = btchStub.Commit()

	_, ok := stub.state["KEY4"]
	assert.Equal(t, false, ok)
}
