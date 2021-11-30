package v1

// Endpoint struct defines the address and port of
// benchmark test server.
type Endpoint struct {
	Address string `yaml:"address"`
	Port    uint   `yaml:"port"`
}
