package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	// 1. 检查命令行参数
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <path_to_markdown_file_or_directory>")
		os.Exit(1)
	}
	inputPath := os.Args[1]

	// 2. 判断输入路径是文件还是目录
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		log.Fatalf("Error accessing path '%s': %v", inputPath, err)
	}

	var wg sync.WaitGroup
	if fileInfo.IsDir() {
		// 3. 如果是目录，则遍历并转换所有 .md 文件
		fmt.Printf("Scanning directory: %s\n", inputPath)
		err := filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				wg.Add(1)
				go func(mdPath string) {
					defer wg.Done()
					if err := convertToPDF(mdPath); err != nil {
						log.Printf("Failed to convert '%s': %v", mdPath, err)
					}
				}(path)
			}
			return nil
		})
		if err != nil {
			log.Fatalf("Error walking directory '%s': %v", inputPath, err)
		}
	} else {
		// 4. 如果是文件，则直接转换
		if !strings.HasSuffix(strings.ToLower(inputPath), ".md") {
			log.Fatalf("Error: Input file '%s' is not a markdown file.", inputPath)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := convertToPDF(inputPath); err != nil {
				log.Printf("Failed to convert '%s': %v", inputPath, err)
			}
		}()
	}

	wg.Wait()
	fmt.Println("All conversions completed.")
}
