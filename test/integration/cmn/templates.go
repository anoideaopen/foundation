package cmn

import (
	"github.com/anoideaopen/foundation/test/integration/cmn/template"
	"github.com/hyperledger/fabric/integration/nwo"
)

// Templates can be used to provide custom templates to GenerateConfigTree.
type TemplatesFound struct {
	*nwo.Templates
	Robot      string `yaml:"robot,omitempty"`
	Connection string `yaml:"connection,omitempty"`
}

func (t *TemplatesFound) RobotTemplate() string {
	if t.Robot != "" {
		return t.Robot
	}
	return template.DefaultRobot
}

func (t *TemplatesFound) ConnectionTemplate() string {
	if t.Connection != "" {
		return t.Connection
	}
	return template.DefaultConnection
}
