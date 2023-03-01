package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "go.uber.org/automaxprocs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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

func router() (mrouter *mux.Router) {
	mrouter = mux.NewRouter()
	mrouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	mrouter.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if hn, err := os.Hostname(); err == nil {
			fmt.Fprintln(w, "hostname:", hn)
		}

		fmt.Fprintln(w, "CPU         :", runtime.NumCPU())
		fmt.Fprintln(w, "GOMAXPROCS  :", runtime.GOMAXPROCS(0))
		fmt.Fprintln(w, "goroutine   :", runtime.NumGoroutine())
		// ll /proc/`pidof hc`/task/
		mn, _ := runtime.ThreadCreateProfile(nil)
		fmt.Fprintln(w, "threadcreate:", mn)

		fmt.Fprintln(w, "config :", configFile)
		fmt.Fprintln(w, "version:", version)
		fmt.Fprintln(w, "message:", msg)
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	mrouter.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		for _, e := range os.Environ() {
			fmt.Fprintln(w, e)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	mrouter.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
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

	mrouter.HandleFunc("/file/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
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

	mrouter.HandleFunc("/file/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
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

	mrouter.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		fd, err := os.ReadDir(dataDir)
		if err != nil {
			slog.Warn("failed", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
		}
		for _, fe := range fd {
			fmt.Fprintln(w, fe.Name())
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)

	}).Methods(http.MethodGet)

	if clientset == nil {
		return
	}

	mrouter.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		itms, err := clientset.CoreV1().Pods("default").List(r.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Warn("k8s client error", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err, "code", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client error", err)
			return
		}
		for _, itm := range itms.Items {
			fmt.Fprintln(w, "pod:", itm.Status.PodIP, itm.Name)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	mrouter.HandleFunc("/deployment", func(w http.ResponseWriter, r *http.Request) {
		itms, err := clientset.AppsV1().Deployments("default").List(r.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Warn("k8s client error", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err, "code", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client error", err)
			return
		}
		for _, itm := range itms.Items {
			fmt.Fprintln(w, "deployment:", itm.Name)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	mrouter.HandleFunc("/deployment/restart/{name:.+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deploymantName := vars["name"]
		// kubectl rollout restart deployment my-deploymnet
		deploymentsClient := clientset.AppsV1().Deployments("default")
		data := fmt.Sprintf(`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`, time.Now().Format("20060102150405"))
		dpm, err := deploymentsClient.Patch(r.Context(), deploymantName, k8stypes.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})

		if err != nil {
			slog.Warn("k8s client error", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err, "code", http.StatusInternalServerError, "name", deploymantName)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client error", err)
			return
		}
		fmt.Fprintln(w, "restart", dpm.Name)

		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK, "name", deploymantName)
	})

	mrouter.HandleFunc("/service", func(w http.ResponseWriter, r *http.Request) {
		itms, err := clientset.CoreV1().Services("default").List(r.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Warn("k8s client error", "uri", r.RequestURI, "client", r.RemoteAddr, "error", err, "code", http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "k8s client error", err)
			return
		}
		for _, itm := range itms.Items {
			fmt.Fprintln(w, "service:", itm.Spec.ClusterIP, itm.Name)
		}
		slog.Info("success", "uri", r.RequestURI, "client", r.RemoteAddr, "code", http.StatusOK)
	})

	return
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
		slog.Info("starting without k8s", "addr", addr, "version", version, "error", err)
	} else {
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			slog.Info("starting without k8s", "addr", addr, "version", version, "error", err)
		} else {
			slog.Info("starting with k8s", "addr", addr, "version", version)
		}
	}

	if err := http.ListenAndServe(addr, router()); err != nil {
		log.Println("listen and serve error", err)
	}
}
