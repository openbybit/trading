package geo

// GeoData geo data
type GeoData interface {
	HasCountryInfo() bool
	HasCityInfo() bool
	GetCountry() Country
	GetCity() City
}

type geoData struct {
	HasCountry bool    `json:"has_country,omitempty"`
	HasCity    bool    `json:"has_city,omitempty"`
	Country    country `json:"country,omitempty"`
	City       city    `json:"city,omitempty"`
}

// HasCountryInfo has country info
func (g geoData) HasCountryInfo() bool {
	return g.HasCountry
}

// HasCityInfo has city info
func (g geoData) HasCityInfo() bool {
	return g.HasCity
}

// GetCountry get country
func (g geoData) GetCountry() Country {
	return g.Country
}

// GetCity get city
func (g geoData) GetCity() City {
	return g.City
}
