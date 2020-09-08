package lepton

func isDynamicLinked(path string) (bool, error) {
	return false, nil
}

// stub
func getSharedLibs(targetRoot string, path string) ([]string, error) {
	var deps []string
	return deps, nil
}
