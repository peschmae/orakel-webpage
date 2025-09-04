package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/peschmae/orakel-webpage/internal/k8s"
	"github.com/peschmae/orakel-webpage/internal/valkey"

	"html/template"

	"github.com/Masterminds/sprig/v3"
	checksv1alpha1 "github.com/fhnw-imvs/fhnw-kubeseccontext/api/v1alpha1"
)

//go:embed static
var static embed.FS

//go:embed templates
var templates embed.FS

var (
	valkeyClient *valkey.ValkeyClient
	valkeyHost   = "valkey"
	valkeyPort   = "6379"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		log.Println(r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {

	flag.StringVar(&valkeyHost, "valkey-host", valkeyHost,
		"The host of the Valkey server.")
	flag.StringVar(&valkeyPort, "valkey-port", valkeyPort,
		"The port of the Valkey server.")

	flag.Parse()

	var err error
	valkeyClient, err = valkey.NewValKeyClient(valkeyHost, valkeyPort)
	if err != nil {
		fmt.Println("Error creating Valkey client:", err)
		log.Fatalf("Failed to create Valkey client: %v", err)
	}

	r := mux.NewRouter()
	r.Use(loggingMiddleware)

	// This will serve files under http://localhost:8000/static/<filename>
	// static files
	staticFS, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.FS(staticFS))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.HandleFunc("/", indexHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Println("Starting server on :8000")

	log.Fatal(srv.ListenAndServe())
}

type WorkloadHardeningCheckInfo struct {
	checksv1alpha1.WorkloadHardeningCheck
}

func (w WorkloadHardeningCheckInfo) Successful() bool {
	for _, c := range w.Status.Conditions {
		if c.Type == checksv1alpha1.ConditionTypeFinished && c.Status != "True" {
			return false
		}
	}

	return true
}

func (w WorkloadHardeningCheckInfo) Running() bool {
	for _, c := range w.Status.Conditions {
		if c.Type == checksv1alpha1.ConditionTypeFinished && c.Status == "True" {
			return true
		}
	}

	return false
}

func (w WorkloadHardeningCheckInfo) GetRecordings() map[string]*valkey.WorkloadRecording {

	recordings := make(map[string]*valkey.WorkloadRecording)
	for _, checkRun := range w.Status.CheckRuns {

		recording, err := valkeyClient.GetRecording(context.Background(), fmt.Sprintf("%s:%s:%s", w.Namespace, w.Spec.Suffix, checkRun.Name))
		if err != nil {
			//log.Printf("Error fetching recording for WorkloadHardeningCheck %s: %v", fmt.Sprintf("%s:%s:%s", w.Namespace, w.Spec.Suffix, checkRun.Name), err)
			continue
		}
		if recording != nil {
			recordings[checkRun.Name] = recording
		}
	}

	return recordings
}

type WorkloadHardeningCheckContext struct {
	Checks []WorkloadHardeningCheckInfo
}

type NamespaceHardeningCheckInfo struct {
	checksv1alpha1.NamespaceHardeningCheck

	WorkloadHardeningChecks []WorkloadHardeningCheckInfo
}

func (n NamespaceHardeningCheckInfo) Successful() bool {
	for _, w := range n.WorkloadHardeningChecks {
		for _, c := range w.Status.Conditions {
			if c.Type == checksv1alpha1.ConditionTypeFinished && c.Status != "True" {
				return false
			}
		}
	}

	return true
}

func (n NamespaceHardeningCheckInfo) Running() bool {
	for _, w := range n.WorkloadHardeningChecks {
		for _, c := range w.Status.Conditions {
			if c.Type == checksv1alpha1.ConditionTypeFinished && c.Status == "True" {
				return true
			}
		}
	}

	return false
}

type NamespaceHardeningCheckContext struct {
	Checks []NamespaceHardeningCheckInfo
}

func prepareNamespaceHardeningChecks(nhc []checksv1alpha1.NamespaceHardeningCheck, whc []checksv1alpha1.WorkloadHardeningCheck) *NamespaceHardeningCheckContext {
	nhcContext := &NamespaceHardeningCheckContext{}
	for _, n := range nhc {
		nhcInfo := NamespaceHardeningCheckInfo{
			NamespaceHardeningCheck: n,
		}
		for _, w := range whc {
			if len(w.ObjectMeta.OwnerReferences) > 0 && w.ObjectMeta.OwnerReferences[0].Name == n.Name {
				nhcInfo.WorkloadHardeningChecks = append(nhcInfo.WorkloadHardeningChecks, WorkloadHardeningCheckInfo{WorkloadHardeningCheck: w})
			}
		}
		nhcContext.Checks = append(nhcContext.Checks, nhcInfo)
	}

	return nhcContext

}

func prepareWorkloadHardeningChecks(whc []checksv1alpha1.WorkloadHardeningCheck) *WorkloadHardeningCheckContext {
	whcContext := &WorkloadHardeningCheckContext{}
	for _, w := range whc {
		whcContext.Checks = append(whcContext.Checks, WorkloadHardeningCheckInfo{WorkloadHardeningCheck: w})
	}

	return whcContext

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	namespaceHardeningChecks, err := k8s.GetNamespaceHardeningChecks()

	if err != nil {
		http.Error(w, "Error fetching NamespaceHardeningChecks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	workloadHardeningChecks, err := k8s.GetWorkloadHardeningChecks()

	if err != nil {
		http.Error(w, "Error fetching WorkloadHardeningChecks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	headerTemplate := loadHeaderTemplate()
	err = headerTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	nhcTemplate := loadNamespaceHardeningCheckTemplate()
	err = nhcTemplate.Execute(w, prepareNamespaceHardeningChecks(namespaceHardeningChecks, workloadHardeningChecks))
	if err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	whcTemplate := loadWorkloadHardeningCheckTemplate()
	err = whcTemplate.Execute(w, prepareWorkloadHardeningChecks(workloadHardeningChecks))
	if err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	footerTemplate := loadFooterTemplate()
	err = footerTemplate.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func loadHeaderTemplate() *template.Template {
	funcMap := sprig.FuncMap()
	tmpl := template.New("header.html").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(templates, "templates/header.html")
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}
	return tmpl
}

func loadFooterTemplate() *template.Template {
	funcMap := sprig.FuncMap()
	tmpl := template.New("footer.html").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(templates, "templates/footer.html")
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}
	return tmpl
}

func loadNamespaceHardeningCheckTemplate() *template.Template {
	funcMap := sprig.FuncMap()
	tmpl := template.New("namespaceHardeningChecks.html").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(templates, "templates/namespaceHardeningChecks.html")
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}
	return tmpl
}

func loadWorkloadHardeningCheckTemplate() *template.Template {
	funcMap := sprig.FuncMap()
	tmpl := template.New("workloadHardeningChecks.html").Funcs(funcMap)
	tmpl, err := tmpl.ParseFS(templates, "templates/workloadHardeningChecks.html")
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}
	return tmpl
}
