package config

import (
	"bytes"
	"io/fs"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/afero"
)

var memeryName = "::memery::"

var _ afero.Fs = &BufferedFs{}
var _ afero.File = &BufferedFile{}
var _ os.FileInfo = &BufferedFileInfo{}

type BufferedFileInfo struct {
	buffer  *bytes.Buffer
	name    string
	modTime time.Time
}

// IsDir implements fs.FileInfo.
func (b *BufferedFileInfo) IsDir() bool {
	return false
}

// ModTime implements fs.FileInfo.
func (b *BufferedFileInfo) ModTime() time.Time {
	return b.modTime
}

// Mode implements fs.FileInfo.
func (b *BufferedFileInfo) Mode() fs.FileMode {
	return 0777
}

// Name implements fs.FileInfo.
func (b *BufferedFileInfo) Name() string {
	return b.name
}

// Size implements fs.FileInfo.
func (b *BufferedFileInfo) Size() int64 {
	return int64(b.buffer.Len())
}

// Sys implements fs.FileInfo.
func (b *BufferedFileInfo) Sys() any {
	return nil
}

type BufferedFile struct {
	info *BufferedFileInfo
}

// Close implements afero.File.
func (b *BufferedFile) Close() error {
	return nil
}

// Name implements afero.File.
func (b *BufferedFile) Name() string {
	return b.info.name
}

// Read implements afero.File.
func (b *BufferedFile) Read(p []byte) (n int, err error) {
	return b.info.buffer.Read(p)
}

// ReadAt implements afero.File.
func (b *BufferedFile) ReadAt(p []byte, off int64) (n int, err error) {
	return bytes.NewReader(b.info.buffer.Bytes()).ReadAt(p, off)
}

// Readdir implements afero.File.
func (b *BufferedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

// Readdirnames implements afero.File.
func (b *BufferedFile) Readdirnames(n int) ([]string, error) {
	return nil, nil
}

// Seek implements afero.File.
func (b *BufferedFile) Seek(offset int64, whence int) (int64, error) {
	return bytes.NewReader(b.info.buffer.Bytes()).Seek(offset, whence)
}

// Stat implements afero.File.
func (b *BufferedFile) Stat() (os.FileInfo, error) {
	return b.info, nil
}

// Sync implements afero.File.
func (b *BufferedFile) Sync() error {
	return nil
}

// Truncate implements afero.File.
func (b *BufferedFile) Truncate(size int64) error {
	var n int
	if size > int64(b.info.buffer.Len()) {
		n = b.info.buffer.Len()
	} else {
		n = int(size)
	}
	b.info.buffer.Truncate(n)
	return nil
}

// Write implements afero.File.
func (b *BufferedFile) Write(p []byte) (n int, err error) {
	return b.info.buffer.Write(p)
}

// WriteAt implements afero.File.
func (b *BufferedFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

// WriteString implements afero.File.
func (b *BufferedFile) WriteString(s string) (ret int, err error) {
	return b.info.buffer.WriteString(s)
}

type BufferedFs struct {
	buffers map[string]*BufferedFileInfo
	lck     sync.Mutex
	id      int64
}

func (b *BufferedFs) newBuf(name string) *BufferedFileInfo {
	b.lck.Lock()
	defer b.lck.Unlock()
	name = memeryName + name
	if v, ok := b.buffers[name]; ok {
		return v
	}
	b.buffers[name] = &BufferedFileInfo{buffer: bytes.NewBuffer(nil), name: name, modTime: time.Now()}
	return b.buffers[name]
}
func (b *BufferedFs) NewID() string {
	b.lck.Lock()
	defer b.lck.Unlock()
	b.id++
	return strconv.FormatInt(b.id, 10)
}

// Chmod implements afero.Fs.
func (b *BufferedFs) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Chown implements afero.Fs.
func (b *BufferedFs) Chown(name string, uid int, gid int) error {
	return nil
}

// Chtimes implements afero.Fs.
func (b *BufferedFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return nil
}

// Create implements afero.Fs.
func (b *BufferedFs) Create(name string) (afero.File, error) {
	return b.Open(name)
}

// Mkdir implements afero.Fs.
func (b *BufferedFs) Mkdir(name string, perm os.FileMode) error {
	return nil
}

// MkdirAll implements afero.Fs.
func (b *BufferedFs) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

// Name implements afero.Fs.
func (b *BufferedFs) Name() string {
	return memeryName
}

// Open implements afero.Fs.
func (b *BufferedFs) Open(name string) (afero.File, error) {
	return &BufferedFile{info: b.newBuf(name)}, nil
}

// OpenFile implements afero.Fs.
func (b *BufferedFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return b.Open(name)
}

// Remove implements afero.Fs.
func (b *BufferedFs) Remove(name string) error {
	b.lck.Lock()
	defer b.lck.Unlock()
	if v, ok := b.buffers[memeryName+name]; ok {
		v.buffer.Reset()
		delete(b.buffers, memeryName+name)
	}
	return nil
}

// RemoveAll implements afero.Fs.
func (b *BufferedFs) RemoveAll(path string) error {
	return nil
}

// Rename implements afero.Fs.
func (b *BufferedFs) Rename(oldname string, newname string) error {
	b.lck.Lock()
	defer b.lck.Unlock()
	if v, ok := b.buffers[oldname]; ok {
		b.buffers[memeryName+newname] = v
		delete(b.buffers, oldname)
	}
	return nil
}

// Stat implements afero.Fs.
func (b *BufferedFs) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

func NewBufferedFs() *BufferedFs {
	return &BufferedFs{
		buffers: make(map[string]*BufferedFileInfo),
	}
}
