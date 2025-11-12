package filesystem

import (
	"os"
	"time"

	"github.com/spf13/afero"
)

type TestFs struct {
	BaseFs        afero.Fs
	MockCreate    func(name string) (afero.File, error)
	MockMkdir     func(name string, perm os.FileMode) error
	MockMkdirAll  func(path string, perm os.FileMode) error
	MockOpen      func(name string) (afero.File, error)
	MockOpenFile  func(name string, flag int, perm os.FileMode) (afero.File, error)
	MockRemove    func(name string) error
	MockRemoveAll func(path string) error
	MockRename    func(oldname, newname string) error
	MockStat      func(name string) (os.FileInfo, error)
	MockChmod     func(name string, mode os.FileMode) error
	MockChtimes   func(name string, atime time.Time, mtime time.Time) error
}

func NewTestFs(baseFs afero.Fs) (test *TestFs) {
	return &TestFs{BaseFs: baseFs}
}

func (*TestFs) Name() (name string) {
	return "TestFs"
}

func (m *TestFs) Create(name string) (file afero.File, err error) {
	if m.MockCreate == nil {
		return m.BaseFs.Create(name)
	}

	return m.MockCreate(name)
}

func (m *TestFs) Mkdir(name string, perm os.FileMode) (err error) {
	if m.MockMkdir == nil {
		return m.BaseFs.Mkdir(name, perm)
	}

	return m.MockMkdir(name, perm)
}

func (m *TestFs) MkdirAll(path string, perm os.FileMode) (err error) {
	if m.MockMkdirAll == nil {
		return m.BaseFs.MkdirAll(path, perm)
	}

	return m.MockMkdirAll(path, perm)
}

func (m *TestFs) Open(name string) (file afero.File, err error) {
	if m.MockOpen == nil {
		return m.BaseFs.Open(name)
	}

	return m.MockOpen(name)
}

func (m *TestFs) OpenFile(name string, flag int, perm os.FileMode) (file afero.File, err error) {
	if m.MockOpenFile == nil {
		return m.BaseFs.OpenFile(name, flag, perm)
	}

	return m.MockOpenFile(name, flag, perm)
}

func (m *TestFs) Remove(name string) (err error) {
	if m.MockRemove == nil {
		return m.BaseFs.Remove(name)
	}

	return m.MockRemove(name)
}

func (m *TestFs) RemoveAll(path string) (err error) {
	if m.MockRemoveAll == nil {
		return m.BaseFs.RemoveAll(path)
	}

	return m.MockRemoveAll(path)
}

func (m *TestFs) Rename(oldname, newname string) (err error) {
	if m.MockRename == nil {
		return m.BaseFs.Rename(oldname, newname)
	}

	return m.MockRename(oldname, newname)
}

func (m *TestFs) Stat(name string) (info os.FileInfo, err error) {
	if m.MockStat == nil {
		return m.BaseFs.Stat(name)
	}

	return m.MockStat(name)
}

func (m *TestFs) Chmod(name string, mode os.FileMode) (err error) {
	if m.MockChmod == nil {
		return m.BaseFs.Chmod(name, mode)
	}

	return m.MockChmod(name, mode)
}

func (m *TestFs) Chtimes(name string, atime time.Time, mtime time.Time) (err error) {
	if m.MockChtimes == nil {
		return m.BaseFs.Chtimes(name, atime, mtime)
	}

	return m.MockChtimes(name, atime, mtime)
}
