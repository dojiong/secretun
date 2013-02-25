package main

import (
	json "encoding/json"
	"flag"
	"log"
	"os"
	"secretun"
)

var cfgfile = flag.String("cfg", "cli.cfg", "configure file path")

func main() {
	flag.Parse()
	var cfg = map[string]map[string]interface{}{}

	if f, err := os.Open(*cfgfile); err != nil {
		log.Println(err)
		return
	} else {
		defer f.Close()
		decoder := json.NewDecoder(f)
		if err := decoder.Decode(&cfg); err != nil {
			log.Println(err)
			return
		}
	}

	if cli, err := secretun.NewClient(cfg); err != nil {
		log.Println(err)
	} else {
		if err := cli.Init(); err != nil {
			log.Println(err)
			return
		}
		if err = cli.Run(); err != nil {
			log.Println(err)
		}
	}
}
