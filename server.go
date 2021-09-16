package main

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

var (
	redisAddress = flag.String("redis-address", lookupEnvOrString("REDIS_ADDRESS", "localhost:6379"), "redis address in format host:port")
	port         = flag.String("port", lookupEnvOrString("PORT", "8080"), "listen port")
)

func main() {
	r := newRouter()

	storage := NewStorage(*redisAddress)

	r.registerRoute(http.MethodGet, "/", func(ctx context.Context, writer http.ResponseWriter, request *http.Request) (any, error) {
		http.ServeFile(writer, request, "."+path.Clean(request.URL.Path))
		return nil, nil
	})

	r.registerRoute(http.MethodGet, "/l/", func(ctx context.Context, writer http.ResponseWriter, request *http.Request) (any, error) {
		const showHTML = "show.html"
		type ShowContext struct {
			CipherText string
		}

		rest, _ := path.Split(request.URL.Path)
		_, id := path.Split(path.Clean(rest))
		res, err := storage.Get(ctx, id)
		if err != nil {
			return nil, err
		}

		tmpl, err := template.ParseFiles(showHTML)
		if err != nil {
			return nil, err
		}
		out := bytes.NewBuffer(nil)
		err = tmpl.Execute(out, ShowContext{
			CipherText: res,
		})
		if err != nil {
			return nil, err
		}

		http.ServeContent(writer, request, showHTML, time.Now().UTC(), bytes.NewReader(out.Bytes()))
		if err := storage.Delete(ctx, id); err != nil {
			return nil, err
		}

		return nil, nil
	})

	r.registerRoute(http.MethodPost, "/", func(ctx context.Context, _ http.ResponseWriter, request *http.Request) (interface{}, error) {
		type StoreRequest struct {
			CipherText string `json:"cipherText"`
		}
		type StoreResponse struct {
			ID string `json:"id"`
		}

		rawBody, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		var req StoreRequest
		if err := json.Unmarshal(rawBody, &req); err != nil {
			return nil, err
		}

		bts := [8]byte{}
		_, err = rand.Read(bts[:])
		if err != nil {
			return nil, err
		}

		key := hex.EncodeToString(bts[:])
		if err := storage.Put(ctx, key, req.CipherText); err != nil {
			return nil, err
		}

		return StoreResponse{ID: key}, nil
	})

	r.serve(context.Background())
}

type router struct {
	routes map[string]map[string]func(writer http.ResponseWriter, request *http.Request)
	mux    *http.ServeMux
}

func newRouter() *router {
	return &router{
		routes: make(map[string]map[string]func(http.ResponseWriter, *http.Request)),
		mux:    http.NewServeMux(),
	}
}

func (r *router) registerRoute(method, path string, handler func(ctx context.Context, writer http.ResponseWriter, request *http.Request) (any, error)) {
	p, ok := r.routes[path]
	if !ok {
		p = map[string]func(http.ResponseWriter, *http.Request){}
		r.routes[path] = p
	}

	p[method] = r.asHandler(handler)
}

func (r *router) initHttpMultiplexer() {
	for path := range r.routes {
		p := path
		r.mux.HandleFunc(p, func(writer http.ResponseWriter, request *http.Request) {
			for m, handler := range r.routes[p] {
				if request.Method == m {
					logger.Printf("%s %s", m, request.URL.Path)
					handler(writer, request)
					return
				}
			}
			writer.WriteHeader(http.StatusMethodNotAllowed)
		})
	}
}

func (r *router) asHandler(call func(context.Context, http.ResponseWriter, *http.Request) (interface{}, error)) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		res, err := call(request.Context(), writer, request)
		defer r.wrapError(writer, err)
		if err != nil {
			return
		}
		if res == nil {
			return
		}
		var out []byte
		out, err = json.Marshal(res)
		if err != nil {
			return
		}

		writer.WriteHeader(http.StatusOK)

		if _, err = writer.Write(out); err != nil {
			return
		}
	}
}

func (r *router) wrapError(writer http.ResponseWriter, err error) {
	type Error struct {
		Error string `json:"message"`
	}

	if err == nil {
		return
	}

	writer.WriteHeader(http.StatusBadRequest)

	out, err := json.Marshal(Error{Error: err.Error()})
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write(out); err != nil {
		panic(err)
	}
}

func (r *router) serve(ctx context.Context) {
	r.initHttpMultiplexer()

	server := &http.Server{
		Addr:    ":" + *port,
		Handler: r.mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	logger.Printf("server started")
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}

	logger.Printf("server closed")
}

var logger = log.Default()

func lookupEnvOrString(key, defaultValue string) string {
	v, ok := os.LookupEnv(key)
	if ok {
		return v
	}
	return defaultValue
}
