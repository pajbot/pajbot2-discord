package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func cleanUpAttachments() {
	// attachmentsMutex.Lock()
	// for i, a := range attachments {
	// 	// TODO: delete old attachments
	// }
	// attachmentsMutex.Unlock()

	fmt.Println("clean up attachments")
	attachmentsFolder := filepath.Join("attachments", "*")
	files, err := filepath.Glob(attachmentsFolder)
	if err != nil {
		fmt.Println("ERROR CLEANING UP ATTACHMENTS:", err)
		return
	}
	for _, path := range files {
		fi, err := os.Stat(path)
		if err != nil {
			fmt.Println("Error stating file:", path)
			continue
		}

		if time.Now().After(fi.ModTime().Add(timeToKeepLocalAttachments)) {
			fmt.Println("Deleting attachment", path)
			err := os.Remove(path)
			if err != nil {
				fmt.Println("Error deleting attachment", path, ":", err)
				continue
			}
		}
	}
}

func startCleanUpAttachments(ctx context.Context) {
	cleanUpAttachments()

	ticker := time.NewTicker(timeBetweenAttachmentCleanups)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			cleanUpAttachments()
		}
	}
}
