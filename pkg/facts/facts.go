package facts

import (
	"context"
	"log/slog"
	"time"

	"github.com/m-horky/elk/internal/exec"
	distributionfacts "github.com/m-horky/elk/internal/facts/distribution"
	lscpufacts "github.com/m-horky/elk/internal/facts/lscpu"
	unanefacts "github.com/m-horky/elk/internal/facts/uname"
	virtwhatfacts "github.com/m-horky/elk/internal/facts/virtwhat"
	"github.com/m-horky/elk/internal/fs"
	"github.com/m-horky/elk/pkg/constants"
)

const (
	// certificateVersion is the version of the system facts certificate format.
	certificateVersion = "3.2"

	// commandTimeout bounds each fact-collection command invocation.
	commandTimeout = 10 * time.Second
)

// TODO Should we keep Facts flat, or expose those structs directly?

// Facts contains the system facts that Elk can publish to a service.
//
// A nil field means that the corresponding fact could not be collected.
type Facts struct {
	// SystemCertificateVersion identifies the system facts certificate format.
	SystemCertificateVersion *string

	// UnameSysname is the kernel name reported by uname.
	UnameSysname *string
	// UnameNodename is the network node name reported by uname.
	UnameNodename *string
	// UnameRelease is the kernel release reported by uname.
	UnameRelease *string
	// UnameVersion is the kernel version reported by uname.
	UnameVersion *string
	// UnameMachine is the hardware identifier reported by uname.
	UnameMachine *string

	// DistributionID is the operating system identifier from os-release.
	DistributionID *string
	// DistributionName is the operating system name from os-release.
	DistributionName *string
	// DistributionVersion is the operating system version from os-release.
	DistributionVersion *string
	// DistributionVersionModifier is the version modifier derived from CPE_NAME.
	DistributionVersionModifier *string

	// VirtIsGuest reports whether the system runs inside a virtual machine.
	VirtIsGuest *bool
	// VirtHostType contains the virtualization types reported by virt-what.
	VirtHostType *string

	// LSCPU contains the normalized lscpu fields, keyed without the "lscpu." prefix.
	// CPU facts are derived from this map by service clients.
	LSCPU map[string]string
}

// sourcePaths contains the resolved locations used by fact collectors.
type sourcePaths struct {
	osRelease string
	lscpu     string
	virtWhat  string
}

// Collect returns the facts currently available from Elk's fact model.
//
// Collection is best effort. A source that cannot be read is logged and
// leaves its facts unset. Command invocations are bounded by commandTimeout.
// The caller context bounds the whole collection. The system certificate
// version is always populated.
func Collect(ctx context.Context) *Facts {
	return collect(
		ctx,
		exec.OSRunner{},
		fs.Filesystem{},
		sourcePaths{
			osRelease: constants.Path(constants.OSReleasePath),
			lscpu:     constants.Path(constants.LSCPUBin),
			virtWhat:  constants.Path(constants.VirtWhatBin),
		},
	)
}

// TODO Should we return instead of mutating?

// collect gathers facts from every source, tolerating individual source failures.
func collect(ctx context.Context, runner exec.Runner, filesystem fs.FS, paths sourcePaths) *Facts {
	slog.Debug("collecting system facts")

	collected := &Facts{SystemCertificateVersion: new(certificateVersion)}
	collectUname(collected)
	collectDistribution(collected, filesystem, paths.osRelease)
	collectVirt(ctx, collected, runner, paths.virtWhat)
	collectLSCPU(ctx, collected, runner, paths.lscpu)

	return collected
}

// collectUname copies kernel identity facts into the result.
func collectUname(collected *Facts) {
	info, err := unanefacts.Collect()
	if err != nil {
		slog.Warn("cannot collect uname facts", "err", err)

		return
	}

	collected.UnameSysname = info.Sysname
	collected.UnameNodename = info.Nodename
	collected.UnameRelease = info.Release
	collected.UnameVersion = info.Version
	collected.UnameMachine = info.Machine
}

// collectDistribution copies operating system identity facts into the result.
func collectDistribution(collected *Facts, filesystem fs.FS, path string) {
	info, err := distributionfacts.Collect(filesystem, path)
	if err != nil {
		slog.Warn("cannot collect distribution facts", "path", path, "err", err)

		return
	}

	collected.DistributionID = info.ID
	collected.DistributionName = info.Name
	collected.DistributionVersion = info.Version
	collected.DistributionVersionModifier = info.VersionModifier
}

// collectVirt copies virtualization facts into the result.
func collectVirt(ctx context.Context, collected *Facts, runner exec.Runner, path string) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	info, err := virtwhatfacts.Collect(ctx, runner, path)
	if err != nil {
		slog.Warn("cannot collect virtualization facts", "path", path, "err", err)

		return
	}

	collected.VirtIsGuest = info.IsGuest
	collected.VirtHostType = info.HostType
}

// collectLSCPU copies the lscpu facts into the result.
func collectLSCPU(ctx context.Context, collected *Facts, runner exec.Runner, path string) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	info, err := lscpufacts.Collect(ctx, runner, path)
	if err != nil {
		slog.Warn("cannot collect lscpu facts", "path", path, "err", err)

		return
	}

	collected.LSCPU = info
}
