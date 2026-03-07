package versioning

import (
	"strconv"
	"sync"
)

type ResourceVersion struct {
	sync.Mutex
	version uint64
}

func NewVersioning() *ResourceVersion {
	return &ResourceVersion{
		version: 1,
	}
}

func (r *ResourceVersion) SetResourceVersion(val string) error {
	r.Lock()
	defer r.Unlock()
	number, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return err
	}
	if number > r.version {
		r.version = number
	}
	return nil
}

func (r *ResourceVersion) UseResourceVersion() string {
	r.Lock()
	defer r.Unlock()
	number := strconv.FormatUint(r.version, 10)
	r.version += 1
	return number
}
