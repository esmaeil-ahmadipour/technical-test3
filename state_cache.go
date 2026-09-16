package ktfunc

// TTL cache for gas-price suggestions.
//
// Contract state that gates a transaction or the lottery seed (StartBlock,
// EpochInterval, ConsensusReq, TlOcFees) is deliberately NOT cached: a stale
// read of any of them makes the node act on an already-rewarded epoch (the
// tx then reverts) or shifts endBlock so two nodes seed the lottery from
// different blocks. Those are always read fresh from the contract. The
// gas-price suggestion is the only value cached here. It is a tx default and
// never gates consensus or a transaction's success, so a stale read is
// harmless and saves an RPC call per loop iteration.

import (
	"context"
	"math/big"
	"sync"
	"time"
)

const (
	gasPriceCacheTTL = 60 * time.Second
)

// cachedValue is a single-entry TTL cache safe for concurrent access. Zero
// value is a fresh cache with no entry. Generic so each call site keeps
// its own type-safe value.
type cachedValue[T any] struct {
	mu        sync.Mutex
	value     T
	expiresAt time.Time
	hasValue  bool
}

// Get returns the cached value and true if a non-expired entry exists.
func (c *cachedValue[T]) Get() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hasValue && time.Now().Before(c.expiresAt) {
		return c.value, true
	}
	var zero T
	return zero, false
}

// Set stores a value with the given TTL.
func (c *cachedValue[T]) Set(v T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = v
	c.expiresAt = time.Now().Add(ttl)
	c.hasValue = true
}

// cachedSuggestGasPrice returns the eth client's gas-price suggestion,
// re-querying at most once per gasPriceCacheTTL. Gas prices fluctuate
// continuously but the consumers here use the result only for INFO logging
// (kt_props.go) or tx-construction defaults (vote_ops.go), so a 60-second
// staleness is fine and saves several RPC calls per loop iteration.
func cachedSuggestGasPrice(cProps *ConnectionProps) (*big.Int, error) {
	if v, ok := cProps.cachedGasPrice.Get(); ok {
		return new(big.Int).Set(v), nil
	}
	v, err := cProps.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	cProps.cachedGasPrice.Set(new(big.Int).Set(v), gasPriceCacheTTL)
	return v, nil
}
