package eventlog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	segmentMaxBytes = 16 << 20
	headerSize      = 28
	magic           = "SFE1"
)

var safeProjectID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Log struct {
	root  string
	fsync bool
	mu    sync.Mutex
	logs  map[string]*projectLog
}

type Record struct {
	ID        uint64
	Timestamp time.Time
	Payload   []byte
}

type projectLog struct {
	mu      sync.Mutex
	dir     string
	nextID  uint64
	segment uint64
	file    *os.File
	size    int64
}

func Open(root string, fsync bool) (*Log, error) {
	if root == "" {
		return nil, errors.New("eventlog root is required")
	}
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		return nil, err
	}
	return &Log{root: root, fsync: fsync, logs: map[string]*projectLog{}}, nil
}

func (l *Log) Append(projectID string, payload []byte) (uint64, error) {
	pl, err := l.openProject(projectID)
	if err != nil {
		return 0, err
	}
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.file == nil || pl.size+headerSize+int64(len(payload)) > segmentMaxBytes {
		if err := pl.rotate(); err != nil {
			return 0, err
		}
	}

	id := pl.nextID
	pl.nextID++
	buf := make([]byte, headerSize+len(payload))
	copy(buf[0:4], magic)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(payload)))
	binary.BigEndian.PutUint64(buf[8:16], id)
	binary.BigEndian.PutUint64(buf[16:24], uint64(time.Now().UnixNano()))
	copy(buf[28:], payload)
	crc := crc32.ChecksumIEEE(buf[0:24])
	crc = crc32.Update(crc, crc32.IEEETable, payload)
	binary.BigEndian.PutUint32(buf[24:28], crc)
	if _, err := pl.file.Write(buf); err != nil {
		return 0, err
	}
	pl.size += int64(len(buf))
	if l.fsync {
		if err := pl.file.Sync(); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (l *Log) ReadAfter(projectID string, after uint64, limit int) ([]Record, error) {
	if limit <= 0 {
		return nil, nil
	}
	dir, err := projectDir(l.root, projectID)
	if err != nil {
		return nil, err
	}
	files, err := segmentFiles(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Record, 0, limit)
	for _, path := range files {
		if err := readSegment(path, after, limit-len(out), &out); err != nil {
			return nil, err
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (l *Log) openProject(projectID string) (*projectLog, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if pl := l.logs[projectID]; pl != nil {
		return pl, nil
	}
	dir, err := projectDir(l.root, projectID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	pl := &projectLog{dir: dir, nextID: 1, segment: 1}
	if err := pl.recover(); err != nil {
		return nil, err
	}
	l.logs[projectID] = pl
	return pl, nil
}

func (pl *projectLog) recover() error {
	files, err := segmentFiles(pl.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, path := range files {
		seg, _ := strconv.ParseUint(filepath.Base(path)[:16], 10, 64)
		lastID, valid, err := scanSegment(path)
		if err != nil {
			return err
		}
		if err := os.Truncate(path, valid); err != nil {
			return err
		}
		if lastID >= pl.nextID {
			pl.nextID = lastID + 1
		}
		if seg >= pl.segment {
			pl.segment = seg
		}
	}
	if len(files) > 0 {
		last := files[len(files)-1]
		f, err := os.OpenFile(last, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		st, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return err
		}
		pl.file, pl.size = f, st.Size()
	}
	return nil
}

func (pl *projectLog) rotate() error {
	if pl.file != nil {
		_ = pl.file.Close()
		pl.segment++
	}
	path := filepath.Join(pl.dir, fmt.Sprintf("%016d.seg", pl.segment))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	pl.file, pl.size = f, st.Size()
	return nil
}

func projectDir(root, projectID string) (string, error) {
	if !safeProjectID.MatchString(projectID) {
		return "", fmt.Errorf("invalid project id %q", projectID)
	}
	return filepath.Join(root, "projects", projectID), nil
}

func segmentFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.seg"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func scanSegment(path string) (lastID uint64, valid int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	for {
		start := valid
		rec, n, err := readRecord(f)
		if err == io.EOF || err == io.ErrUnexpectedEOF || errChecksum(err) {
			return lastID, start, nil
		}
		if err != nil {
			return 0, 0, err
		}
		lastID = rec.ID
		valid += n
	}
}

func readSegment(path string, after uint64, limit int, out *[]Record) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for len(*out) < limit {
		rec, _, err := readRecord(f)
		if err == io.EOF || err == io.ErrUnexpectedEOF || errChecksum(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if rec.ID > after {
			*out = append(*out, rec)
		}
	}
	return nil
}

func readRecord(r io.Reader) (Record, int64, error) {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Record{}, 0, err
	}
	if string(header[0:4]) != magic {
		return Record{}, 0, checksumError{}
	}
	length := binary.BigEndian.Uint32(header[4:8])
	if length > 8<<20 {
		return Record{}, 0, fmt.Errorf("eventlog corrupt length %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Record{}, 0, err
	}
	storedCRC := binary.BigEndian.Uint32(header[24:28])
	crc := crc32.ChecksumIEEE(header[0:24])
	crc = crc32.Update(crc, crc32.IEEETable, payload)
	if crc != storedCRC {
		return Record{}, 0, checksumError{}
	}
	id := binary.BigEndian.Uint64(header[8:16])
	ts := int64(binary.BigEndian.Uint64(header[16:24]))
	return Record{ID: id, Timestamp: time.Unix(0, ts), Payload: payload}, int64(headerSize + length), nil
}

type checksumError struct{}

func (checksumError) Error() string { return "eventlog checksum mismatch" }

func errChecksum(err error) bool {
	var c checksumError
	return errors.As(err, &c)
}
