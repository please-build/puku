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
}

func (k *Kind) IsProvided(i string) bool {
	for _, dep := range k.ProvidedDeps {
		if i == dep {
			return true
		}
	}
	return false
}

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
}
