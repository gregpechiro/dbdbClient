package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cagnosolutions/mockdb"
	"github.com/cagnosolutions/web"
	"github.com/cagnosolutions/web/tmpl"
)

var db = mockdb.NewMockDB("backup.json", 5)

var ts = tmpl.NewTemplateStore(true)

var config = mockdb.NewMockDB("config.json", 5)

func main() {
	mux := web.NewMux("CTIXID", (web.HOUR / 2))
	mux.Get("/", Root)
	mux.Get("/:db", DB)
	mux.Get("/:db/:store", Store)
	mux.Get("/:db/:store/new", NewRecord)
	mux.Get("/:db/:store/search", Search)
	mux.Post("/:db/:store/search", PostSearch)
	mux.Post("/:db/:store/search/save", SaveSearch)
	mux.Post("/:db/:store/add", AddRecord)
	mux.Get("/:db/:store/:record", Record)
	mux.Post("/:db/:store/:record", SaveRecord)
	mux.Post("/:db/:store/:record/del", DelRecord)
	mux.Serve(":8080")
}

// GET render all saved DBs
func Root(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	ts.Render(w, "index.tmpl", tmpl.Model{
		msgk:  msgv,
		"dbs": []string{"test"},
	})
}

// GET render specified DB
func DB(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	ts.Render(w, "db.tmpl", tmpl.Model{
		msgk:     msgv,
		"db":     c.GetPathVar("db"),
		"stores": OrderStores(db.GetAllStores()),
	})
	return
}

// GET render specified store from specified DB
func Store(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	ts.Render(w, "store.tmpl", tmpl.Model{
		msgk:          msgv,
		"savedSearch": GetSavedSearches(c.GetPathVar("store")),
		"db":          c.GetPathVar("db"),
		"stores":      OrderStores(db.GetAllStores()),
		"store":       OrderStore(*db.GetStore(c.GetPathVar("store"))),
		"storeName":   c.GetPathVar("store"),
	})
	return
}

// GET render empty record for specified store from specified DB
func NewRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	ts.Render(w, "record.tmpl", tmpl.Model{
		msgk:        msgv,
		"db":        c.GetPathVar("db"),
		"stores":    OrderStores(db.GetAllStores()),
		"storeName": c.GetPathVar("store"),
		"record":    "",
	})
	return
}

// GET render complex search for specified store from specified DB
func Search(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	var query map[string]string
	config.GetAs("search", c.GetPathVar("store"), &query)
	ts.Render(w, "search.tmpl", tmpl.Model{
		msgk:          msgv,
		"savedSearch": GetSavedSearches(c.GetPathVar("store")),
		"query":       query[r.FormValue("query")],
		"db":          c.GetPathVar("db"),
		"stores":      OrderStores(db.GetAllStores()),
		"store":       OrderStore(*db.GetStore(c.GetPathVar("store"))),
		"storeName":   c.GetPathVar("store"),
	})
	return
}

// GET render specified record from specified store from specified DB
func Record(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()
	ts.Render(w, "record.tmpl", tmpl.Model{
		msgk:        msgv,
		"db":        c.GetPathVar("db"),
		"stores":    OrderStores(db.GetAllStores()),
		"storeName": c.GetPathVar("store"),
		"record":    db.Get(c.GetPathVar("store"), c.GetPathVar("record")),
	})
	return
}

// POST submit simple search for specified store from specified DB
func PostSearch(w http.ResponseWriter, r *http.Request, c *web.Context) {
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
		"db":          c.GetPathVar("db"),
		"stores":      OrderStores(db.GetAllStores()),
		"store":       result,
		"storeName":   c.GetPathVar("store"),
	})
	return
}

// POST
func SaveSearch(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var savedSearch map[string]string
	config.GetAs("search", c.GetPathVar("store"), &savedSearch)
	if savedSearch == nil {
		savedSearch = make(map[string]string)
	}
	savedSearch[r.FormValue("name")] = r.FormValue("search")
	config.Set("search", c.GetPathVar("store"), savedSearch)
	c.SetFlash("alertSuccess", "Successfully saved search")
	http.Redirect(w, r, fmt.Sprintf("/%s/%s/search", c.GetPathVar("db"), c.GetPathVar("store")), 303)
	return
}

// POST submit new record to specified store from specified DB to add
func AddRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	rec["Id"] = mockdb.UUID4()
	db.Set(c.GetPathVar("store"), rec["Id"].(string), rec)
	c.SetFlash("alertSuccess", "Successfully saved record")
	http.Redirect(w, r, fmt.Sprintf("/%s/%s", c.GetPathVar("db"), c.GetPathVar("store")), 303)
	return
}

// POST submit existing record to specified store from specified DB to save
func SaveRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	rec["Id"] = c.GetPathVar("record")
	db.Set(c.GetPathVar("store"), rec["Id"].(string), rec)
	c.SetFlash("alertSuccess", "Successfully saved Record")
	http.Redirect(w, r, fmt.Sprintf("/%s/%s", c.GetPathVar("db"), c.GetPathVar("store")), 303)
	return
}

// POST submit existing record id to specified store from specified DB to delete
func DelRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	db.Del(c.GetPathVar("store"), c.GetPathVar("record"))
	c.SetFlash("alertSuccess", "Successfully deleted record")
	http.Redirect(w, r, fmt.Sprintf("/%s/%s", c.GetPathVar("db"), c.GetPathVar("store")), 303)
	return
}

// helper functions

func GetSavedSearches(searchStore string) []string {
	var savedSearch map[string]string
	config.GetAs("search", searchStore, &savedSearch)
	return OrderKeys(savedSearch)
}

type StoreStat struct {
	Name     string
	DocCount uint64
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
		sss = append(sss, StoreStat{Name: k, DocCount: uint64(len(*v))})
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
