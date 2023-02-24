package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"golang.org/x/exp/slog"

	"github.com/gobike/envflag"
)

var (
	debug      bool
	version    string = "0.0"
	addr       string = ":80"
	msg        string = "default message"
	configFile string = "config.json"
	clientset  *kubernetes.Clientset
)

func router() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr)
		w.Write([]byte(fmt.Sprintf("config : %v", configFile)))
		w.Write([]byte(fmt.Sprintf("version: %v", version)))
		w.Write([]byte(fmt.Sprintf("status : %v", "Ok")))
	})

	http.HandleFunc("/msg", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr)
		w.Write([]byte(msg))
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr)
		for _, es := range os.Environ() {
			w.Write([]byte(es))
		}
	})

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		fd, err := os.Open(configFile)
		if err != nil {
			slog.Info("failed", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer fd.Close()
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr)
		io.Copy(w, fd)
	})

	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		if clientset == nil {
			slog.Info("k8s client not ready", "uri", r.RequestURI, "client", r.RemoteAddr)
			return
		}

		pods, err := clientset.CoreV1().Pods("default").List(r.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Info("k8s client not ready", "uri", r.RequestURI, "client", r.RemoteAddr)
			return
		}
		for _, pod := range pods.Items {
			w.Write([]byte("pod: " + pod.Name + "\n"))
		}

	})
}

func main() {
	ver := flag.Bool("v", false, "show version")
	flag.BoolVar(&debug, "debug", debug, "debug log level")
	flag.StringVar(&msg, "msg", msg, "server message")
	flag.StringVar(&addr, "addr", addr, "server serve address")
	flag.StringVar(&configFile, "config", configFile, "server config file")
	envflag.Parse()

	if *ver {
		fmt.Println("version", version)
		return
	}

	config, err := rest.InClusterConfig()
	if err == nil {
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			slog.Warn("k8s client configuration failed", "error", err.Error())
		}
	} else {
		slog.Info("k8s client not set")
	}

	router()

	slog.Info("starting", "addr", addr, "version", version)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Println("listen and serve error", err)
	}
}
