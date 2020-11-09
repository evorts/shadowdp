package main

import (
	"fmt"
	"github.com/evorts/shadowdp/config"
	"log"
	"net/http"
	"os"
)

func main() {
	var dir string
	if dir = os.Getenv("SHADOWDP_CONFIG_DIR"); len(dir) < 1 {
		dir, _ = os.Getwd()
	}
	cfg, err := config.NewConfig(dir, "config.main.yaml", "config.yaml").Initiate()
	if err != nil {
		log.Fatal("error reading configuration")
		return
	}
	o := http.NewServeMux()
	o.Handle("/", WithMethodFilter(
		http.MethodGet,
		WithInjection(
			http.HandlerFunc(goRodRender),
			map[string]interface{}{
				"cfg": cfg,
			},
		),
	))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.GetConfig().App.Port), o); err != nil {
		log.Fatal(err)
	}
}
