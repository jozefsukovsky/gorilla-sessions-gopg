# gorilla/sessions go-pg store

Not ready for serious use

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-pg/pg"
	"github.com/gorilla/sessions"
	gsgopg "github.com/jozefsukovsky/gorilla-sessions-gopg"
)

var db *pg.DB
var store *gsgopg.GoPgStore

func handler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-key")
	if err != nil {
		log.Fatalf(err.Error())
	}

	if session.Values["foo"] == nil {
		session.Values["foo"] = "bar"
	}

	session.Save(r, w)

	fmt.Fprintf(w, "%v\n", store)
	fmt.Fprintf(w, "Stored value: %s\n", session.Values["foo"])
}

func main() {
	db = pg.Connect(&pg.Options{
		User:     "dbuser",
		Password: "dbpassword",
		Database: "dbname",
	})
	defer db.Close()
	var err error
	store, err = gsgopg.NewGoPgStore(db, []byte("<SecretKey>"))
	store.Options = &sessions.Options{
		MaxAge:   86400,
		HttpOnly: true,
	}
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":1234", nil)
}
```