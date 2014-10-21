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
