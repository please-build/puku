package kinds

type Type int

const (
	Lib Type = iota
	Test
	Bin
	ThirdParty
)

// Kind is a kind of build target, e.g. go_library. These can either be library, test or binaries. They can also provide
// dependencies e.g. you could wrap go_test to add a common testing library, in which case, we should not add it as a
// dep.
//
// Currently, puku assumes these use the same args for e.g. srcs, and deps as go_library, go_test and go_binary. This
// may become configurable if the need arises.
type Kind struct {
	Name              string
	Type              Type
	ProvidedDeps      []string
	DefaultVisibility []string
	// NonGoSources indicates the puku that the sources to this rule are not go so we shouldn't try to parse them to
	// infer their deps, for example, proto_library.
	NonGoSources bool
}

// IsProvided returns whether the dependency is already provided by the kind, and therefore can be omitted from the deps
// list.
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
	"go_repo": {
		Name:              "go_repo",
		Type:              ThirdParty,
		DefaultVisibility: []string{"PUBLIC"},
	},
	"go_module": {
		Name: "go_repo",
		Type: ThirdParty,
	},
}
