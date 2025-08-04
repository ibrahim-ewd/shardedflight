package shardedflight

import "errors"

// // // // // // // //

var ErrInvalidShards = errors.New("ConfObj.Shards must be a power of two")
