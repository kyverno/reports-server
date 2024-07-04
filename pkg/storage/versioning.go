package storage

import (
	"strconv"
	"sync"

	"github.com/kyverno/reports-server/pkg/storage/api"
)

type resourceVersion struct {
	sync.Mutex
	version uint64
}

func NewVersioning() api.Versioning {
	return &resourceVersion{
		version: 1,
	}
}

func (r *resourceVersion) SetResourceVersion(val string) error {
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

func (r *resourceVersion) UseResourceVersion() string {
	r.Lock()
	defer r.Unlock()
	number := strconv.FormatUint(r.version, 10)
	r.version += 1
	return number
}
