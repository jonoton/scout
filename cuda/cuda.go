package cuda

import "io/ioutil"

// HasCudaInstalled on system
func HasCudaInstalled() bool {
	out, err := ioutil.ReadFile("/usr/local/cuda/version.txt")
	outStr := string(out)
	if err != nil || outStr == "" {
		return false
	}
	return true
}
