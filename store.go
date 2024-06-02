package elastic

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
)

var (
	profile_id = "general"

	cacheFilename = "/var/lib/setting-management/cache-"
)

type StoreConfig struct {
	Index           string
	Default         interface{}
	RefreshDuration time.Duration
}

type store struct {
	cnf  *StoreConfig
	Data *map[string]interface{} `json:"data"`
}

func NewStore(c *StoreConfig) func() (*store, error) {
	return func() (*store, error) {

		if c.RefreshDuration.Seconds() == 0 {
			c.RefreshDuration = time.Minute
		}
		if c.Default == nil {
			return nil, fmt.Errorf("default not specified on store")
		}

		var s *store = &store{
			cnf:  c,
			Data: &map[string]interface{}{},
		}

		if err := s.load(); err != nil {
			return nil, err
		}

		go func() {
			for {
				time.Sleep(c.RefreshDuration)
				s.Refresh()
			}
		}()

		return s, nil
	}

}

func (s *store) load() error {
	var res []byte
	for {
		var err error
		res, err = Get(s.cnf.Index, profile_id)
		if err != nil {

			if strings.Contains(err.Error(), "404") {
				log.Println("failed to load settings : ", err.Error())
				err := s.createDefault()
				if err != nil {
					return err
				}

				time.Sleep(time.Second * 10)
				continue
			}
			return nil
		}
		break
	}

	err := json.Unmarshal(res, s.Data)
	if err != nil {
		return err
	}

	if err := s.cacheToFile(res); err != nil {
		log.Println("erorr in write to cahce : ", err.Error())
	}

	return nil
}

func (s *store) Read(myvar interface{}) error {
	if myvar == nil {
		return fmt.Errorf("you specified variable is nil and is not struct!")
	}
	if reflect.ValueOf(myvar).Kind() != reflect.Pointer {
		return fmt.Errorf("you must specify a pointer not variable")
	}

	return Unmarshal(s.Data, myvar)
}

func (s *store) Refresh() error {
	res, err := Get(s.cnf.Index, profile_id)
	if err != nil {
		return err
	}
	err = json.Unmarshal(res, s.Data)
	if err != nil {
		return err
	}

	if err := s.cacheToFile(res); err != nil {
		log.Println("erorr in write to cahce : ", err.Error())
	}

	return nil
}

func (c *store) Write(s interface{}) error {
	data, _ := json.Marshal(s)
	err := Index(c.cnf.Index, data, profile_id)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, c.Data)
	if err != nil {
		return err
	}

	if err := c.cacheToFile(data); err != nil {
		log.Println("erorr in write to cahce : ", err.Error())
	}

	return nil
}

func (s *store) Reset() error {
	if err := DeleteIndex(s.cnf.Index); err != nil {
		return err
	}
	err := s.createDefault()
	if err != nil {
		return err
	}
	return nil
}

func (s *store) createDefault() error {
	log.Println("creating default setting...")
	// unmarshal default as data
	data, err := s.readCache()

	if err != nil || len(data) == 0 {
		log.Printf("erorr in read cahce %v or data is empty", err)
		err := Unmarshal(s.cnf.Default, s.Data)
		if err != nil {
			return err
		}
	} else {
		log.Println("load from cache...")
		err = json.Unmarshal([]byte(data), s.Data)
		if err != nil {
			return err
		}
	}

	// marshal default to byte
	jsondata, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}
	// index data byte to index
	err = Index(s.cnf.Index, jsondata, profile_id)
	if err != nil {
		return err
	}
	return nil
}

func Unmarshal(a, b interface{}) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, b)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) cacheToFile(content []byte) error {
	var data []byte = content
	// if len(content) > 0 {
	// 	base64Data := base64.StdEncoding.EncodeToString(content)
	// 	data = []byte(base64Data)
	// }

	err := os.WriteFile(cacheFilename+s.cnf.Index, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *store) readCache() (string, error) {

	data, err := os.ReadFile(cacheFilename + s.cnf.Index)
	if err != nil {
		return "", err
	}

	// decoded, err := base64.StdEncoding.DecodeString(string(data))
	// if err != nil {
	// 	return "", err
	// }

	return string(data), nil
}
