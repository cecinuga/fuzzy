package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
)

// reportFile is where a run writes its report. An incremental run writes it several
// times, each write replacing the whole file in one step, so a reader, or a run killed
// halfway through a write, always finds a complete report.
type reportFile struct {
	path string
	mode os.FileMode

	// direct is set for the single write of a run that isn't incremental, or when
	// the path can't be replaced: a pipe or a device, which must not be swapped for
	// a regular file, or a writable file in a directory that isn't.
	direct *os.File
}

// openReportFile creates the file up front, so that a bad path fails before the
// fuzzing, and empties it, so that an old report can't pass for this run's.
func openReportFile(path string, incremental bool) (*reportFile, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if !incremental {
		return &reportFile{path: path, direct: file}, nil
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	// Replacing a symlink's target keeps the link, instead of turning it into a file.
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}
	r := &reportFile{path: target, mode: info.Mode().Perm()}
	if !info.Mode().IsRegular() || !canCreateIn(filepath.Dir(target)) {
		r.direct = file
		return r, nil
	}
	return r, file.Close()
}

func canCreateIn(dir string) bool {
	probe, err := os.CreateTemp(dir, ".fuzzy-*")
	if err != nil {
		return false
	}
	probe.Close()
	return os.Remove(probe.Name()) == nil
}

// replaceable tells whether the file can take more than one write.
func (r *reportFile) replaceable() bool { return r.direct == nil }

// write stores what render produces as the file's new content.
func (r *reportFile) write(render func(io.Writer) error) error {
	if r.direct != nil {
		buffered := bufio.NewWriter(r.direct)
		if err := render(buffered); err != nil {
			return err
		}
		return buffered.Flush()
	}

	tmp, err := os.CreateTemp(filepath.Dir(r.path), "."+filepath.Base(r.path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // fails harmlessly once the file has been renamed
	defer tmp.Close()

	buffered := bufio.NewWriter(tmp)
	if err := render(buffered); err != nil {
		return err
	}
	if err := buffered.Flush(); err != nil {
		return err
	}
	if err := tmp.Chmod(r.mode); err != nil {
		return err
	}
	// Without this, a power cut after the rename could leave an empty report.
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), r.path)
}

// close finishes a direct write. A replaceable file has nothing left to close.
func (r *reportFile) close() error {
	if r.direct == nil {
		return nil
	}
	direct := r.direct
	r.direct = nil
	return direct.Close()
}
