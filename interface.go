package config

type ConfigInterface interface {
	// Read config file
	Open(file ...string) error

	// Read integer property. If property is not exists or empty will return 0
	GetInt(name string) int

	// Read integer property or return defValue if property is not exists or empty
	GetIntOr(name string, defValue int) int

	// Read string property
	GetString(name string) string

	// Read string property or retun defValue if property is not exists or empty
	GetStringOr(name string, defValue string) string

	// Read boolean property
	GetBool(name string) bool

	// Read boolean property or return defValue if property is not exists or empty
	GetBoolOr(name string, defValue bool) bool
}
