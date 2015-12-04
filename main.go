package main

import (
	"archive/tar"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cagnosolutions/dbdb"
	"github.com/cagnosolutions/mockdb"
	"github.com/cagnosolutions/web"
	"github.com/cagnosolutions/webc/tmpl"
)

var rpc = dbdb.NewClient()

var ts *web.TmplCache
var config = mockdb.NewMockDB("config.json", 5)

func init() {
	web.Funcs["title"] = strings.Title
	web.Funcs["json"] = func(v interface{}) string {
		b, err := json.Marshal(v)
		if err != nil {
			log.Println(err)
		}
		return string(b)
	}
	web.Funcs["pretty"] = func(v interface{}) string {
		b, err := json.MarshalIndent(v, "", "\t")
		if err != nil {
			log.Println(err)
		}
		return string(b)
	}
	ts = web.NewTmplCache()
}

func main() {
	mux := web.NewMux()
	mux.AddRoutes(root, addConnection, saveConnection, delConnection, connect, disconnect)
	mux.AddRoutes(saveStore, exportDB, importDB, eraseDB, getStore, delStore, saveQuery)
	mux.AddRoutes(newRecord, addRecord, importStore, getRecord, saveRecord, delRecord)
	log.Println(http.ListenAndServe(":8888", mux))
}

var root = web.Route{"GET", "/", func(w http.ResponseWriter, r *http.Request) {
	// Not connected display all DBs
	if !rpc.Alive() {
		ts.Render(w, r, "index.tmpl", tmpl.Model{
			"dbs":   GetSavedDBs(),
			"conns": config.GetStore("connections"),
		})
		return
	}

	db := web.Get(r, "db")

	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	ts.Render(w, r, "db.tmpl", tmpl.Model{
		"db":     db,
		"stores": rpc.GetAllStoreStats(),
	})
	return
}}

// POST add new db connection
var addConnection = web.Route{"POST", "/connection", func(w http.ResponseWriter, r *http.Request) {
	var connection map[string]string
	config.GetAs("connections", r.FormValue("name"), &connection)
	if connection == nil {
		connection = make(map[string]string)
	}
	connection["address"] = r.FormValue("address")
	connection["token"] = r.FormValue("token")
	config.Set("connections", r.FormValue("name"), connection)
	web.SetSuccessRedirect(w, r, "/", "Successfully added connection")
	return
}}

// POST update existing DB connection
var saveConnection = web.Route{"POST", "/connection/save", func(w http.ResponseWriter, r *http.Request) {
	var connection map[string]string
	ok := config.GetAs("connections", r.FormValue("name"), &connection)
	if r.FormValue("name") != r.FormValue("oldName") {
		if ok {
			web.SetErrorRedirect(w, r, "/", "Error connection name already exists")
			return
		}
		config.Del("connections", r.FormValue("oldName"))
	}
	if connection == nil {
		connection = make(map[string]string)
	}
	connection["address"] = r.FormValue("address")
	connection["token"] = r.FormValue("token")
	config.Set("connections", r.FormValue("name"), connection)
	web.SetSuccessRedirect(w, r, "/", "Successfully updated connection")
	return
}}

// POST delete saved DB connection
var delConnection = web.Route{"POST", "/connection/:db/del", func(w http.ResponseWriter, r *http.Request) {
	config.Del("connections", r.FormValue(":db"))
	web.SetSuccessRedirect(w, r, "/", "Successfully deleted connection")
	return
}}

// GET connect to DB
var connect = web.Route{"GET", "/connect/:db", func(w http.ResponseWriter, r *http.Request) {
	var connection map[string]string
	db := r.FormValue(":db")
	if config.GetAs("connections", db, &connection) {
		if address, ok := connection["address"]; ok && address != "" {
			if rpc.Connect(address, connection["token"]) {
				web.Put(w, "db", db)
				web.SetSuccessRedirect(w, r, "/", "Successfully connected to database")
				return
			}
		}
	}
	web.SetErrorRedirect(w, r, "/", "Error connecting to the database")
	return
}}

// GET disconnect from DB
var disconnect = web.Route{"GET", "/disconnect", func(w http.ResponseWriter, r *http.Request) {
	rpc.Disconnect()
	web.Delete(w, r, "db")
	web.SetSuccessRedirect(w, r, "/", "Disconnected")
	return
}}

// GET create .tar file from connected DB of all its stores
// and records and return download link
var exportDB = web.Route{"GET", "/export", func(w http.ResponseWriter, r *http.Request) {
	var response = make(map[string]interface{})
	if !rpc.Alive() {
		response["complete"] = true
		response["path"] = "/"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		response["complete"] = true
		response["path"] = "/disconnect"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	response["complete"] = false
	path := "static/export/"
	fullPath := path + strings.Replace(db, " ", "_", -1) + "_" + time.Now().Format("2006-01-02") + ".tar"
	err := os.MkdirAll(path, 0755)
	if HasError("/", err, w, response) {
		return
	}
	tarFile, err := os.Create(fullPath)
	if HasError("/", err, w, response) {
		return
	}
	defer tarFile.Close()
	tw := tar.NewWriter(tarFile)
	defer tw.Close()
	exportData := rpc.Export()
	var dat map[string][]map[string]interface{}
	err = json.Unmarshal(exportData, &dat)
	if HasError("/", err, w, response) {
		return
	}
	for store, docs := range dat {
		b, err := json.Marshal(docs)
		if HasError("/", err, w, response) {
			return
		}
		hdr := &tar.Header{
			Name: store + ".json",
			Mode: 0600,
			Size: int64(len(b)),
		}
		err = tw.WriteHeader(hdr)
		if HasError("/", err, w, response) {
			return
		}
		_, err = tw.Write(b)
		if HasError("/", err, w, response) {
			return
		}
	}

	response["complete"] = true
	response["path"] = fullPath
	b, _ := json.Marshal(response)
	if HasError("/", err, w, response) {
		return
	}
	fmt.Fprintf(w, "%s", b)
	return
}}

// POST upload .tar file of .json files to add stores and records to connected DB
var importDB = web.Route{"POST", "/import", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	if web.Get(r, "db") == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}

	r.ParseMultipartForm(32 << 20) // 32 MB
	tarFile, handler, err := r.FormFile("import")
	if err != nil || len(handler.Header["Content-Type"]) < 1 {
		fmt.Printf("dbdbClient >> ImportDB() >> lenfile header: %v", err)
		web.SetErrorRedirect(w, r, "/import", "Error Uploading file")
		return
	}
	defer tarFile.Close()

	if handler.Header["Content-Type"][0] != "application/z-tar" {
		web.SetErrorRedirect(w, r, "/", "Incorrect file type")
		return
	}

	tarReader := tar.NewReader(tarFile)
	tarData := make(map[string][]map[string]interface{})
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			fmt.Printf("dbdbClient >> ImportDB() >> tarReader.Next(): %v", err)
			web.SetErrorRedirect(w, r, "/", "Error uploading file")
			return
		}

		buf := new(bytes.Buffer)
		io.Copy(buf, tarReader)
		b := buf.Bytes()
		var st []map[string]interface{}
		json.Unmarshal(b, &st)
		tarData[strings.Split(hdr.Name, ".")[0]] = st
	}

	b, err := json.Marshal(tarData)
	if err != nil {
		web.SetErrorRedirect(w, r, "/", "Error reading file")
		return
	}
	rpc.Import(b)
	web.SetSuccessRedirect(w, r, "/", "Successfully imported database")
	return
}}

var eraseDB = web.Route{"POST", "/erase", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	if web.Get(r, "db") == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	rpc.ClearAll()
	web.SetSuccessRedirect(w, r, "/", "Successfully erased database")
	return
}}

// POST add store to connected DB
var saveStore = web.Route{"POST", "/new", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	if web.Get(r, "db") == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	name := r.FormValue("name")
	if ok := rpc.AddStore(name); ok {
		web.SetSuccessRedirect(w, r, "/", "Successfully saved store")
		return
	}
	web.SetErrorRedirect(w, r, "/", "Error saving store")
	return
}}

// GET render specified store from specified DB
var getStore = web.Route{"GET", "/:store", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	if !rpc.HasStore(store) {
		web.SetErrorRedirect(w, r, "/"+store, "Invalid store")
		return
	}
	ts.Render(w, r, "store.tmpl", tmpl.Model{
		"savedSearch": GetSavedSearches(db, store),
		"db":          db,
		"stores":      rpc.GetAllStoreStats(),
		"store":       rpc.GetAll(store),
		"storeName":   store,
		"query":       GetSavedSearch(db, store, r.FormValue("query")),
	})
	return
}}

// POST delete store from connected DB
var delStore = web.Route{"POST", "/:store", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	if web.Get(r, "db") == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue("store")
	println(store)
	if !rpc.DelStore(store) {
		web.SetErrorRedirect(w, r, "/", "Error deleting store")
		return
	}
	web.SetSuccessRedirect(w, r, "/", "Successfully deleted store "+store)
	return
}}

// POST save search made on store
var saveQuery = web.Route{"POST", "/:store/search/save", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	var savedSearch map[string]map[string]string

	config.GetAs("search", db, &savedSearch)
	if savedSearch == nil {
		savedSearch = make(map[string]map[string]string)
	}
	if savedSearch[store] == nil {
		savedSearch[store] = make(map[string]string)
	}
	savedSearch[store][r.FormValue("name")] = r.FormValue("search")
	config.Set("search", db, savedSearch)
	web.SetSuccessRedirect(w, r, "/"+store, "Successfully saved search")
	return
}}

// GET render empty record for specified store from connected DB
var newRecord = web.Route{"GET", "/:store/new", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	ts.Render(w, r, "record.tmpl", tmpl.Model{
		"db":        db,
		"stores":    rpc.GetAllStoreStats(),
		"storeName": store,
		"record":    "",
	})
	return
}}

// POST add record to specified store in connected DB
var addRecord = web.Route{"POST", "/:store/add", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	rpc.Add(store, rec)
	web.SetSuccessRedirect(w, r, "/"+store, "Successfully added record")
	return
}}

// POST upload .json file of records to add to store
var importStore = web.Route{"POST", "/:store/import", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	r.ParseMultipartForm(32 << 20) // 32 MB
	file, handler, err := r.FormFile("data")
	if err != nil || len(handler.Header["Content-Type"]) < 1 {
		fmt.Println(err)
		web.SetErrorRedirect(w, r, "/"+store, "Error uploading file")
		return
	}
	defer file.Close()
	var m []map[string]interface{}
	switch handler.Header["Content-Type"][0] {
	case "application/json":
		dec := json.NewDecoder(file)
		err = dec.Decode(&m)
	case "text/csv":
		m, err = DecodeCSV(file)
	default:
		web.SetErrorRedirect(w, r, "/"+store, "Error uploading file")
		return
	}

	if err != nil {
		fmt.Println(err)
		web.SetErrorRedirect(w, r, "/"+store, "Error uploading file")
		return
	}
	for _, doc := range m {
		SanitizeMap(&doc)
		rpc.Add(store, doc)
	}
	web.SetSuccessRedirect(w, r, "/"+store, "Successfully imported data")
	return
}}

// GET render specified record from specified store in connected DB
var getRecord = web.Route{"GET", "/:store/:record", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	record := rpc.Get(store, GetId(r.FormValue(":record")))
	if record == nil {
		web.SetErrorRedirect(w, r, "/"+store, "Invalid Record")
		return
	}
	ts.Render(w, r, "record.tmpl", tmpl.Model{
		"db":        db,
		"stores":    rpc.GetAllStoreStats(),
		"storeName": store,
		"record":    record,
	})
	return
}}

// POST save record to specified store in connected DB
var saveRecord = web.Route{"POST", "/:store/:record", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	record := r.FormValue("record")
	var rec map[string]interface{}
	json.Unmarshal([]byte(record), &rec)
	if ok := rpc.Set(store, GetId(record), rec); ok {
		web.SetSuccessRedirect(w, r, "/"+store, "Successfully saved record")
		return
	}
	web.SetErrorRedirect(w, r, "/"+store, "Error saving record")
	return
}}

// POST delete record from specified store in connected DB
var delRecord = web.Route{"POST", "/:store/:record/del", func(w http.ResponseWriter, r *http.Request) {
	if !rpc.Alive() {
		web.SetErrorRedirect(w, r, "/", "Error no connection to a database")
		return
	}
	db := web.Get(r, "db")
	if db == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	store := r.FormValue(":store")
	rpc.Del(store, GetId(r.FormValue("record")))
	web.SetSuccessRedirect(w, r, "/"+store, "Successfully deleted record")
	return
}}

// helper functions

// return saved store searches from local DB
func GetSavedSearches(db, store string) []string {
	var savedSearch map[string]map[string]string
	config.GetAs("search", db, &savedSearch)
	var keys []string
	if storeSearch, ok := savedSearch[store]; ok {
		for k := range storeSearch {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// return single saved search by name from local DB
func GetSavedSearch(db, store, query string) string {
	var qry string
	if query != "" {
		var allQuery map[string]map[string]string
		config.GetAs("search", db, &allQuery)
		if storeQuery, ok := allQuery[store]; ok {
			qry = storeQuery[query]
		}
	}
	return qry
}

// return all saved DB connections from local DB
func GetSavedDBs() []string {
	var dbs []string
	for k := range *config.GetStore("connections") {
		dbs = append(dbs, k)
	}
	sort.Strings(dbs)
	return dbs
}

func GetId(s string) uint64 {
	id, _ := strconv.ParseUint(s, 10, 64)
	return id
}

// decode csv as []map[string]interface{} for importing store
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

//  sanitize doc for importing to  store
func SanitizeMap(m *map[string]interface{}) {
	for k, v := range *m {
		delete(*m, k)
		(*m)[strings.ToLower(k[0:1])+k[1:]] = v
	}
	runtime.GC()
}

// check for error and redirect accrodingly
func HasError(redirect string, err error, w http.ResponseWriter, response map[string]interface{}) bool {
	if err != nil {
		fmt.Printf("%v\n", err)
		response["complete"] = false
		b, err2 := json.Marshal(response)
		if err2 != nil {
			fmt.Fprintf(w, "ERROR")
		}
		fmt.Fprintf(w, "%s", b)
		return true
	}
	return false
}
