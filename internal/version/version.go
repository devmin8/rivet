package version

import "fmt"

var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func String() string {
	if Commit == "" {
		return Version
	}
	return fmt.Sprintf("%s (%s)", Version, Commit)
}
