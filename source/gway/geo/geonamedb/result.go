package geonamedb

import (
	"fmt"
)

type GeoNameQueryResult struct {
	city    *GeoNameCity
	country *GeonameCountry
	hasCity bool
}

func (rst GeoNameQueryResult) String() string {
	label := "Country"
	id := rst.country.GeoNameId
	if rst.hasCity {
		label = "City"
		id = rst.city.GeoNameId
	}

	return fmt.Sprintf("[%s:%s#%d]", label, rst.Name(), id)
}

func (rst GeoNameQueryResult) City() *GeoNameCity {
	return rst.city
}

func (rst GeoNameQueryResult) Country() *GeonameCountry {
	return rst.country
}

func (rst GeoNameQueryResult) HasCity() bool {
	return rst.hasCity
}

func (rst GeoNameQueryResult) Name() string {
	if rst.hasCity {
		return rst.city.Name
	} else {
		return rst.country.Country
	}
}
