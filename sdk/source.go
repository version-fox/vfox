package sdk

import (
	"github.com/aooohan/version-fox/env"
	"net/url"
)

type Source interface {
	DownloadUrl(handler *Handler, version Version) *url.URL
	FileExt(handler *Handler) string
	EnvKeys(handler *Handler, version Version) []*env.KV
	Name() string
}
