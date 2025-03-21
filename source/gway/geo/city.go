package geo

import (
	"code.bydev.io/fbu/gateway/gway.git/geo/geoipdb"
)

// City city
type City interface {
	GetGeoNameID() int64
	GetName() string
	GetSubdivisions() []geoipdb.Subdivisions
	GetSubdivision() geoipdb.Subdivisions
}

type city struct {
	GeoNameID    int64                  `json:"geo_name_id"`
	Name         string                 `json:"name"`
	Subdivisions []geoipdb.Subdivisions `json:"subdivisions"`
}

// GetGeoNameID get city geo name id
func (c city) GetGeoNameID() int64 {
	return c.GeoNameID
}

// GetName get city name
func (c city) GetName() string {
	return c.Name
}

// GetSubdivisions get Subdivisions
func (c city) GetSubdivisions() []geoipdb.Subdivisions {
	return c.Subdivisions
}

// GetSubdivision get subdivision
func (c city) GetSubdivision() geoipdb.Subdivisions {
	if len(c.Subdivisions) > 0 {
		return c.Subdivisions[0]
	}
	return geoipdb.Subdivisions{}
}
