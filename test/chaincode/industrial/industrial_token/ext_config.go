package industrialtoken

import (
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ core.ExternalConfigurable = &IndustrialToken{}

func (it *IndustrialToken) ValidateExtConfig(cfgBytes []byte) error {
	var (
		ec      ExtConfig
		cfgFull proto.Config
	)

	if err := protojson.Unmarshal(cfgBytes, &cfgFull); err != nil {
		return fmt.Errorf("unmarshalling config: %w", err)
	}

	if cfgFull.ExtConfig.MessageIs(&ec) {
		if err := cfgFull.ExtConfig.UnmarshalTo(&ec); err != nil {
			return fmt.Errorf("unmarshalling ext config: %w", err)
		}
	}

	if err := ec.Validate(); err != nil {
		return fmt.Errorf("validating ext config data: %w", err)
	}

	return nil
}

func (it *IndustrialToken) ApplyExtConfig(cfgBytes []byte) error {
	var (
		ec      ExtConfig
		cfgFull proto.Config
	)

	if err := protojson.Unmarshal(cfgBytes, &cfgFull); err != nil {
		return fmt.Errorf("unmarshalling config: %w", err)
	}

	if cfgFull.ExtConfig.MessageIs(&ec) {
		if err := cfgFull.ExtConfig.UnmarshalTo(&ec); err != nil {
			return fmt.Errorf("unmarshalling ext config: %w", err)
		}
	}

	it.extConfig = &ec

	return nil
}
