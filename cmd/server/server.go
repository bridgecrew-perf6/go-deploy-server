package main

import (
	"bufio"
	"embed"
	"errors"
	"github.com/cute-angelia/go-utils/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go-deploy/config"
	"go-deploy/internal/controller"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"time"
)

//go:embed vue/dist/*
var assets embed.FS

const TYPE_SITE = "go-deploy-server"

func main() {
	// config
	config.InitConfig()

	// 日志
	logger.NewLogger(TYPE_SITE, !config.C.Debug)

	// router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.ThrottleBacklog(20, 500, time.Second))

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// 中间件
	r.Route("/", func(r chi.Router) {
		r.Mount("/list", controller.List{}.Routes())
		r.Mount("/deploy", controller.Deploy{}.Routes())
		r.Mount("/rollback", controller.Rollback{}.Routes())
		r.Mount("/showlog", controller.ShowLog{}.Routes())
	})

	r.NotFound(NotFoundHandler)

	// Ping
	for addr := range config.C.UniqAddr {
		go Ping(addr)
	}

	log.Println(TYPE_SITE + "启动成功~ " + config.C.ListenHttp)
	if err := http.ListenAndServe(config.C.ListenHttp, r); err != nil {
		log.Println("启动错误：", err)
	}
}

func Ping(addr string) {
	for {
		func() {
			//connect to this socket
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Println("Error connect to client:", err)
				return
			}

			//connect success
			setClientOnlineStatus(addr, true)

			//remote client closed
			defer func() {
				setClientOnlineStatus(addr, false)
				conn.Close()
			}()

			//read message from client
			go func(conn net.Conn) {
				defer conn.Close()
				for {
					message, err := bufio.NewReader(conn).ReadString('\n')
					if err != nil {
						log.Println("Client closed", err.Error())
						return
					}
					if config.C.Debug {
						log.Print(conn.RemoteAddr(), " -> Message Received from client:", message)
					}
				}
			}(conn)

			ticker := time.Tick(time.Second * 15)
			for {
				select {
				case <-ticker:
					conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
					_, err := conn.Write([]byte("PING\n"))
					if err != nil {
						log.Println("Error writing to stream:", err)
						return
					}
				}
			}
		}()
		time.Sleep(time.Second * 5)
	}
}

//set client online or offline
func setClientOnlineStatus(addr string, online bool) {
	for key, app := range config.C.Apps {
		for k, node := range app.Node {
			if node.Addr == addr {
				config.C.Apps[key].Node[k].Online = online
			}
		}
	}
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	err := tryRead(assets, "vue/dist", r.URL.Path, w)
	if err == nil {
		return
	}
	err = tryRead(assets, "vue/dist", "index.html", w)
	if err != nil {
		log.Println(err)
	}
}
var ErrDir = errors.New("path is dir")
func tryRead(fs embed.FS, prefix, requestedPath string, w http.ResponseWriter) error {
	f, err := fs.Open(path.Join(prefix, requestedPath))
	if err != nil {
		return err
	}
	defer f.Close()

	// Goのfs.Openはディレクトリを読みこもとうしてもエラーにはならないがここでは邪魔なのでエラー扱いにする
	stat, _ := f.Stat()
	if stat.IsDir() {
		return ErrDir
	}
	contentType := mime.TypeByExtension(filepath.Ext(requestedPath))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=864000")
	_, err = io.Copy(w, f)
	return err
}