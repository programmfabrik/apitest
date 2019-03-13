# fylr apitest

The fylr apitesting tool helps you to build automated apitests that can be run after every build to ensure a constant product quality.

A single testcase is also a perfect definition of an occuring problem and helps the developers to fix your issues faster!

# Configuration file
For configuring the apitest tool, add the follwing section to your 'fylr.yml' configuration file.

The report parametere of this config can be overwritten via a command line flag. So you should set your intended standard values in the config.

 **After the first setup you don't need to touch the config again. (You should not do that, to prevent errors based on rash changes in the config)**

```yaml
apitest:
   server: "http://5.simon.pf-berlin.de/api/v1" //The base url to the api you want to fire the apitests against. Important: don’t add a trailing ‘/’
   report: //Configures the maschine report. For usage with jenkis or any other CI tool
      file: "apitest_report" //Filename of the report file. The file gets saved in the same directory of the fylr binary
      format: "json" //Format of the report. (Supported formats: json or junit)
```

## Command line interface

You start the apitest tool with the following command

```bash
./fylr apitest
```

This starts the command with the following default settings:

- Runs all tests that are in the current directory, or in any of its subdirectories
- Logs to console
- Writes the machine log, to the given file in the fylr.yml
- Logs only the request & responses if a test fails

### Configure which tests should be run

- `--directory testDirectory` or `-d testDirectory`: Defines which directory should be used for running the tests in it. The tool walks recursively trough all subdirectories and runs alls tests that have a "manifest.json" file in alphabetical order of the folder names. (Depth-First-Search)
- `--single path/to/a/single/manifest.json` or `-s path/to/a/single/manifest.json`: Run only a single test. The path needs to point directly to the manifest file. (Not the directory containing it)

### Configure logging

- `--verbosity 1` or `-v 1`: Set the verbosity of the logger. (Verbosity defines what kind of communication with the server should be shown) Can be realy helpfull to find out why a test fails.
	- `-v -1` (default): Only normal test ouput
	- `-v 0`: All from '-1' plus failed test responses
	- `-v 1`: All from '-1' plus all responses
	- `-v 2`:  All from '1' plus all requests
	

You can also set the log verbosity per single testcase. The greater verbosity wins.


#### Console logging

- `--log-console-enable false`: If you want to see a log in the console this parameter needs to be "true" (what is also the default)
- `--log-console-level debug`: Sets the loglevel which controls what kind of output should be displayed in the console
	- `--log-console-level info` (default): Shows only critical information
	- `--log-console-level warn`: Shows more verbose log output
	- `--log-console-level debug`: Shows all possible log output

#### SQLite logging

- `--log-sqlite-enable false`: If you want to save the into a sqlite databasethis parameter needs to be "true"
- `--log-sqlite-file newLog.db`: Defines the filename in which the sqlite log should be saved
- `--log-sqlite-level debug`: Sets the loglevel which controls what kind of output should be saved into the sqlite database
	- `--log-sqlite-level info` (default): Saves only critical information
	- `--log-sqlite-level warn`: Saves more verbose log output
	- `--log-sqlite-level debug`: Saves all possible log output

### Overwrite config parameters

- `--config subfolder/newConfigFile` or `-c subfolder/newConfigFile`: Overwrites the path of the fylr config file (default "./fylr.yml") with "subfolder/newConfigFile"
- `--report-file newReportFile`: Overwrites the report file name from the fylr.yml config with "newReportFile"
- `--report-format junit`: Overwrites the report format from the fylr.yml config with "junit"

### Examples

- Run all tests in the directory **apitests** display **all server communication** and safe the maschine report as **junit** for later parsing it with *jenkins*

```bash
./fylr apitest --directory apitests --verbosity 2 --report-format junit
```

- Only run a single test **apitests/test1/manifest.json** with **no console output** and save the maschine report to the standard file defined in the fylr.yml

```bash
./fylr apitest --single apitests/test1/manifest.json --log-console-enable false
```

# Manifest

Manifest is loaded as **template**, so you can use variables, Go **range** and **if** and others.

```json
{
    //General info about the testuite. Try to explain your problem indepth here. So that someone who works on the test years from now knows what is happening
    "description": "search api tests for filename", 
    //Testname. Should be the ticket number if the test is based on a ticket
    "name": "ticket_48565", 
    // init store
    "store": {
        "custom": "data"
    },
    //Testsuites your want to run upfront (e.g. a setup). Paths are relative to the current test manifest
    "require": [ 
        "setup_manifests/purge.yaml",
        "setup_manifests/config.yaml",
        "setup_manifests/upload_datamodel.yaml"
    ],
    //Array of single testcases. Add es much as you want. They get executed in chronological order
    "tests": [         
        //[SINGLE TESTCASE]: See below for more information
        //[SINGLE TESTCASE]: See below for more information
        //[SINGLE TESTCASE]: See below for more information
        
        //We also support the external loading of a complete test:
        "@pathToTest.json"
    ]
}

```

## Single Testcase Definition

### Basic Format
```json
{
    //Define if the testuite should continue even if this test fails. (default:false)
    "continue_on_failure": true, 
    //Name to identify this single test. Is important for the log. Try to give an explaning name
    "name": "Testname", 
    // Store custom values to the datastore
    "store": {
        "key1": "value1",
        "key2": "value2"
    },
    // Specify a unique log level only for this single test. (If cli has a greate log verbosity than this, cli wins)
    "log_verbosity": 2,
    //Defines what gets send to the server
    "request": { 
		//What endpoint we want to target. You find all possible endpoints in the api documentation
        "endpoint": "suggest", 
		//How the endpoint should be accessed. The api documentations tells your which methods are possible for an endpoint. All HTTP methods are possible.
        "method": "GET", 
		//Parameters that will be added to the url. e.g. http://5.testing.pf-berlin.de/api/v1/session?token=testtoken&number=2 would be defined as follows
        "query_params": { 
            "number": 2,
            "token": "testtoken"
        },
		//Additional headers that should be added to the request
		"header":{
			"header1":"value",
			"header2":"value"
		},
		// With header_from_you set a header to the value of the datastore field
		// In this example we set the "Content-Type" header to the value "application/json"
		// As "application/json" is stored as string in the datastore on index "contentType"
		"header_from_store": {
		  "Content-Type": "contentType"
		}
		//All the content you want to send in the http body. Is a JSON Object
		"body":{
			"flower":"rose",
			"animal":"dog"
		},
		//If the body should be marshaled in a special way, you can define this here. Is not a required attribute. Standart is to marshal the body as json. Possible: [multipart,urlencoded]
		"body_type":"urlencoded"
    },
    //Define how the response should look like. Testtool checks against this response
    "response": { 
    	//Expected http status code. See api documentation vor the right ones
        "statuscode": 200,
		//If you expect certain response headers, you can define them here. A single key can have mulitble headers (as defiend in rfc2616)
		"header":{
			"key1":[
				"val1",
				"val2",
				"val3"
			],
			"x-easydb-token":[
				"csdklmwerf8ßwji02kopwfjko2"
			]
		},
		//The body we want to assert on
        "body": {
            "objecttypes": [
                "pictures"
            ]
        }
    },
	// Store parts of the repsonse into the datastore
    "store_response_qjson": {
        "eas_id": "body.0.eas._id"
	},
	//Delay the request by x msec
	 "delay_ms":5000,
    //With the poll we can make the testing tool redo the request to wait for certain events (Only the timeout_msec is required)
    // timeout_ms:* If this timeout is done, no new redo will be started
    //  -1: No timeout - run endless
    // break_response: [Array] [Logical OR] If one of this responses occures, the tool fails the test and tells it found a break repsponse
    // collect_response:  [Array] [Logical AND] If this is set, the tool will check if all reponses occure in the response (even in different poll runs)
    "timeout_ms":5000,
    "break_response":["@break_response.json"],
    "collect_response":["@continue_response_pending.json","@continue_response_processing.json"]
}

```

## Datastore

The datastore is a storage for arbitrary data. It can be set directly or set using values received from a response. It has two parts:

* Custom storage with custom key
* Sequential response store per test suite (one manifest)

The custom storage is persistent throughout the **apitest** run, so all requirements, all manifests, all tests. Sequential storage is cleared at the start of each manifest.

The custom store uses a **string** as index and can store any type of data.

**Array**: If an key ends in `[]`, the value is assumed to be an Array, and is appended. If no Array exists, an array is created.

**Map**: If an key ends in `[key]`, the value is assumed to be an map, and writes the data into the map at that key. If no map exists, an map is created.

```django
{
  "store": {
    "eas_ids[]": 15,
    "mapStorage[keyIWantToStore]": "value"
  }
}
```

This example would create an Array in index **eas_ids** and append **15** to it.

Arrays are useful using the Go-Template **range** function.

### Set Data in Custom Store

To set data in custom store, you can use 4 methods:

* Use `store`on the **manifest.json** top level, the data is set before the session authentication (if any)
* Use `store_response_qjson`in `authentication.store_response_qjson`
* Use `store`on the **test** level, the data is set before **request** and **response** are evaluated
* Use `store_respsonse_qjson`on the test level, the data is set after each **response** 

All methods use a Map as value, the keys of the map are **string**, the values can be anything. If the key (or **index**) ends in `[]`and Array is created if the key does not yet exists, or the value is appended to the Array if it does exist.

The methods `store_respsonse_qjson`take only **string** as value. This qjson-string is used to parse the current response using the **qjson** feature. The return value from the qjson call is then stored in the datastore.

### Get Data from Custom Store

The data from the custom store is retrieved using the `datastore <key>`Template function. `key`must be used in any store method before it is requested. If the key is unset, the datastore function returns an empty **string**. 

### Get Data from Sequential Store

To get the data from the sequential store an integer number has to be given to the datastore function as **string**. So `datastore "0"`would be a valid request. This would return the response from first test of the current manifest. `datastore "-1"`returns the last response from the current manifest. `datastore "-2"`returns second to last from the current manifest. If the index is wrong the function returns an error.

## Use control structures

We support certain control structures in the **response definition**. You can use this control structures when ever you 
are able to set keys in the json (so you have to be inside a object).
Some of them also need a value and some don't. For those which don't need a value you can just setup the control structure
without a second key with some weird value. When you give a value the tool always tries to deep check if that value is
correct and present in the actual reponse. So be aware of this behavior as it could interfere with your intended test 
behavior.

### Define a control structure
In the example we use the jsonObject `test` and define some control structures on it. A control structure uses the key it 
is attached to plus `:control`. So for our case it would be `test:control`. The tool gets that this two keys `test` and
`test:control` are in relationship with each other.

```json
{
  "test":{
    "hallo":2,
    "hello":3
  },
  "test:control":{
    "is_object":true,
    "no_extra":true
  }
}
```


### Available controls

Their are several controls available. The first two `no_extra` and `order_matters` always need their responding real key
and value to function as intended. The others can be used without a real key.

Default behavior for all keys is `=false`. So you only have to set them if you want to explicit use them as `true`

#### no_extra
This commands defines an exact match, if it is set, their are no more fields allowed in the response as defined in the 
testcase.

`no_extra` is available for objects and arrays

The following response would **fail**  as their are to many fields in the actual response

##### expected response defined with no_extra

```json
{
  "body":{
    "testObject":{
      "a":"z",
      "b":"y"
    },
    "testObject:control":{
      "no_extra":true
    }
  }
}
```
##### actual response

```json
{
  "body":{
    "testObject":{
      "a":"z",
      "b":"y",
      "c":"to much, so we fail"
    }
  }
}
```

#### order_matters
This commands defines that the order in an array should be checked.

`order_matters` is available **only for arrays**

E.g. the following response would **fail**  as the order in the actual response is wrong

##### expected response defined with order_matters

```json
{
  "body":{
    "testArray":[
      "a",
      "b",
      "c"
    ],
    "testArray:control":{
      "order_matters":true
    }
  }
}
```
##### actual response
```json
{
  "body":{
    "testArray":[
      "c",
      "b",
      "a"
    ]
  }
}
```


#### must_exist
Check if a certain value does exist in the reponse (no matter what its content is)

`must_exist` is available for all types. 

This control can be used without a "real" key. So only the `:control` key is present.

E.g. the following response would **fail**  as `"iShouldExists"` is  **not** in the actual response


##### expected response defined with must_exist

```json
{
  "body":{
    "iShouldExists:control":{
      "must_exist":true
    }
  }
}
```


#### element_count
Check if the size of an array equals the element_count

`element_count` is available only for arrays

This control can be used without a "real" key. So only the `:control` key is present.

E.g. the following response would **fail**  as `"count"` is has the wrong length


##### expected response defined with must_exist

```json
{
  "body":{
    "count:control":{
      "element_count":2
    }
  }
}
```

##### actual response

```json
{
  "body":{
    "count":[
      1,
      2,
      3
    ]
   }
}
```

#### element_no_extra
Passes the no extra to the underlying structure in an array

`must_exist` is available only for arrays

This control can be used without a "real" key. So only the `:control` key is present.

E.g. the following response would **fail**  as `"hasExtra"` is has extras


##### expected response defined with must_exist

```json
{
  "body":{
    "count":[
      {
       "fine": true,
      }
    ],
    "count:control":{
      "element_no_extra":true
    }
  }
}
```

##### actual response

```json
{
  "body":{
    "count":[
      {
       "fine": true,
       "extra": "shouldNotBeHere"
      }
    ]
   }
}
```

#### must_not_exist
Check if a certain value does not exist in the reponse

`must_not_exist` is available for all types. 

This control can be used without a "real" key. So only the `:control` key is present.

E.g. the following response would **fail**  as `"iShouldNotExists"` is in the actual response

##### expected response defined with must_exist

```json
{
  "body":{
    "iShouldNotExists:control":{
      "must_not_exist":true
    }
  }
}
```

##### actual response

```json
{
  "body":{
    "iShouldNotExists":"i exist, hahahah"
   }
}
```

#### Type checkers
With `is_string`, `is_bool`, `is_object`, `is_array` and `is_number` you can check if your field has a certain type

The type checkers are available for all types. It implicit also checks `must_exist` for the value as there is no sense in 
type checking a value that does not exist.

This control can be used without a "real" key. So only the `:control` key is present.

E.g. the following response would **fail**  as `"beANumber"` is no number in the actual response

##### expected response defined with is_number

```json
{
  "body":{
    "testNumber:control":{
      "is_number":true
    }
  }
}
```

##### actual response

```json
{
  "body":{
    "testNumber":false
    }
}
```

## Use external file

In the request and response part of the single testcase you also can load the content from an external file
This is exspecially helpfull for keeping the manifest file simpler/smaller and keep a better overview. **On top: You can use so called template functions in the external file.** (We will dig deeper into the template functions later)

A single test could look as simple as following:

```json
{
	"name": "Test loading request & response from external file",
	"request": "@path/to/requestFile.json",
	"response": "@path/to/responseFile.json"
}
```

**Important: The paths to the external files start with a '@' and are relative to the location of the manifest.json or
can be web urls e.g. https://programmfabrik.de/testfile.json**

The content of the request and response file are execatly the same as if you would place the json code inline:
**Request:**

```json
{
    "body": {
        "animal": "dog",
        "flower": "rose"
    },
    "body_type": "urlencoded",
    "endpoint": "suggest",
    "header": {
        "header1": "value",
        "header2": "value"
    },
    "method": "GET",
    "query_params": {
        "number": 2,
        "token": "testtoken"
    }
}
```

**Response:**

```json
{
    "body": {
        "objecttypes": [
            "pictures"
        ],
        "query": ">>>[0-9]*<<<"
    },
    "header": {
        "key1": [
            "val1",
            "val2",
            "val3"
        ],
        "x-easydb-token": [
            "csdklmwerf8\u00dfwji02kopwfjko2"
        ]
    },
    "statuscode": 200
}
```

# Template functions
As described before, if you use an external file you can make use of so called template functions. What they are and how they work for the apitesting tool is described in the following part.

Template Functions are invoked using the tags `{{ }}` and upon returning substitutes the function call with
its result. We use the golang "text/template" package so all functions provided there are also supported here.
For a reference see [https://golang.org/pkg/text/template/]

```sequence
manifest.json->external file: load external file
external file->another file: render template with file parameter "hello"
another file->external file: return rendered template "hello world"
external file->manifest.json: return rendered template
```

#### Example
Assume that the template function `myfunc`, given the arguments `1 "foo"`, returns
`"bar"`. The call `{{ myfunc 1 "foo" }}` would translate to `bar`.
Consequently, rendering `Lets meet at the {{ myfunc 1 "foo" }}` results in an invitation to the `bar`.

We provide the following functions:

### file "relative/path/" [param1] ... [param4]
```
Helper function to load contents of a file; if this file contains templates; it will render these templates
with the up to 4 parameters provided in the call
@param1-4: string; can be accessed from the loaded file via {{ .Param1-4 }}; see example below
```
Loads the file with the relative path ( to the file this template function is invoked in ) "relative/path" or a weburl
e.g. https://docs.google.com/test/tmpl.txt
The loaded file will be rendered with the (up to) 4 provided parameters, that can be accessed using .Param1-4.
#### Example

Content of file at `some/path/example.tmpl`:
`{{ load_file "../target.tmpl" "hello" }}`

Content of file at `some/target.tmpl`:
`{{ .Param1 }} world`

Rendering `example.tmpl` will result in `hello world`

### rows_to_map "keyColumn" "valueColumn" [input]

Generates a key-value map from your input rows.

Assume you have the following structure in your sheet:

| column_a | column_b | column_c |
| -------- | -------- | -------- |
| row1a    | row1b    | row1c    |
| row2a    | row2b    | row2c    |

If you parse this now to CSV and then load it via `file_csv` you get the following JSON structure:

```go
[
    {
        column_a: "row1a",
        column_b: "row1b",
        column_c: "row1c",
    },
    {
        column_a: "row2a",
        column_b: "row2b",
        column_c: 22,
    }
]
```

For mapping now certain values to a map you can use ` rows_to_map "column_a" "column_c" `  and the output will be a map with the following content:

```go
{
    "row1a":"row1c",
    "row2a":22,
}
```

### group_rows "groupColumn" [rows]

Generates an Array of rows from input rows. The **groupColumn** needs to be set to a column which will be used for grouping the rows into the Array.

The column needs to:

* be an **int64** column
* use integers between 0 and 999

The Array will group all rows with identical values in the **groupColumn**.

#### Example

The CSV can look at follows, use **file_csv** to read it and pipe into **group_rows**

| batch | reference | title |
| -------- | -------- | -------- |
| int64 | string |  string |
| 1    | ref1a    | title1a |
| 1    | ref1b | title1b |
| 4    | ref4 | title4  |
| 3    | ref3   | titlte2  |

Produces this output (presented as **json** for better readability:

```json
[
    [
        {
            "batch": 1,
            "reference": "ref1a",
            "title": "title1a"
        },
        {
            "batch": 1,
            "reference": "ref1b",
            "title": "title1b"
        }
    ]
    ,
    [
        {
            "batch": 3,
            "reference": "ref3",
            "title": "title3"
        }
    ]
    ,
    [
        {
            "batch": 4,
            "reference": "ref4",
            "title": "title4"
        }
    ]
]
```



#### Template Example

With the parameters `keyColumn` and `valueColumn` you can select the two columns you want to use for map. (Only two are supported)

The `keyColumn`  **must** be of the type string, as it functions as map index (which is of type string)


```django
{{ unmarshal "[{\"column_a\": \"row1a\",\"column_b\": \"row1b\",\"column_c\": \"row1c\"},{\"column_a\": \"row2a\",\"column_b\": \"row2b\",\"column_c\": \"row2c\"}]" | rows_to_map "column_a" "column_c"  | marshal  }}
```

Rendering that will give you :
```django
{
    "row1a":"row1c",
    "row2a":"row2c"
}
```

#### Behavior in corner cases

##### No keyColumn given

The function returns an empty map

For `rows_to_map`:

```go
{}
```

##### No valueColumn given

The complete row gets mapped

For `rows_to_map "column_a"`:

```go
{
    "row1a":{
        column_a: "row1a",
        column_b: "row1b",
        column_c: "row1c",
    },
    "row2a":{
        column_a: "row2a",
        column_b: "row2b",
        column_c: "row2c",
    }
}
```

##### A row does not contain a key column

The row does get skipped

**Input:**

```go
[
    {
        column_a: "row1a",
        column_b: "row1b",
        column_c: "row1c",
    },
    {
        column_b: "row2b",
        column_c: "row2c",
    }
    {
        column_a: "row3a",
        column_b: "row3b",
        column_c: "row3c",
    }
]
```

For `rows_to_map "column_a" "column_c" `:

```go
{
    "row1a":"row1c",
    "row3a":"row3c",
}
```

##### A row does not contain a value column

The value will be set to `""` (empty string)

**Input:**

```go
[
    {
        column_a: "row1a",
        column_b: "row1b",
        column_c: "row1c",
    },
    {
        column_a: "row2a",
        column_b: "row2b",
    }
    {
        column_a: "row3a",
        column_b: "row3b",
        column_c: "row3c",
    }
]
```

For `rows_to_map "column_a" "column_c" `:

```go
{
	"row1a":"row1c",
    "row2a":"",
    "row3a":"row3c",
}
```

### datastore \[key\]

Helper function to query the datastore; used most of the time in conjunction with `qjson`.

The `key`can be an int, or int64 accessing the store of previous responses. The responses are accessed in the order received. Using a negative value access the store from the back, so a value of **-2** would access the second to last response struct.

This function returns a string, if the `key`does not exists, an empty string is returned.

If the `key` is a string, the datastore is accessed directly, allowing access to custom set values using `store` or `store_response_qjson`parameters.

The datastore stores all responses in a list. We can retrieve the response (as a json string) by using this
template function. `{{ datastore 0  }}` will render to

```json
{
    "statuscode": 200,
    "header": {
        "foo": [
		"bar", 
		"baz"
	]
    },
    "body": "..."
}

```

This function is intended to be used with the `qjson` template function.

The key `-` has a special meaning, it returns the entire custom datastore (not the sequentially stored responses)

### qjson \[path\] \[json\]

```
Helper function to extract fields from the 'json'.
@path: string; a description of the location of the field to extract. For array access use integers; for object
access use keys. Example: 'body.1.field'; see below for more details
@json_string: string; a valid json blob to be queried; can supplied via pipes from 'datastore idx'
@result: the content of the json blob at the specified path
```

#### Example
The call 

```django
{{ qjson "foo.1.bar" "{\"foo": [{\"bar\": \"baz\"}, 42]}" }}
```

 would return `baz`.

As an example with pipes, the call

```django
{{ datastore idx | qjson "header.foo.1" }}
```

 would return`bar` given the response above.

See [gjson](https://github.com/tidwall/gjson/blob/master/README.md)



### file_csv [path] [delimiter]

```
Helper function to load a csv file. 
@path: string; a path to the csv file that should be loaded. The path is either relative to the manifest or a weburl
@delimiter: rune; The delimiter that is used in the given csv e.g. ','
@result: the content of the csv as json array so we can work on this data with qjson
```

The CSV **must** have a certain structur. If the structure of the given CSV differs, the apitest tool will fail with a
error

- In the first row must be the names of the fields
- In the seconds row must be the types of the fields

**Valid types**
- int64
- string
- float64
- bool
- int64,array
- string,array
- float64,array
- bool,array
- json

#### Example

Content of file at `some/path/example.csv`:
```csv
id,name
int64,string
1,simon
2,martin
```

The call

```django
{{ csv "some/path/example.csv" ','}}
```

would result in

```go
[map[id:1 name:simon] map[id:2 name:martin]]
```

As an example with pipes, the call 

```django
{{ csv "some/path/example.csv" ',' | marshal | qjson "1.name" }}
```

 would result in `martin` given the response above.

#### Cornercases

There are some corner cases that trigger a certain behavior you should keep in mind

##### No format for a column given

The column gets skipped in every row

**Input**

```csv
id,name
int64,
1,simon
2,martin
```

**Result**

```go
[map[id:1] map[id:2]]
```



##### No name for a column given 

The column gets skipped in every row

**Input**

```csv
,name
int64,string
1,simon
2,martin
```

**Result**

```go
[map[name:simon] map[name:martin]]
```



##### Comment or empty line

If there is a comment marked with `#` , or a empty line that does not get rendered into the result

**Input**

```csv
id,name
int64,string
1,simon
2,martin
#3,philipp
4,roman
#5,markus


6,klaus
7,sebastian
```

**Result**

```go
[map[name:simon] map[name:martin] map[name:roman] map[name:klaus] map[name:sebastian]]
```



### slice \[parms...\]

Returns a slice with the given parameters as elements. Use this for **range** in templates.

### add \[a\] \[b\]

Returns the sum of `a`and `b`. `a`can be any numeric type or string. The function returns a numeric type, depending on the input. With `string`we return `int64`.

### unmarshal \[string\]

Returns a `util.GenericJson` Object (go: `interface{}`) of the unmarshalled  `JSON` string.

### marshal \[interface{}\]

Returns a `string` of the marshalled  `interface{}` object.

### str_escape [string]

Returns a `string`where all `"`are escaped to `\"`. This is useful in Strings which need to be concatenated.

### printf [interface{}...]

Just for reference, this is a Go Template [built-in](https://golang.org/pkg/text/template/#hdr-Functions).

### N [float64|int64|int]

Returns a slice of n 0-sized elements, suitable for ranging over.

Example how to range over 100 objects

```django
        "body":	[
            {{ range $idx, $v := N 100 }}
            ...
            {{ end }}
        ]
    }
```
