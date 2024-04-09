package industrialtoken

import (
	"encoding/json"
	"fmt"

	"github.com/anoideaopen/foundation/core"
)

var _ core.ExternalConfigurable = &IndustrialToken{}

func (it *IndustrialToken) ValidateExtConfig(cfgBytes []byte) error {
	var ec ExtConfig
	if err := json.Unmarshal(cfgBytes, &ec); err != nil {
		return fmt.Errorf("unmarshalling ext config data: %w", err)
	}

	if err := ec.Validate(); err != nil {
		return fmt.Errorf("validating ext config data: %w", err)
	}

	return nil
}

func (it *IndustrialToken) ApplyExtConfig(cfgBytes []byte) error {
	var ec ExtConfig
	if err := json.Unmarshal(cfgBytes, &ec); err != nil {
		return fmt.Errorf("unmarshalling ext config: %w", err)
	}

	it.extConfig = &ec

	return nil
}
