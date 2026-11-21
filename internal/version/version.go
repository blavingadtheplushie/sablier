// go build -ldflags "-X github.com/espcaa/sablier/internal/version.Version=v1.2.3 ..."
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func String() string {
	v, c, d := Version, Commit, Date

	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if c == "" && len(s.Value) >= 7 {
					c = s.Value[:7]
				}
			case "vcs.time":
				if d == "" {
					d = s.Value
				}
			}
		}
	}

	out := "sablier " + v
	if c != "" {
		out += fmt.Sprintf(" (%s", c)
		if d != "" {
			out += ", " + d
		}
		out += ")"
	}
	return out
}

// sent with every request, so the server knows who is on what version
func UserAgent() string {
	return "sablier/" + Version + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
}
