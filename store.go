package elastic

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"
)

var (
	profile_id = "general"
)

type StoreConfig struct {
	Index           string
	Default         interface{}
	RefreshDuration time.Duration
}

type Store struct {
	cnf  *StoreConfig
	data map[string]interface{}
}

func NewStore(c *StoreConfig) (*Store, error) {
	if c.RefreshDuration.Seconds() == 0 {
		c.RefreshDuration = time.Minute
	}
	if c.Default == nil {
		return nil, fmt.Errorf("default not specified on store")
	}

	var s *Store = &Store{
		cnf:  c,
		data: make(map[string]interface{}),
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

func (s *Store) load() error {
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

func (s *Store) Read(myvar interface{}) error {
	if myvar == nil {
		return fmt.Errorf("you specified variable is nil and is not struct!")
	}
	if reflect.ValueOf(myvar).Kind() != reflect.Pointer {
		return fmt.Errorf("you must specify a pointer not variable")
	}

	return Unmarshal(s.data, myvar)
}

func (s *Store) Refresh() error {
	res, err := Get(s.cnf.Index, profile_id)
	if err != nil {
		return err
	}
	err = json.Unmarshal(res, &s.data)
	if err != nil {
		return err
	}

	return nil
}

func (c *Store) Write(s interface{}) error {
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

func (s *Store) createDefault() error {
	log.Println("creating default setting...")
	// unmarshal default as data
	err := Unmarshal(s.cnf.Default, s.data)
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

func Unmarshal(a, b interface{}) error {
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
