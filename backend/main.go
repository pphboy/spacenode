package main

import (
	"flag"
	"spacenode/modules/db"
	"spacenode/modules/spacehttp"

	"github.com/sirupsen/logrus"
)

var (
	httpPort = flag.Int("http-port", 58083, "HTTP server port")
	dbpath   = flag.String("dbpath", "/lzcapp/var/space.db", "db path")
)

func main() {
	flag.Parse()
	logrus.SetReportCaller(true)

	logrus.SetLevel(logrus.DebugLevel)
	db.InitDB(*dbpath)

	s, err := spacehttp.NewServer(*httpPort)
	if err != nil {
		panic(err)
	}

	logrus.Infoln("Starting server on port", *httpPort)

	if err := s.Start(); err != nil {
		logrus.Errorln(err)
		panic(err)
	}
}
