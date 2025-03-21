package geoipdb

// Country country
type Country interface {
	GetGeoNameID() uint
	GetISO() string
	IsInEuropeanUnion() bool
	GetNames() map[string]string
}

type country struct {
	geoNameID         uint
	isoCode           string
	isInEuropeanUnion bool
	names             map[string]string
}

// GetGeoNameID get geo name id
func (c country) GetGeoNameID() uint {
	return c.geoNameID
}

// GetISO get iso
func (c country) GetISO() string {
	return c.isoCode
}

// IsInEuropeanUnion is in European Union
func (c country) IsInEuropeanUnion() bool {
	return c.isInEuropeanUnion
}

// GetNames get names
func (c country) GetNames() map[string]string {
	if c.names == nil {
		return map[string]string{}
	}
	return c.names
}
