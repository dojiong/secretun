package main

import (
	"flag"
	"log"
	"secretun"
)

var cfgfile = flag.String("cfg", "ser.cfg", "configure file path")

func main() {
	flag.Parse()
	cfg, err := secretun.ConfigFromJson(*cfgfile)
	if err != nil {
		log.Println(err)
		return
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
