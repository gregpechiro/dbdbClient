package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/cagnosolutions/dbdb"
	"github.com/cagnosolutions/mockdb"
	"github.com/cagnosolutions/web"
	"github.com/cagnosolutions/web/tmpl"
)

//var db = mockdb.NewMockDB("backup.json", 5)

var rpc = dbdb.NewClient()
var ts = tmpl.NewTemplateStore(true)
var config = mockdb.NewMockDB("config.json", 5)

func main() {
	mux := web.NewMux("CTIXID", (web.HOUR / 2))

	// db managment
	mux.Get("/", Root)
	mux.Post("/connection", AddConnection)
	mux.Post("/connection/:db", SaveConnection)
	mux.Post("/connection/:db/del", DelConnection)

	mux.Get("/connect/:db", Connect)
	mux.Get("/disconnect", Disconnect)

	// store managment
	mux.Get("/new", NewStore)
	mux.Post("/new", SaveStore)
	mux.Get("/:store", Store)
	mux.Post("/:store", DelStore)

	// store search
	mux.Get("/:store/search", Search)
	mux.Post("/:store/search", MakeSearch)
	mux.Post("/:store/search/save", SaveSearch)

	// record managment
	mux.Get("/:store/new", NewRecord)
	mux.Post("/:store/add", AddRecord)
	mux.Get("/:store/import", ImportRecords)
	mux.Post("/:store/import", UploadRecords)
	mux.Get("/:store/:record", Record)
	mux.Post("/:store/:record", SaveRecord)
	mux.Post("/:store/:record/del", DelRecord)

	mux.Serve(":8080")
}

// GET render all saved DBs
func Root(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	// Not connected display all DBs
	if !rpc.State {
		ts.Render(w, "index.tmpl", tmpl.Model{
			msgk:    msgv,
			"dbs":   GetSavedDBs(),
			"conns": config.Get("db", "connections"),
		})
		return
	}
	ts.Render(w, "db.tmpl", tmpl.Model{
		msgk:     msgv,
		"db":     c.Get("db"),
		"stores": rpc.GetAllStoreStats(),
	})
	return
}

// POST save new db connection
func AddConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var savedConns map[string]string
	config.GetAs("db", "connections", &savedConns)
	if savedConns == nil {
		savedConns = make(map[string]string)
	}
	savedConns[r.FormValue("name")] = r.FormValue("address")
	config.Set("db", "connections", savedConns)
	c.SetFlash("alertSuccess", "Successfully added connection")
	http.Redirect(w, r, "/", 303)
	return
}

func SaveConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var savedConns map[string]string
	config.GetAs("db", "connections", &savedConns)
	if savedConns == nil {
		savedConns = make(map[string]string)
	}
	savedConns[r.FormValue("name")] = r.FormValue("address")
	if r.FormValue("name") != r.FormValue("oldName") {
		delete(savedConns, r.FormValue("oldName"))
	}
	config.Set("db", "connections", savedConns)
	c.SetFlash("alertSuccess", "Successfully updated connection")
	http.Redirect(w, r, "/", 303)
	return
}

func DelConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var savedConns map[string]string
	config.GetAs("db", "connections", &savedConns)
	if savedConns == nil {
		savedConns = make(map[string]string)
	}
	delete(savedConns, c.GetPathVar("db"))
	config.Set("db", "connections", savedConns)
	c.SetFlash("alertSuccess", "Successfully deleted connection")
	http.Redirect(w, r, "/", 303)
	return
}

func Connect(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var addrs map[string]string
	config.GetAs("db", "connections", &addrs)
	if addr, ok := addrs[c.GetPathVar("db")]; ok && addr != "" {
		if err := rpc.Connect(addr); err == nil {
			c.SetFlash("alertSuccess", "Successfully connected to database")
			c.Set("db", c.GetPathVar("db"))
			http.Redirect(w, r, "/", 303)
			return
		}
	}
	rpc.State = false
	c.SetFlash("alertError", "Error connecting to the database")
	http.Redirect(w, r, "/", 303)
	return
}

func Disconnect(w http.ResponseWriter, r *http.Request, c *web.Context) {
	rpc.Disconnect()
	c.SetFlash("alertSuccess", "Dissconnected")
	http.Redirect(w, r, "/", 303)
	return
}

// POST add new tore to connected db
func NewStore(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	ts.Render(w, "newStore.tmpl", tmpl.Model{
		msgk: msgv,
		//"db":     c.GetPathVar("db"),
		"stores": rpc.GetAllStoreStats(),
	})
}

func SaveStore(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	name := r.FormValue("name")
	if ok := rpc.AddStore(name); ok {
		c.SetFlash("alertSuccess", "Successfully saved store")
	} else {
		c.SetFlash("alertError", "Error saving store")
	}
	http.Redirect(w, r, "/", 303)
	return
}

// GET render specified store from specified DB
func Store(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	if !rpc.HasStore(c.GetPathVar("store")) {
		c.SetFlash("alertError", "Invalid store")
		http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("db")), 303)
		return
	}
	ts.Render(w, "store.tmpl", tmpl.Model{
		msgk:          msgv,
		"savedSearch": GetSavedSearches(c.GetPathVar("store")),
		"db":          c.Get("db"),
		"stores":      rpc.GetAllStoreStats(),
		"store":       rpc.GetAll(c.GetPathVar("store")),
		"storeName":   c.GetPathVar("store"),
	})
	return
}

func DelStore(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if !rpc.DelStore(c.GetPathVar("store")) {
		c.SetFlash("alertError", "Error deleteing store")
	} else {
		c.SetFlash("alertSuccess", "Successfully deleted store")
	}
	http.Redirect(w, r, "/", 303)
	return
}

// GET render complex search for specified store from specified DB
func Search(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	if !rpc.HasStore(c.GetPathVar("store")) {
		c.SetFlash("alertError", "Invalid store")
		http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("db")), 303)
		return
	}
	var query map[string]string
	config.GetAs("search", c.GetPathVar("store"), &query)
	ts.Render(w, "search.tmpl", tmpl.Model{
		msgk:          msgv,
		"savedSearch": GetSavedSearches(c.GetPathVar("store")),
		"query":       query[r.FormValue("query")],
		"db":          c.Get("db"),
		"stores":      rpc.GetAllStoreStats(),
		"store":       rpc.GetAll(c.GetPathVar("store")),
		"storeName":   c.GetPathVar("store"),
	})
	return
}

// POST submit simple search for specified store from specified DB
func MakeSearch(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	// TODO:  needs refactored to work with dbdb
	/*
		msgk, msgv := c.GetFlash()
		query := r.FormValue("query")
		var qry map[string]interface{}
		json.Unmarshal([]byte(query), &qry)
		var result []map[string]interface{}
		db.QueryAll(c.GetPathVar("store"), qry, &result)
		ts.Render(w, "store.tmpl", tmpl.Model{
			msgk:          msgv,
			"savedSearch": GetSavedSearches(c.GetPathVar("store")),
			"query":       query,
			"db":          c.Get("db"),
			"stores":      OrderStores(db.GetAllStores()),
			"store":       result,
			"storeName":   c.GetPathVar("store"),
		})
	*/
	http.Redirect(w, r, "/"+c.GetPathVar("store"), 303)
	return
}

// POST
func SaveSearch(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	var savedSearch map[string]string
	config.GetAs("search", c.GetPathVar("store"), &savedSearch)
	if savedSearch == nil {
		savedSearch = make(map[string]string)
	}
	savedSearch[r.FormValue("name")] = r.FormValue("search")
	config.Set("search", c.GetPathVar("store"), savedSearch)
	c.SetFlash("alertSuccess", "Successfully saved search")
	http.Redirect(w, r, fmt.Sprintf("/%s/search", c.GetPathVar("store")), 303)
	return
}

// GET render empty record for specified store from specified DB
func NewRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	ts.Render(w, "record.tmpl", tmpl.Model{
		msgk:        msgv,
		"db":        c.Get("db"),
		"stores":    rpc.GetAllStoreStats(),
		"storeName": c.GetPathVar("store"),
		"record":    "",
	})
	return
}

// POST submit new record to specified store from specified DB to add
func AddRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	rpc.Add(c.GetPathVar("store"), rec)
	c.SetFlash("alertSuccess", "Successfully added record")

	http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("store")), 303)
	return
}

func ImportRecords(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	ts.Render(w, "import.tmpl", tmpl.Model{
		msgk:        msgv,
		"db":        c.Get("db"),
		"stores":    rpc.GetAllStoreStats(),
		"storeName": c.GetPathVar("store"),
	})
}

func UploadRecords(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	r.ParseMultipartForm(32 << 20) // 32 MB
	file, handler, err := r.FormFile("data")
	if err != nil || len(handler.Header["Content-Type"]) < 1 {
		fmt.Println(err)
		c.SetFlash("alertError", "Error uploading file")
		http.Redirect(w, r, "/"+c.GetPathVar("store")+"/import", 303)
		return
	}
	defer file.Close()
	var m []map[string]interface{}
	switch handler.Header["Content-Type"][0] {
	case "application/json":
		dec := json.NewDecoder(file)
		err = dec.Decode(&m)
	case "text/xml":

	case "text/csv":
		m, err = DecodeCSV(file)
	default:
		c.SetFlash("alertError", "Error uploading file")
		http.Redirect(w, r, "/"+c.GetPathVar("store")+"/import", 303)
		return
	}

	if err != nil {
		fmt.Println(err)
		c.SetFlash("alertError", "Error uploading file")
		http.Redirect(w, r, "/"+c.GetPathVar("store")+"/import", 303)
		return
	}

	for _, doc := range m {
		SanitizeMap(&doc)
		rpc.Add(c.GetPathVar("store"), doc)
	}
	c.SetFlash("alertSuccess", "Successfully imported data")
	http.Redirect(w, r, "/"+c.GetPathVar("store"), 303)
	return
}

// GET render specified record from specified store from specified DB
func Record(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	msgk, msgv := c.GetFlash()
	record := rpc.Get(c.GetPathVar("store"), GetId(c.GetPathVar("record")))
	if record == nil {
		c.SetFlash("alertError", "Invalid Record")
		http.Redirect(w, r, fmt.Sprintf("/%s/%s", c.GetPathVar("db"), c.GetPathVar("store")), 303)
		return
	}
	ts.Render(w, "record.tmpl", tmpl.Model{
		msgk:        msgv,
		"db":        c.Get("db"),
		"stores":    rpc.GetAllStoreStats(),
		"storeName": c.GetPathVar("store"),
		"record":    rpc.Get(c.GetPathVar("store"), GetId(c.GetPathVar("record"))),
	})
	return
}

// POST submit existing record to specified store from specified DB to save
func SaveRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	if ok := rpc.Set(c.GetPathVar("store"), GetId(c.GetPathVar("record")), rec); ok {
		c.SetFlash("alertSuccess", "Successfully saved Record")
	} else {
		c.SetFlash("alertError", "Error saving record")
	}
	http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("store")), 303)
	return
}

// POST submit existing record id to specified store from specified DB to delete
func DelRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.State {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	rpc.Del(c.GetPathVar("store"), GetId(c.GetPathVar("record")))
	c.SetFlash("alertSuccess", "Successfully deleted record")
	http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("store")), 303)
	return
}

// helper functions

func GetSavedSearches(searchStore string) []string {
	var savedSearch map[string]string
	config.GetAs("search", searchStore, &savedSearch)
	var keys []string
	for k := range savedSearch {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func GetSavedDBs() []string {
	var savedConns map[string]string
	config.GetAs("db", "connections", &savedConns)
	var dbs []string
	for k := range savedConns {
		dbs = append(dbs, k)
	}
	sort.Strings(dbs)
	return dbs
}

func GetId(s string) uint64 {
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}

func DecodeCSV(data io.Reader) ([]map[string]interface{}, error) {
	fd := csv.NewReader(data)
	keys, err := fd.Read()
	if err != nil || err == io.EOF {
		log.Println(err)
		return nil, err
	}
	var docs []map[string]interface{}
	for {
		row, err := fd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return nil, err
		}
		doc := make(map[string]interface{})
		for i := 0; i < len(keys); i++ {
			doc[keys[i]] = row[i]
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func SanitizeMap(m *map[string]interface{}) {
	for k, v := range *m {
		delete(*m, k)
		(*m)[strings.ToLower(k[0:1])+k[1:]] = v
	}
	runtime.GC()
}

/*
type StoreStat struct {
	Name string
	Docs uint64
}

type StoreStatSorted []StoreStat

func (sss StoreStatSorted) Len() int {
	return len(sss)
}

func (sss StoreStatSorted) Less(i, j int) bool {
	return sss[i].Name < sss[j].Name
}

func (sss StoreStatSorted) Swap(i, j int) {
	sss[i], sss[j] = sss[j], sss[i]
}

func OrderStores(stores map[string]*map[string]interface{}) []StoreStat {
	var sss StoreStatSorted
	for k, v := range stores {
		sss = append(sss, StoreStat{Name: k, Docs: uint64(len(*v))})
	}
	sort.Sort(sss)
	return sss
}

func OrderStore(store map[string]interface{}) []map[string]interface{} {
	var ss []string
	var ret []map[string]interface{}
	for k := range store {
		ss = append(ss, k)
	}
	sort.Strings(ss)
	for _, v := range ss {
		ret = append(ret, store[v].(map[string]interface{}))
	}
	return ret
}

func OrderKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func GetCrumbs(path string) []string {
	ss := strings.Split(path, "/")
	return ss[1:len(ss)]
}
*/
