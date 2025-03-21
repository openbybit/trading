package geonamedb

import (
	"bufio"
	"bytes"
	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"errors"
	"os"
	"path"
	"sync"
)

var (
	ErrNotFound = errors.New("not found")
)

type GeoName interface {
	QueryCityByGeoNameID(geonameID int64) (gr *GeoNameCity, err error)
	QueryByGeoNameID(geonameID int64) (rst GeoNameQueryResult, err error)
	QueryCountryByISO(iso string) (gc *GeonameCountry, err error)
	QueryCountryByGeoNameID(geonameID int64) (gc *GeonameCountry, err error)
}

type geoName struct {
	path                string
	loadCityOnce        sync.Once
	cities              map[int64]*GeoNameCity
	loadCountryOnce     sync.Once
	countries           map[string]*GeonameCountry
	geoNameIdCountryISO map[int64]string
}

func NewGeonames(path string) (GeoName, error) {
	if err := filesystem.MkdirAll(path); err != nil {
		return nil, err
	}
	return &geoName{
		path: path,
	}, nil
}

func (g *geoName) openCity() error {
	var err error

	g.loadCityOnce.Do(func() {
		var f *os.File
		f, err = os.OpenFile(path.Join(g.path, "cities500.txt"), os.O_RDONLY, os.FileMode(0644))
		if err != nil {
			return
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		recordMap := make(map[int64]*GeoNameCity)

		for scanner.Scan() {
			parts := bytes.Split(scanner.Bytes(), []byte("\t"))
			row := make([]string, len(parts))
			for idx, part := range parts {
				row[idx] = string(part)
			}

			r := geoNameRecordFromStringSlice(row)
			recordMap[r.GeoNameId] = r
		}
		err = scanner.Err()
		if err != nil {
			return
		}

		g.cities = recordMap
	})

	if err != nil {
		g.loadCityOnce = sync.Once{}
	}
	return nil
}

func (g *geoName) openCountry() error {
	var err error
	g.loadCountryOnce.Do(func() {
		var f *os.File
		f, err = os.OpenFile(path.Join(g.path, "countryInfo.txt"), os.O_RDONLY, os.FileMode(0644))
		if err != nil {
			return
		}
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		countriesMap := make(map[string]*GeonameCountry)
		geoNameIDMap := make(map[int64]string)

		for scanner.Scan() {
			line := scanner.Bytes()
			if line[0] == byte('#') {
				continue
			}

			parts := bytes.Split(line, []byte("\t"))
			row := make([]string, len(parts))
			for idx, part := range parts {
				row[idx] = string(part)
			}

			r := geoNameCountryFromStringSlice(row)
			countriesMap[r.Iso] = r
			geoNameIDMap[r.GeoNameId] = r.Iso
		}
		err = scanner.Err()
		if err != nil {
			return
		}

		g.countries = countriesMap
		g.geoNameIdCountryISO = geoNameIDMap
	})

	if err != nil {
		g.loadCountryOnce = sync.Once{}
	}

	return err
}

func (g *geoName) QueryCityByGeoNameID(geonameID int64) (gr *GeoNameCity, err error) {
	err = g.openCity()
	if err != nil {
		return
	}
	gr, ok := g.cities[geonameID]
	if !ok {
		err = ErrNotFound
	}
	return
}
func (g *geoName) QueryByGeoNameID(geonameID int64) (rst GeoNameQueryResult, err error) {
	city, err := g.QueryCityByGeoNameID(geonameID)
	switch {
	case err == nil:
	case errors.Is(err, ErrNotFound):
		err = nil
		var country *GeonameCountry
		country, err = g.QueryCountryByGeoNameID(geonameID)
		if err != nil {
			return
		}
		rst = GeoNameQueryResult{
			country: country,
			hasCity: false,
		}
		return
	default:
		return
	}

	var country *GeonameCountry
	country, err = g.QueryCountryByISO(city.Country)
	if err != nil {
		return
	}
	rst = GeoNameQueryResult{
		city:    city,
		country: country,
		hasCity: true,
	}
	return
}
func (g *geoName) QueryCountryByISO(iso string) (gc *GeonameCountry, err error) {
	if g.countries == nil {
		err = g.openCountry()
		if err != nil {
			return
		}
	}
	gc, ok := g.countries[iso]
	if !ok {
		err = ErrNotFound
	}
	return
}
func (g *geoName) QueryCountryByGeoNameID(geonameID int64) (gc *GeonameCountry, err error) {
	if g.geoNameIdCountryISO == nil {
		err = g.openCountry()
		if err != nil {
			return
		}
	}
	iso, ok := g.geoNameIdCountryISO[geonameID]
	if !ok {
		err = ErrNotFound
		return
	}
	gc, err = g.QueryCountryByISO(iso)
	return
}
