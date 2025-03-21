package geo

// Country country
type Country interface {
	GetGeoNameID() int64
	GetISO() string
	GetISO3() string
	GetCurrencyCode() string
}

type country struct {
	GeoNameId    int64  `json:"geo_name_id"`
	Iso          string `json:"iso"`
	Iso3         string `json:"iso3"`
	CurrencyCode string `json:"currency_code"`
}

// GetGeoNameID get geo name id
func (c country) GetGeoNameID() int64 {
	return c.GeoNameId
}

// GetISO get iso
func (c country) GetISO() string {
	return c.Iso
}

// GetISO3 get iso3
func (c country) GetISO3() string {
	return c.Iso3
}

// GetCurrencyCode get currency code
func (c country) GetCurrencyCode() string {
	return c.CurrencyCode
}
