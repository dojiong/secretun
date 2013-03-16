package main

import (
	"flag"
	"log"
	"secretun"
)

var cfgfile = flag.String("cfg", "cli.cfg", "configure file path")

func main() {
	flag.Parse()
	cfg, err := secretun.ConfigFromJson(*cfgfile)
	if err != nil {
		log.Println(err)
		return
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
