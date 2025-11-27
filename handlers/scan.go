package handlers

import (
	"bytes"
	"fmt"
	"os"

	"github.com/dutchcoders/go-clamd"
)

func ScanWithClamAVDaemon(filePath string) error {
	c := clamd.NewClamd(os.Getenv("CLAMAV_IP"))

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("unable to read file: %v", err)
	}

	reader := bytes.NewReader(fileBytes)
	response, err := c.ScanStream(reader, make(chan bool))
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
