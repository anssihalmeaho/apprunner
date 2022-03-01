# apprunner
FunL application execution service

## Concepts

Microservices are usually implemented as OS processes.
Services implemented with [FunL](https://github.com/anssihalmeaho/funl) can be run as language-level entities by using __apprunner__.
This kind of service is called **app** (or **application**).
Many app's can be run in context of one apprunner -process.

### App (application)
**App** is a fiber/goroutine (started by __apprunner__) which executes main procedure (of main module)
of given **package**. **Package** is stored first to __apprunner__.

App is able to see (import) all modules which are in its package (not the ones in other app's packages).

### Package
[Package](https://github.com/anssihalmeaho/funl/wiki/packages) is concept of FunL programming language.
Package contains several FunL modules in one file and can be executed so that all import paths
are targeted to that same package.

Apprunner stores packages to file ([bbolt](https://github.com/etcd-io/bbolt) is used as storage).

## Possible apprunner configurations

Apprunner contains two parts:

1. Code Server (storing packages)
2. Executor (runs app's, uses Code Server for getting packages)

Apprunner can be run as one process so that its Executor part uses
its own Code Server part for getting packages.

Multiple apprunner processes can be run also so that some of those
are in Executor role and use one apprunner which acts as Code Server.

### Options

With **-csaddr** option apprunner is told to use Code Server
from given address (instead of own internal one which is default option).

With **-port** option port used for HTTP requests can be defined
(default is 8080).

With **-file** option target file for storing packages (by **bbolt**)
can be given (default is "packs.db" in current working directory).

## API

There are REST (HTTP) API's provided by apprunner (Code Server and Executor parts).

### Code Server API's

#### POST /packs/:package-name

Adds package file contents to Code Server.
Content is in request body as binary data.

Status code in response is:

* 201 (Created): operation ok
* 400 (Bad Request): content could not be read
* 500 (Internal Server Error): error in storing package

#### GET /packs

Get array of package names of stored packages (as JSON array).
With **name** -query parameter package name can be defined.

Status code in response is 200 (OK) if no query parameter is given.

If **name** -query parameter was given then:

* In success status code is 200 (OK) and array in response contains that package name
* If given package name is not found then status code is 404 (Not Found)

#### GET /packs/:package-name

Get package content (binary data) for given package name.
If package was found status code is 200 (OK) and response body
contains package content as binary data.
If package was not found status code is 404 (Not Found).

#### DELETE /packs/:package-name

Removes package with given name from Code Server.
Status code in response is 200 (OK).

### Executor API's

#### POST /app

Starts app.

JSON object in request body contains:

| name | value |
| ---- | ----- |
| name | app name (string) |
| pack | package name (string) |
| args | arguments for main procedure (array) |
| ctx-last | context given as last argument to main (bool) |
| ctx-1st | context given as first argument to main (bool) |

If "ctx-last" and "ctx-last" are **false** or missing then no context is given
to main procedure as argument.
Context is map which contains additional information for app to use.

Status code in response is:

* 201 (Created): operation ok
* 400 (Bad Request): invalid request body
* 404 (Not Found): package not found
* 500 (Internal Server Error): error in writing response

#### GET /app

Gets information about currently running app's.
App information in response is JSON array of JSON objects.

JSON object contains:

| name | value |
| ---- | ----- |
| id | app id (int) |
| name | app name (string) |

Status code in response is 200 (OK).

#### DELETE /app/:app-id

Stops app with given id.
**Note.** this requires that app supports stopping by having 
context as argument and listening to **exit-channel** (in context with key "exit-chan").

Status code in response is 200 (OK).

## Get started

### Install

Go language need to be installed first. After that get apprunner repository from Github:

```
git clone https://github.com/anssihalmeaho/apprunner.git
cd apprunner
```

Then run make (Linux/Unix/Cygwin/MacOS) to create executable (**apprunner**):

```
make
```

Building executable in Windows can be made as:

```
go build -o apprunner.exe -v .
```

### Starting

Start appserver:

```
./apprunner
2022/02/09 14:32:54 .../apprunner started
```

or in Windows:

```
apprunner.exe
2022/02/09 19:08:06 apprunner.exe started
```

### Stopping

And shutting down is done by CTRL-C (SIGINT):

```
2022/02/09 19:04:31 .../apprunner exit
```

## App implementation issues

There are several things from apprunner that can be visible to app
implementation if allowed.
Those things are supported by extra argument (so-called **context** map).

context (ctx) map contains following name-value pairs:

| name | value |
| ---- | ----- |
| 'name' | app name (string) |
| 'id' | app id (string) |
| 'log' | logger procedure (proc) |
| 'exit-chan' | exit channel (chan) |


### Logging

Logger procedure from context (with key 'log') provides way
to printout meaningful messages from app.
Apprunner adds app name and id to printout.

### Shutdown

For shutdown app gets exit-channel from context map (with key 'exit-chan').
App needs to listen exit-channel and when value is received there app needs
to shutdown its action and return from main procedure.

## Example App: Simple HTTP Server

This example app just replies to GET /hello request with "Hi".
It gets port number as argument to main procedure.

In this example main module is the only module and
no context-map is given as argument.

Source code: **simpleserver.fnl**:

```
ns main

import stdhttp

main = proc(port)
	handler = proc(w r)
		import stdbytes
		call(stdhttp.write-response w 200 call(stdbytes.str-to-bytes 'Hi'))
	end

	mux = call(stdhttp.mux)
	_ = call(stdhttp.reg-handler mux '/hello' handler)
	call(stdhttp.listen-and-serve mux plus(':' str(port)))
end

endns
```

First make package with __tar__-command:

```
tar -cvf simpleserver.fpack ./simpleserver.fnl
```

This creates package file **simpleserver.fpack**.

Then put package to apprunner (Code Server):

```
curl -X POST --data-binary @simpleserver.fpack http://localhost:8080/packs/simpleserver.fpack
```

And start app:

```
curl -X POST -d '{"name": "myserver", "pack": "simpleserver.fpack", "args": ["8003"]}' http://localhost:8080/app

{"id":"11"}
```

Now try GET /hello API provided by app:

```
curl http://localhost:8003/hello

Hi
```

## Example App: HTTP server with multiple modules and context-map support

This example is funtionally similar as earlier simple HTTP server but
this one has one other module in addition to main module (both in same package).

This app also receives context-map as argument. Logging and exit-channel are
used from context-map.
This app is able to shutdown its service when it receives value from exit-channel.

Source code, main module: **ctxserver.fnl**:

```
ns main

import stdhttp
import servhandler

main = proc(ctx port)
	exit-chan = get(ctx 'exit-chan')
	log = get(ctx 'log')

	mux = call(stdhttp.mux)

	_ = spawn(call(proc()
		_ = call(log 'exit channel: ' recv(exit-chan))
		print(call(stdhttp.shutdown mux))
	end))

	_ = call(stdhttp.reg-handler mux '/hello' servhandler.handler)
	_ = call(log 'exit -> ' call(stdhttp.listen-and-serve mux plus(':' str(port))))
	'server done'
end

endns
```

Source code, module: **servhandler.fnl**:

```
ns servhandler

import stdhttp
import stdbytes

handler = proc(w r)
	call(stdhttp.write-response w 200 call(stdbytes.str-to-bytes 'Hi'))
end

endns
```

First make package with __tar__-command:

```
tar -cvf ../ctxserver.fpack ./*.*
```

Then put package to apprunner (Code Server):

```
curl -X POST --data-binary @ctxserver.fpack http://localhost:8080/packs/ctxserver.fpack
```

When inquiring available packages this package is seen:

```
curl http://localhost:8080/packs

["ctxserver.fpack"]
```

And start app:

```
curl -X POST -d '{"name": "myserver", "pack": "ctxserver.fpack", "ctx-1st": true, "args": ["8003"]}' http://localhost:8080/app

{"id":"11"}
```

Now try GET /hello API provided by app:

```
curl http://localhost:8003/hello

Hi
```

Checking currently running app's:

```
curl http://localhost:8080/app

[{"id":11,"name":"myserver"}]
```

Then stopping app:

```
curl -X DELETE http://localhost:8080/app/11
```

Printout by apprunner:

```
app 11 (myserver): 'exit channel: ' 'exit-from-user'
app 11 (myserver): 'exit -> ' 'http: Server closed'
App exit: 11 (myserver): 'server done'
```

It can be seen that app is not anymore running:

```
curl http://localhost:8080/app

[]
```

## Extensions

There is possibility to add extension modules to be built-in to **apprunner**.

Extension modules are such that app's can import and use those without giving those as part of package as those are part of apprunner -executable.

It's easy to add new extensions without need for modifying other code.

### Go extension modules

There are extension modules implemented with Go.

Those register themselves in **init** -function by using **funl.AddExtensionInitializer**
(which actually stores actual initializer function).

No modification for other apprunner code is needed when Go extension is added.

### FunL extension module

Extension modules implemented with **FunL** are done just by adding FunL source file (with extension **.fnl**) to **extensions** -directory.

Source codes are embedded to executable (by using Go's file embedding) and parsed when apprunner starts.

### List of extensions

Currently there are following extensions:

name | type | description
---- | ---- | -----------
appval | Go | see explanation later
valuez | Go | https://github.com/anssihalmeaho/fuvaluez
fields | FunL | https://github.com/anssihalmeaho/fields
httprouter | FunL | https://github.com/anssihalmeaho/httprouter

### appval -extension

Module **appval** is used for sharing FunL -values between separate app's.

For example, communication via channels can be done by sharing channels
via **appval**.

Values are identified by names (string) and by tokens (string).

Token forms its own namespace for name-value pairs.

#### setval

Writes value by given token and name:

```
call(appval.setval <token:string> <name:string> <value>) -> true (bool)
```

#### getval

Reads value by given token and name:

```
call(appval.getval <token:string> <name:string>) -> list(<found:bool> <value>)
```

