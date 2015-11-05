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
	"github.com/cagnosolutions/web/tmpl"
)

var rpc = dbdb.NewClient()
var ts = tmpl.NewTemplateStore(true)
var config = mockdb.NewMockDB("config.json", 5)

func main() {
	mux := web.NewMux("CTIXID", (web.HOUR / 2))

	// db managment
	mux.Get("/test", Test)
	mux.Get("/", Root)
	mux.Post("/connection", AddConnection)
	mux.Post("/connection/save", SaveConnection)
	mux.Post("/connection/:db/del", DelConnection)

	mux.Get("/connect/:db", Connect)
	mux.Get("/disconnect", Disconnect)

	mux.Get("/export", ExportDB)
	mux.Post("/import", ImportDB)
	mux.Post("/erase", EraseDB)

	// store managment
	mux.Post("/new", SaveStore)
	mux.Get("/:store", Store)
	mux.Post("/:store", DelStore)

	// store search
	mux.Post("/:store/search/save", SaveSearch)

	// record managment
	mux.Get("/:store/new", NewRecord)
	mux.Post("/:store/add", AddRecord)
	mux.Post("/:store/import", UploadRecords)
	mux.Get("/:store/:record", Record)
	mux.Post("/:store/:record", SaveRecord)
	mux.Post("/:store/:record/del", DelRecord)

	mux.Serve(":8080")
}

// GET render all saved DBs or show currently cinnected DB
func Root(w http.ResponseWriter, r *http.Request, c *web.Context) {
	msgk, msgv := c.GetFlash()

	// Not connected display all DBs
	if !rpc.Alive() {
		ts.Render(w, "index.tmpl", tmpl.Model{
			msgk:    msgv,
			"dbs":   GetSavedDBs(),
			"conns": config.GetStore("connections"),
		})
		return
	}

	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	ts.Render(w, "db.tmpl", tmpl.Model{
		msgk:     msgv,
		"db":     c.Get("db"),
		"stores": rpc.GetAllStoreStats(),
	})
	return
}

// POST add new db connection
func AddConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var connection map[string]string
	config.GetAs("connections", r.FormValue("name"), &connection)
	if connection == nil {
		println(3)
		connection = make(map[string]string)
	}
	connection["address"] = r.FormValue("address")
	connection["token"] = r.FormValue("token")
	config.Set("connections", r.FormValue("name"), connection)
	c.SetFlash("alertSuccess", "Successfully added connection")
	http.Redirect(w, r, "/", 303)
	return
}

// POST update existing DB connection
func SaveConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var connection map[string]string
	ok := config.GetAs("connections", r.FormValue("name"), &connection)
	if r.FormValue("name") != r.FormValue("oldName") {
		if ok {
			c.SetFlash("alertError", "Error connection name already exists")
			http.Redirect(w, r, "/", 303)
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
	c.SetFlash("alertSuccess", "Successfully updated connection")
	http.Redirect(w, r, "/", 303)
	return
}

// POST delete saved DB connection
func DelConnection(w http.ResponseWriter, r *http.Request, c *web.Context) {
	config.Del("connections", c.GetPathVar("db"))
	c.SetFlash("alertSuccess", "Successfully deleted connection")
	http.Redirect(w, r, "/", 303)
	return
}

// GET connect to DB
func Connect(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var connection map[string]string
	if config.GetAs("connections", c.GetPathVar("db"), &connection) {
		if address, ok := connection["address"]; ok && address != "" {
			if rpc.Connect(address, connection["token"]) {
				c.SetFlash("alertSuccess", "Successfully connected to database")
				c.Set("db", c.GetPathVar("db"))
				http.Redirect(w, r, "/", 303)
				return
			}
		}
	}
	c.SetFlash("alertError", "Error connecting to the database")
	http.Redirect(w, r, "/", 303)
	return
}

// GET disconnect from DB
func Disconnect(w http.ResponseWriter, r *http.Request, c *web.Context) {
	rpc.Disconnect()
	c.SetFlash("alertSuccess", "Disconnected")
	http.Redirect(w, r, "/", 303)
	return
}

func Test(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var response = make(map[string]interface{})
	if !rpc.Alive() {
		response["complete"] = true
		response["path"] = "/"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		response["complete"] = true
		response["path"] = "/disconnect"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	exportData := rpc.Export()
	var dat map[string][]map[string]interface{}
	json.Unmarshal([]byte(exportData), &dat)
	fmt.Fprintf(w, "%+#v", dat)
	return
	/*
		response["complete"] = false
		path := "static/export/"
		fullPath := path + strings.Replace(c.Get("db").(string), " ", "_", -1) + "_" + time.Now().Format("2006-01-02") + ".json"

		if _, err := os.Stat(fullPath); err == nil {
			response["complete"] = true
			response["path"] = fullPath
			b, _ := json.Marshal(response)
			fmt.Fprintf(w, "%s", b)
			return
		}

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

		for _, stat := range rpc.GetAllStoreStats() {
			var docs []map[string]interface{}
			for _, doc := range rpc.GetAll(stat.Name) {
				docs = append(docs, doc.Data)
			}

			b, err := json.Marshal(docs)
			if HasError("/", err, w, response) {
				return
			}
			hdr := &tar.Header{
				Name: stat.Name + ".json",
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
		exportData := rpc.Export()
		var dat map[string][]map[string]interface{}
		err = json.Unmarshal([]byte(exportData), &dat)
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
		return*/
}

// GET create .tar file from connected DB of all its stores
// and records and return download link
func ExportDB(w http.ResponseWriter, r *http.Request, c *web.Context) {
	var response = make(map[string]interface{})
	if !rpc.Alive() {
		response["complete"] = true
		response["path"] = "/"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		response["complete"] = true
		response["path"] = "/disconnect"
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}
	response["complete"] = false
	path := "static/export/"
	fullPath := path + strings.Replace(c.Get("db").(string), " ", "_", -1) + "_" + time.Now().Format("2006-01-02") + ".tar"

	/*if _, err := os.Stat(fullPath); err == nil {
		response["complete"] = true
		response["path"] = fullPath
		b, _ := json.Marshal(response)
		fmt.Fprintf(w, "%s", b)
		return
	}*/

	err := os.MkdirAll(path, 0755)
	if HasError("/", err, w, response) {
		return
	}

	//tarFile, err := os.OpenFile(fullPath, os.O_TRUNC|os.O_WRONLY, 0644)
	tarFile, err := os.Create(fullPath)
	if HasError("/", err, w, response) {
		return
	}
	defer tarFile.Close()
	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	/*for _, stat := range rpc.GetAllStoreStats() {
		var docs []map[string]interface{}
		for _, doc := range rpc.GetAll(stat.Name) {
			docs = append(docs, doc.Data)
		}

		b, err := json.Marshal(docs)
		if HasError("/", err, w, response) {
			return
		}
		hdr := &tar.Header{
			Name: stat.Name + ".json",
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

	}*/

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
}

// POST upload .tar file of .json files to add stores and records to connected DB
func ImportDB(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}

	r.ParseMultipartForm(32 << 20) // 32 MB
	tarFile, handler, err := r.FormFile("import")
	if err != nil || len(handler.Header["Content-Type"]) < 1 {
		fmt.Printf("dbdbClient >> ImportDB() >> lenfile header: %v", err)
		c.SetFlash("alertError", "Error uploading file")
		http.Redirect(w, r, "/"+c.GetPathVar("store")+"/import", 303)
		return
	}
	defer tarFile.Close()
	// TODO: check type

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
			c.SetFlash("alertError", "Error uploading file")
			http.Redirect(w, r, "/", 303)
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
	// TODO: handle err
	rpc.Import(b)

	/*for store, data := range tarData {
		rpc.AddStore(store)
		for _, doc := range data {
			rpc.Add(store, doc)
		}
	}*/

	c.SetFlash("alertSuccess", "Successfully imported database")
	http.Redirect(w, r, "/", 303)
	return
}

func EraseDB(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	rpc.ClearAll()
	c.SetFlash("alertSuccess", "Successfully erased database")
	http.Redirect(w, r, "/", 303)
	return
}

// POST add store to connected DB
func SaveStore(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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
		"savedSearch": GetSavedSearches(c.Get("db").(string), c.GetPathVar("store")),
		"db":          c.Get("db"),
		"stores":      rpc.GetAllStoreStats(),
		"store":       rpc.GetAll(c.GetPathVar("store")),
		"storeName":   c.GetPathVar("store"),
		"query":       GetSavedSearch(c.Get("db").(string), c.GetPathVar("store"), r.FormValue("query")),
	})
	return
}

// POST delete store from connected DB
func DelStore(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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

// POST save search made on store
func SaveSearch(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	var savedSearch map[string]map[string]string

	config.GetAs("search", c.Get("db").(string), &savedSearch)
	if savedSearch == nil {
		savedSearch = make(map[string]map[string]string)
	}
	if savedSearch[c.GetPathVar("store")] == nil {
		savedSearch[c.GetPathVar("store")] = make(map[string]string)
	}
	savedSearch[c.GetPathVar("store")][r.FormValue("name")] = r.FormValue("search")
	config.Set("search", c.Get("db").(string), savedSearch)
	c.SetFlash("alertSuccess", "Successfully saved search")
	http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("store")), 303)
	return
}

// GET render empty record for specified store from connected DB
func NewRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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

// POST add record to specified store in connected DB
func AddRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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

// POST upload .json file of records to add to store
func UploadRecords(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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

// GET render specified record from specified store in connected DB
func Record(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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
		"record":    record,
	})
	return
}

// POST save record to specified store in connected DB
func SaveRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
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

// POST delete record from specified store in connected DB
func DelRecord(w http.ResponseWriter, r *http.Request, c *web.Context) {
	if !rpc.Alive() {
		http.Redirect(w, r, "/", 303)
		c.SetFlash("alertError", "Error no connection to a database")
		return
	}
	if c.Get("db") == nil || c.Get("db").(string) == "" {
		http.Redirect(w, r, "/disconnect", 303)
		return
	}
	rpc.Del(c.GetPathVar("store"), GetId(c.GetPathVar("record")))
	c.SetFlash("alertSuccess", "Successfully deleted record")
	http.Redirect(w, r, fmt.Sprintf("/%s", c.GetPathVar("store")), 303)
	return
}

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
