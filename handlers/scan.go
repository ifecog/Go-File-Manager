package handlers

import (
	"fmt"
	"os"

	"github.com/dutchcoders/go-clamd"
)

func ScanWithClamAVDaemon(filePath string) error {
	c := clamd.NewClamd(os.Getenv("CLAMAV_IP"))

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file: %v", err)
	}
	defer f.Close()

	response, err := c.ScanStream(f, make(chan bool))
	if err != nil {
		return fmt.Errorf("clamd scan failed: %v", err)
	}

	for scan := range response {
		fmt.Println(scan)
		if scan.Status == clamd.RES_FOUND {
			return fmt.Errorf("virus detected: %s", scan.Description)
		}
	}

	return nil
}
