package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"golang.org/x/exp/slog"

	"github.com/gobike/envflag"
	"github.com/gorilla/mux"
)

var (
	debug      bool
	version    string = "0.0"
	addr       string = ":80"
	msg        string = "default message"
	configFile string = "config.json"
	dataDir    string = os.TempDir()
	clientset  *kubernetes.Clientset
)

func router() *mux.Router {
	router := mux.NewRouter()
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "config :", configFile)
		fmt.Fprintln(w, "version:", version)
		fmt.Fprintln(w, "status :", "OK")
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	http.HandleFunc("/msg", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, msg)
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		for _, e := range os.Environ() {
			fmt.Fprintln(w, e)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		fd, err := os.Open(configFile)
		if err != nil {
			slog.Warn("failed", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}
		defer fd.Close()
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
		io.Copy(w, fd)
	})

	router.HandleFunc("/upload/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := filepath.Join(dataDir, vars["name"])
		fd, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			slog.Warn("failed", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
		}
		defer fd.Close()
		io.Copy(fd, r.Body)
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	}).Methods(http.MethodPut)

	router.HandleFunc("/download/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := filepath.Join(dataDir, vars["name"])
		fd, err := os.Open(filename)
		if err != nil {
			slog.Warn("failed", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
		}
		defer fd.Close()
		io.Copy(w, fd)
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	}).Methods(http.MethodGet)

	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		if clientset == nil {
			slog.Info("k8s client not ready", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client not ready")
			return
		}

		pods, err := clientset.CoreV1().Pods("default").List(r.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Info("k8s client error", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err, "code", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client error", err)
			return
		}
		for _, pod := range pods.Items {
			fmt.Fprintln(w, "pod:", pod.Name)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})
}

func main() {
	ver := flag.Bool("v", false, "show version")
	flag.BoolVar(&debug, "debug", debug, "debug log level")
	flag.StringVar(&msg, "msg", msg, "server message")
	flag.StringVar(&addr, "addr", addr, "server serve address")
	flag.StringVar(&dataDir, "data-dir", dataDir, "server data dir")
	flag.StringVar(&configFile, "config", configFile, "server config file")
	envflag.Parse()

	if *ver {
		fmt.Println("version", version)
		return
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		slog.Warn("k8s inCluster init failed", "error", err)
	} else {
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			slog.Warn("k8s client configuration failed", "error", err)
		}
	}

	slog.Info("starting", "addr", addr, "version", version)
	if err := http.ListenAndServe(addr, router()); err != nil {
		log.Println("listen and serve error", err)
	}
}
