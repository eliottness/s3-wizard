package main

import (
	fuseFs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type S3FS struct {
	/// Path of the loopback filesystem
	loopbackPath string

	/// Path of the mountpoint
	mountPath string

	/// All file handle by paths
	fhmap  map[string][]*S3File
	mutex  sync.Mutex
	logger *log.Logger

	config *ConfigPath

	server *fuse.Server
}

func NewS3FS(loopbackPath, mountPath string, config *ConfigPath) *S3FS {
	return &S3FS{
		loopbackPath: loopbackPath,
		mountPath:    mountPath,
		fhmap:        make(map[string][]*S3File),
		logger:       log.New(os.Stderr, mountPath+": ", log.LstdFlags),
		config:       config,
	}
}

/// We want to run this function in a goroutine
/// This function manages 1 Rule for 1 mountpoint
func (fs *S3FS) Run(debug bool) error {

	loopbackRoot, err := NewS3Root(fs.loopbackPath, fs)
	opts := &fuseFs.Options{}

	opts.MountOptions.Options = append(opts.MountOptions.Options, "default_permissions")
	// First column in "df -T": original dir
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname="+fs.loopbackPath)
	// Second column in "df -T" will be shown as "fuse." + Name
	opts.MountOptions.Name = "s3-agent"
	// Leave file permissions on "000" files as-is
	opts.NullPermissions = true

	// Linux will manage locks for us
	opts.MountOptions.EnableLocks = false
	opts.MountOptions.Debug = debug

	server, err := fuseFs.Mount(fs.mountPath, loopbackRoot, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
		fs.logger.Panicln(err)
	}

	fs.catchSignals()

	fs.server = server
	fs.server.Wait()

	return nil
}

func (fs *S3FS) Stop() error {
	return fs.server.Unmount()
}

// We have 7 Hooks on the Fuse calls
// 1. Rename        -> Rename entry in the DB
// 2. Unlink        -> Remove entry from the DB + if remote, remove the file from the S3
// 3. Download      -> The user needs the bytes in the file
// 4. GetSize       -> We need the replace the size of the file with the one from the S3
// 5. Create        -> Create a new file in the DB and register the file handler
// 6. RegisterFH    -> Register the file handle to the list of file handle related to the file
// 7. UnregisterFH  -> Unregister the file handle

/// Rename entry in the DB
func (fs *S3FS) Rename(oldPath, newPath string) error {

	fs.logger.Printf("Rename: %v -> %v\n", oldPath, newPath)

	// If the path does not point to a file, then we don't treat it
	if !fs.regFile(oldPath) {
		return nil
	}

	db := Open(fs.config)

	// The file does not need to be tracked or is local
	if IsEntryLocal(db, oldPath) {
		return nil
	}

	RenameEntry(db, oldPath, newPath)
	return nil
}

/// Remove entry from the DB + if remote, remove the file from the S3
func (fs *S3FS) Unlink(path string) error {

	fs.logger.Printf("Unlink: %v\n", path)

	// If the path does not point to a file, then we don't treat it
	if !fs.regFile(path) {
		return nil
	}

	db := Open(fs.config)

	var entries []S3NodeTable
	db.Where("Path = ?", path).Limit(1).Find(&entries)

	// The file does not need to be tracked
	if len(entries) == 0 {
		return nil
	}

	if !entries[0].IsLocal {
		// TODO Remove file from S3
	}

	DeleteEntry(db, &entries[0])

	return nil
}

/// We add the entry to the DB and we register the file handle
func (fs *S3FS) Create(fh *S3File) error {

	fs.logger.Printf("Create: %v\n", fh)

	stat, err := os.Stat(fh.Path)
	if err != nil {
		fs.logger.Printf("Error statting file: %v", err)
		return err
	}

	// If the path does not point to a file, then we don't treat it
	if !stat.Mode().IsRegular() {
		return nil
	}

	db := Open(fs.config)
	entry := NewEntry(fh.Path, stat.Size())
	db.Create(&entry).Commit()
	return fs.RegisterFH(fh)
}

/// if the file is local, returns
/// Find all file handle related to the file
/// Lock them all to be the only one to use the file
/// Download the file if it is not downloaded
/// Replace all file descritor by the new ones
/// Unlock all file handles
func (fs *S3FS) Download(path string) error {

	fs.logger.Printf("Download: %v\n", path)

	// If the path does not point to a file, then we don't treat it
	if !fs.regFile(path) {
		return nil
	}

	db := Open(fs.config)
	entry := GetEntry(db, path)

	// The file does not need to be tracked or the file is local
	if entry == nil || entry.IsLocal {
		return nil
	}

	// Lock all file handle related to the file
	fs.lockFHs(path)
	defer fs.unlockFHs(path)

	// TODO Download the file
	// Maybe flock the file but not sure if rclone will work as it will be a child process

	// Replace all file descriptor by the new ones
	if err := fs.reloadFds(path); err != nil {
		fs.logger.Printf("Error reloading file descriptors: %v", err)
		return err
	}

	RetriveFromServer(db, entry)

	return nil
}

func (fs *S3FS) GetSize(path string) (int64, error) {

	fs.logger.Printf("GetSize: %v\n", path)

	stat, err := os.Stat(path)
	if err != nil {
		fs.logger.Printf("Error statting file: %v", err)
		return -1, err
	}

	if !stat.Mode().IsRegular() {
		return stat.Size(), nil
	}

	db := Open(fs.config)
	entry := GetEntry(db, path)

	// The file does not need to be tracked or the file is local
	if entry == nil || entry.IsLocal {
		return stat.Size(), nil
	}

	// The file is remote so we need the fake size
	return entry.Size, nil

}

func (fs *S3FS) RegisterFH(fh *S3File) error {

	fs.logger.Printf("RegisterFH: %v\n", fh)

	// If the file handle does not point to a file, we do not register it
	if !fs.regFile(fh.Path) {
		return nil
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if _, ok := fs.fhmap[fh.Path]; !ok {
		fs.fhmap[fh.Path] = make([]*S3File, 0)
	}

	// Check that we don't have already the file handle in the map
	// If we do, and we don't check this we will lock twice the same mutex
	// and we will have a deadlock
	for _, otherFh := range fs.fhmap[fh.Path] {
		if otherFh == fh {
			return nil
		}
	}

	fs.fhmap[fh.Path] = append(fs.fhmap[fh.Path], fh)

	return nil
}

func (fs *S3FS) UnregisterFH(fh *S3File) error {
	fs.logger.Printf("UnregisterFH: %v\n", fh)

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if _, ok := fs.fhmap[fh.Path]; !ok {
		fs.logger.Println("WARN: UnregisterFH: file handle not found")
		return nil
	}

	index := -1
	for i, otherFh := range fs.fhmap[fh.Path] {
		if otherFh == fh {
			index = i
			break
		}
	}

	if index == -1 {
		fs.logger.Println("WARN: UnregisterFH: file handle not found")
		return nil
	}

	if len(fs.fhmap[fh.Path]) == 1 {
		delete(fs.fhmap, fh.Path)
	} else {
		ret := make([]*S3File, 0)
		ret = append(ret, fs.fhmap[fh.Path][:index]...)
		fs.fhmap[fh.Path] = append(ret, fs.fhmap[fh.Path][index+1:]...)
	}

	return nil
}

func (fs *S3FS) lockFHs(path string) {
	for _, fh := range fs.fhmap[path] {
		fh.Mutex.Lock()
	}
}

func (fs *S3FS) unlockFHs(path string) {
	for _, fh := range fs.fhmap[path] {
		fh.Mutex.Unlock()
	}
}

func (fs *S3FS) reloadFds(path string) error {
	for _, fh := range fs.fhmap[path] {
		fd, err := syscall.Open(fh.Path, int(fh.Flags), 0)
		if err != nil {
			return err
		}

		if fh.Fd != -1 {
			syscall.Close(fh.Fd)
		}

		fh.Fd = fd
	}

	return nil
}

func (fs *S3FS) regFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		fs.logger.Printf("Error statting file: %v", err)
		return false
	}

	return stat.Mode().IsRegular()
}

func (fs *S3FS) catchSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fs.logger.Printf("Unmounting: %v (Signal: %v)\n", fs.mountPath, sig)
		fs.Stop()
	}()
}
