package geoipdb

// City city
type City interface {
	GetGeoNameID() uint
	GetNames() map[string]string
	GetCountryGeoNameID() uint
	GetISO() string
	IsInEuropeanUnion() bool
	GetCountryNames() map[string]string
	GetSubdivisions() []Subdivisions
}

type city struct {
	geoNameID         uint
	names             map[string]string
	countryGeoNameID  uint
	iso               string
	isInEuropeanUnion bool
	countryNames      map[string]string
	subdivisions      []Subdivisions
}

// GetGeoNameID get geo name id
func (c city) GetGeoNameID() uint {
	return c.geoNameID
}

// GetNames get names
func (c city) GetNames() map[string]string {
	if c.names == nil {
		return map[string]string{}
	}
	return c.names
}

// GetCountryGeoNameID get country geo name id
func (c city) GetCountryGeoNameID() uint {
	return c.countryGeoNameID
}

// GetISO get iso
func (c city) GetISO() string {
	return c.iso
}

// IsInEuropeanUnion is european union
func (c city) IsInEuropeanUnion() bool {
	return c.isInEuropeanUnion
}

// GetCountryNames get country names
func (c city) GetCountryNames() map[string]string {
	if c.countryNames == nil {
		return map[string]string{}
	}
	return c.countryNames
}

// GetSubdivisions get Subdivisions
func (c city) GetSubdivisions() []Subdivisions {
	return c.subdivisions
}

// Subdivisions Subdivisions
type Subdivisions struct {
	GeoNameID uint              `json:"geo_name_id"`
	IsoCode   string            `json:"iso_code"`
	Names     map[string]string `json:"names"`
}

// GetGeoNameID get geo name id
func (s Subdivisions) GetGeoNameID() uint {
	return s.GeoNameID
}

// GetIsoCode get iso code
func (s Subdivisions) GetIsoCode() string {
	return s.IsoCode
}

// GetNames get names
func (s Subdivisions) GetNames() map[string]string {
	if s.Names == nil {
		return map[string]string{}
	}
	return s.Names
}
