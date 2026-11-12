package experimental

import "github.com/elaous/xango/internal/transpile"

func Loaders() map[string]transpile.LoaderFunc {
	return map[string]transpile.LoaderFunc{
		".hcl":   LoadHCL,
		".proto": LoadProto,
		".capnp": LoadCapnProto,
		".sql":   LoadSQL,
	}
}
