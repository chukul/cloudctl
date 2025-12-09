package internal

import "runtime"

// IsMacOS checks if the runtime OS is darwin
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}
