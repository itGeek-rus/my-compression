package service

import (
	"context"
	"fmt"
	"io"
	"my-compression/internal/archive"
	"my-compression/internal/job"
	"os"
	"path/filepath"
	"strings"
)

type Processor struct {
	jobs    *job.Store
	tempDir string
}

func NewProcessor(jobs *job.Store, tempDir string) *Processor {
	return &Processor{jobs: jobs, tempDir: tempDir}
}

type Input struct {
	JobID    string
	Action   string // archive | extract
	Format   string // zip | tar.gz
	Filename string
	SrcPath  string
	WorkDir  string
}

func (p *Processor) Run(ctx context.Context, in Input) {
	_, _ = p.jobs.Update(in.JobID, func(j *job.Job) {
		j.Status = job.StatusProcessing
		j.Message = "Processing"
		j.Progress = 5
	})

	progress := func(percent int, message string) {
		_, _ = p.jobs.Update(in.JobID, func(j *job.Job) {
			j.Status = job.StatusProcessing
			j.Progress = percent
			j.Message = message
		})
	}

	format, err := archive.ParseFormat(in.Format)
	if err != nil {
		p.fail(in.JobID, err)
		return
	}

	outDir := filepath.Join(in.WorkDir, "out")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		p.fail(in.JobID, err)
		return
	}

	var res archive.Result
	switch in.Action {
	case "archive":
		res, err = archive.Archive(ctx, in.SrcPath, outDir, format, progress)
	case "extract":
		res, err = archive.Extract(ctx, in.SrcPath, outDir, format, progress)
	default:
		err = fmt.Errorf("unsupported action: %s", in.Action)
	}
	if err != nil {
		p.fail(in.JobID, err)
		return
	}

	_, _ = p.jobs.Update(in.JobID, func(j *job.Job) {
		j.Status = job.StatusDone
		j.Message = "Done"
		j.Progress = 100
		j.OriginalSize = res.OriginalSize
		j.ResultSize = res.ResultSize
		j.ResultPath = res.OutputPath
		j.DownloadName = res.OutputName
		j.Error = ""
	})
}

func (p *Processor) fail(id string, err error) {
	_, _ = p.jobs.Update(id, func(j *job.Job) {
		j.Status = job.StatusFailed
		j.Message = "Failed"
		j.Progress = 100
		j.Error = err.Error()
	})
}

func SanitizeFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "upload.bin"
	}
	return name
}

func SaveUpload(dstPath string, r io.Reader) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
		return 0, err
	}
	f, err := os.Create(dstPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return n, err
	}
	return n, f.Sync()
}
