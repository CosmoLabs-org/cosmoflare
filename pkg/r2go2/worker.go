package r2go2

// WorkerService provides operations for Cloudflare Workers. [Phase 3]
type WorkerService struct{}

// NewWorkerService creates a new Workers service client. [Phase 3]
func NewWorkerService() *WorkerService {
	return &WorkerService{}
}
