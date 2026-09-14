package version

import "fmt"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("skiller %s\ncommit: %s\nbuilt: %s", Version, Commit, Date)
}
