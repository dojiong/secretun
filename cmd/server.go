package main

import (
	json "encoding/json"
	"flag"
	"log"
	"os"
	"secretun"
)

var cfgfile = flag.String("cfg", "ser.cfg", "configure file path")

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

	if ser, err := secretun.NewServer(cfg); err != nil {
		log.Println(err)
	} else {
		if err := ser.Init(); err != nil {
			log.Println(err)
			return
		}
		if err = ser.Run(); err != nil {
			log.Println(err)
		}
	}
}
