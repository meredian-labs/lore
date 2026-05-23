package cli

import "errors"

var (
	ErrNotALoreRepo = errors.New("not a lore repository (or any parent up to mount point /)")
	ErrUsage        = errors.New("usage error")
)
