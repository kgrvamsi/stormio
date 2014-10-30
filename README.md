CloudIO
=========

Installation & Building
--------------
Installing CloudIO 
```
mkdir -p ~/Buildspace/cloudio
export GOROOT=/usr/local/go
export GOPATH=~/Buildspace/cloudio
export GOBIN=~/bin
git clone https://github.com/clearpath-networks/cloudio.git
cd $GOPATH/src/stormstack.org/cloudio
go get .
go install
```
Once the build is complete, it places the executable `cloudio` inside $GOBIN
The other files required to run CloudIO is explained in <i>Running</i> section

Running
---------
CloudIO src folder contains configuration files with an extension cfg
Here is a sample configuration for running it on local

[application]<br>
name=CloudIO


[default]<br>

log-conf=log/seelog.xml

[server]<br>

host=localhost
port=8080

[database]<br>

db-name=cloudio
username=cloudio
password=password
host=localhost
port=3306


[external]<br>

vertex-url=http://localhost:9090/VertexPlatform


[web-app]<br>

context-path=/CloudIO

CloudIO looks for the configuration in the following order

    1. From the command line
    cloudio -profile localhost.cfg
    2. From the environment variable CLOUDIOCONFIG.
    export CLOUDIOCONFIG=path of the configuration file
    3. From the default location.
    /etc/cloudio/default.cfg
In case 2 & 3  just the command is enough
```
    $ cloudio
``` 

### License
The MIT License (MIT)

Copyright (c) 2014 StormStack

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all 
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE 
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE 
SOFTWARE.


### Contributors
* [Devananda Reddy](DReddy@clearpathnet.com)
* [Ravi Chunduru](RChunduru@clearpathnet.com)
* [Peter Lee](plee@Clearpathnet.com)


## Let's collaborate
Feel free to [file issues](https://github.com/stormstack/cloudio/issues) if you have feedback/questions/suggestions
