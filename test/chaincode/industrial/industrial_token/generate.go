package industrialtoken

// Chaincode configuration
//go:generate protoc -I=. -I=../../../../proto/ --go_out=paths=source_relative:. --validate_out=lang=go,paths=source_relative:. ext_config.proto
