package gateway

// I might also add the ability to hotswap different cache stores such
// as using something other than Redis, let me know. I prefer not too as
// redis has a lot of nice things we use like HSET and adding compatibility
// might be a pain. If you want to use a different DB, let me know and ill
// see if theres a point in adding it.

type StoreClient interface {
	String() string
}
