package options

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// ReleaseServer is the main context object for release controller
type ReleaseServer struct {
	// Kubeconfig specify the api server.
	Kubeconfig string
	// Controllers is the list of controllers to enable.
	Controllers []string
	// ConcurrentGCSyncs is the number of target objects that are
	// allowed to sync concurrently. Larger number = more responsive jobs,
	// but more CPU (and network) load.
	ConcurrentGCSyncs int32
	// ConcurrentStatusSyncs is the number of target objects that are
	// allowed to sync concurrently. Larger number = more responsive jobs,
	// but more CPU (and network) load.
	ConcurrentStatusSyncs int32
	// ResyncPeriod describes the period of informer resync.
	ResyncPeriod time.Duration
	// ReleaseResyncPeriod is the resync period to invoke informer event handler.
	ReleaseResyncPeriod time.Duration

	HealthzPort int

	// The number of releaseHistory to retain to allow rollback.
	// Defaults to 50.
	HistoryLimit int32
}

// NewReleaseServer creates a new CMServer with a default config.
func NewReleaseServer() *ReleaseServer {
	return &ReleaseServer{
		ConcurrentGCSyncs:     5,
		ConcurrentStatusSyncs: 5,
		ResyncPeriod:          5 * time.Minute,
		ReleaseResyncPeriod:   30 * time.Second,
	}
}

// AddFlags adds flags for a specific ReleaseServer to the specified FlagSet
func (s *ReleaseServer) AddFlags(fs *pflag.FlagSet, allControllers []string) {
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information")
	fs.StringSliceVar(&s.Controllers, "controllers", allControllers, fmt.Sprintf(""+
		"A list of controllers to enable. All controllers: %s", strings.Join(allControllers, ", ")))
	fs.Int32Var(&s.ConcurrentGCSyncs, "concurrent-gc-syncs", s.ConcurrentGCSyncs, "The number of garbage collector worker that are allowed to sync concurrently")
	fs.Int32Var(&s.ConcurrentStatusSyncs, "concurrent-status-syncs", s.ConcurrentStatusSyncs, "The number of status controller worker that are allowed to sync concurrently")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "ResyncPeriod describes the period of informer resync")
	fs.DurationVar(&s.ReleaseResyncPeriod, "handler-resync-period", s.ReleaseResyncPeriod, "ReleaseResyncPeriod is the resync period to invoke informer event handler")
	fs.IntVar(&s.HealthzPort, "healthz-port", 8080, "The port of the localhost healthz endpoint")
	fs.Int32Var(&s.HistoryLimit, "history-limit", 50, "The number of releaseHistory to retain to allow rollback")
}
