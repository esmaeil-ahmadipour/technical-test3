package ktfunc

// Tests for the gas-price TTL cache and the generic cachedValue helper.
// Contract state that gates a tx or the seed is no longer cached, so there
// are no StartBlock/EpochInterval/ConsensusReq/TlOcFees cache tests here.

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCachedValue_ExpiresAfterTTL(t *testing.T) {
	var c cachedValue[int]
	c.Set(42, 10*time.Millisecond)
	v, ok := c.Get()
	assert.True(t, ok)
	assert.Equal(t, 42, v)

	time.Sleep(15 * time.Millisecond)
	_, ok = c.Get()
	assert.False(t, ok, "value should have expired")
}

func TestCachedSuggestGasPrice_HitsClientOnceWithinTTL(t *testing.T) {
	mockClient := &MockEthClient{}
	cProps := &ConnectionProps{Client: mockClient}
	mockClient.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(20_000_000_000), nil).Once()

	for i := 0; i < 5; i++ {
		v, err := cachedSuggestGasPrice(cProps)
		assert.NoError(t, err)
		assert.Equal(t, uint64(20_000_000_000), v.Uint64())
	}
	mockClient.AssertNumberOfCalls(t, "SuggestGasPrice", 1)
}
