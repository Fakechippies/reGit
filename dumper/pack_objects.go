package dumper

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

type packedRecord struct {
	offset     int64
	objectType int
	baseOffset int64
	baseSHA    string
	data       []byte
	resolved   *resolvedObject
}

type resolvedObject struct {
	objectType string
	data       []byte
}

func (d *Dumper) fetchPackedObject(sha string, looseErr error) ([]byte, error) {
	if err := d.loadPacks(); err != nil {
		return nil, looseErr
	}

	object, ok := d.packedObjects[sha]
	if !ok {
		return nil, looseErr
	}

	return object, nil
}

func (d *Dumper) loadPacks() error {
	d.packsMu.Lock()
	defer d.packsMu.Unlock()

	if d.packsLoaded {
		return nil
	}
	d.packedObjects = map[string][]byte{}

	packs, err := d.fetchPackNames()
	if err != nil {
		return err
	}

	for _, pack := range packs {
		data, err := d.fetchPack(pack)
		if err != nil {
			continue
		}

		objects, err := parsePack(data)
		if err != nil {
			continue
		}

		for sha, object := range objects {
			d.packedObjects[sha] = object
		}
	}

	d.packsLoaded = true
	return nil
}

func parsePack(data []byte) (map[string][]byte, error) {
	if len(data) < 12 || string(data[:4]) != "PACK" {
		return nil, fmt.Errorf("invalid pack")
	}

	count := int(binary.BigEndian.Uint32(data[8:12]))
	reader := bytes.NewReader(data[12:])
	records := make([]*packedRecord, 0, count)
	recordsByOffset := map[int64]*packedRecord{}

	for range count {
		offset := int64(len(data)) - int64(reader.Len())
		record, err := readPackedRecord(reader, offset)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
		recordsByOffset[offset] = record
	}

	objects := map[string][]byte{}
	recordsBySHA := map[string]*packedRecord{}
	for _, record := range records {
		resolved, err := resolvePackedRecord(record, recordsByOffset, recordsBySHA)
		if err != nil {
			continue
		}

		encoded := encodeObject(resolved.objectType, resolved.data)
		sha := objectSHA(resolved.objectType, resolved.data)
		recordsBySHA[sha] = record
		objects[sha] = encoded
	}

	return objects, nil
}

func readPackedRecord(reader *bytes.Reader, offset int64) (*packedRecord, error) {
	first, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	objectType := int((first >> 4) & 0x7)
	for first&0x80 != 0 {
		first, err = reader.ReadByte()
		if err != nil {
			return nil, err
		}
	}

	record := &packedRecord{
		offset:     offset,
		objectType: objectType,
	}

	switch objectType {
	case 6:
		baseOffset, err := readOffsetDeltaBase(reader, offset)
		if err != nil {
			return nil, err
		}
		record.baseOffset = baseOffset
	case 7:
		base := make([]byte, 20)
		if _, err := io.ReadFull(reader, base); err != nil {
			return nil, err
		}
		record.baseSHA = hex.EncodeToString(base)
	}

	zlibReader, err := zlib.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer zlibReader.Close()

	record.data, err = io.ReadAll(zlibReader)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func readOffsetDeltaBase(reader *bytes.Reader, currentOffset int64) (int64, error) {
	value, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}

	offset := int64(value & 0x7f)
	for value&0x80 != 0 {
		value, err = reader.ReadByte()
		if err != nil {
			return 0, err
		}
		offset = ((offset + 1) << 7) | int64(value&0x7f)
	}

	return currentOffset - offset, nil
}

func resolvePackedRecord(record *packedRecord, byOffset map[int64]*packedRecord, bySHA map[string]*packedRecord) (*resolvedObject, error) {
	if record.resolved != nil {
		return record.resolved, nil
	}

	switch record.objectType {
	case 1, 2, 3, 4:
		record.resolved = &resolvedObject{
			objectType: packedTypeName(record.objectType),
			data:       record.data,
		}
		return record.resolved, nil
	case 6:
		base, ok := byOffset[record.baseOffset]
		if !ok {
			return nil, fmt.Errorf("missing offset delta base")
		}
		return resolveDelta(record, base, byOffset, bySHA)
	case 7:
		base, ok := bySHA[record.baseSHA]
		if !ok {
			return nil, fmt.Errorf("missing ref delta base")
		}
		return resolveDelta(record, base, byOffset, bySHA)
	default:
		return nil, fmt.Errorf("unsupported pack object type %d", record.objectType)
	}
}

func resolveDelta(record, base *packedRecord, byOffset map[int64]*packedRecord, bySHA map[string]*packedRecord) (*resolvedObject, error) {
	baseObject, err := resolvePackedRecord(base, byOffset, bySHA)
	if err != nil {
		return nil, err
	}

	data, err := applyDelta(baseObject.data, record.data)
	if err != nil {
		return nil, err
	}

	record.resolved = &resolvedObject{
		objectType: baseObject.objectType,
		data:       data,
	}
	return record.resolved, nil
}

func applyDelta(base, delta []byte) ([]byte, error) {
	reader := bytes.NewReader(delta)
	if _, err := readDeltaSize(reader); err != nil {
		return nil, err
	}
	targetSize, err := readDeltaSize(reader)
	if err != nil {
		return nil, err
	}

	output := make([]byte, 0, targetSize)
	for reader.Len() > 0 {
		opcode, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		if opcode&0x80 != 0 {
			offset := 0
			size := 0
			if opcode&0x01 != 0 {
				value, _ := reader.ReadByte()
				offset |= int(value)
			}
			if opcode&0x02 != 0 {
				value, _ := reader.ReadByte()
				offset |= int(value) << 8
			}
			if opcode&0x04 != 0 {
				value, _ := reader.ReadByte()
				offset |= int(value) << 16
			}
			if opcode&0x08 != 0 {
				value, _ := reader.ReadByte()
				offset |= int(value) << 24
			}
			if opcode&0x10 != 0 {
				value, _ := reader.ReadByte()
				size |= int(value)
			}
			if opcode&0x20 != 0 {
				value, _ := reader.ReadByte()
				size |= int(value) << 8
			}
			if opcode&0x40 != 0 {
				value, _ := reader.ReadByte()
				size |= int(value) << 16
			}
			if size == 0 {
				size = 0x10000
			}
			if offset+size > len(base) {
				return nil, fmt.Errorf("delta copy outside base")
			}
			output = append(output, base[offset:offset+size]...)
			continue
		}

		if opcode == 0 {
			return nil, fmt.Errorf("invalid delta opcode")
		}

		insert := make([]byte, int(opcode))
		if _, err := io.ReadFull(reader, insert); err != nil {
			return nil, err
		}
		output = append(output, insert...)
	}

	if len(output) != targetSize {
		return nil, fmt.Errorf("invalid delta target size")
	}

	return output, nil
}

func readDeltaSize(reader *bytes.Reader) (int, error) {
	shift := 0
	size := 0

	for {
		value, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		size |= int(value&0x7f) << shift
		if value&0x80 == 0 {
			return size, nil
		}
		shift += 7
	}
}

func packedTypeName(objectType int) string {
	switch objectType {
	case 1:
		return "commit"
	case 2:
		return "tree"
	case 3:
		return "blob"
	case 4:
		return "tag"
	default:
		return ""
	}
}

func encodeObject(objectType string, data []byte) []byte {
	header := fmt.Sprintf("%s %d\x00", objectType, len(data))
	return append([]byte(header), data...)
}

func objectSHA(objectType string, data []byte) string {
	sum := sha1.Sum(encodeObject(objectType, data))
	return hex.EncodeToString(sum[:])
}
