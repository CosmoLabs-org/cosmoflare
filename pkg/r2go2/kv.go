package r2go2

// KVService provides operations for Cloudflare KV namespaces. [Phase 3]
type KVService struct{}

// NewKVService creates a new KV service client. [Phase 3]
func NewKVService() *KVService {
	return &KVService{}
}
