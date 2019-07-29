package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net"

	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
)

var (
	cli  *clientv3.Client // v3 client
	kapi client.KeysAPI   // v2 client
)

var (
	separator string
	usetls    bool
	cacert    string
	cert      string
	keyfile   string
	address   string
	endpoints string
	help      bool
)

func init() {
	flag.StringVar(&separator, "sep", "/", "separator")
	flag.BoolVar(&usetls, "tls", false, "enable tls to connect etcd server")
	flag.StringVar(&cacert, "cacert", "", "CA certificates file")
	flag.StringVar(&cert, "cert", "", "certificate file")
	flag.StringVar(&keyfile, "key", "", "key file")
	flag.StringVar(&address, "bind", "0.0.0.0:8000", "etcdkeeper bind ip:port address")
	flag.StringVar(&endpoints, "etcd", "127.0.0.1:2379", "connect etcd server endpoints")
	flag.BoolVar(&help, "help", false, "usage")
}

func main() {

	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	// v2
	http.HandleFunc("/v2/separator", getSeparator)
	http.HandleFunc("/v2/connect", connectV2)
	http.HandleFunc("/v2/put", putV2)
	http.HandleFunc("/v2/get", getV2)
	http.HandleFunc("/v2/delete", delV2)

	// dirctory mode
	http.HandleFunc("/v2/getpath", getPathV2)

	// v3
	http.HandleFunc("/v3/separator", getSeparator)
	http.HandleFunc("/v3/connect", connect)
	http.HandleFunc("/v3/put", put)
	http.HandleFunc("/v3/get", get)
	http.HandleFunc("/v3/delete", del)

	// dirctory mode
	http.HandleFunc("/v3/getpath", getPath)

	http.Handle("/", http.FileServer(http.Dir("./assets"))) // view static directory

	log.Printf("listening on %s\n", address)

	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// v2 api
func connectV2(w http.ResponseWriter, r *http.Request) {
	var err error
	
	urls := make([]string, 0)
	epts := strings.Split(endpoints, ",")
	for _, v := range epts {
		if usetls {
			urls = append(urls, "https://"+v)
		}else {
			urls = append(urls, "http://"+v)
		}
	}

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if usetls {
		tlsInfo := transport.TLSInfo{
			CertFile:      cert,
			KeyFile:       keyfile,
			TrustedCAFile: cacert,
			InsecureSkipVerify: true,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	defaultHTTPTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}

	var defaultTransport client.CancelableTransport = defaultHTTPTransport

	cfg := client.Config{
		Endpoints:               urls,
		HeaderTimeoutPerRequest: 5 * time.Second,
		Transport:               defaultTransport,
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Println(r.Method, "v2", "connect fail.")
		io.WriteString(w, string(err.Error()))
	} else {
		kapi = client.NewKeysAPI(c)
		log.Println(r.Method, "v2", "connect success.")
		io.WriteString(w, "ok")
	}
}

func putV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	dir := r.FormValue("dir")
	log.Println("PUT", "v2", key)

	var isDir bool
	if dir != "" {
		isDir, _ = strconv.ParseBool(dir)
	}
	var err error
	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			log.Println(err.Error())
		}
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{TTL: time.Duration(sec) * time.Second, Dir: isDir})
	} else {
		_, err = kapi.Set(context.Background(), key, value, &client.SetOptions{Dir: isDir})
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive: true, Sort: true}); err != nil {
			data["errorCode"] = err.Error()
		} else {
			if resp.Node != nil {
				node := make(map[string]interface{})
				node["key"] = resp.Node.Key
				node["value"] = resp.Node.Value
				node["dir"] = resp.Node.Dir
				node["ttl"] = resp.Node.TTL
				node["createdIndex"] = resp.Node.CreatedIndex
				node["modifiedIndex"] = resp.Node.ModifiedIndex
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func getV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v2", key)

	if resp, err := kapi.Get(context.Background(), key, &client.GetOptions{Recursive: true, Sort: true}); err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp.Node == nil {
			data["errorCode"] = 500
			data["message"] = "The node does not exist."
		} else {
			data["node"] = getNode(resp.Node)
		}
	}
	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func getNode(node *client.Node) map[string]interface{} {
	nm := make(map[string]interface{})
	nm["key"] = node.Key
	nm["value"] = node.Value
	nm["dir"] = node.Dir
	nm["ttl"] = node.TTL
	nm["createdIndex"] = node.CreatedIndex
	nm["modifiedIndex"] = node.ModifiedIndex
	nm["nodes"] = make([]map[string]interface{}, 0)
	if len(node.Nodes) != 0 {
		for _, n := range node.Nodes {
			nm["nodes"] = append(nm["nodes"].([]map[string]interface{}), getNode(n))
		}
	}
	return nm
}

func delV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v2", key)

	isDir, _ := strconv.ParseBool(dir)
	if isDir {
		if _, err := kapi.Delete(context.Background(), key, &client.DeleteOptions{Recursive: true, Dir: true}); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	} else {
		if _, err := kapi.Delete(context.Background(), key, nil); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}

	io.WriteString(w, "ok")
}

func getPathV2(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	log.Println("GET", "v2", key)
	getV2(w, r)
}

func strings_compare(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, bv := range b {
		var flag bool
		for _, av := range a {
			if 0 == strings.Compare(bv, av) {
				flag = true
			}
		}

		if flag == false {
			return false
		}
	}

	return true
}

// v3 api
func connect(w http.ResponseWriter, r *http.Request) {
	if cli != nil {
		etcdHost := cli.Endpoints()
		etcdEnds := strings.Split(endpoints, ",")
		if strings_compare(etcdHost, etcdEnds) {
			io.WriteString(w, "running")
			return
		} else {
			if err := cli.Close(); err != nil {
				log.Println(err.Error())
			}
		}
	}

	var err error

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if usetls {
		tlsInfo := transport.TLSInfo{
			CertFile:      cert,
			KeyFile:       keyfile,
			TrustedCAFile: cacert,
			InsecureSkipVerify: true,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	cli, err = clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(endpoints, ","),
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	})

	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		io.WriteString(w, string(err.Error()))
	} else {
		log.Println(r.Method, "v3", "connect success.")
		io.WriteString(w, "ok")
	}
}

func getSeparator(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, separator)
}

func put(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	value := r.FormValue("value")
	ttl := r.FormValue("ttl")
	log.Println("PUT", "v3", key)

	var err error
	data := make(map[string]interface{})
	if ttl != "" {
		var sec int64
		sec, err = strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			log.Println(err.Error())
		}
		var leaseResp *clientv3.LeaseGrantResponse
		leaseResp, err = cli.Grant(context.TODO(), sec)
		_, err = cli.Put(context.Background(), key, value, clientv3.WithLease(leaseResp.ID))
	} else {
		_, err = cli.Put(context.Background(), key, value)
	}
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix()); err != nil {
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func get(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	data := make(map[string]interface{})
	log.Println("GET", "v3", key)

	if resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix()); err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if r.FormValue("prefix") == "true" {
			pnode := make(map[string]interface{})
			pnode["key"] = key
			pnode["nodes"] = make([]map[string]interface{}, 0)
			for _, kv := range resp.Kvs {
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				if key == string(kv.Key) {
					node["ttl"] = getTTL(kv.Lease)
				} else {
					node["ttl"] = 0
				}
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				nodes := pnode["nodes"].([]map[string]interface{})
				pnode["nodes"] = append(nodes, node)
			}
			data["node"] = pnode
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = string(kv.Key)
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			} else {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			}
		}
	}
	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func getPath(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	log.Println("GET", "v3", key)
	var (
		data = make(map[string]interface{})
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		*/
		all       = make(map[int][]map[string]interface{})
		min       int
		max       int
		prefixKey string
	)
	// parent
	presp, err := cli.Get(context.Background(), key)
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
		dataByte, _ := json.Marshal(data)
		io.WriteString(w, string(dataByte))
		return
	}
	if key == separator {
		min = 1
		prefixKey = separator
	} else {
		min = len(strings.Split(key, separator))
		prefixKey = key + separator
	}
	max = min
	all[min] = []map[string]interface{}{{"key": key}}
	if presp.Count != 0 {
		all[min][0]["value"] = string(presp.Kvs[0].Value)
		all[min][0]["ttl"] = getTTL(presp.Kvs[0].Lease)
		all[min][0]["createdIndex"] = presp.Kvs[0].CreateRevision
		all[min][0]["modifiedIndex"] = presp.Kvs[0].ModRevision
	}
	all[min][0]["nodes"] = make([]map[string]interface{}, 0)

	//child
	resp, err := cli.Get(context.Background(), prefixKey, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
		dataByte, _ := json.Marshal(data)
		io.WriteString(w, string(dataByte))
		return
	}

	for _, kv := range resp.Kvs {
		if string(kv.Key) == separator {
			continue
		}
		keys := strings.Split(string(kv.Key), separator) // /foo/bar
		var begin bool
		for i := range keys { // ["", "foo", "bar"]
			k := strings.Join(keys[0:i+1], separator)
			if k == "" {
				continue
			}
			if key == separator {
				begin = true
			} else if k == key {
				begin = true
				continue
			}
			if begin {
				node := map[string]interface{}{"key": k}
				if node["key"].(string) == string(kv.Key) {
					node["value"] = string(kv.Value)
					if key == string(kv.Key) {
						node["ttl"] = getTTL(kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
				}
				level := len(strings.Split(k, separator))
				if level > max {
					max = level
				}

				if _, ok := all[level]; !ok {
					all[level] = make([]map[string]interface{}, 0)
				}
				levelNodes := all[level]
				var isExist bool
				for _, n := range levelNodes {
					if n["key"].(string) == k {
						isExist = true
					}
				}
				if !isExist {
					node["nodes"] = make([]map[string]interface{}, 0)
					all[level] = append(all[level], node)
				}
			}
		}
	}

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 {
					pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
					pa["dir"] = true
				} else {
					if strings.HasPrefix(a["key"].(string), pa["key"].(string)+separator) {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					}
				}
			}
		}
	}
	data = all[min][0]
	if dataByte, err := json.Marshal(map[string]interface{}{"node": data}); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func del(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	dir := r.FormValue("dir")
	log.Println("DELETE", "v3", key)

	if _, err := cli.Delete(context.Background(), key); err != nil {
		io.WriteString(w, err.Error())
		return
	}

	if dir == "true" {
		if _, err := cli.Delete(context.Background(), key+separator, clientv3.WithPrefix()); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, "ok")
}

func getTTL(lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}
