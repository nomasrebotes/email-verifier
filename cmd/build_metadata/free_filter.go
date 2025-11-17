package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const workerCount = 10
const srcPath = "free.txt"          // file with one domain per line
const dstPath = "free_valid_mx.txt" // file with one domain per line (only those with valid MX record)

// filterFreeDomainsWithValidMX filters out free domains that do not have a valid MX record
func filterFreeDomainsWithValidMX() {

	// open source file for reading
	srcFile, err := os.Open(srcPath)
	if err != nil {
		log.Fatalf("failed to open source file %s: %v", srcPath, err)
	}
	defer func() {
		if cerr := srcFile.Close(); cerr != nil {
			log.Printf("error closing source file %s: %v", srcPath, cerr)
		}
	}()

	// open destination file for writing (truncate if exists)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("failed to create destination file %s: %v", dstPath, err)
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil {
			log.Printf("error closing destination file %s: %v", dstPath, cerr)
		}
	}()

	jobs := make(chan string)
	results := make(chan string)
	var wg sync.WaitGroup

	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for domain := range jobs {
				if hasValidMX(domain) {
					results <- domain
				}
			}
		}()
	}

	// close results channel once all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// writer: consume results and write to destination file
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for domain := range results {
			if _, err := fmt.Fprintln(dstFile, domain); err != nil {
				log.Printf("failed to write domain %s to %s: %v", domain, dstPath, err)
			}
		}
	}()

	// reader: scan source file and send domains to workers
	scanner := bufio.NewScanner(srcFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		jobs <- line
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error reading from source file %s: %v", srcPath, err)
	}
	close(jobs)

	// wait for writer to finish consuming all results
	writerWg.Wait()
}

// hasValidMX checks if a domain has a valid MX record
func hasValidMX(domain string) bool {
	mx, err := net.LookupMX(domain)
	if err != nil {
		return false
	}

	// Check if MX records exist
	if len(mx) == 0 {
		return false
	}

	// Check if MX record has a valid host
	if mx[0].Host == "" {
		return false
	}

	// a "." value means no SMTP service enabled
	if mx[0].Host == "." {
		return false
	}

	// All checks passed
	return true
}
