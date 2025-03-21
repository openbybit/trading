package geoipdb

import "errors"

var (
	ErrBadIPString = errors.New("bad ip string")

	// invalidDatabase = errors.New("invalid database")
	invalidSha256 = errors.New("invalid sha256 data")
)

const (
	template = "https://download.maxmind.com/app/geoip_download?edition_id={{id}}&license_key={{license}}&suffix=tar.gz"

	cityPrefix    = "GeoIP2-City"
	countryPrefix = "GeoIP2-Country"

	cityDataBase    = "GeoIP2-City.mmdb"
	countryDataBase = "GeoIP2-Country.mmdb"

	cityLiteDataBase    = "GeoLite2-City.mmdb"
	countryLiteDataBase = "GeoLite2-Country.mmdb"
)
