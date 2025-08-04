package shardedflight

import (
	"golang.org/x/sync/singleflight"
	"sync/atomic"
)

// // // // // // // //

type ModObj struct {
	conf *ConfObj

	shards   []*singleflight.Group
	inFlight atomic.Int64
	mask     uint64
}

func New(conf ConfObj) (*ModObj, error) {
	if conf.Shards <= 0 || conf.Shards&(conf.Shards-1) != 0 {
		return nil, ErrInvalidShards
	}

	obj := ModObj{conf: &conf}

	if conf.BuildKey == nil {
		conf.BuildKey = defaultBuilder
	}
	if conf.Hash == nil {
		conf.Hash = defaultHash
	}

	obj.shards = make([]*singleflight.Group, conf.Shards)
	for i := range obj.shards {
		obj.shards[i] = new(singleflight.Group)
	}

	obj.mask = uint64(conf.Shards - 1)

	return &obj, nil
}

// //

// InFlight Returns the number of active (not yet completed) calls
func (obj *ModObj) InFlight() int64 { return obj.inFlight.Load() }

// Forget tells the singleflight to forget about a key.  Future calls
// to Do for this key will call the function rather than waiting for
// an earlier call to complete.
func (obj *ModObj) Forget(keyParts ...string) {
	key := obj.conf.BuildKey(keyParts...)
	idx := obj.conf.Hash(key) & obj.mask
	obj.shards[idx].Forget(key)
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// The return value shared indicates whether v was given to multiple callers.
func (obj *ModObj) Do(fn func() (any, error), keyParts ...string) (v any, err error, shared bool) {
	key := obj.conf.BuildKey(keyParts...)
	idx := obj.conf.Hash(key) & obj.mask
	obj.inFlight.Add(1)
	v, err, shared = obj.shards[idx].Do(key, fn)
	obj.inFlight.Add(-1)
	return
}

// DoChan is like Do but returns a channel that will receive the
// results when they are ready.
//
// The returned channel will not be closed.
func (obj *ModObj) DoChan(fn func() (any, error), keyParts ...string) <-chan singleflight.Result {
	key := obj.conf.BuildKey(keyParts...)
	idx := obj.conf.Hash(key) & obj.mask
	obj.inFlight.Add(1)

	ch := obj.shards[idx].DoChan(key, fn)
	out := make(chan singleflight.Result, 1)
	go func() {
		res := <-ch
		obj.inFlight.Add(-1)
		out <- res
	}()

	return out
}
