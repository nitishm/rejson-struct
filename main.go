package main

import (
	"encoding/json"
	"flag"
	"fmt"
	rejson "go-rejson"
	"log"

	"github.com/gomodule/redigo/redis"
)

var addr = flag.String("Server", "localhost:6379", "Redis server address")

// Name - student name
type Name struct {
	First  string `json:"first,omitempty"`
	Middle string `json:"middle,omitempty"`
	Last   string `json:"last,omitempty"`
}

// Student - student object
type Student struct {
	// CHECKPOINT -
	// Info is an embedded pointer. This is where redigo's base
	// library faces the issue.
	Info *StudentDetails `json:"info,omitempty"`
	Rank int             `json:"rank,omitempty"`
}

type StudentDetails struct {
	FirstName string
	LastName  string
	Major     string
}

func main() {
	flag.Parse()

	conn, err := redis.Dial("tcp", *addr)
	if err != nil {
		log.Fatalf("Failed to connect to redis-server @ %s", *addr)
		return
	}

	student := Student{
		Info: &StudentDetails{
			FirstName: "John",
			LastName:  "Doe",
			Major:     "CSE",
		},
		Rank: 1,
	}

	// CHECKPOINT -
	// Add the student object to the store as a HMSET.
	err = addStructHash(conn, "JohnDoeHash", student)
	if err != nil {
		log.Fatalf("Failed to addStructHash - %s", err)
		return
	}

	// CHECKPOINT -
	// Add the student object to the store as a JSON.SET
	err = addStructReJSON(conn, "JohnDoeJSON", student)
	if err != nil {
		log.Fatalf("Failed to addStructReJSON - %s", err)
		return
	}

	// CHECKPOINT -
	// Redigo stores embedded objects as MultiBulk entries, which are essentially
	// just string objects. All key/value pairs are stored as strings.
	// The only way we can read the stored Hash members is to use the HGETALL and return
	// the values as a map[string]string.
	outStudentMap, err := redis.StringMap(getStructHash(conn, "JohnDoeHash"))
	if err != nil {
		log.Fatalf("Failed to getStructHash - %s", err)
		return
	}
	fmt.Printf("[HASH] Student Info %v [Type %T]\n", outStudentMap["Info"], outStudentMap["Info"])
	// OUTPUT :
	// [HASH] Student Info &{John Doe CSE} [Type string]
	// =====================================
	// 127.0.0.1:6379> HGET JohnDoeHash Info
	// Info
	// &{John Doe CSE}
	// =====================================

	outJSON, err := getStructReJSON(conn, "JohnDoeJSON")
	if err != nil {
		log.Fatalf("Failed to getStructReJSON - %s", err)
		return
	}

	outStudent := &Student{}
	err = json.Unmarshal(outJSON.([]byte), outStudent)
	if err != nil {
		log.Fatalf("Failed to JSON Unmarshal - %s", err)
		return
	}
	fmt.Printf("[ReJSON] Student Info %v [Type %T]\n", outStudent.Info, outStudent.Info)
	// OUTPUT :
	// [ReJSON] Student Info &{John Doe CSE} [Type *main.StudentDetails]
	// =====================================
	// 127.0.0.1:6379> JSON.GET JohnDoeJSON INDENT "\t" NEWLINE "\n" SPACE " " .info
	// {
	// 		"FirstName": "John",
	// 		"LastName": "Doe",
	// 		"Major": "CSE"
	// }
	// =====================================

	// CHECKPOINT :
	// Alternatively we could still use Redigo HSET to store our object as a JSON string
	err = addStructHashWithJSON(conn, "JohnDoeHashJSON", student)
	if err != nil {
		log.Fatalf("Failed to addStructHashWithJSON - %s", err)
		return
	}

	outHashJSON, err := getStructHashWithJSON(conn, "JohnDoeHashJSON")
	if err != nil {
		log.Fatalf("Failed to getStructHashWithJSON - %s", err)
		return
	}

	outHashJSONStudent := &Student{}
	err = json.Unmarshal(outHashJSON.([]byte), outHashJSONStudent)
	if err != nil {
		log.Fatalf("Failed to JSON Unmarshal - %s", err)
		return
	}
	fmt.Printf("[HSET JSON] Student Info %v [Type %T]\n", outHashJSONStudent.Info, outHashJSONStudent.Info)
	// OUTPUT :
	// [HSET JSON] Student Info &{John Doe CSE} [Type *main.StudentDetails]
	// =====================================
	// 127.0.0.1:6379> HGETALL JohnDoeHashJSON
	// JSON
	// {"info":{"FirstName":"John","LastName":"Doe","Major":"CSE"},"rank":1}
	// =====================================

}

func addStructHash(conn redis.Conn, key string, value interface{}) (err error) {
	_, err = conn.Do("HMSET", redis.Args{key}.AddFlat(value)...)
	if err != nil {
		return
	}

	return
}

func getStructHash(conn redis.Conn, key string) (value interface{}, err error) {
	return conn.Do("HGETALL", "JohnDoeHash")
}

func addStructReJSON(conn redis.Conn, key string, value interface{}) (err error) {
	_, err = rejson.JSONSet(conn, key, ".", value, false, false)
	if err != nil {
		return
	}
	return
}

func getStructReJSON(conn redis.Conn, key string) (value interface{}, err error) {
	return rejson.JSONGet(conn, key, "")
}

func addStructHashWithJSON(conn redis.Conn, key string, value interface{}) (err error) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, err = conn.Do("HSET", key, "JSON", string(b))
	if err != nil {
		return
	}
	return
}

func getStructHashWithJSON(conn redis.Conn, key string) (value interface{}, err error) {
	value, err = conn.Do("HGET", key, "JSON")
	if err != nil {
		return
	}
	return
}
