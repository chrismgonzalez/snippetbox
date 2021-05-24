package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"chrismgonzalez.com/snippetbox/pkg/models"
	"chrismgonzalez.com/snippetbox/pkg/models/mysql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golangcollege/sessions"
)

type contextKey string

const contextKeyIsAuthenticated = contextKey("isAuthenticated")
type application struct {
		debug bool
		errorLog *log.Logger
		infoLog *log.Logger
		session *sessions.Session
		snippets interface {
			Insert(string, string, string) (int, error)
			Get(int) (*models.Snippet, error)
			Latest() ([]*models.Snippet, error)
		}
		templateCache map[string]*template.Template
		users interface {
			Insert(string, string, string) error
			Authenticate(string, string) (int, error)
			Get(int) (*models.User, error)
			ChangePassword(int, string, string) error
		}
	}

func main() {
	dsn := flag.String("dsn", "web:Sgcik634814!@/snippetbox?parseTime=true", "MySQL data source name")
	addr := flag.String("addr", ":4000", "HTTP network address")
	debug := flag.Bool("debug", false, "enable debug mode")
	secret := flag.String("secret", "s6Ndh+pPbnzHbS*+9Pk8qGWhTzbpa@ge", "Secret key" )
	flag.Parse()
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}

	defer db.Close()

	// initialize a template cache
	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		errorLog.Fatal(err)
	}

	session := sessions.New([]byte(*secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = true

	app := &application{
		debug: *debug,
		errorLog: errorLog,
		infoLog: infoLog,
		session: session,
		snippets: &mysql.SnippetModel{DB: db,},
		templateCache: templateCache,
		users: &mysql.UserModel{DB: db},
	}
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr: *addr,
		ErrorLog: errorLog,
		Handler: app.routes(),
		TLSConfig: tlsConfig,
		IdleTimeout: time.Minute,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)

}

func openDB(dsn string) (*sql.DB, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		if err = db.Ping(); err != nil {
			return nil, err
		}
		return db, nil
	}