# Using Redis HMSET versus RedisLab's ReJSON for golang objects
In this example I present an alternative way to use Redis to store (embedded) structs like -
```golang
type Student struct {
	Info *StudentDetails
	Rank int            
}

type StudentDetails struct {
	FirstName string
	LastName  string
	Major     string
}
```

## Why can't we just use [Redigo](https://github.com/gomodule/redigo) ?
I have used [redigo](https://github.com/gomodule/redigo), for the past year, to interact with Redis and absolutely love the library.
However the biggest problem I face is when working with embedded structs. Redigo works fine if we stick to standard data types and avoid embedded struct. The [`HMSET`](https://redis.io/commands/hmset) arguments can be used in conjuction with `redis.Args{key}.AddFlat(value)...` to flatten a data-structure to be stored in Redis. The data can be read back on an [`HGETALL`](https://redis.io/commands/hgetall) using the helper `redis.ScanStruct(value, &obj)`.

The problem arises when this is used with an embedded struct instance. The resulting data stored in Redis looks like below - 

> [Key] Info - [Value] &{John Doe CSE} [Type string]

When the data is read back into the structure using `redis.ScanStruct(value, &student)`, it fails to port the data into the embedded object, and returns an **error**.

# Solution#1
## JSON Marshal
A way of getting around this is to store the object as a JSON string. 

Add the object to the DB using :

```golang
// Add it into Redis against the JSON field (This can be done with a regular SET as well) 
b, err := json.Marshal(value)
if err != nil {
	return
}
_, err = conn.Do("HSET", key, "JSON", string(b))
if err != nil {
	return
}

// Read it from Redis and Unmarshal back into the struct 
s, err := redis.String(conn.Do("HGET", key, "JSON"))
if err != nil {
	return
}

err = json.Unmarshal([]byte(s), res)
if err != nil {
	return
}
```

This works well if all you need to do is cache the entire object and not worry about ever accessing or modifying the individual members of the object.

However, if you wish to read / modify the fields in the object you will have to Unmarshal the object, modify the field and then re-add the object back into Redis.

# Solution#2
## [ReJSON](https://github.com/RedisLabsModules/ReJSON/)
With `ReJSON` you can instead store the object into Redis directly as a JSON object (mind you not Marshaled as JSON string). The object is added to Redis using the [`JSON.SET`](http://rejson.io/commands/#jsonset) command. The best part is that we can now `GET` any part of our JSON object back from Redis using the [`JSON.GET`](http://rejson.io/commands/#jsondel) and specifying the path to the member field.

To add the object into Redis using the ReJSON module I use [go-rejson](https://github.com/nitishm/go-rejson), a helper library that I wrote to easily use the commands with redigo.

```golang
// func JSONSet(conn redis.Conn, key string, path string, obj interface{}, NX bool, XX bool) (res interface{}, err error)
_, err = rejson.JSONSet(conn, key, "", value, false, false)
if err != nil {
	return
}
return
```

And each field in the object can be read using :
```golang
// func JSONGet(conn redis.Conn, key string, path string) (res interface{}, err error)
res, err := rejson.JSONGet(conn, key, path)
if err != nil {
	return
}
```

There is a whole bunch of documentation around using the ReJSON module available at [rejson.io](http://rejson.io/).

# Example
## Running
### Docker
Run the docker container provided by ReJSON as follows,

```
docker run -p 6379:6379 --name redis-rejson redislabs/rejson:latest
```

Once the container has spun up, run the `main.go` by performing,

```
go run main.go
```

## Output
Running the example would generate the entries shown below : 
```
127.0.0.1:6379> keys *
1) "JohnDoeJSON"
2) "JohnDoeHashJSON"
3) "JohnDoeHash"
```

**Re-JSON (with pretty print options)**
```
127.0.0.1:6379> JSON.GET JohnDoeJSON INDENT "\t" NEWLINE "\n" SPACE " "
{
        "info": {
                "FirstName": "John",
                "LastName": "Doe",
                "Major": "CSE"
        },
        "rank": 1
}
```

**HGETALL with key/value pair in odd/even numbers**
```
127.0.0.1:6379> HGETALL JohnDoeHash
Info
&{John Doe CSE}
Rank
1
```

**HGETALL (stored as JSON)**
```
127.0.0.1:6379> HGETALL JohnDoeHashJSON
JSON
{"info":{"FirstName":"John","LastName":"Doe","Major":"CSE"},"rank":1}
```

# The fancy bits
## Getting an object field using ReJSON
```
127.0.0.1:6379> JSON.GET JohnDoeJSON INDENT "\t" NEWLINE "\n" SPACE " " .info
{
        "FirstName": "John",
        "LastName": "Doe",
        "Major": "CSE"
}
```

## Setting an object field using ReJSON
```
127.0.0.1:6379> JSON.SET JohnDoeJSON info.Major '"EEE"'
OK

127.0.0.1:6379> JSON.GET JohnDoeJSON INDENT "\t" NEWLINE "\n" SPACE " " .info
{
        "FirstName": "John",
        "LastName": "Doe",
        "Major": "EEE"
}
```