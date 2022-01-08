package sandboxworker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/cretz/temporal-sdk-go-advanced/temporaldeterminism/sandboxworker/sandboxrt"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"golang.org/x/mod/modfile"
)

type Options struct {
	// Required
	TaskQueue string

	// This must reference a package-level function. No lambda or receiver methods
	// allowed. Caller should set options and register workflows and activities.
	// Note, this will be called multiple times, one for activity side and one for
	// workflow side.
	//
	// Required.
	InitWorker func(*worker.Options, worker.Registry) error

	// This must reference a package-level function. No lambda or receiver methods
	// allowed. Caller should set options. Note, this will be called multiple
	// times, one for activity side and one for workflow side.
	//
	// Optional.
	InitClient func(*client.Options) error

	// Any modules not included but referenced will let Go derive them on its own.
	IncludeModules []*Module

	// Defaults to os.TempDir().
	TempDirRoot string

	// Default is no log
	Logf func(string, ...interface{})
}

type Module struct {
	Name           string
	Version        string
	ReplaceVersion string
	// This should be an absolute path
	ReplaceDir string
}

type Worker struct {
	// These fields are not populated if
	TempDir string
	Cmd     *exec.Cmd

	activityOnlyWorker worker.Worker
}

type pendingActivityRegistration struct {
	activity interface{}
	options  activity.RegisterOptions
}

type pendingWorkflowRegistration struct {
	workflow interface{}
	options  workflow.RegisterOptions
}

// Worker is sometimes returned even when an error is, but it does not have to
// be closed unless there is no error.
func Build(options Options) (*Worker, error) {
	if options.InitWorker == nil {
		return nil, fmt.Errorf("must have InitWorker top-level function")
	}
	if options.InitClient == nil {
		options.InitClient = sandboxrt.DefaultInitClient
	}
	if options.Logf == nil {
		options.Logf = func(string, ...interface{}) {}
	}

	// Get function references
	initWorkerRef, err := newFuncRef(options.InitWorker)
	if err != nil {
		return nil, fmt.Errorf("invalid InitWorker function: %w", err)
	}
	if initClientRef, err := newFuncRef(options.InitClient); err != nil {
		return nil, fmt.Errorf("invalid InitClient function: %w", err)
	}

	// Create worker, closing on failure
	w := &Worker{}
	w.TempDir, err = os.MkdirTemp(options.TempDirRoot, "temporal-sandbox-")
	if err != nil {
		return nil, err
	}
	options.Logf("Created temp dir at %v", w.TempDir)
	success := false
	defer func() {
		if !success {
			w.Cleanup()
		}
	}()

	// Build
	options.Logf("Building executable")
	if err := w.build(&options, newClientRef, newWorkerRef); err != nil {
		return w, fmt.Errorf("failed building: %w", err)
	}

	success = true
	return w, nil
}

func (w *Worker) Start()

func (w *Worker) Close() error {
	if w.TempDir != "" {
		return os.RemoveAll(w.TempDir)
	}
}

func (w *Worker) build(options *Options, newClientRef, newWorkerRef *funcRef) error {
	// Write go.mod
	if err := w.writeGoModFile(options); err != nil {
		return fmt.Errorf("failed writing go.mod: %w", err)
	}

	// Write main.go
	if err := w.writeMainGoFile(newClientRef, newWorkerRef); err != nil {
		return fmt.Errorf("failed writing main.go: %w", err)
	}

	// Build
	outFile := "sandboxworker"
	if runtime.GOOS == "windows" {
		outFile += ".exe"
	}
	// TODO(cretz): Overlay
	args := []string{"build", "-o", outFile, "."}
	cmd := exec.Command("go", args...)
	cmd.Dir = w.TempDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed building executable (err: %w), output:\n%s", err, bytes.TrimSpace(out))
	}

	// Set the command on the worker
	w.Cmd = exec.Command(outFile)
	return nil
}

func (w *Worker) writeGoModFile(options *Options) error {
	var goMod modfile.File
	if err := goMod.AddModuleStmt("go.temporal.io/sandbox-worker-" + uuid.NewString()); err != nil {
		return err
	} else if err := goMod.AddGoStmt("1.17"); err != nil {
		return err
	}
	for _, mod := range options.IncludeModules {
		// If there is no version, we use the pseudo version
		vers := mod.Version
		if vers == "" {
			vers = "v0.0.0-00010101000000-000000000000"
		}
		// Add require
		if err := goMod.AddRequire(mod.Name, vers); err != nil {
			return fmt.Errorf("failed adding require for module %v: %w", mod.Name, err)
		}

		// Add replace
		if mod.ReplaceDir != "" || mod.ReplaceVersion != "" {
			var dir string
			if mod.ReplaceDir != "" {
				// We have to relativize it
				var err error
				if dir, err = filepath.Rel(w.TempDir, mod.ReplaceDir); err != nil {
					return fmt.Errorf("unable to relativize path from %v to %v: %w", w.TempDir, mod.ReplaceDir, err)
				}
				dir = filepath.ToSlash(dir)
				if !strings.HasPrefix(dir, ".") {
					dir = "./" + dir
				}
			}
			if err := goMod.AddReplace(mod.Name, "", dir, mod.ReplaceVersion); err != nil {
				return fmt.Errorf("failed replacing module %v: %w", mod.Name, err)
			}
		}
	}

	// Write
	b, err := goMod.Format()
	if err != nil {
		return fmt.Errorf("failed formatting: %w", err)
	}
	return os.WriteFile(filepath.Join(w.TempDir, "go.mod"), b, 0644)
}

func (w *Worker) writeMainGoFile(newClientRef, newWorkerRef *funcRef) error {
	return os.WriteFile(filepath.Join(w.TempDir, "main.go"), []byte(`package main

import (
	__new_client__ "`+strconv.Quote(newClientRef.pkg)+`"
	__new_worker__ "`+strconv.Quote(newWorkerRef.pkg)+`"
	__sandboxrt__ "github.com/cretz/temporal-sdk-go-advanced/temporaldeterminism/sandboxworker/sandboxrt"
)

func main() {
	__sandboxrt__.RunMain(__new_client__.`+newClientRef.name+`, __new_worker__.`+newWorkerRef.name+`)
}
`), 0644)
}
