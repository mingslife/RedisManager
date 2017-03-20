package main

import (
	"encoding/json"
	"fmt"
	"github.com/alphazero/Go-Redis"
)

const (
	UNKNOWN = iota
	MALE
	FEMALE
)

type Student struct {
	Id   int
	Name string
	Sex  int
}

func (student *Student) introduce() {
	fmt.Printf("Hello, I am %s.\n", student.Name)
}

func main0() {
	spec := redis.DefaultSpec().Db(0).Password("")
	client, err := redis.NewSynchClientWithSpec(spec)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	key := "students/1"
	student := &Student{
		Id:   1,
		Name: "Ming",
	}
	b, errStringify := json.Marshal(student)
	if errStringify != nil {
		fmt.Println(errStringify.Error())
		return
	}
	errSet := client.Set(key, b)
	if errSet != nil {
		fmt.Println(errSet.Error())
		return
	}
	testStudent := &Student{}
	testB, err := client.Get(key)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	errParse := json.Unmarshal(testB, testStudent)
	if errParse != nil {
		fmt.Println(errParse)
		return
	}
	fmt.Println(testStudent.Id)
	fmt.Println(testStudent.Name)
}

const (
	DevelopmentMode = iota
	ProductionMode
)

var mode = ProductionMode

type Log struct{}

func (log *Log) Error(v interface{}) {
	if mode == DevelopmentMode {
		fmt.Println(v)
	}
}
func (log *Log) Debug(v interface{}) {
	if mode == DevelopmentMode {
		fmt.Println(v)
	}
}

var log = &Log{}

func main() {
	student := &Student{
		Id:   1,
		Name: "Ming",
		Sex:  MALE,
	}
	redisMgr := NewRedisManager("127.0.0.1", 6379, "", 0)
	redisMgr.SetObject("students/3", student)
	testStudent := &Student{}
	redisMgr.GetObject("students/3", testStudent)
	fmt.Println(testStudent)

	// student1 := &Student{
	// 	Id:   1,
	// 	Name: "Ming",
	// 	Sex:  MALE,
	// }
	// student2 := &Student{
	// 	Id:   2,
	// 	Name: "Mary",
	// 	Sex:  MALE,
	// }
	// student3 := &Student{
	// 	Id:   3,
	// 	Name: "Jack",
	// 	Sex:  FEMALE,
	// }
	// redisMgr.SetObjects("test", []Student{})

	// students := make([]Student, 100)
	// students = append(students, *student1)
	// students = append(students, *student2)
	// students = append(students, *student3)
	// redisMgr.SetObject("test", students)
}

// RedisManagerError

// No use yet
type RedisManagerError struct {
	error
	msg  string
	code int
}

const (
	RedisManagerUncheckError = iota
	RedisManagerCheckError
	RedisManagerDirtyError
	RedisManagerConnectError
	RedisManagerJsonEncodeError
	RedisManagerJsonDecodeError
)

func newRedisManagerError(message string, errorCode int) *RedisManagerError {
	return &RedisManagerError{
		msg:  message,
		code: errorCode,
	}
}

func (err *RedisManagerError) IsRedisManagerError() bool {
	return true
}

func (err *RedisManagerError) Error() string {
	return fmt.Sprintf("RedisManagerError: ", err.msg)
}

// RedisManager

// Const for status
const (
	RedisManagerStatusEmpty     = "0"
	RedisManagerStatusUnchecked = "1"
	RedisManagerStatusChecked   = "2"
	RedisManagerStatusDirty     = "3"
)

type RedisManager struct {
	// Private
	client redis.Client

	// Public
	Host     string
	Port     int
	Password string
	Db       int
}

func NewRedisManager(host string, port int, password string, db int) *RedisManager {
	return &RedisManager{
		client:   nil,
		Host:     host,
		Port:     port,
		Password: password,
		Db:       db,
	}
}

// No Use yet
// func (redisMgr *RedisManager) Run() {}

func (redisMgr *RedisManager) getClient() (redis.Client, error) {
	if redisMgr.client != nil && redisMgr.client.Ping() != nil {
		return redisMgr.client, nil
	}
	spec := redis.DefaultSpec().Host(redisMgr.Host).Port(redisMgr.Port).Password(redisMgr.Password).Db(redisMgr.Db)
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
		log.Error(err.Error())
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
		log.Error(err.Error())
		return "", err
	}
	strBytes, err := client.Get(key)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}
	return string(strBytes), nil
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
		log.Error(err.Error())
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
		log.Error(err.Error())
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

// TODO Should be implemented here
func (redisMgr *RedisManager) SetObjects(key string, objs []interface{}) {
	log.Debug(key)
	for _, obj := range objs {
		log.Debug(obj)
	}
}
