package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/go-checkpoint"
	"github.com/mitchellh/packer/packer"
)

func init() {
	packer.VersionChecker = packerVersionCheck
	checkpointResult = make(chan *checkpoint.CheckResponse, 1)
}

var checkpointResult chan *checkpoint.CheckResponse

// runCheckpoint runs a HashiCorp Checkpoint request. You can read about
// Checkpoint here: https://github.com/hashicorp/go-checkpoint.
func runCheckpoint(c *config) {
	// If the user doesn't want checkpoint at all, then return.
	if c.DisableCheckpoint {
		log.Printf("[INFO] Checkpoint disabled. Not running.")
		checkpointResult <- nil
		return
	}

	configDir, err := ConfigDir()
	if err != nil {
		log.Printf("[ERR] Checkpoint setup error: %s", err)
		checkpointResult <- nil
		return
	}

	version := packer.Version
	if packer.VersionPrerelease != "" {
		version += fmt.Sprintf(".%s", packer.VersionPrerelease)
	}

	signaturePath := filepath.Join(configDir, "checkpoint_signature")
	if c.DisableCheckpointSignature {
		log.Printf("[INFO] Checkpoint signature disabled")
		signaturePath = ""
	}

	resp, err := checkpoint.Check(&checkpoint.CheckParams{
		Product:       "packer",
		Version:       version,
		SignatureFile: signaturePath,
		CacheFile:     filepath.Join(configDir, "checkpoint_cache"),
	})
	if err != nil {
		log.Printf("[ERR] Checkpoint error: %s", err)
		resp = nil
	}

	checkpointResult <- resp
}

// packerVersionCheck implements packer.VersionCheckFunc and is used
// as the version checker.
func packerVersionCheck(current string) (packer.VersionCheckInfo, error) {
	info := <-checkpointResult
	if info == nil {
		var zero packer.VersionCheckInfo
		return zero, nil
	}

	alerts := make([]string, len(info.Alerts))
	for i, a := range info.Alerts {
		alerts[i] = a.Message
	}

	return packer.VersionCheckInfo{
		Outdated: info.Outdated,
		Latest:   info.CurrentVersion,
		Alerts:   alerts,
	}, nil
}
