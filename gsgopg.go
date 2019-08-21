package gsgopg

import (
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// Session schema
type Session struct {
	tableName struct{}  `sql:"gorilla_session"`
	Key       string    `sql:",type:varchar(52),pk"`
	Data      string    `sql:",notnull"`
	Expire    time.Time `sql:",default:now()"`
}

// Creates session table if not present
func createSessionTable(db *pg.DB) error {
	for _, model := range []interface{}{(*Session)(nil)} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GoPgStore implements Session store
type GoPgStore struct {
	Codecs  []securecookie.Codec
	Options *sessions.Options
	db      *pg.DB
}

func NewGoPgStore(db *pg.DB, keyPairs ...[]byte) (*GoPgStore, error) {
	createSessionTable(db)
	store := &GoPgStore{
		db:     db,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
	}
	return store, nil
}

func (s *GoPgStore) load(session *sessions.Session) error {
	sess := &Session{Key: session.ID}
	err := s.db.Model(sess).Limit(1).Select()
	if err != nil {
		return err
	}
	return securecookie.DecodeMulti(session.Name(), string(sess.Data),
		&session.Values, s.Codecs...)

}

func (s *GoPgStore) save(session *sessions.Session) error {
	var err error
	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}
	sess := &Session{
		Key:    session.ID,
		Data:   encoded,
		Expire: time.Now().Add(time.Second * time.Duration(session.Options.MaxAge)),
	}
	if session.IsNew == true {
		err = s.db.Insert(sess)
	} else {
		_, err = s.db.Model(sess).Column("data").WherePK().Update()
	}

	if err != nil {
		return err
	}
	return nil
}

func (s *GoPgStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// TODO: Delete expired sessions

	if session.ID == "" {
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
	}

	if err := s.save(session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return err
	}

	newCook := sessions.NewCookie(session.Name(), encoded, session.Options)
	http.SetCookie(w, newCook)
	return nil
}

func (s *GoPgStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	options := *s.Options
	session.Options = &options
	session.IsNew = true
	var err error
	if cook, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, cook.Value, &session.ID, s.Codecs...)
		if err == nil {
			err = s.load(session)
			if err == nil {
				session.IsNew = false
			} else {
				err = nil
			}
		}
	}
	return session, err
}

func (s *GoPgStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}
