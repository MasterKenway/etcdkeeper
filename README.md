## ETCD Keeper
* Lightweight etcd web client.
* Support etcd 2.x and etcd 3.x.
* The server uses the etcd go client interface, and the server compiles with the etcd client package.
* Based easyui framework to achieve(easyui license [easyui website](http://www.jeasyui.com)).

## build
```
go build -o etcdkeeper.bin src/etcdkeeper/main.go
```

## Usage
* Run etcdkeeper.exe (windows version)
```
Usage of etcdkeeper.exe:
  -bind string
        etcdkeeper bind ip:port address (default "0.0.0.0:8000")
  -cacert string
        CA certificates file
  -cert string
        certificate file
  -etcd string
        connect etcd server endpoints (default "127.0.0.1:2379")
  -help
        usage
  -key string
        key file
  -sep string
        separator (default "/")
  -tls
        enable tls to connect etcd server
```

# run
```
etcdkeeper.exe -etcd 192.168.0.100:2379
```

* Open your browser and enter the address: http://127.0.0.1:8000/etcdkeeper
* Click on the version of the title to select the version of ETCD. The default is V3. Reopening will remember your choice.
* Right click on the tree node to add or delete.
* Etcd address can be modified by default to the localhost. If you change, press the Enter key to take effect.

## Features
* Etcd client view, Add, update or delete nodes.
* Content edits use the ace editor([Ace editor](https://ace.c9.io)). Support toml,ini,yaml,json,xml and so on to highlight view.
* Content format. (Currently only support json, Other types can be extended later) Thanks jim3ma for his contribution.[@jim3ma]( https://github.com/jim3ma)

## Future Features
* Import and export.
* Content simple filter search.

## Special Note
Because the etcdv3 version uses the new storage concept, without the catalog concept, the client uses the previous default "/" delimiter to view. See the documentation for etcdv3 [clientv3 doc](https://godoc.org/github.com/coreos/etcd/clientv3).

## License
MIT
