package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/zinrai/debinstaller-go/internal/config"
	"github.com/zinrai/debinstaller-go/internal/installer"
	"github.com/zinrai/debinstaller-go/internal/utils"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	logger := utils.NewLogger(cfg.LogFile)
	defer logger.Close()

	inst := installer.NewInstaller(cfg, logger)

	if err := inst.Install(); err != nil {
		logger.Error("Installation failed: %v", err)
		os.Exit(1)
	}

	fmt.Println("Debian installation completed successfully")
}
