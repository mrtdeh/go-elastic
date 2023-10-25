package elastic

import (
	"encoding/json"
	"log"
	"strings"
	"time"
)

var (
	profile_id = "general"
)

type StoreConfig struct {
	Index   string
	Default interface{}
}

type setting struct {
	cnf  *StoreConfig
	data map[string]interface{}
}

func NewStore(c *StoreConfig) (*setting, error) {

	var s *setting = &setting{
		cnf:  c,
		data: make(map[string]interface{}),
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *setting) load() error {
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

	err := json.Unmarshal(res, &s.data)
	if err != nil {
		return err
	}

	return nil
}

func (s *setting) Read(myvar interface{}) error {
	return unmarshal(s.data, myvar)
}

func (c *setting) Write(s interface{}) error {
	data, _ := json.Marshal(s)
	err := Index(c.cnf.Index, data, profile_id)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &c.data)
	if err != nil {
		return err
	}
	return nil
}

func (s *setting) createDefault() error {
	log.Println("creating default setting...")
	// unmarshal default as data
	err := unmarshal(s.cnf.Default, s.data)
	if err != nil {
		return err
	}
	// marshal default to byte
	data, err := json.Marshal(s.cnf.Default)
	if err != nil {
		return err
	}
	// index data byte to index
	err = Index(s.cnf.Index, data, profile_id)
	if err != nil {
		return err
	}
	return nil
}

func unmarshal(a, b interface{}) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &b)
	if err != nil {
		return err
	}
	return nil
}
