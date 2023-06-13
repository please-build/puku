package kinds

type Type int

const (
	Lib Type = iota
	Test
	Bin
)

type Kind struct {
	Name         string
	Type         Type
	ProvidedDeps []string
	// NonGoSources indicates the puku that the sources to this rule are not go so we shouldn't try to parse them to
	// infer their deps, for example, proto_library.
	NonGoSources bool
}

func (k *Kind) IsProvided(i string) bool {
	for _, dep := range k.ProvidedDeps {
		if i == dep {
			return true
		}
	}
	return false
}

// DefaultKinds are the base kinds that puku supports out of the box
var DefaultKinds = map[string]*Kind{
	"go_library": {
		Name: "go_library",
		Type: Lib,
	},
	"go_binary": {
		Name: "go_binary",
		Type: Bin,
	},
	"go_test": {
		Name: "go_test",
		Type: Test,
	},
	"proto_library": {
		Name:         "proto_library",
		Type:         Lib,
		NonGoSources: true,
	},
	"grpc_library": {
		Name:         "proto_library",
		Type:         Lib,
		NonGoSources: true,
	},
}
