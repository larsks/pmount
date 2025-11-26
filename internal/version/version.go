package version

import (
	"fmt"
	"runtime/debug"

	"github.com/larsks/gobot/tools"
)

var (
	Version string = "dev"
)

func GetVersion(progName string) string {
	vs := fmt.Sprintf("%s version %s", progName, Version)

	if bi, ok := debug.ReadBuildInfo(); ok {
		bim := tools.BuildInfoMap(bi)
		vs = fmt.Sprintf("%s %s/%s", vs, bim["GOOS"], bim["GOARCH"])
		if vcs, ok := bim["vcs"]; ok && vcs == "git" {
			vs = fmt.Sprintf("%s rev %s on %s", vs, bim["vcs.revision"][:10], bim["vcs.time"])
		}
	}

	return vs
}
