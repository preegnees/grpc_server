package main

import (
	"log"
	"fmt"
	"os"
	"strconv"
	"strings"

	m "streaming/pkg/models"
	s "streaming/pkg/server"
	"github.com/joho/godotenv"
)

type config struct {
	Addr                 string `env:"ADDR"`
	ShutdownTimeout      int    `env:"SHUTDOWN_TIMEOUT"`
	CertPem              string `env:"CERT_PEM"`
	KeyPem               string `env:"KEY_PEM"`
	Debug                bool   `env:"DEBUG"`
	StreamStoragePathLog string `env:"STREAM_STORAGE_PATH_LOG"`
	ServerPathLog        string `env:"SERVER_PATH_LOG"`
}

func readConf() config {

	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("Передайте путь до файла конифгурации")
		os.Exit(1)
	}

	if _, err := os.Stat(args[0]); err != nil {
		fmt.Println("invalid PATH_TO_ENV, is not exists")
		os.Exit(1)
	}

	err := godotenv.Load(args[0])
	if err != nil {
		fmt.Println("Ошибка при загрузки файла с переменными")
		os.Exit(1)
	}

	a := os.Getenv("ADDR")
	if a == "" || len(strings.Split(a, ":")) != 2 {
		fmt.Println("invalid ADDR")
		os.Exit(1)
	}

	st := os.Getenv("SHUTDOWN_TIMEOUT")
	stInt, err := strconv.Atoi(st)
	if err != nil {
		fmt.Println("SHUTDOWN_TIMEOUT ADDR")
		os.Exit(1)
	}

	cp := os.Getenv("CERT_PEM")
	if _, err := os.Stat(cp); err != nil {
		fmt.Println("invalid CERT_PEM, is not exists")
		os.Exit(1)
	}

	kp := os.Getenv("KEY_PEM")
	if _, err := os.Stat(kp); err != nil {
		fmt.Println("invalid KEY_PEM, is not exists")
		os.Exit(1)
	}

	d := os.Getenv("DEBUG")
	dbool, err := strconv.ParseBool(d)
	if err != nil {
		fmt.Println("invalid DEBUG, mast been bool var")
		os.Exit(1)
	}

	sspl := os.Getenv("STREAM_STORAGE_PATH_LOG")
	if _, err := os.Stat(sspl); err != nil {
		fmt.Println("invalid STREAM_STORAGE_PATH_LOG, is not exists")
		os.Exit(1)
	}

	spl := os.Getenv("SERVER_PATH_LOG")
	if _, err := os.Stat(spl); err != nil {
		fmt.Println("invalid SERVER_PATH_LOG, is not exists")
		os.Exit(1)
	}

	return config{
		Addr:                 a,
		ShutdownTimeout:      stInt,
		CertPem:              cp,
		KeyPem:               kp,
		Debug:                dbool,
		StreamStoragePathLog: sspl,
		ServerPathLog:        spl,
	}
}

func main() {
	cnf := readConf()
	server := s.New()
	err := server.Run(
		m.CnfServer{
			Addr: cnf.Addr,
			ShutdownTimeout: cnf.ShutdownTimeout,
			CertPem: cnf.CertPem,
			KeyPem: cnf.KeyPem,
			Debug: cnf.Debug,
			StreamStoragePathLog: cnf.StreamStoragePathLog,
			ServerPathLog: cnf.ServerPathLog,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
