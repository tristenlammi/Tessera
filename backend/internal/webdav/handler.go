package webdav

import (
	"context"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/models"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/storage"
)

// FileSystem implements a WebDAV file system backed by Tessera
type FileSystem struct {
	fileRepo *repository.FileRepository
	storage  *storage.MinIOStorage
	log      zerolog.Logger
}

// NewFileSystem creates a new WebDAV file system
func NewFileSystem(fileRepo *repository.FileRepository, storage *storage.MinIOStorage, log zerolog.Logger) *FileSystem {
	return &FileSystem{
		fileRepo: fileRepo,
		storage:  storage,
		log:      log,
	}
}

// FileInfo represents file information for WebDAV
type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *FileInfo) Name() string       { return fi.name }
func (fi *FileInfo) Size() int64        { return fi.size }
func (fi *FileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *FileInfo) IsDir() bool        { return fi.isDir }
func (fi *FileInfo) Sys() interface{}   { return nil }

// File represents a WebDAV file
type File struct {
	fs       *FileSystem
	userID   string
	fileID   string
	name     string
	isDir    bool
	size     int64
	modTime  time.Time
	reader   io.ReadCloser
	children []os.FileInfo
	childIdx int
}

func (f *File) Close() error {
	if f.reader != nil {
		return f.reader.Close()
	}
	return nil
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.reader == nil {
		return 0, io.EOF
	}
	return f.reader.Read(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, os.ErrInvalid
	}

	if f.children == nil {
		ctx := context.Background()
		userUUID, _ := uuid.Parse(f.userID)
		var parentUUID *uuid.UUID
		if f.fileID != "" {
			id, _ := uuid.Parse(f.fileID)
			parentUUID = &id
		}
		files, err := f.fs.fileRepo.ListByParent(ctx, userUUID, parentUUID, false)
		if err != nil {
			return nil, err
		}

		f.children = make([]os.FileInfo, len(files))
		for i, file := range files {
			mode := os.FileMode(0644)
			if file.IsFolder {
				mode = os.FileMode(0755) | os.ModeDir
			}
			f.children[i] = &FileInfo{
				name:    file.Name,
				size:    file.Size,
				mode:    mode,
				modTime: file.UpdatedAt,
				isDir:   file.IsFolder,
			}
		}
	}

	if count <= 0 {
		return f.children, nil
	}

	if f.childIdx >= len(f.children) {
		return nil, io.EOF
	}

	end := f.childIdx + count
	if end > len(f.children) {
		end = len(f.children)
	}

	result := f.children[f.childIdx:end]
	f.childIdx = end

	return result, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	mode := os.FileMode(0644)
	if f.isDir {
		mode = os.FileMode(0755) | os.ModeDir
	}
	return &FileInfo{
		name:    f.name,
		size:    f.size,
		mode:    mode,
		modTime: f.modTime,
		isDir:   f.isDir,
	}, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	return 0, os.ErrPermission
}

// OpenFile opens a file for WebDAV access
func (fs *FileSystem) OpenFile(ctx context.Context, userID string, name string, flag int, perm os.FileMode) (*File, error) {
	fs.log.Debug().Str("path", name).Str("user", userID).Msg("WebDAV OpenFile")

	if name == "/" || name == "" {
		return &File{
			fs:      fs,
			userID:  userID,
			fileID:  "",
			name:    "/",
			isDir:   true,
			modTime: time.Now(),
		}, nil
	}

	name = path.Clean("/" + name)
	file, err := fs.resolveFile(ctx, userID, name)
	if err != nil {
		return nil, err
	}

	f := &File{
		fs:      fs,
		userID:  userID,
		fileID:  file.ID.String(),
		name:    file.Name,
		isDir:   file.IsFolder,
		size:    file.Size,
		modTime: file.UpdatedAt,
	}

	if !file.IsFolder && (flag&os.O_RDONLY != 0 || flag == 0) {
		reader, err := fs.storage.Download(ctx, file.StorageKey)
		if err != nil {
			return nil, err
		}
		f.reader = reader
	}

	return f, nil
}

// Stat returns file info
func (fs *FileSystem) Stat(ctx context.Context, userID string, name string) (os.FileInfo, error) {
	fs.log.Debug().Str("path", name).Str("user", userID).Msg("WebDAV Stat")

	if name == "/" || name == "" {
		return &FileInfo{
			name:    "/",
			mode:    os.FileMode(0755) | os.ModeDir,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	name = path.Clean("/" + name)
	file, err := fs.resolveFile(ctx, userID, name)
	if err != nil {
		return nil, err
	}

	mode := os.FileMode(0644)
	if file.IsFolder {
		mode = os.FileMode(0755) | os.ModeDir
	}

	return &FileInfo{
		name:    file.Name,
		size:    file.Size,
		mode:    mode,
		modTime: file.UpdatedAt,
		isDir:   file.IsFolder,
	}, nil
}

// Mkdir creates a directory
func (fs *FileSystem) Mkdir(ctx context.Context, userID string, name string, perm os.FileMode) error {
	fs.log.Debug().Str("path", name).Str("user", userID).Msg("WebDAV Mkdir")

	name = path.Clean("/" + name)
	dir := path.Dir(name)
	baseName := path.Base(name)

	userUUID, _ := uuid.Parse(userID)
	var parentUUID *uuid.UUID
	if dir != "/" && dir != "" {
		parent, err := fs.resolveFile(ctx, userID, dir)
		if err != nil {
			return err
		}
		parentUUID = &parent.ID
	}

	folder := &models.File{
		OwnerID:  userUUID,
		ParentID: parentUUID,
		Name:     baseName,
		IsFolder: true,
	}
	return fs.fileRepo.Create(ctx, folder)
}

// RemoveAll removes a file or directory
func (fs *FileSystem) RemoveAll(ctx context.Context, userID string, name string) error {
	fs.log.Debug().Str("path", name).Str("user", userID).Msg("WebDAV RemoveAll")

	name = path.Clean("/" + name)
	file, err := fs.resolveFile(ctx, userID, name)
	if err != nil {
		return err
	}

	return fs.fileRepo.MoveToTrash(ctx, file.ID)
}

// Rename moves/renames a file
func (fs *FileSystem) Rename(ctx context.Context, userID string, oldName, newName string) error {
	fs.log.Debug().Str("old", oldName).Str("new", newName).Str("user", userID).Msg("WebDAV Rename")

	oldName = path.Clean("/" + oldName)
	newName = path.Clean("/" + newName)

	file, err := fs.resolveFile(ctx, userID, oldName)
	if err != nil {
		return err
	}

	oldDir := path.Dir(oldName)
	newDir := path.Dir(newName)
	newBaseName := path.Base(newName)

	if oldDir == newDir {
		file.Name = newBaseName
		return fs.fileRepo.Update(ctx, file)
	}

	var newParentUUID *uuid.UUID
	if newDir != "/" && newDir != "" {
		parent, err := fs.resolveFile(ctx, userID, newDir)
		if err != nil {
			return err
		}
		newParentUUID = &parent.ID
	}

	file.Name = newBaseName
	file.ParentID = newParentUUID
	return fs.fileRepo.Update(ctx, file)
}

// resolveFile resolves a path to a file
func (fs *FileSystem) resolveFile(ctx context.Context, userID string, filePath string) (*models.File, error) {
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil, os.ErrNotExist
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, os.ErrNotExist
	}

	var currentParentID *uuid.UUID
	var currentFile *models.File

	for _, part := range parts {
		if part == "" {
			continue
		}

		file, err := fs.fileRepo.GetByName(ctx, userUUID, currentParentID, part)
		if err != nil {
			return nil, os.ErrNotExist
		}

		currentFile = file
		currentParentID = &file.ID
	}

	if currentFile == nil {
		return nil, os.ErrNotExist
	}

	return currentFile, nil
}
