package main

import (
	"encoding/json"
	// "fmt"
	"github.com/alphazero/Go-Redis"
)

var log = &Log{}

// Const for status
const (
	RedisManagerStatusEmpty     = "0"
	RedisManagerStatusUnchecked = "1"
	RedisManagerStatusChecked   = "2"
	RedisManagerStatusDirty     = "3"
)

const (
	RedisManagerExpireTime int64 = 21600 // 6 hours
)

type RedisManager struct {
	// Private
	client redis.Client
	// Constructor args
	expireTime int64
	host       string
	port       int
	password   string
	db         int
}

func NewRedisManager(host string, port int, password string, db int) *RedisManager {
	return &RedisManager{
		client:     nil,
		expireTime: RedisManagerExpireTime,
		host:       host,
		port:       port,
		password:   password,
		db:         db,
	}
}

func NewRedisManagerWithExpireTime(host string, port int, password string, db int, expireTime int64) *RedisManager {
	return &RedisManager{
		client:     nil,
		expireTime: expireTime,
		host:       host,
		port:       port,
		password:   password,
		db:         db,
	}
}

// No Use yet
// func (redisMgr *RedisManager) Run() {}

func (redisMgr *RedisManager) getClient() (redis.Client, error) {
	if redisMgr.client != nil && redisMgr.client.Ping() != nil {
		return redisMgr.client, nil
	}
	spec := redis.DefaultSpec().Host(redisMgr.host).Port(redisMgr.port).Password(redisMgr.password).Db(redisMgr.db)
	client, err := redis.NewSynchClientWithSpec(spec)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	redisMgr.client = client // Cache client
	return client, nil
}

func (redisMgr *RedisManager) getStatusKey(key string) string {
	return key + "/status"
}

func (redisMgr *RedisManager) Set(key string, str string) error {
	client, err := redisMgr.getClient()
	if err != nil {
		return err
	}
	err = client.Set(key, []byte(str))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func (redisMgr *RedisManager) Get(key string) (string, error) {
	client, err := redisMgr.getClient()
	if err != nil {
		return "", err
	}
	strBytes, err := client.Get(key)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}
	return string(strBytes), nil
}

func (redisMgr *RedisManager) Del(key string) (bool, error) {
	client, err := redisMgr.getClient()
	if err != nil {
		return false, err
	}
	return client.Del(key)
}

func (redisMgr *RedisManager) SetObject(key string, obj interface{}) (string, error) {
	bytes, err := json.Marshal(obj)
	if err != nil {
		log.Error(err.Error())
		return RedisManagerStatusEmpty, err
	}
	log.Debug(bytes)
	client, err := redisMgr.getClient()
	if err != nil {
		return RedisManagerStatusEmpty, err
	}
	exists, err := client.Exists(key)
	if err != nil {
		log.Error(err.Error())
		return RedisManagerStatusEmpty, err
	}
	statusKey := redisMgr.getStatusKey(key)
	status := RedisManagerStatusUnchecked
	if exists {
		statusBytes, err := client.Get(statusKey)
		if err != nil { // Suck error catching!
			log.Error(err.Error())
			return RedisManagerStatusEmpty, err
		}
		// If unchecked, mark dirty.
		if string(statusBytes) == RedisManagerStatusUnchecked {
			status = RedisManagerStatusDirty
		}
	}
	client.Set(key, bytes)
	client.Set(statusKey, []byte(status))
	return status, nil
}

func (redisMgr *RedisManager) GetObject(key string, obj interface{}) (string, error) {
	client, err := redisMgr.getClient()
	if err != nil {
		return RedisManagerStatusEmpty, err
	}
	exists, err := client.Exists(key)
	if err != nil {
		log.Error(err.Error())
		return RedisManagerStatusEmpty, err
	}
	statusKey := redisMgr.getStatusKey(key)
	if exists {
		statusBytes, err := client.Get(statusKey)
		if err != nil { // Suck error catching!
			log.Error(err.Error())
			return RedisManagerStatusEmpty, err
		}
		bytes, err := client.Get(key)
		if err != nil {
			log.Error(err.Error())
			return RedisManagerStatusEmpty, err
		}
		errParse := json.Unmarshal(bytes, obj)
		if errParse != nil {
			log.Error(err.Error())
			return RedisManagerStatusEmpty, errParse
		}
		return string(statusBytes), nil
	} else {
		// TODO Should I declare an error type?
		log.Error("RedisManagerError: Empty Data")
		obj = nil
		return RedisManagerStatusEmpty, nil
	}

}

func (redisMgr *RedisManager) DelObject(key string) (bool, error) {
	client, err := redisMgr.getClient()
	if err != nil {
		return false, err
	}
	statusKey := redisMgr.getStatusKey(key)
	delResult, err := client.Del(key)
	if err != nil {
		log.Error(err.Error())
		return false, err
	}
	delStatusResult, err := client.Del(statusKey)
	if err != nil {
		log.Error(err.Error())
		return false, err
	}
	return delResult && delStatusResult, nil
}

func (redisMgr *RedisManager) CheckObject(key string) error {
	client, err := redisMgr.getClient()
	if err != nil {
		return err
	}
	statusKey := redisMgr.getStatusKey(key)
	exists, err := client.Exists(key)
	if err != nil {
		log.Error(err.Error())
		return err
	} else if !exists {
		// TODO Should I declare an error type?
		return nil
	} else if statusBytes, err := client.Get(statusKey); err != nil || string(statusBytes) == RedisManagerStatusChecked {
		// TODO Should I declare an error type?
		return nil
	}
	errSet := client.Set(statusKey, []byte(RedisManagerStatusChecked))
	if errSet != nil {
		return errSet
	}
	return nil
}

// TODO Should be implemented here
// func (redisMgr *RedisManager) SetObjects(key string, objs []interface{}) {
// 	log.Debug(key)
// 	for _, obj := range objs {
// 		log.Debug(obj)
// 	}
// }
