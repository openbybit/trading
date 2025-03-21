package geoipdb

import (
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/oschwald/geoip2-golang"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"code.bydev.io/fbu/gateway/gway.git/gcore/recovery"
)

// GeoIP geo ip
type GeoIP interface {
	GetCountryInfo(ip string) (Country, error)
	GetCityInfo(ip string) (City, error)
	Close(file string) error
	CloseAll() error
}

var (
	defaultGeoService *geoIP
	once              sync.Once
)

type geoIP struct {
	downloader downloader
	cfg        *config
	db         map[string]*geoip2.Reader
	cache      *freecache.Cache
	sync.RWMutex
}

// New create geo ip
func New(storePath, licence string, cache *freecache.Cache, options ...Option) (GeoIP, error) {
	err := filesystem.MkdirAll(storePath)
	if err != nil {
		return nil, err
	}
	if defaultGeoService == nil {
		once.Do(func() {
			c := &config{
				storePath:        storePath,
				license:          licence,
				timeout:          time.Second * 300,
				autoUpdateEnable: true,
				autoUpdatePeriod: time.Hour * 24,
			}
			if len(options) > 0 {
				for _, o := range options {
					o.apply(c)
				}
			}
			defaultGeoService = &geoIP{
				downloader: newDownloader(c.license, c.storePath, c.timeout),
				cfg:        c,
				db:         make(map[string]*geoip2.Reader),
				cache:      cache,
			}
			defaultGeoService.keepGeoIpUpdate()
		})
	}
	if err != nil {
		return nil, err
	}
	return defaultGeoService, nil
}

// GetCountryInfo get country info
func (g *geoIP) GetCountryInfo(ip string) (Country, error) {
	parseIP := net.ParseIP(ip)
	if parseIP == nil {
		return nil, ErrBadIPString
	}

	reader, err := g.openDBWithLock(countryLiteDataBase)
	if err != nil {
		return nil, err
	}

	countryInfo, err := reader.Country(parseIP)
	if err != nil {
		return nil, err
	}
	return country{
		geoNameID:         countryInfo.Country.GeoNameID,
		isoCode:           countryInfo.Country.IsoCode,
		isInEuropeanUnion: countryInfo.Country.IsInEuropeanUnion,
		names:             countryInfo.Country.Names,
	}, nil
}

// GetCityInfo get city info
func (g *geoIP) GetCityInfo(ip string) (City, error) {
	parseIP := net.ParseIP(ip)
	if parseIP == nil {
		return nil, ErrBadIPString
	}

	reader, err := g.openDBWithLock(cityLiteDataBase)
	if err != nil {
		return nil, err
	}

	cityInfo, err := reader.City(parseIP)
	if err != nil {
		return nil, err
	}

	c := city{
		geoNameID:         cityInfo.City.GeoNameID,
		names:             cityInfo.City.Names,
		countryGeoNameID:  cityInfo.Country.GeoNameID,
		iso:               cityInfo.Country.IsoCode,
		isInEuropeanUnion: cityInfo.Country.IsInEuropeanUnion,
		countryNames:      cityInfo.Country.Names,
		subdivisions:      make([]Subdivisions, 0, len(cityInfo.Subdivisions)),
	}
	for _, subdivision := range cityInfo.Subdivisions {
		s := Subdivisions{
			GeoNameID: subdivision.GeoNameID,
			IsoCode:   subdivision.IsoCode,
			Names:     subdivision.Names,
		}
		c.subdivisions = append(c.subdivisions, s)
	}
	return c, nil
}

func (g *geoIP) openDBWithLock(dbname string) (*geoip2.Reader, error) {
	g.RLock()
	reader, ok := g.db[dbname]
	g.RUnlock()
	if ok {
		return reader, nil
	}

	g.Lock()
	defer g.Unlock()
	if reader, ok := g.db[dbname]; ok {
		return reader, nil
	}

	db, err := geoip2.Open(path.Join(g.cfg.storePath, dbname))
	if err != nil {
		return nil, err
	}
	g.db[dbname] = db

	return db, nil
}

func (g *geoIP) open(dbname string) (*geoip2.Reader, error) {
	reader, ok := g.db[dbname]
	if ok {
		return reader, nil
	}

	db, err := geoip2.Open(path.Join(g.cfg.storePath, dbname))
	if err != nil {
		return nil, err
	}
	g.db[dbname] = db
	return db, nil
}

// Close close db
func (g *geoIP) Close(file string) error {
	g.Lock()
	defer g.Unlock()
	return g.close(file)
}

func (g *geoIP) close(file string) error {
	reader, ok := g.db[file]
	if !ok {
		return nil
	}
	err := reader.Close()
	delete(g.db, file)
	return err
}

// CloseAll close all
func (g *geoIP) CloseAll() error {
	var err error
	for name := range g.db {
		err = g.Close(name)
	}
	return err
}

func (g *geoIP) keepGeoIpUpdate() {
	if !g.cfg.autoUpdateEnable {
		return
	}
	recovery.Go(func() {
		g.autoUpdate()
	}, nil)
}

func (g *geoIP) autoUpdate() {
	interval := g.cfg.autoUpdatePeriod
	if interval == 0 {
		interval = 24 * time.Hour
	} else if interval < 24*time.Hour {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// update for first
	g.update()
	for {
		<-ticker.C
		g.update()
	}
}

func (g *geoIP) update() {
	log.Println("geoIP downloader Do:", cityPrefix)
	filename, cityTar, err := g.downloader.Do(cityPrefix)
	if err != nil {
		log.Printf("geoIP downloader Do: %s, error: %s\n", cityPrefix, err.Error())
		return
	}
	var updateCache bool
	if filename != "" {
		if err := g.updateDb(filename); err != nil {
			log.Println("geoIP updateDb error:", cityPrefix, err)
		}
		updateCache = true
	} else {
		log.Println("geoIP updateDb: no update", cityPrefix)
	}
	log.Println("geoIP downloader Do success:", cityPrefix)
	g.clear(cityTar, cityPrefix)

	log.Println("geoIP downloader Do:", countryPrefix)
	filename, countryTar, err := g.downloader.Do(countryPrefix)
	if err != nil {
		log.Printf("geoIP downloader Do: %s, error: %s\n", countryPrefix, err.Error())
		return
	}
	if filename != "" {
		if err := g.updateDb(filename); err != nil {
			log.Println("geoIP updateDb error:", countryPrefix, err)
		}
		updateCache = true
	} else {
		log.Println("geoIP updateDb: no update", countryPrefix)
	}
	log.Println("geoIP downloader Do success:", countryPrefix)
	g.clear(countryTar, countryPrefix)
	if updateCache && g.cache != nil {
		log.Println("geo cache update")
		g.cache.Clear()
	}
}

// updateDb rename
func (g *geoIP) updateDb(filename string) error {
	dbName := strings.TrimSuffix(filename, ".tmp")

	g.Lock()
	defer g.Unlock()
	// close db connection
	if err := g.close(dbName); err != nil {
		log.Printf("geoIP close %s error: %s\n", dbName, err.Error())
	}

	if err := os.Remove(g.getFilePath(dbName)); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("geoIP Remove %s error: %s\n", dbName, err.Error())
			return err
		}
	}

	if err := os.Rename(g.getFilePath(filename), g.getFilePath(dbName)); err != nil {
		log.Printf("geoIP Rename %s error: %s\n", dbName, err.Error())
		return err
	}

	// recover db connection
	_, err := g.open(dbName)
	if err != nil {
		log.Printf("geoIP open %s error: %s\n", dbName, err.Error())
		return err
	}

	return nil
}

func (g *geoIP) clear(tarName string, prefix string) {
	if tarName == "" {
		log.Println("clear tarName is nil")
		return
	}
	// clear old tar file
	files, err := filesystem.GetFilesInDir(g.cfg.storePath, ".tar.gz")
	if err != nil {
		log.Println(g.cfg.storePath, "GetFilesInDir error", err, tarName)
		return
	}
	for _, file := range files {
		name := file.Name()
		if name == tarName || !strings.HasPrefix(name, prefix) {
			log.Println(g.cfg.storePath, "skip clear current tar", name, tarName)
			continue
		}
		if err = os.Remove(g.getFilePath(name)); err != nil {
			log.Println(g.cfg.storePath, "Remove error", err, name)
			continue
		}
	}
}

func (g *geoIP) getFilePath(fileName string) string {
	return g.cfg.storePath + "/" + fileName
}
