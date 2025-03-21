package geoip

type countryInfo string

const (
	countryIso   countryInfo = "iso"
	countryIso3  countryInfo = "iso3"
	countryFiat  countryInfo = "currencycode"
	countryGeoID countryInfo = "geonameid"
)

type cityInfo string

const (
	cityName  cityInfo = "name"
	cityGeoID cityInfo = "geonameid"
)

type geoRule struct {
	bannedRule
	metadataCountry
	metadataCity
}

type metadataCountry struct {
	needTransfer bool // check needTransfer
	iso          bool // iso
	iso3         bool // iso3
	fiat         bool // currencyCode
	geoID        bool // geonameid
}

type metadataCity struct {
	needTransfer bool // check needTransfer
	name         bool // name
	geoID        bool // geonameid
}

type geoMetadata struct {
	Country []string `json:"country"`
	City    []string `json:"city"`
}

type bannedRule struct {
	bannedCountries string
}

func (t metadataCountry) IsOpen() bool {
	return t.iso || t.iso3 || t.fiat || t.geoID
}

func (t metadataCity) IsOpen() bool {
	return t.name || t.geoID
}
