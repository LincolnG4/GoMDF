package mf4

type VersionError struct {
}

func (e *VersionError) Error() string {
	return "File version is not >= 4.00"
}

