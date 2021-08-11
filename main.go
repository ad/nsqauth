package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ad/nsqauth/clickhouse"

	"github.com/alexcesaro/log/stdlog"
)

var (
	addr  = ":7755"
	useDB bool

	file string

	db     *sql.DB
	dbAddr string

	ttl = 3600

	identities Identities
	sess       *Session

	err error
)

// Authorization.
type Authorization struct {
	Topic       string   `json:"topic"`
	Channels    []string `json:"channels"`
	Permissions []string `json:"permissions"`
}

// Identity.
type Identity struct {
	Username       string
	Authorizations []Authorization
}

// Identities.
type Identities map[string]*Identity

// Session struct.
type Session struct {
	mux     *sync.RWMutex
	secrets map[string]*Identity
}

func main() {
	flag.StringVar(&addr, "addr", LookupEnvOrString("ADDR", addr), "listen addr")
	flag.StringVar(&file, "file", LookupEnvOrString("FILE", file), "identity csv file")
	flag.StringVar(&dbAddr, "db", LookupEnvOrString("DB", dbAddr), "db address")
	flag.IntVar(&ttl, "ttl", LookupEnvOrInt("TTL", ttl), "ttl in seconds")
	flag.Parse()

	logger := stdlog.GetFromFlags()

	if dbAddr == "" && file == "" {
		logger.Error("provide db or file")
		os.Exit(1)
	}

	if file == "" {
		useDB = true
	}

	if !useDB {
		identities, err = ParseIdentity(file)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}

		sess = NewSession()

		for _, identity := range identities {
			_ = sess.Set(identity)
		}
	} else {
		db, err = clickhouse.InitClickhouse(dbAddr)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		// remoteAddr := r.FormValue("remote_ip")
		// tls := r.FormValue("tls")

		secret := r.FormValue("secret")

		if len(secret) <= 1 {
			logger.Debug("invalid secret:", secret)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		logger.Debug("auth request:", secret)

		var auths []Authorization

		if useDB {
			if secrets, err := clickhouse.GetSecretsInfo(db, clickhouse.Secret{UUID: secret}); err != nil {
				logger.Error("secret", secret, err)
				w.WriteHeader(http.StatusForbidden)

				return
			} else {
				for _, sec := range secrets {
					auths = append(auths, Authorization{Topic: sec.Topic, Channels: strings.Split(sec.Channels, ";"), Permissions: strings.Split(sec.Permissions, ";")})
				}
			}
		} else {
			if ident, err := sess.Get(secret); err == nil {
				auths = ident.Authorizations
			} else {
				logger.Error("secret", secret, err)
				w.WriteHeader(http.StatusForbidden)

				return
			}
		}

		state := struct {
			TTL            int             `json:"ttl"`
			Authorizations []Authorization `json:"authorizations"`
			Identity       string          `json:"identity"`
			IdentityURL    string          `json:"identity_url"`
		}{
			TTL:            ttl,
			Authorizations: auths,
			Identity:       secret,
			IdentityURL:    fmt.Sprintf("http://%s/secret", addr),
		}

		logger.Debug("auth response:", secret, state)

		if err := json.NewEncoder(w).Encode(state); err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusForbidden)

			return
		}
	})

	logger.Info("starting nsqauth on address", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Error(err)
	}
}

// ParseIdentity.
func ParseIdentity(db string) (Identities, error) {
	logger := stdlog.GetFromFlags()

	identities := make(Identities)

	fhandler, err := os.Open(db)
	if err != nil {
		return identities, err
	}

	defer fhandler.Close()

	reader := csv.NewReader(fhandler)

	for {
		line, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				logger.Error("error read identity csv:", err)
			}

			break
		}

		if len(line) < 4 {
			logger.Error("error parse indentity:", line)

			continue
		}

		if strings.HasPrefix(line[0], "#") {
			continue
		}

		secret := line[0]

		authorization := Authorization{
			Topic:       line[1],
			Channels:    strings.Split(line[2], ";"),
			Permissions: strings.Split(line[3], ";"),
		}

		if _, ok := identities[secret]; ok {
			identities[secret].Authorizations = append(identities[secret].Authorizations, authorization)
		} else {
			identities[secret] = &Identity{
				Username:       secret,
				Authorizations: []Authorization{authorization},
			}
		}
	}

	return identities, nil
}

// New session.
func NewSession() *Session {
	return &Session{
		mux:     &sync.RWMutex{},
		secrets: make(map[string]*Identity),
	}
}

// Get session.
func (s *Session) Get(secret string) (*Identity, error) {
	s.mux.RLock()
	identity, ok := s.secrets[secret]
	s.mux.RUnlock()

	if ok {
		return identity, nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}

// Set session.
func (s *Session) Set(identity *Identity) string {
	secret := identity.Username // uuid.NewString()

	s.mux.Lock()
	s.secrets[secret] = identity
	s.mux.Unlock()

	return secret
}

// LookupEnvOrString ...
func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultVal
}

// LookupEnvOrInt ...
func LookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}

	return defaultVal
}
