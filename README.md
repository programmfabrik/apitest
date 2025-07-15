# apitest

[![release](https://github.com/programmfabrik/apitest/actions/workflows/release.yml/badge.svg?branch=production)](https://github.com/programmfabrik/apitest/actions/workflows/release.yml)
[![unit-tests](https://github.com/programmfabrik/apitest/actions/workflows/unit-tests.yml/badge.svg)](https://github.com/programmfabrik/apitest/actions/workflows/unit-tests.yml)
[![Twitter](https://img.shields.io/twitter/follow/programmfabrik.svg?label=Follow&style=social)](https://twitter.com/programmfabrik)
[![GoReportCard](https://goreportcard.com/badge/github.com/programmfabrik/apitest)](https://goreportcard.com/report/github.com/programmfabrik/apitest)


The apitest tool helps you to build automated apitests that can be run after every build to ensure a constant product quality.

A single testcase is also a perfect definition of an occuring problem and helps the developers to fix your issues faster!

# Configuration file

For configuring the apitest tool, add the following section to your `apitest.yml` configuration file.

The report parameters of this config can be overwritten via a command line flag. So you should set your intended standar

```yaml
apitest:
  # The base url to the api you want to fire the apitests against
  # Important: don’t add a trailing ‘/’
  server: "http://5.simon.pf-berlin.de/api/v1"

  log:
    # Configures minimal logs by default for all tests
    short: true

  # Configures the maschine report
  # For usage with jenkis or any other CI tool
  report:
    # Filename of the report file
    # The file gets saved in the same directory of the apitest binary
    file: "apitest_report.xml"
    # Format of the report.
    # Supported formats: json, junit or stats
    format: "json.junit"

  # initial values for the datastore, parsed as map[string]interface{}
  store:
    email.server: smtp.google.com

  # Map of client-config for oAuth clients
  oauth2_client:
    # oauth Client ID
    my_client:
      # endpoints on the oauth server
      endpoint:
        auth_url: "http://auth.myserver.de/oauth/auth"
        token_url: "http://auth.myserver.de/oauth/token"
      # oauth Client secret
      secret: "foobar"
      # redirect, usually on client side
      redirect_url: "http://myfancyapp.de/auth/receive-fancy-token"
```

The YAML config is optional. All config values can be overwritten/set by command line parameters: see [Overwrite config parameters](#overwrite-config-parameters)

## Command line interface

You start the apitest tool with the following command

```bash
./apitest
```

This starts the command with the following default settings:

- Runs all tests that are in the current directory, or in any of its subdirectories
- Logs to console
- Writes the machine log, to the given file in the `apitest.yml`
- Logs only the request & responses if a test fails

### Configure which tests should be run

| Parameter                   |                    | Description                                                                                                                                                                                                                                   |
| ---                         | ---                | ---                                                                                                                                                                                                                                           |
| `--directory testDirectory` | `-d testDirectory` | Defines which directory should be used for running the tests in it. The tool walks recursively trough all subdirectories and runs alls tests that have a `manifest.json` file in alphabetical order of the folder names. (Depth-First-Search) |
| `--single manifest.json`    | `-s manifest.json` | Run only a single test. The path needs to point directly to the manifest file. (Not the directory containing it)                                                                                                                              |

### Stop on fail

| Parameter      | Description                                               |
| ---            | ---                                                       |
| `stop-on-fail` | Stop execution of later test suites if a test suite fails |

### Keep running

- `keep-running`: Wait for a keyboard interrupt after each test suite invocation.
  This can be useful for keeping the HTTP / SMTP server for manual inspection.

### Configure logging

Per default request and response of a request will be logged on test failure. If you want to see more information you
can configure the tool with additional log flags

| Parameter          |      | Description                                                                |
| ---                | ---  | ---                                                                        |
| `--log-network`    | `-n` | Log all network traffic                                                    |
| `--log-datastore`  |      | Logs datastore operations into datastore                                   |
| `--log-verbose`    | `-v` | `--log-network`, `--log-datastore` and a few additional trace informations |
| `--log-short`      |      | Show minimal logs, useful for CI chains                                    |
| `--log-timestamp`  | `-t` | Log the timestamp of the log message into the console                      |
| `--curl-bash`      |      | Log the request as curl command                                            |
| `--limit-request`  |      | Limit the lines of request log output. Configure limit in `apitest.yml`    |
| `--limit-response` |      | Limit the lines of response log output. Configure limit in `apitest.yml`   |

You can also set the log verbosity per single testcase. The greater verbosity wins.

### Overwrite config parameters

| Parameter                      |                    | Description                                                                           |
| ---                            | ---                | ---                                                                                   |
| `--config newConfigFile`       | `-c newConfigFile` | Overwrites the path of the config file (default `./apitest.yml`) with `newConfigFile` |
| `--server URL`                 |                    | Overwrites base url to the api                                                        |
| `--report-file newReportFile`  |                    | Overwrites the report file name from the `apitest.yml` config with `newReportFile`    |
| `--report-format junit`        |                    | Overwrites the report format from the `apitest.yml` config with `junit`               |
| `--replace-host host`          |                    | Overwrites built-in server host in template function `replace_host`                   |

### Additional parameters

| Parameter                       | Description                                                                           |
| ---                             | ---                                                                                   |
| `--report-format-stats-group 3` | Sets the number of groups for manifests distrubution when using report format `stats` |

### Examples

- Run all tests in the directory **apitests** display **all server communication** and save the maschine report as **junit** for later parsing it with *jenkins*

```bash
./apitest --directory apitests --verbosity 2 --report-format junit
```

- Only run a single test **apitests/test1/manifest.json** with **no console output** and save the maschine report to the standard file defined in the `apitest.yml`

```bash
./apitest --single apitests/test1/manifest.json --log-console-enable false
```

- Run all tests in the directory **apitests** with **http server host replacement** for those templates using **replace_host** template function
```bash
./apitest -d apitests --replace-host my.fancy.host
```

# Manifest

Manifest is loaded as **template**, so you can use variables, Go **range** and **if** and others.

```jsonc
{
    // Testname
    // Should include the ticket number if the test is based on a ticket
    "name": "ticket_48565",

    // General info about the testuite.
    // Try to explain your problem indepth here.
    // So that someone who works on the test years from now knows what is happening
    "description": "search api tests for filename",

    // init store
    "store": {
        "custom": "data"
    },

    // Testsuites your want to run upfront (e.g. a setup).
    // Paths are relative to the current test manifest
    "require": [
        "setup_manifests/purge.yaml",
        "setup_manifests/config.yaml",
        "setup_manifests/upload_datamodel.yaml"
    ],

    // Array of single testcases. Add es much as you want.
    // They get executed in chronological order
    "tests": [
        // [SINGLE TESTCASES]: See below for more information
        // ...

        // We also support the external loading of a complete test:
        "@pathToTest.json",

        // By prefixing it with a number, the testtool runs that many instances of
        // the included test file in parallel to each other.
        // Only tests directly included by the manifest are allowed to run in parallel.
        "5@pathToTestsThatShouldRunInParallel.json"
    ]
}
```

## Testcase Definition

| **Key**                            | **Description** |
|------------------------------------|-----------------|
| `name`                             | Name to identify this single test. Is important for the log. Try to give an explaining name |
| `store`                            | Store custom values to the datastore |
| `http_server`                      | Optional temporary [HTTP Server](#http-server) |
| `smtp_server`                      | Optional temporary [SMTP Server](#smtp-server) |
| `log_network`                      | Log network only for this single test |
| `log_verbose`                      | Verbose logging only for this single test |
| `log_short`                        | Show or disable minimal logs for this test |
| `request.endpoint`                 | What endpoint we want to target. You find all possible endpoints in the api documentation |
| `request.server_url`               | The server url to connect can be set directly for a request, overwriting the configured server url |
| `request.method`                   | How the endpoint should be accessed. The api documentations tells your which methods are possible for an endpoint. All HTTP methods are possible |
| `request.no_redirect`              | If set to `true`, don't follow redirects |
| `request.query_params`             | Parameters that will be added to the url |
| `request.query_params_from_store`  | With this set a query parameter to the value of the datastore field |
| `request.header`                   | Additional headers that should be added to the request |
| `request.cookies`                  | Cookies can be added to the request |
| `request.header-x-test-set-cookie` | Special headers `X-Test-Set-Cookie` can be populated in the request (on per entry). Used in the built-in `http_server` |
| `request.header_from_store`        | With this you set a header to the value of the datastore field |
| `request.body`                     | All the content you want to send in the http body. Is a JSON object or array |
| `request.body_type`                | If the body should be marshaled in a special way, you can define this here. Possible: [`multipart`, `urlencoded`, `file`] |
| `request.body_file`                | If `body_type` is `file`, `body_file` points to the file to be sent as binary body |
| `response.statuscode`              | Expected http [status code](#statuscode). See api documentation for the endpoint to decide which code to expect |
| `response.header`                  | If you expect certain response headers, you can define them here. A single key can have multiple headers |
| `response.cookie`                  | Cookies will be under this key, in a map `name => cookie` |
| `response.format`                  | Optionally, the expected format of the response can be specified or [preprocessed](#preprocessing-responses) so that it can be converted into json and can be checked. Formats are: [`binary`](#binary-data-comparison), [`xml`](#xml-data-comparison), [`html`](#html-data-comparison), [`csv`](#csv-data-comparison), [`text`](#text-data-comparison) |
| `response.body`                    | The body we want to assert on |
| `store_response_gjson`             | Store parts of the response into the datastore |
| `store_response_gjson.sess_cookie` | Cookies are stored in `cookie` map |
| `wait_before_ms`                   | Pauses right before sending the test request `<n>` milliseconds |
| `wait_after_ms`                    | Pauses right after sending the test request `<n>` milliseconds |
| `delay_ms`                         | Delay the request by `<n>` milliseconds |
| `timeout_ms`                       | With this the testing tool will repeat the request to wait for certain events. The timeout is `<n>` milliseconds before the test fails |
| `break_response`                   | If one of this responses occurs, the tool fails the test and tells it found a break response |
| `collect_response`                 | The tool will check if all responses occur in the response (even in different poll runs) |
| `reverse_test_result`              | If set to true, the test case will consider its failure as a success, and the other way around |
| `continue_on_failure`              | Define if the test suite should continue even if this test fails. (default: false) |

The `response` definition is optional. If it is not included in the test case, a status code of `200` and no specific body is expected.

#### Statuscode

Expected http status code, if the response has another status code, the test case fails.

- If the `statuscode` key is not set, the default value `200` is used
- If `"statuscode": 0` is set, the status code is ignored and any status code from the response is accepted

### manifest.json

```jsonc
{
    // Name to identify this single test.
    "name": "Testname",

    // Store custom values to the datastore
    "store": {
        "key1": "value1",
        "key2": "value2"
    },

    // Optional temporary HTTP Server (see below)
    "http_server": {
        "addr": ":1234",
        "dir": ".",
        "testmode": false
    },

    // Optional temporary SMTP Server (see below)
    "smtp_server": {
        "addr": ":9025",
        "max_message_size": 1000000
    },

    // Specify a unique log behavior only for this single test.
    "log_network": true,
    "log_verbose": false,

    // Show or disable minimal logs for this test
    "log_short": false,

    // Defines what gets send to the server
    "request": {

        // What endpoint we want to target. You find all possible endpoints in the api documentation
        "endpoint": "suggest",

        // the server url to connect can be set directly for a request, overwriting the configured server url
        "server_url": "",

        // How the endpoint should be accessed.
        // The api documentations tells your which methods are possible for an endpoint.
        // All HTTP methods are possible.
        "method": "GET",

        // If set to true, don't follow redirects.
        "no_redirect": false,

        // Parameters that will be added to the url.
        // e.g. http:// 5.testing.pf-berlin.de/api/v1/session?token=testtoken&number=2 would be defined as follows
        "query_params": {
            "number": 2,
            "token": "testtoken"
        },

        // With query_params_from_store set a query parameter to the value of the datastore field
        "query_params_from_store": {
            "format": "formatFromDatastore",
            // If the datastore key starts with an ?, we do not throw an error if the key could not be found,
            // but just do not set the query param.
            // If the key "a" is not found it datastore, the query parameter test will not be set
            "test": "?a"
        },

        // Additional headers that should be added to the request
        "header": {
            "header1": "value",
            "header2": [
                "value1",
                "value2"
            ]
        },

        // Cookies can be added to the request
        "cookies": {
            // name of a cookie to be set
            "cookie1": {
                // A cookie can be get parsed from store if it was saved before
                // It will ignore the cookie if it is not set
                "value_from_store": "sess_cookie",
                // Or its values can be directly set, overriding the one from store, if defined
                "value": "value"
            },
            "cookie2": {
                "value_from_store": "ads_cookie",
            }
        },

        // Special headers `X-Test-Set-Cookie` can be populated in the request (on per entry)
        // It is used in the builting `http_server` to automatically set those cookies on response
        // So it is useful for mocking them for further testing
        "header-x-test-set-cookie": [
            {
                "name": "sess",
                "value": "myauthtoken"
            },
            {
                "name": "jwtoken",
                "value": "tokenized",
                "path": "/auth",
                "domain": "mydomain",
                "expires": "2021-11-10T10:00:00Z",
                "max_age": 86400,
                "secure": false,
                "http_only": true,
                "same_site": 1
            }
        ],

        // With header_from_store you set a header to the value of the datastore field
        // In this example we set the "Content-Type" header to the value "application/json"
        // "application/json" is stored as string in the datastore on index "contentType"
        "header_from_store": {
            "Content-Type": "contentType",
            // If the datastore key starts with an ?, wo do not throw an error if the key could not be found, but just
            // do not set the header. If the key "a" is not found it datastore, the header Range will not be set
            "Range": "?a"
        },

        // All the content you want to send in the http body. Is a JSON Object
        "body": {
            "flower": "rose",
            "animal": "dog"
        },

        // If the body should be marshaled in a special way, you can define this here. Is not a required attribute. Standart is to marshal the body as json. Possible: [multipart,urlencoded, file]
        "body_type": "urlencoded",

        // If body_type is file, "body_file" points to the file to be sent as binary body
        "body_file": "<path|url>"
    },

    // Define how the response should look like. Testtool checks against this response
    "response": {

        // Expected http status code. Default: 200. Use 0 to accept any statuscode
        "statuscode": 200,

        // If you expect certain response headers, you can define them here. A single key can have multiple headers (as defined in rfc2616)
        "header": {
            "key1": [
                "val1",
                "val2",
                "val3"
            ],
            // Headers sharing the same key are concatenated using ";", if the comparison value is a simple string,
            // thus "key1" can also be checked like this:
            "key1": "val1;val2;val3",
            // :control in header is always applied to the flat format
            "key1:control": {
                // see below, this is not applied against the array
            },
            "x-easydb-token": [
                "csdklmwerf8ßwji02kopwfjko2"
            ]
        },

        // Cookies will be under this key, in a map name => cookie
        "cookie": {
            "sess": {
                "name": "sess",
                "value": "myauthtoken"
            },
            "jwtoken": {
                "name": "jwtoken",
                "value": "tokenized",
                "path": "/auth",
                "domain": "mydomain",
                "expires": "2021-11-10T10:00:00Z",
                "max_age": 86400,
                "secure": false,
                "http_only": true,
                "same_site": 1
            }
        },

        // optionally, the expected format of the response can be specified so that it can be converted into json and can be checked
        "format": {
            "type": "csv",
            "csv": {
                "comma": ";"
            }
        },

        // The body we want to assert on
        "body": {
            "objecttypes": [
                "pictures"
            ]
        }
    },

    // Store parts of the response into the datastore
    "store_response_gjson": {
        "eas_id": "body.0.eas._id",
        // Cookies are stored in `cookie` map
        "sess_cookie": "cookie.sess"
    },

    // wait_before_ms pauses right before sending the test request <n> milliseconds
    "wait_before_ms": 0,

    // wait_after_ms pauses right before sending the test request <n> milliseconds
    "wait_after_ms": 0,

    // Delay the request by x msec
    "delay_ms": 5000,

    // With the poll we can make the testing tool redo the request to wait for certain events (Only timeout_ms is required)
    // timeout_ms:                             If this timeout is done, no new redo will be started (-1: No timeout - run endless)
    // break_response:   [Array] [Logical OR]  If one of this responses occures, the tool fails the test and tells it found a break repsponse
    // collect_response: [Array] [Logical AND] If this is set, the tool will check if all reponses occure in the response (even in different poll runs)
    "timeout_ms": 5000,
    "break_response": [
        "@break_response.json"
    ],
    "collect_response": [
        "@continue_response_pending.json",
        "@continue_response_processing.json"
    ],

    // If set to true, the test case will consider its failure as a success, and the other way around
    "reverse_test_result": false,

    // Define if the test suite should continue even if this test fails. (default: false)
    "continue_on_failure": true
}
```

## Override template delimiters

Go template delimiters can be redefined as part of a single line comment in any of these syntax:

```jsonc
// template-delims: <delim_left> <delim_right>
/* template-delims: <delim_left> <delim_right> */
```

Examples:

```jsonc
// template-delims: /* */
/* template-delims: // // */
// template-delims {{ }}
/* template-delims: {* *} */
```

All external tests/requests/responses inherit those delimiters if not overriden in their template.

## Remove template 'placeholders'

Go templates may break the proper JSONC format even when separators are comments. So we could use placeholders for filling missing parts then strip them.

```jsonc
// template-remove-tokens: <token> [<token>]*
/* template-remove-tokens: <token> [<token>] */
```

Example:

```jsonc
// template-delims: /* */
// template-remove-tokens: "delete_me"
{
    "prop": /* datastore "something" */"delete_me"
}
```

This would be an actual proper JSONC as per the `"delete_me"` string. However that one will be stripped before parsing the template, which would be just:

```jsonc
{
    "prop": /* datastore "something" */
}
```

Unlike with delimiters, external tests/requests/responses don't inherit those removals, and need to be specified per file.

## Run tests in parallel

The tool is able to run tests in parallel to themselves. You activate this mechanism by including an external test file with `N@pathtofile.json`, where `N` is the number of parallel "clones" you want to have of the included tests.

The included tests themselves are still run serially, only the entire set of tests will run in parallel for the specified number of replications.

This is useful e.g. for stress-testing an API.

Only tests directly included by a manifest are allowed to run in parallel.

Using `"0@file.json"` will not run that specific test.

```json
{
    "name": "Example Manifest",
    "tests": [
        "@setup.json",
        "123@foo.json",
        "@cleanup.json"
    ]
}
```

## Binary data comparison

The tool is able to do a comparison with a binary file. Here we take a MD5 hash of the file and and then later compare that hash.

For comparing a binary file, simply point the response to the binary file:

```js
{
    "name": "Binary Comparison",
    "request": {
        "endpoint": "suggest",
        "method": "GET"
    },
    "response": {
        "format": {
            "type": "binary"
        },
        "body": {
            // Path to binary file with @
            "md5sum": {{ md5sum "@simple.bin" || marshal }}
        }
    }
}
```

The format must be specified as `"type": "binary"`

## XML Data comparison

If the response format is specified as `"type": "xml"` or `"type": "xml2"`, we internally marshal that XML into json using [github.com/clbanning/mxj](https://github.com/clbanning/mxj).

The format `"xml"` uses `NewMapXmlSeq()`, whereas the format `"xml2"` uses `NewMapXml()`, which provides a simpler json format.

See also template [`file_xml2json`](#file_xml2json-path).

On that json you can work as you are used to with the json syntax. For seeing how the converted json looks you can use the `--log-verbose` command line flag

## HTML Data comparison

If the response format is specified as `"type": "html"`, we internally marshal that HTML into json using [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery).

This marshalling is less strict than for [XHTML](#file_xhtml2json-path). For example it will not raise errors for unclosed tags like `<p>` or `<hr>`, as well as Javascript code inside the HTML code. But it is possible that unclosed tags are missing in the resulting JSON if the tokenizer can not find a matching closing tag.

See also template [`file_html2json`](#file_html2json-path).

## XHTML Data comparison

If the response format is specified as `"type": "xhtml"`, we internally marshal that XHTML into json using [github.com/clbanning/mxj](https://github.com/clbanning/mxj).

The XHTML code in the response must comply to the [XHTML standard](https://www.w3.org/TR/xhtml1/), which means it must be parsable as XML.

See also template [`file_xhtml2json`](#file_xhtml2json-path).

## CSV Data comparison

If the response format is specified as `"type": "csv"`, we internally marshal that CSV into json.

You can also specify the delimiter (`comma`) for the CSV format (default: `,`):

```json
{
    "name": "CSV comparison",
    "request": {
        "endpoint": "export/1/files/file.csv",
        "method": "GET"
    },
    "response": {
        "format": {
            "type": "csv",
            "csv": {
                "comma": ";"
            }
        },
        "body": {}
    }
}
```

## Text Data comparison

If the response format is specified as `"type": "text"`, the content of the response is returned in a JSON object.

The object contains the following keys:

* `text`: the response text without any changes
* `text_trimmed`: the response text, leading and trailing whitespaces have been trimmed
* `lines`: the response text, split into lines
* `float64`: if the text can be parsed into a float64 value, the numerical value is returned, else `null`
* `int64`: if the text can be parsed into a int64 value, the numerical value is returned, else `null`

Assume we get the content of this text file in the response, including whitespaces and newlines:

```
    42.35

```

```json
{
    "name": "Text comparison",
    "request": {
        "endpoint": "export/1/files/number.txt",
        "method": "GET"
    },
    "response": {
        "text": "    42.35\n",
        "text_trimmed": "42.35",
        "lines": [
            "    42"
        ],
        "float64": 42.35,
        "int64": 42,
    }
}
```

## Preprocessing responses

Responses in arbitrary formats can be preprocessed by calling any command line tool that can produce JSON, XML, CSV or binary output. In combination with the `type` parameter in `format`, non-JSON output can be [formatted after preprocessing](#reading-metadata-from-a-file-xml-format). If the result is already in JSON format, it can be [checked directly](#reading-metadata-from-a-file-json-format).

The response body is piped to the `stdin` of the tool and the result is read from `stdout`. The result of the command is then used as the actual response and is checked. The response is formatted as JSON string if it is not parsable as JSON.

To define a preprocessing for a response, add a `format` object that defines the `pre_process` to the response definition:

```json
{
    "response": {
        "format": {
            "pre_process": {
                "cmd": {
                    "name": "...",
                    "args": [ ],
                    "output": "stdout"
                }
            }
        }
    }
}
```

|                                 |                          |                                                                                                            |
| ---                             | ---                      | ---                                                                                                        |
| `format.pre_process.cmd.name`   | (string, mandatory)      | name of the command line tool                                                                              |
| `format.pre_process.cmd.args`   | (string array, optional) | list of command line parameters                                                                            |
| `format.pre_process.cmd.output` | (string, optional)       | what command output to use as result response, it can be one of `exitcode`, `stderr` or `stdout` (default) |

### Examples

#### Basic usage: pipe response without changes

This basic example shows how to use the `pre_process` feature. The response is piped through `cat` which returns the input without any changes. This command takes no arguments.

```json
{
    "response": {
        "format": {
            "pre_process": {
                "cmd": {
                    "name": "cat"
                }
            }
        }
    }
}
```

#### Advanced usage: compare binary image with local one

This example shows how to use the `pre_process` feature with `stderr` output. The response is the metric result of running `imagemagick compare` which returns the absolute error between 2 images given a threshold (0 if identical, number of different pixels otherwise). The arguments are the piped binary from the response and the image to compare against (local file using `file_path` template function) .

```js
{
    "response": {
        "format": {
            "pre_process": {
                "cmd": {
                    "name": "compare",
                    "args": [
                        "-metric",
                        "AE",
                        "-fuzz",
                        "2%",
                        "-",
                        {{ file_path "other/file.jpg" | marshal }},
                        "/dev/null"
                    ],
                    "output": "stderr"
                }
            }
        },
        "body": 0
    }
}
```

* `format.pre_process`:
    * Command: `compare -metric AE -fuzz 2% - /path/to/other/file.jpg /dev/null`
    * Parameters:
        * `-metric AE`: metric to use for image comparison
        * `-fuzz 2%`: threshold for allowed pixel color difference
        * `-`: read first image from `stdin` instead loading a saved file
        * `/path/to/other/file.jpg `: read second image from local path (result from template function above)
        * `/dev/null`: discard stdout (it contains a binary we don't want, we use stderr output)

#### Reading metadata from a file (JSON Format)

To check the file metadata of a file that is directly downloaded as a binary file using the `eas/download` API, use `exiftool` to read the file and output the metadata in JSON format.

If there is a file with the asset ID `1`, and the apitest needs to check that the MIME type is `image/jpeg`, create the following test case:

```json
{
    "request": {
        "endpoint": "eas/download/1/original",
        "method": "GET"
    },
    "response": {
        "format": {
            "pre_process": {
                "cmd": {
                    "name": "exiftool",
                    "args": [
                        "-j",
                        "-g",
                        "-"
                    ]
                }
            }
        },
        "body": [
            {
                "File": {
                    "MIMEType": "image/jpeg"
                }
            }
        ]
    }
}
```

* `format.pre_process`:
    * Command: `exiftool -j -g -`
    * Parameters:
        * `-j`: output in JSON format
        * `-g`: group output by tag class
        * `-`: read from `stdin` instead loading a saved file

#### Reading metadata from a file (XML Format)

This example shows the combination of `pre_process` and `type`. Instead of calling `exiftool` with JSON output, it can also be used with XML output, which then will be formatted to JSON by the apitest tool.

```json
{
    "request": {
        "endpoint": "eas/download/1/original",
        "method": "GET"
    },
    "response": {
        "format": {
            "pre_process": {
                "cmd": {
                    "name": "exiftool",
                    "args": [
                        "-X",
                        "-"
                    ]
                }
            },
            "type": "xml"
        },
        "body": [
            {
                "File": {
                    "MIMEType": "image/jpeg"
                }
            }
        ]
    }
}
```

* `format.pre_process`:
    * Command: `exiftool -X -`
    * Parameters:
        * `-X`: output in XML format
        * `-`: read from `stdin` instead loading a saved file
* `format.type`:
    * `xml`: convert the output of `exiftool`, which is expected to be in XML format, into JSON

### Error handling

If there is any error during the call of the command line tool, the error is formatted as a JSON object and returned instead of the expected response:

```json
{
    "command": "cat --INVALID",
    "error": "exit status 1",
    "exit_code": 1,
    "stderr": "cat: unrecognized option '--INVALID'\nTry 'cat --help' for more information.\n"
}
```

|             |                                                                         |
| ---         | ---                                                                     |
| `command`   | the command that was executed (consisting of `cmd.name` and `cmd.args`) |
| `error`     | error message (message of internal `exec.ExitError`)                    |
| `exit_code` | integer value of the exit code                                          |
| `stderr`    | additional error information from `stderr` of the command line tool     |

If such an error is expected as a result, this formatted error message can be checked as the response.

# Datastore

The datastore is a storage for arbitrary data. It can be set directly or set using values received from a response. It has two parts:

* Custom storage with custom key
* Sequential response store per test suite (one manifest)

The custom storage is persistent throughout the **apitest** run, so all requirements, all manifests, all tests. Sequential storage is cleared at the start of each manifest.

The custom store uses a **string** as index and can store any type of data.

**Array**: If an key ends in `[]`, the value is assumed to be an Array, and is appended. If no Array exists, an array is created.

**Map**: If an key ends in `[key]`, the value is assumed to be an map, and writes the data into the map at that key. If no map exists, an map is created.

```json
{
    "store": {
        "eas_ids[]": 15,
        "mapStorage[keyIWantToStore]": "value"
    }
}
```

This example would create an Array in index **eas_ids** and append **15** to it.

Arrays are useful using the Go-Template **range** function.

## Set Data in Custom Store

To set data in custom store, you can use 4 methods:

|                                                                 |                                                                                                                                                                                                                                                                                                                           |
| ---                                                             | ---                                                                                                                                                                                                                                                                                                                       |
| `store` on the `manifest.json` top level                        | the data is set before the session authentication (if any)                                                                                                                                                                                                                                                                |
| `store_response_gjson` in `authentication.store_response_gjson` |                                                                                                                                                                                                                                                                                                                           |
| `store` on the **test** level                                   | the data is set before **request** and **response** are evaluated                                                                                                                                                                                                                                                         |
| `store_response_gjson` on the test level                        | the data is set after each **response** (If you want the datestore to delete the current entry if no new one could be found with `gjson`. Just prepend the `gjson` key with a `!`. E.g. `"eventId":"!body.0._id"` will delete the `eventId` entry from the datastore if `body.0._id` could not be found in the response json) |

All methods use a Map as value, the keys of the map are **string**, the values can be anything. If the key (or **index**) ends in `[]` and Array is created if the key does not yet exist, or the value is appended to the Array if it does exist.

The method `store_response_gjson` takes only **string** as value. This `gjson`-string is used to parse the current response using the [`gjson`](#gjson-path-json) feature. The return value from the `gjson` call is then stored in the datastore.

## Get Data from Custom Store

The data from the custom store is retrieved using the `datastore <key>`Template function. `key`must be used in any store method before it is requested. If the key is unset, the datastore function returns an empty **string**. Use the special key `-` to return the entire datastore.

Slices allow the backwards index access. If you have a slice of length 3 and access it at index `-1` you get the last element in the slice (original index `2`)

If you access an invalid index for datastore `map[index]` or `slice[]` you get an empty string. No error is thrown.

## Get Data from Sequential Store

To get the data from the sequential store an integer number has to be given to the datastore function as **string**. So `datastore "0"` would be a valid request. This would return the response from first test of the current manifest. `datastore "-1"` returns the last response from the current manifest. `datastore "-2"` returns second to last from the current manifest. If the index is wrong the function returns an error.

The sequential store stores the body and header of all responses. Use `gjson` to access values in the responses. See template functions [`datastore`](#datastore-key) and [`gjson`](#gjson-path-json).

When using relative indices (negative indices), use the same index to get values from the datastore to use in the request and response definition. Especially, for evaluating the current response, it has not yet been stored. So, `datastore "-1"` will still return the last response in the datastore. The current response will be appended after it was evaluated, and then will be returned with `datastore "-1"`.

# Use control structures

We support certain control structures in the **response definition**. You can use this control structures when ever you are able to set keys in the json (so you have to be inside a object).

Some of them also need a value and some don't. For those which don't need a value you can just setup the control structure without a second key with some weird value. When you give a value the tool always tries to deep check if that value is correct and present in the actual reponse. So be aware of this behavior as it could interfere with your intended test behavior.

## Define a control structure

In the example we use the jsonObject `test` and define some control structures on it. A control structure uses the key it is attached to plus `:control`. So for our case it would be `test:control`. The tool gets that this two keys `test` and `test:control` are in relationship with each other.

```json
{
    "test": {
        "hallo": 2,
        "hello": 3
    },
    "test:control": {
        "is_object": true,
        "no_extra": true
    }
}
```

### `body:control`

All controls, which are defined below, can also be applied to the complete response body itself by setting `body:control`. The control check functions work the same as on any other key. This can be combined with other controls inside the body.

## Available controls

There are several controls available. The first two `no_extra` and `order_matters` always need their responding real key and value to function as intended. The others can be used without a real key.

Default behavior for all keys is `false`. So you only have to set them if you want to explicit use them as `true`.

### `no_extra`

This command defines an exact match. If it is set, there are no more fields allowed in the response as defined in the testcase.

`no_extra` is available for objects and arrays.

The following response would **fail** as there are to many fields in the actual response:

#### expected response defined with `no_extra`

```json
{
    "body": {
        "testObject": {
            "a": "z",
            "b": "y"
        },
        "testObject:control": {
            "no_extra": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testObject": {
            "a": "z",
            "b": "y",
            "c": "to much, so we fail"
        }
    }
}
```

### `order_matters`

This command defines that the order in an array should be checked.

`order_matters` is available **only for arrays**

E.g. the following response would **fail** as the order in the actual response is wrong:

#### expected response defined with `order_matters`

```json
{
    "body": {
        "testArray": [
            "a",
            "b",
            "c"
        ],
        "testArray:control": {
            "order_matters": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testArray": [
            "c",
            "b",
            "a"
        ]
    }
}
```

### `depth`

This setting defines the depth that the `no_extra` and `order_matters` should consider when matching arrays.

`depth` is available only for arrays.

The possible values of `depth` are:

|      |                            |
| ---: | ---                        |
| `-1` | full depth                 |
| `0`  | top element only (default) |
| `N`  | N elements deep            |


The following response would **fail** as there are too many entries in the actual response inner array:

#### expected response defined with `no_extra` and `depth`

```json
{
    "body": {
        "testArray": [
            [1, 3, 5],
            [2, 4, 6]
        ],
        "testObject:control": {
            "no_extra": true,
            "depth": 1
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testArray": [
            [1, 3, 5],
            [2, 4, 6, 8]
        ]
    }
}
```

### `must_exist`

Check if a certain value does exist in the reponse (no matter what its content is).

`must_exist` is available for all types.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"iShouldExist"` is not in the actual response:

#### expected response defined with `must_exist`

```json
{
    "body": {
        "iShouldExist:control": {
            "must_exist": true
        }
    }
}
```

#### actual response

```json
{
    "body": {}
}
```

### `element_count`

Check if the size of an array equals the `element_count`.

`element_count` is available only for arrays.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"count"` has the wrong length:

#### expected response defined with `element_count`

```json
{
    "body": {
        "count:control": {
            "element_count": 2
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "count": [
            1,
            2,
            3
        ]
    }
}
```

### `element_no_extra`

Passes the `no_extra` check to the underlying structure in an array.

`element_no_extra` is only available for arrays.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"extra"` has an extra element:

#### expected response defined with `element_no_extra`

```json
{
    "body": {
        "count": [
            {
                "fine": true,
            }
        ],
        "count:control": {
            "element_no_extra": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "count": [
            {
                "fine": true,
                "extra": "shouldNotBeHere"
            }
        ]
    }
}
```

### `must_not_exist`

Check if a certain value does not exist in the reponse.

`must_not_exist` is available for all types.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"iShouldNotExist"` is in the actual response:

#### expected response defined with `must_not_exist`

```json
{
    "body": {
        "iShouldNotExist:control": {
            "must_not_exist": true
        }
    }
}
```

##### actual response

```json
{
    "body": {
        "iShouldNotExist": "i exist, hahahah"
    }
}
```

### `not_equal`

Check if a field is not equal to a specific value.

This check is available for the types `string`, `number`, `array` and `bool`.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testNumber"` has the value `5`:

#### expected response defined with `not_equal`

```json
{
    "body": {
        "testNumber:control": {
            "not_equal": 5
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testNumber": 5
    }
}
```

### `match`

Check if a string value matches a given [regular expression](https://gobyexample.com/regular-expressions). If the input is not string, it is rendered as string using `fmt.Sprintf("%v", input)`.

E.g. the following response would **fail** as `"text"` does not match the regular expression:

#### expected string response checked with a regex:

```json
{
    "body": {
        "text:control": {
            "match": ".+-\\d+"
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "text": "valid_string-123"
    }
}
```

### `not_match`

Check if a string value does not match a given regular expression. If the input is not string, it is rendered as string using `fmt.Sprintf("%v", input)`.

This is the opposite check function of [match](#match).

### `starts_with`

Check if a string value starts with a given string prefix.

E.g. the following response would **fail** as `"text"` does not have the prefix:

#### expected string response checked with a prefix

```json
{
    "body": {
        "text:control": {
            "starts_with": "abc-"
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "text": "abc-123"
    }
}
```

### `ends_with`

Check if a string value ends with a given string suffix.

E.g. the following response would **fail** as `"text"` does not have the suffix:

#### expected string response checked with a suffix

```json
{
    "body": {
        "text:control": {
            "ends_with": "-123"
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "text": "abc-123"
    }
}
```

### `is_string`

Check if the field has the type `string`.

It implicitly also checks `must_exist` for the value as there is no sense in type checking a value that does not exist.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testString"` is not a string in the actual response:

#### expected response defined with `is_string`

```json
{
    "body": {
        "testString:control": {
            "is_string": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testString": 555
    }
}
```

### `is_bool`

Check if the field has the type `bool`.

It implicitly also checks `must_exist` for the value.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testBool"` is no boolean value in the actual response:

#### expected response defined with `is_bool`

```json
{
    "body": {
        "testBool:control": {
            "is_bool": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testBool": "not a boolean"
    }
}
```

### `is_number`

Check if the field has the type `number`.

It implicitly also checks `must_exist` for the value.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testNumber"` is no numeric value in the actual response:

#### expected response defined with `is_number`

```json
{
    "body": {
        "testNumber:control": {
            "is_number": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testNumber": "not a number"
    }
}
```

### `is_object`

Check if the field is a JSON object.

It implicitly also checks `must_exist` for the value.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testObj"` is not an object in the actual response:

#### expected response defined with `is_object`

```json
{
    "body": {
        "testObj:control": {
            "is_object": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testObj": "not an object"
    }
}
```

### `is_array`

Check if the field is a JSON array.

It implicitly also checks `must_exist` for the value.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"testArr"` is not an array in the actual response:

#### expected response defined with `is_array`

```json
{
    "body": {
        "testArr:control": {
            "is_array": true
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "testArr": "not an array"
    }
}
```

### `number_gt`

With `number_gt` (`>`), you can check if your field of type number (implicit check) is greater than a specific number.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"beGreater"` is equal to the expected number:

#### expected response defined with `number_gt`

```json
{
    "body": {
        "beGreater:control": {
            "number_gt": 5
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "beGreater": 5
    }
}
```

### `number_ge`

With `number_ge` (`=>`), you can check if your field of type number (implicit check) is equal or greater than a specific number.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"beGreaterOrEqual"` is less than the expected number:

#### expected response defined with `number_ge`

```json
{
    "body": {
        "beGreaterOrEqual:control": {
            "number_ge": 5
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "beGreaterOrEqual": 3
    }
}
```

### `number_lt`

With `number_lt` (`<`), you can check if your field of type number (implicit check) is less than a specific number.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"beLess"` is equal to the expected number:

#### expected response defined with `number_lt`

```json
{
    "body": {
        "beLess:control": {
            "number_lt": 5
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "beLess": 5
    }
}
```

### `number_le`

With `number_le` (`<=`), you can check if your field of type number (implicit check) is less or equal than a specific number.

This control can be used without the actual key. So only the `:control` key is present.

E.g. the following response would **fail** as `"beLessOrEqual"` is greater than the expected number:

#### expected response defined with `number_le`

```json
{
    "body": {
        "beLessOrEqual:control": {
            "number_le": 5
        }
    }
}
```

#### actual response

```json
{
    "body": {
        "beLessOrEqual": 7
    }
}
```

# Use external file

In the request and response part of the single testcase you also can load the content from an external file.

This is exspecially helpfull for keeping the manifest file simpler/smaller and keep a better overview. **On top: You can use so called template functions in the external file.** (We will dig deeper into the template functions later)

A single test could look as simple as following:

```json
{
    "name": "Test loading request & response from external file",
    "request": "@path/to/requestFile.json",
    "response": "@path/to/responseFile.json"
}
```

Important: The paths to the external files start with a `@` and are relative to the location of the `manifest.json` or can be web urls e.g. https://programmfabrik.de/testfile.json

The content of the request and response file are execatly the same as if you would place the json code inline:

## Request:

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

## Response:

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

**apitest** supports the [Sprig template](http://masterminds.github.io/sprig/) function library in v3. Internally provided functions like `add` overwrite the `Sprig` function.

As described before, if you use an external file you can make use of so called template functions. What they are and how they work for the apitesting tool is described in the following part.

Template Functions are invoked using the tags `{{ }}` and upon returning substitutes the function call with its result. We use the golang "text/template" package so all functions provided there are also supported here.

For a reference see [https://golang.org/pkg/text/template](https://golang.org/pkg/text/template)

|                                    |                                               |
| ---                                | ---                                           |
| `manifest.json` -> external file   | load external file                            |
| external file   -> another file    | render template with file parameter `"hello"` |
| another file    -> external file   | return rendered template `"hello world"`      |
| external file   -> `manifest.json` | return rendered template                      |

### Example

Assume that the template function `myfunc`, given the arguments `1 "foo"`, returns `"bar"`. The call `{{ myfunc 1 "foo" }}` would translate to `bar`. Consequently, rendering `Lets meet at the {{ myfunc 1 "foo" }}` results in an invitation to the `bar`.

We provide the following functions:

## `file_render "relative/path/" [param, ...]`

Helper function to load contents of a file; if this file contains templates; it will render these templates with the parameters provided in the can be accessed from the loaded file via `{{ .Param1-n }};` see example below

Loads the file with the relative path ( to the file this template function is invoked in ) "relative/path" or a weburl e.g. https://docs.google.com/test/tmpl.txt. Returns string.

## `file "relative/path/"`

Loads the file with the relative path ( to the file this template function is invoked in ) "relative/path" or a weburl e.g. https://docs.google.com/test/tmpl.txt. Returns string.

### Example

Content of file at `some/path/example.tmpl`:

```js
{{ load_file "../target.tmpl" "hello" }}
```

Content of file at `some/target.tmpl`:

```js
{{ .Param1 }} world
```

Rendering `example.tmpl` will result in `hello world`

## `file_path "relative/path/"`

Returns the relative path (to the file this template function is invoked in) "relative/path" or a weburl e.g. https://docs.google.com/test/tmpl.txt

### Example

Absolute path of file at `some/path/myfile.cpp`:
```js
{{ file_path "../myfile.tmpl" }}
```

## `pivot_rows` "keyColumn" "typeColumn" [input]

Read a CSV map and turn rows into columns and columns into rows.

Assume you have the following structure in your sheet:

|           |           |           |        |
| --------  | --------  | --------  | ------ |
| key       | type      | 1         | 2      |
| string    | string    | string    | string |
| name      | string    | bicyle    | car    |
| wheels    | int64     | 2         | 4      |

As a convention the data columns need to be named `1`, `2`, ... Allowed types are:

* `string`
* `int64`
* `number` (JSON type number)
* `float64`

Calling

```
pivot_rows("key","type",(file_csv "file.csv" ','))
```

returns

```json
[
    {
        "filename": "bicyle",
        "wheels": 2
    },
    {
        "filename": "car",
        "wheels": 4
    }
]
```

## `rows_to_map "keyColumn" "valueColumn" [input]`

Generates a key-value map from your input rows.

Assume you have the following structure in your sheet:

| column_a | column_b | column_c |
| -------- | -------- | -------- |
| row1a    | row1b    | row1c    |
| row2a    | row2b    | row2c    |

If you parse this now to CSV and then load it via `file_csv` you get the following JSON structure:

```json
[
    {
        "column_a": "row1a",
        "column_b": "row1b",
        "column_c": "row1c"
    },
    {
        "column_a": "row2a",
        "column_b": "row2b",
        "column_c": 22
    }
]
```

For mapping now certain values to a map you can use ` rows_to_map "column_a" "column_c" `  and the output will be a map with the following content:

```json
{
    "row1a": "row1c",
    "row2a": 22
}
```

## `group_rows "groupColumn" [rows]`

Generates an Array of rows from input rows. The **groupColumn** needs to be set to a column which will be used for grouping the rows into the Array.

The column needs to:

* be an **int64** column
* use integers between `0` and `999`

The Array will group all rows with identical values in the **groupColumn**.

### Example

The CSV can look at follows, use **file_csv** to read it and pipe into **group_rows**

| batch    | reference | title    |
| -------- | --------- | -------- |
| int64    | string    | string   |
| 1        | ref1a     | title1a  |
| 1        | ref1b     | title1b  |
| 4        | ref4      | title4   |
| 3        | ref3      | titlte2  |

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
    ],
    [
        {
            "batch": 3,
            "reference": "ref3",
            "title": "title3"
        }
    ],
    [
        {
            "batch": 4,
            "reference": "ref4",
            "title": "title4"
        }
    ]
]
```

## `group_map_rows "groupColumn" [rows]`

Generates an Map of rows from input rows. The **groupColumn** needs to be set to a column which will be used for grouping the rows into the Array.

The column needs to be a **string** column.

The Map will group all rows with identical values in the **groupColumn**.

### Example

The CSV can look at follows, use **file_csv** to read it and pipe into **group_rows**

| batch    | reference | title    |
| -------- | --------  | -------- |
| string   | string    | string   |
| one      | ref1a     | title1a  |
| one      | ref1b     | title1b  |
| 4        | ref4      | title4   |
| 3        | ref3      | titlte2  |

Produces this output (presented as **json** for better readability:

```json
{
    "one": [
        {
            "batch": "one",
            "reference": "ref1a",
            "title": "title1a"
        },
        {
            "batch": "one",
            "reference": "ref1b",
            "title": "title1b"
        }
    ],
    "4": [
        {
            "batch": "4",
            "reference": "ref3",
            "title": "title3"
        }
    ],
    "3": [
        {
            "batch": "3",
            "reference": "ref4",
            "title": "title4"
        }
    ]
}
```

### Template Example

With the parameters `keyColumn` and `valueColumn` you can select the two columns you want to use for map. (Only two are supported)

The `keyColumn`  **must** be of the type string, as it functions as map index (which is of type string)


```js
{{ unmarshal "[{\"column_a\": \"row1a\",\"column_b\": \"row1b\",\"column_c\": \"row1c\"},{\"column_a\": \"row2a\",\"column_b\": \"row2b\",\"column_c\": \"row2c\"}]" | rows_to_map "column_a" "column_c"  | marshal  }}
```

Rendering that will give you :
```json
{
    "row1a": "row1c",
    "row2a": "row2c"
}
```

### Behavior in corner cases

#### No keyColumn given

The function returns an empty map

For `rows_to_map`:

```json
{}
```

#### No valueColumn given

The complete row gets mapped

For `rows_to_map "column_a"`:

```json
{
    "row1a": {
        "column_a": "row1a",
        "column_b": "row1b",
        "column_c": "row1c",
    },
    "row2a": {
        "column_a": "row2a",
        "column_b": "row2b",
        "column_c": "row2c",
    }
}
```

#### A row does not contain a key column

The row does get skipped

**Input:**

```json
[
    {
        "column_a": "row1a",
        "column_b": "row1b",
        "column_c": "row1c",
    },
    {
        "column_b": "row2b",
        "column_c": "row2c",
    }
    {
        "column_a": "row3a",
        "column_b": "row3b",
        "column_c": "row3c",
    }
]
```

For `rows_to_map "column_a" "column_c" `:

```json
{
    "row1a": "row1c",
    "row3a": "row3c",
}
```

#### A row does not contain a value column

The value will be set to `""` (empty string)

**Input:**

```json
[
    {
        "column_a": "row1a",
        "column_b": "row1b",
        "column_c": "row1c",
    },
    {
        "column_a": "row2a",
        "column_b": "row2b",
    }
    {
        "column_a": "row3a",
        "column_b": "row3b",
        "column_c": "row3c",
    }
]
```

For `rows_to_map "column_a" "column_c" `:

```json
{
    "row1a": "row1c",
    "row2a": "",
    "row3a": "row3c",
}
```

## `datastore [key]`

Helper function to query the datastore; used most of the time in conjunction with [`gjson`](#gjson-path-json).

The `key`can be an int, or int64 accessing the store of previous responses. The responses are accessed in the order received. Using a negative value access the store from the back, so a value of **-2** would access the second to last response struct.

This function returns a string, if the `key`does not exist, an empty string is returned.

If the `key` is a string, the datastore is accessed directly, allowing access to custom set values using `store` or `store_response_gjson`parameters.

The datastore stores all responses in a list. We can retrieve the response (as a json string) by using this template function. `{{ datastore 0  }}` will render to

```json
{
    "statuscode": 200,
    "header": {
        "foo": "bar;baz"
    },
    "body": "..."
}
```

This function is intended to be used with the [`gjson`](#gjson-path-json) template function.

The key `-` has a special meaning, it returns the entire custom datastore (not the sequentially stored responses)

## `gjson [path] [json]`

Helper function to extract fields from the `json`. It uses `gjson` syntax. For more information, see the [external documentation](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

| Parameter      | Type     | Description                                                                                                                                                           |
| ---            | ---      | ---                                                                                                                                                                   |
| `@path`        | `string` | a description of the location of the field to extract. For array access use integers; for object access use keys. Example: `body.1.field`; see below for more details |
| `@json_string` | `string` | a valid json blob to be queried; can be supplied via pipes from `datastore idx`                                                                                       |
| `@result`      |          | the content of the json blob at the specified path                                                                                                                    |

### Example

The call

```js
{{ gjson "foo.1.bar" "{\"foo": [{\"bar\": \"baz\"}, 42]}" }}
```

would return `baz`.

As an example with pipes, the call

```js
{{ datastore idx | gjson "header.foo.1" }}
```

would return`bar` given the response above.


## `file_csv [path] [delimiter]`

Helper function to load a csv file

| Parameter    | Type     | Description                                                                                           |
| ---          | ---      | ---                                                                                                   |
| `@path`      | `string` | A path to the csv file that should be loaded. The path is either relative to the manifest or a weburl |
| `@delimiter` | `rune`   | The delimiter that is used in the given csv e.g. `,` Defaults to `,`                                  |
| `@result`    |          | The content of the csv as json array so we can work on this data with `gjson`                         |

The CSV **must** have a certain structur. If the structure of the given CSV differs, the apitest tool will fail with a error

- In the first row must be the names of the fields
- In the seconds row must be the types of the fields

**Valid types**
- `int64`
- `int`
- `string`
- `float64`
- `bool`
- `int64,array`
- `string,array`
- `float64,array`
- `bool,array`
- `json`

All types can be prefixed with `*` to return a pointer to the value. Empty strings initialize the Golang zero value for the type,  for type array the empty string inialized an empty array. The empty string returns an untyped **nil**.

### Example

Content of file at `some/path/example.csv`:
```csv
id,name
int64,string
1,simon
2,martin
```

The call

```js
{{ file_csv "some/path/example.csv" ','}}
```

would result in

```go
[map[id:1 name:simon] map[id:2 name:martin]]
```

As an example with pipes, the call

```js
{{ file_csv "some/path/example.csv" ',' | marshal | gjson "1.name" }}
```

would result in `martin` given the response above.

### Corner Cases

There are some corner cases that trigger a certain behavior you should keep in mind

#### No format for a column given

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

#### No name for a column given

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

#### Comment or empty line

If there is a comment marked with `#`, or a empty line that does not get rendered into the result

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

## `file_xml2json [path]`

Helper function to parse an XML file and convert it into json
- `@path`: string; a path to the XML file that should be loaded. The path is either relative to the manifest or a weburl

This function uses the function `NewMapXml()` from [github.com/clbanning/mxj](https://github.com/clbanning/mxj).

### Example

Content of XML file `some/path/example.xml`:

```xml
<objects xmlns="https://schema.easydb.de/EASYDB/1.0/objects/">
    <obj>
        <_standard>
            <de-DE>Beispiel Objekt</de-DE>
            <en-US>Example Object</en-US>
        </_standard>
        <_system_object_id>123</_system_object_id>
        <_id>45</_id>
        <name type="text_oneline"
            column-api-id="263">Example</name>
    </obj>
</objects>
```

The call

```js
{{ file_xml2json "some/path/example.xml" }}
```

would result in

```json
{
    "objects": {
        "-xmlns": "https://schema.easydb.de/EASYDB/1.0/objects/",
        "obj": {
            "_id": "45",
            "_standard": {
                "de-DE": "Beispiel Objekt",
                "en-US": "Example Object"
            },
            "_system_object_id": "123",
            "name": {
                "#text": "Example",
                "-column-api-id": "263",
                "-type": "text_oneline"
            }
        }
    }
}
```

## `file_html2json [path]`

Helper function to parse an HTML file and convert it into json
- `@path`: string; a path to the HTML file that should be loaded. The path is either relative to the manifest or a weburl

This marshalling is less strict than for [XHTML](#file_xhtml2json-path). For example it will not raise errors for unclosed tags like `<p>` or `<hr>`, as well as Javascript code inside the HTML code. But it is possible that unclosed tags are missing in the resulting JSON if the [goquery](https://github.com/PuerkitoBio/goquery) tokenizer can not find a matching closing tag.

### Example

Content of HTML file `some/path/example.html`:

```html
<!DOCTYPE html>
<html lang="en">

	<head>
		<meta charset="utf-8" />
		<title>fylr</title>
		<meta name="description" content="fylr - manage your data" />
		<script>
			function onInputHandler(event) {
				const form = event.currentTarget;
				submitForm(form);
			}
		</script>
	</head>

	<body>
		<div class="container">
			<h1>Register</h1>

			<p class="required-information"><sup>*</sup>Mandatory fields<br>
			<p class="error-summary">Form has errors

			<hr>
		</div>
	</body>

</html>
```

The call

```js
{{ file_html2json "some/path/example.html" }}
```

would result in

```json
{
    "html": {
        "-lang": "en",
        "head": {
            "meta": [
                {
                    "-charset": "utf-8"
                },
                {
                    "-content": "fylr - manage your data",
                    "-name": "description"
                }
            ],
            "title": {
                "#text": "fylr"
            },
            "script": {
                "#text": "function onInputHandler(event) {\n\t\t\t\tconst form = event.currentTarget;\n\t\t\t\tsubmitForm(form);\n\t\t\t}"
            }
        },
        "body": {
            "div": {
                "-class": "container",
                "h1": {
                    "#text": "Register"
                },
                "p": [
                    {
                        "-class": "required-information",
                        "sup": {
                            "#text": "*"
                        },
                        "br": {}
                    },
                    {
                        "#text": "Form has errors",
                        "-class": "error-summary"
                    }
                ],
                "hr": {}
            }
        }
    }
}
```

## `file_xhtml2json [path]`

Helper function to parse an XHTML file and convert it into json
- `@path`: string; a path to the XHTML file that should be loaded. The path is either relative to the manifest or a weburl

### Example

Content of XHTML file `some/path/example.xhtml`:

```html
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
    <head>
        <link href="/css/easydb.css" rel="stylesheet" type="text/css" />
        <title>easydb documentation</title>
    </head>
    <body>
        <h1 id="welcome-to-the-easydb-documentation">Welcome to the easydb documentation</h1>
    </body>
</html>
```

The call

```js
{{ file_xhtml2json "some/path/example.xhtml" }}
```

would result in

```json
{
    "html": {
        "-xmlns": "http://www.w3.org/1999/xhtml",
        "head": {
            "link": {
                "-href": "/css/easydb.css",
                "-rel": "stylesheet",
                "-type": "text/css"
            },
            "title": "easydb documentation"
        },
        "body": {
            "h1": {
                "#text": "Welcome to the easydb documentation",
                "-id": "welcome-to-the-easydb-documentation"
            }
        }
    }
}
```

## `file_sqlite [path] [statement]`

Helper function to return the result of an SQL statement from a sqlite3 file

| Parameter    | Type   | Description                                                                                              |
| ---          | ---    | ---                                                                                                      |
| `@path`      | string | a path to the sqlite file that should be loaded. The path is either relative to the manifest or a weburl |
| `@statement` | string | a SQL statement that returns data (`SELECT`)                                                             |
| `@result`    |        | the result of the statement as a json array so we can work on this data with `gjson`                     |

### Example

Content of sqlite file at `some/path/example.sqlite`:

Table `names`:
- column `id`: type `INTEGER`
- column `name`: type `TEXT`

| id  | name     |
| --- | ---      |
| `2` | `martin` |
| `3` | NULL     |
| `1` | `simon`  |

The call

```js
{{ file_sqlite "some/path/example.sqlite" `
    SELECT id, name FROM names
    WHERE name IS NOT NULL
    ORDER BY id ASC
` }}
```

would result in

```go
[map[id:1 name:simon] map[id:2 name:martin]]
```

### Working with `NULL` values

`NULL` values in the database are returned as `nil` in the template. To check if a value in the sqlite file is `NULL`, us a comparison to `nil`:

The call

```js
{{ file_sqlite "some/path/example.sqlite" `
    SELECT id, name FROM names
    ORDER BY id ASC
` }}
```

would result in

```go
[map[id:1 name:simon] map[id:2 name:martin] map[id:3 name:nil]]
```

The `NULL` value in `name` can be checked with

```js
{{ if ne $row.name nil }}
    // use name, else skip
{{ end }}
```

## `slice [parms...]`

Returns a slice with the given parameters as elements. Use this for **range** in templates.

## `split s sep`

Returns a string slice with `s` split by `sep`.

## `add [a] [b]`

Returns the sum of `a`and `b`. `a, b` can be any numeric type or string. The function returns a numeric type, depending on the input. With `string` we return `int64`.

## `subtract [a] [b]`

Returns `a - b`. `a, b` can be any numeric type or string. The function returns a numeric type, depending on the input. With `string` we return `int64`.

## `multiply [a] [b]`

Returns `a * b`. `a, b` can be any numeric type or string. The function returns a numeric type, depending on the input. With `string` we return `int64`.

## `divide [a] [b]`

Returns `a / b`. `a, b` can be any numeric type or string. The function returns a numeric type, depending on the input. With `string` we return `int64`.

## `unmarshal [string]`

Returns a `util.GenericJson` Object (go: `interface{}`) of the unmarshalled  `JSON` string.

## `marshal [interface{}]`

Returns a `string` of the marshalled  `interface{}` object.

## `md5sum [filepath]`

Returns a `string` of the MD5 sum of the file found in `filepath`.

## `str_escape [string]`

Returns a `string` where all `"` are escaped to `\"`. This is useful in Strings which need to be concatenated.

## `query_escape [string]`

Returns a `string` as the result of escaping input as if it was intended for use in a URL query string.

## `query_unescape [string]`

Returns a `string` as the result of unescaping input as if it was coming from a URL query string.

## `base64_encode [string]`

Returns a `string` as the result of encoding input into base64.

## `base64_decode [string]`

Returns a `string` as the result of decoding input from base64.

## `url_path_escape [string]`

Uses [Url.PathEscape](https://pkg.go.dev/net/url?tab=doc#PathEscape) to escape given `string` to use in `endpoint` or `server_url`. Returns `string`.

## `match [regex] [text]`

Returns a `bool` value. If `text` matches the [regular expression](https://gobyexample.com/regular-expressions) `regex`, it returns `true`, else `false`. This is useful inside `{{ if ... }}` templates.

## `printf [interface{}...]`

Just for reference, this is a Go Template [built-in](https://golang.org/pkg/text/template/#hdr-Functions).

## `N [float64|int64|int]`

Returns a slice of n 0-sized elements, suitable for ranging over.

Example how to range over 100 objects

```js
{
    "body":  [
        {{ range $idx, $v := N 100 }}
        ...
        {{ end }}
    ]
}
```

## `replace_host [url]`

**replace_host** replaces the host in the given `url` with the actual address of the built-in HTTP server (see below). This address, taken from the `manifest.json` can be overwritten with the command line parameter `--replace-host`.

As an example, the URL `http://localhost/myimage.jpg` would be changed into `http://localhost:8788/myimage.jpg` following the example below.

## `server_url`

**server_url** returns the server url, which can be globally provided in the config file or directly by the command line parameter `--server`. This is a `*url.URL`.

## `server_url_no_user`

**server_url_no_user** returns the server url, which can be globally provided in the config file or directly by the command line parameter `--server`. Any information about the user authentification is removed. This is a `*url.URL`.

If the **server_url** is in the form of `http://user:password@localhost`, **server_url_no_user** will return `http://localhost`.

## `is_zero`

**is_zero** returns **true** if the passed value is the Golang zero value of the type.

## `oauth2_password_token [client] [username] [password]`

**oauth2_password_token** returns an **oauth token** for a configured client and given some user credentials. Such token is an object which contains several properties, being **access_token** one of them. It uses the `trusted` oAuth2 flow

Example:

```js
{
    "store": {
        "access_token": {{ oauth2_password_token "my_client" "john" "pass" | marshal | gjson "access_token" }}
    }
}
```

## `oauth2_client_token [client]`

**oauth2_client_token** returns an **oauth token** for a configured client. Such token is an object which contains several properties, being **access_token** one of them. It uses the `client credentials` oAuth2 flow.

Example:

```js
{
    "store": {
        "access_token": {{ oauth2_client_token "my_client" | marshal | gjson "access_token" }}
    }
}
```

## `oauth2_code_token [client] ...[[key] [value]]`

**oauth2_code_token** returns an **oauth token** for a configured client and accepts a variable number of key/value parameters. Such token is an object which contains several properties, being **access_token** one of them. It uses the `code grant` oAuth2 flow.

Behind the scenes the function will do a GET request to the `auth URL`, adding such parameters to it, and interpret the last URL such request was redirected to, extracting the code from it and passing it to the last step of the regular flow.

Example:

```js
{
    "store": {
        "access_token": {{ oauth2_code_token "my_client" "username" "myuser" "password" "mypass" | marshal | gjson "access_token" }}
    }
}
```

Or:

```js
{
    "store": {
        "access_token": {{ oauth2_code_token "my_client" "guess_access" "true" | marshal | gjson "access_token" }}
    }
}
```

## `oauth2_implicit_token [client] ...[[key] [value]]`

**oauth2_implicit_token** returns an **oauth token** for a configured client and accepts a variable number of key/value parameters. Such token is an object which contains several properties, being **access_token** one of them. It uses the `implicit grant` oAuth2 flow.

Behind the scenes the function will do a GET request to the `auth URL`, adding such parameters to it, and interpret the last URL such request was redirected to, extracting the token from its fragment.

Example:

```js
{
    "store": {
        "access_token": {{ oauth2_password_token "my_client" "myuser" "mypass" | marshal | gjson "access_token" }}
    }
}
```

## `oauth2_client [client]`

**oauth2_client** returns a configured **oauth client** given its `client_id`. Result is an object which contains several properties.

Example:

```js
{
    "store": {
        "oauth2_client_config": {{ oauth2_client "my_client" | marshal }}
    }
}
```

## `oauth2_basic_auth [client]`

**oauth2_basic_auth** returns the authentication header for basic authentication for the given oauth client.

## `semver_compare [version 1] [version 2]`

**semver_compare** compares to semantic version strings. This calls https://pkg.go.dev/golang.org/x/mod/semver#Compare, so check there for additional documentation. If the version is `""` the version `v0.0.0` is assumed. Before comparing, the function checks if the strings are valid. In case they are not, an error is returned.

## `log` [msg] [args...]

Write **msg** to log output. Args can be given. This uses logrus.Debugf to output.

## `remove_from_url` [key] [url]

Removes from **key** from **url**'s query, returns the **url** with the **key** removed. In case of an error, the **url** is returned as is. Unparsable urls are ignored and the **url** is returned.

## `value_from_url` [key]

Returns the **value** from the **url**'s query for **key**. In case of an error, an empty string is returned. Unparsable urls are ignored and an empty string is returned.

## `parallel_run_idx`

Returns the index of the Parallel Run that the template is executed in, or -1 if it is not executed within a parallel run.

# HTTP Server

The apitest tool includes an HTTP Server. It can be used to serve files from the local disk temporarily. The HTTP Server can run in test mode. In this mode, the apitest tool does not run any tests, but starts the HTTP Server in the foreground, until CTRL-C in pressed.
It is possible to define a proxy in the server which accepts and stores request data.
It is useful if there is need to test that expected webhook calls are properly performed.

Different stores can be configured within the proxy.

To configure a HTTP Server, the manifest need to include these lines:

```jsonc
{
    "http_server": {
        "addr": ":8788",           // address to listen on
        "dir": "",                 // directory to server, relative to the manifest.json, defaults to "."
        "testmode": false,         // boolean flag to switch test mode on / off
        "proxy": {                 // proxy configuration
            "test": {              // proxy store configuration
                "mode": "passthru" // proxy store mode
            }
        }
    }
}
```

The proxy `mode` parameter supports these values:
- `passthru` : The request is stored as it is, without further processing

The HTTP Server is started and stopped per test.

## HTTP Endpoints

The server provides endpoints to serve local files and return responses based on request data.

### Static files

To access any static file, use the path relative to the server directory (`dir`) as the endpoint:

```json
{
    "request": {
        "endpoint": "path/to/file.jpg",
        "method": "GET"
    }
}
```

If there is any error (for example wrong path), a HTTP error repsonse will be returned.

#### No Content-Length header

For some tests, you may not want the Content-Length header to be sent alongside the asset

In this case, add `no-content-length=1` to the query string of the asset url:

```json
{
    "request": {
        "endpoint": "path/to/file.jpg?no-content-length=1",
        "method": "GET"
    }
}
```

### `bounce`

The endpoint `bounce` returns the binary of the request body, as well as the request headers and query parameters as part of the response headers.

```json
{
    "request": {
        "endpoint": "bounce",
        "method": "POST",
        "query_params": {
            "param1": "abc"
        },
        "header": {
            "header1": "123"
        },
        "body": {
            "file": "@path/to/file.jpg"
        },
        "body_type": "multipart"
    }
}
```

The file that is specified is relative to the apitest file, not relative to the http server directory. The response will include the binary of the file, which can be handled with [`pre_process` and `format`](#preprocessing-responses).

Request headers are included in the response header with the prefix `X-Req-Header-`, request query parameters are included in the response header with the prefix `X-Req-Query-`:

```json
{
    "response": {
        "header": {
            "X-Req-Query-Param1": [
                "abc"
            ],
            "X-Req-Header-Header1": [
                "123"
            ]
        }
    }
}
```

### `bounce-json`

The endpoint `bounce-json` returns the a response that includes `header`, `query_params` and `body` in the body.

```json
{
    "request": {
        "endpoint": "bounce-json",
        "method": "POST",
        "query_params": {
            "param1": "abc"
        },
        "header": {
            "header1": 123
        },
        "body": {
            "value1": "test",
            "value2": {
                "hello": "world"
            }
        }
    }
}
```

will return this response:

```json
{
    "response": {
        "body": {
            "query_params": {
                "param1": [
                    "abc"
                ]
            },
            "header": {
                "Header1": [
                    "123"
                ]
            },
            "body": {
                "value1": "test",
                "value2": {
                    "hello": "world"
                }
            }
        }
    }
}
```

### `bounce-query`

The endpoint `bounce-query` returns the a response that includes in its `body` the request `query string` as it is.

This is useful in endpoints where a body cannot be configured, like oAuth urls, so we can simulate responses in the request for testing.

```json
{
    "request": {
        "endpoint": "bounce-query?here=is&all=stuff",
        "method": "POST",
        "body": {}
    }
}
```

will return this response:

```json
{
    "response": {
        "body": "here=is&all=stuff"
    }
}
```

## HTTP Server Proxy

The proxy different stores can be used to both store and read their stored requests.

The configuration, as already defined in [HTTP Server](#http-server), is as follows:

```jsonc
"proxy": {                 // proxy configuration
    "<store_name>": {      // proxy store configuration
        "mode": "passthru" // proxy store mode
    }
}
```

| Key            | Value Type     | Value description                                                        |
|----------------|----------------|--------------------------------------------------------------------------|
| `proxy`        | JSON Object    | An object with the store names as keys and their configuration as values |
| `<store_name>` | JSON Object    | An object with the store configuration                                   |
| `mode`         | string         | The mode the store runs on (see below)                                   |

Store modes:

| Value        | Description                                                                           |
|--------------|---------------------------------------------------------------------------------------|
| `passthru`   | The request to the proxy store will be stored as it is without any further processing |

### Write to proxy store

Perform a request against the http server path `/proxywrite/<store_name>`. Where `<store_name>` is a key (store name) inside the `proxy` object in the configuration.

The expected response will have either `200` status code and the used offset as body or another status and an error body.

Given this request:

```json
{
    "endpoint": "/proxywrite/test",
    "method": "POST",
    "query_params": {
        "some": "param"
    },
    "header": {
        "X-My-Header": 0
    },
    "body": {
        "post": {
            "my": [
                "body",
                "here"
            ]
        }
    }
}
```

The expected response:

```json
{
    "statuscode": 200,
    "body": {
        "offset": 0
    }
}
```

### Read from proxy store

Whatever request performed against the server path `/proxyread/<store_name>?offset=<offset>`.

Where:
- `<store_name>` is a key inside the `proxy` object in the server configuration, aka the proxy store name
- `<offset>` represents the entry to be retrieved in the proxy store requests collection. If not provided, 0 is assumed.

Given this request:

```json
{
    "endpoint": "/proxyread/test",
    "method": "GET",
    "query_params": {
        "offset": 0
    }
}
```

The expected response:

```jsonc
{
    // Merged headers. original request headers prefixed with 'X-Request`
    "header": {
        // The method of the request to the proxy store
        "X-Apitest-Proxy-Request-Method": [
            "POST"
        ],
        // The url path requested (including query string)
        "X-Apitest-Proxy-Request-Path": [
            "/proxywrite/test"
        ],
        // The request query string only
        "X-Apitest-Proxy-Request-Query": [
            "is=here&my=data&some=value"
        ],
        // Original request custom header
        "X-My-Header": [
            "blah"
        ],
        // The number of requests stored
        "X-Apitest-Proxy-Store-Count": [
            "7"
        ],
        // The next offset in the store
        "X-Apitest-Proxy-Store-Next-Offset": [
            "1"
        ]
        // [...]
        // All other standard headers sent with the original request (like Content-Type)
    },
    // The body of this request to the proxy store, always in binary format
    "body": {
        // Content-Type header will reveal its format on client side, in this case, it's JSON, but it could be a byte stream of an image, etc.
        "whatever": [
            "is",
            "here"
        ]
    }
}
```

## SMTP Server

### Summary and Configuration

The apitest tool can run a mock SMTP server intended to catch locally sent emails for testing purposes.

To add the SMTP Server to your test, put the following in your manifest:

```jsonc
{
    "smtp_server": {
        "addr":             ":9025", // address to listen on
        "max_message_size": 1000000  // maximum accepted message size in bytes
                                     // (defaults to 30MiB)
    }
}
```

The server will then listen on the specified address for incoming emails.
Incoming messages are stored in memory and can be accessed using the HTTP
endpoints described further below. No authentication is performed when
receiving messages.

If the test mode is enabled on the HTTP server and an SMTP server is also
configured, both the HTTP and the SMTP server will be available during
interactive testing.

### HTTP Endpoints

On its own, the SMTP server has only limited use, e.g. as an email sink for
applications that require such an email sink to function. But when combined
with the HTTP server (see above in section [HTTP Server](#http-server)),
the messages received by the SMTP server can be reproduced in JSON format.

When both the SMTP server and the HTTP server are enabled, the following
additional endpoints are made available on the HTTP server:

#### `/smtp/gui`

A very basic HTML/JavaScript GUI that displays and auto-refreshes the received
messages is made available on the `/smtp/gui` endpoint.

#### `/smtp`

On the `/smtp` endpoint, an index of all received messages will be made
available as JSON in the following schema:

```json
{
    "count": 3,
    "messages": [
        {
            "from": [
                "testsender@programmfabrik.de"
            ],
            "idx": 0,
            "isMultipart": false,
            "receivedAt": "2024-07-02T11:23:31.212023129+02:00",
            "smtpFrom": "testsender@programmfabrik.de",
            "smtpRcptTo": [
                "testreceiver@programmfabrik.de"
            ],
            "to": [
                "testreceiver@programmfabrik.de"
            ]
        },
        {
            "from": [
                "testsender2@programmfabrik.de"
            ],
            "idx": 1,
            "isMultipart": true,
            "receivedAt": "2024-07-02T11:23:31.212523916+02:00",
            "smtpFrom": "testsender2@programmfabrik.de",
            "smtpRcptTo": [
                "testreceiver2@programmfabrik.de"
            ],
            "subject": "Example Message",
            "to": [
                "testreceiver2@programmfabrik.de"
            ]
        },
        {
            "from": [
                "testsender3@programmfabrik.de"
            ],
            "idx": 2,
            "isMultipart": false,
            "receivedAt": "2024-07-02T11:23:31.212773829+02:00",
            "smtpFrom": "testsender3@programmfabrik.de",
            "smtpRcptTo": [
                "testreceiver3@programmfabrik.de"
            ],
            "to": [
                "testreceiver3@programmfabrik.de"
            ]
        }
    ]
}
```

You can filter messages by passing one of more query parameters `header`. `header` can either be a JSON array of strings, or just a string. The filter checks that all headers (regexp format) match headers of the filtered email.

Headers that were encoded according to RFC2047 are decoded first.

#### `/smtp/$idx`

On the `/smtp/$idx` endpoint (e.g. `/smtp/1`), metadata about the message with the corresponding index is made available as JSON:

```json
{
    "bodySize": 306,
    "contentType": "multipart/mixed",
    "contentTypeParams": {
        "boundary": "d36c3118be4745f9a1cb4556d11fe92d"
    },
    "from": [
        "testsender2@programmfabrik.de"
    ],
    "headers": {
        "Content-Type": [
            "multipart/mixed; boundary=\"d36c3118be4745f9a1cb4556d11fe92d\""
        ],
        "Date": [
            "Tue, 25 Jun 2024 11:15:57 +0200"
        ],
        "From": [
            "testsender2@programmfabrik.de"
        ],
        "Mime-Version": [
            "1.0"
        ],
        "Subject": [
            "Example Message"
        ],
        "To": [
            "testreceiver2@programmfabrik.de"
        ]
    },
    "idx": 1,
    "isMultipart": true,
    "multiparts": [
        {
            "bodySize": 15,
            "contentType": "text/plain",
            "contentTypeParams": {
                "charset": "utf-8"
            },
            "headers": {
                "Content-Type": [
                    "text/plain; charset=utf-8"
                ]
            },
            "idx": 0,
            "isMultipart": false
        },
        {
            "bodySize": 39,
            "contentType": "text/html",
            "contentTypeParams": {
                "charset": "utf-8"
            },
            "headers": {
                "Content-Type": [
                    "text/html; charset=utf-8"
                ]
            },
            "idx": 1,
            "isMultipart": false
        }
    ],
    "multipartsCount": 2,
    "receivedAt": "2024-07-02T12:54:44.443488367+02:00",
    "smtpFrom": "testsender2@programmfabrik.de",
    "smtpRcptTo": [
        "testreceiver2@programmfabrik.de"
    ],
    "subject": "Example Message",
    "to": [
        "testreceiver2@programmfabrik.de"
    ]
}
```

Headers that were encoded according to RFC2047 are decoded first.

#### `/smtp/$idx/body`

On the `/smtp/$idx/body` endpoint (e.g. `/smtp/1/body`), the message body (excluding message headers, including multipart part headers) is made availabe for the message with the corresponding index.

If the message was sent with a `Content-Transfer-Encoding` of either `base64` or `quoted-printable`, the endpoint returns the decoded body.

If the message was sent with a `Content-Type` header, it will be passed through to the HTTP response.

#### `/smtp/$idx/multipart`

For multipart messages, the `/smtp/$idx/multipart` endpoint (e.g. `/smtp/1/multipart`) will contain an index of that messages multiparts in the following schema:

```json
{
    "multiparts": [
        {
            "bodySize": 15,
            "contentType": "text/plain",
            "contentTypeParams": {
                "charset": "utf-8"
            },
            "headers": {
                "Content-Type": [
                    "text/plain; charset=utf-8"
                ]
            },
            "idx": 0,
            "isMultipart": false
        },
        {
            "bodySize": 39,
            "contentType": "text/html",
            "contentTypeParams": {
                "charset": "utf-8"
            },
            "headers": {
                "Content-Type": [
                    "text/html; charset=utf-8"
                ]
            },
            "idx": 1,
            "isMultipart": false
        }
    ],
    "multipartsCount": 2
}
```

#### `/smtp/$idx[/multipart/$partIdx]+`

On the `/smtp/$idx/multipart/$partIdx` endpoint (e.g. `/smtp/1/multipart/0`), metadata about the multipart with the corresponding index is made available:

```json
{
    "bodySize": 15,
    "contentType": "text/plain",
    "contentTypeParams": {
        "charset": "utf-8"
    },
    "headers": {
        "Content-Type": [
            "text/plain; charset=utf-8"
        ]
    },
    "idx": 0,
    "isMultipart": false
}
```

Headers that were encoded according to RFC2047 are decoded first.

The endpoint can be called recursively for nested multipart messages, e.g. `/smtp/1/multipart/0/multipart/1`.

#### `/smtp/$idx[/multipart/$partIdx]+/body`

On the `/smtp/$idx/multipart/$partIdx/body` endpoint (e.g. `/smtp/1/multipart/0/body`), the body of the multipart (excluding headers) is made available.

If the multipart was sent with a `Content-Transfer-Encoding` of either `base64` or `quoted-printable`, the endpoint returns the decoded body.

If the message was sent with a `Content-Type` header, it will be passed through to the HTTP response.

The endpoint can be called recursively for nested multipart messages, e.g. `/smtp/1/multipart/0/multipart/1/body`.
