package geonamedb

import (
	"strconv"
)

func geoNameRecordFromStringSlice(record []string) *GeoNameCity {
	if len(record) < 19 {
		panic("insufficient geoname record len")
	}
	geoNameId, err := strconv.Atoi(record[0])
	panicif(err)

	population, err := strconv.Atoi(record[14])
	panicif(err)

	var elevation int
	if len(record[15]) > 0 {
		elevation, err = strconv.Atoi(record[15])
		panicif(err)
	}

	var dem int
	if len(record[16]) > 0 {
		dem, err = strconv.Atoi(record[16])
		panicif(err)
	}

	return &GeoNameCity{
		GeoNameId:        int64(geoNameId),
		Name:             record[1],
		AsciiName:        record[2],
		AlternateNames:   record[3],
		Latitude:         record[4],
		Longitude:        record[5],
		FeatureClass:     record[6],
		FeatureCode:      record[7],
		Country:          record[8],
		Cc_2:             record[9],
		Admin_1:          record[10],
		Admin_2:          record[11],
		Admin_3:          record[12],
		Admin_4:          record[13],
		Population:       int64(population),
		Elevation:        int64(elevation),
		Dem:              int64(dem),
		Timezone:         record[17],
		ModificationDate: record[18],
	}
}

func geoNameCountryFromStringSlice(record []string) *GeonameCountry {
	if len(record) < 19 {
		panic("insufficient geoname country record len")
	}

	area, err := strconv.ParseFloat(record[6], 64)
	panicif(err)

	population, err := strconv.Atoi(record[7])
	panicif(err)

	geoNameId, err := strconv.Atoi(record[16])
	panicif(err)

	return &GeonameCountry{
		Iso:                record[0],
		Iso_3:              record[1],
		IsoNumeric:         record[2],
		Fips:               record[3],
		Country:            record[4],
		Capital:            record[5],
		Area:               area,
		Population:         int64(population),
		Continent:          record[8],
		Tld:                record[9],
		CurrencyCode:       record[10],
		CurrencyName:       record[11],
		Phone:              record[12],
		PostalCodeFormat:   record[13],
		PostalCodeRegex:    record[14],
		Languages:          record[15],
		GeoNameId:          int64(geoNameId),
		Neighbours:         record[17],
		EquivalentFipsCode: record[18],
	}
}

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

type GeoNameCity struct {
	GeoNameId        int64  // integer id of record in geonames database
	Name             string // name of geographical point (utf8) varchar(200)
	AsciiName        string // name of geographical point in plain ascii characters, varchar(200)
	AlternateNames   string // alternatenames, comma separated, ascii names automatically transliterated, convenience attribute from alternatename table, varchar(10000)
	Latitude         string // latitude in decimal degrees (wgs84)
	Longitude        string // longitude in decimal degrees (wgs84)
	FeatureClass     string // see http://www.geonames.org/export/codes.html, char(1)
	FeatureCode      string // see http://www.geonames.org/export/codes.html, varchar(10)
	Country          string // ISO-3166 2-letter country code, 2 characters
	Cc_2             string // alternate country codes, comma separated, ISO-3166 2-letter country code, 200 characters
	Admin_1          string // fipscode (subject to change to iso code), see exceptions below, see file admin1Codes.txt for display names of this code; varchar(20)
	Admin_2          string // code for the second administrative division, a county in the US, see file admin2Codes.txt; varchar(80)
	Admin_3          string // code for third level administrative division, varchar(20)
	Admin_4          string // code for fourth level administrative division, varchar(20)
	Population       int64  // bigint (8 byte int)
	Elevation        int64  // in meters, integer
	Dem              int64  // digital elevation model, srtm3 or gtopo30, average elevation of 3''x3'' (ca 90mx90m) or 30''x30'' (ca 900mx900m) area in meters, integer. srtm processed by cgiar/ciat.
	Timezone         string // the iana timezone id (see file timeZone.txt) varchar(40)
	ModificationDate string // date of last modification in yyyy-MM-dd format
}

type GeonameCountry struct {
	Iso                string // CN
	Iso_3              string // CHN
	IsoNumeric         string
	Fips               string
	Country            string  // China
	Capital            string  // Beijing
	Area               float64 // sq km
	Population         int64
	Continent          string // AS
	Tld                string // .cn
	CurrencyCode       string // CNY
	CurrencyName       string // Yuan Renminbi
	Phone              string // 86
	PostalCodeFormat   string
	PostalCodeRegex    string
	Languages          string // zh-CN,yue,wuu,dta,ug,za
	GeoNameId          int64  // 1814991
	Neighbours         string // LA,BT,TJ,KZ,MN,AF,NP,MM,KG,PK,KP,RU,VN,IN
	EquivalentFipsCode string
}
