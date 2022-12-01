package docwithrefs

import (
  "bytes"
  "encoding/binary"
)

type DocWithRefs struct {
	Data []byte
	Refs uint32
}

func (i DocWithRefs) Marshal() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(len(i.Data)))
	binary.Write(buf, binary.BigEndian, i.Data)
	binary.Write(buf, binary.BigEndian, uint32(i.Refs))
	return buf.Bytes()
}

func Unmarshal(b []byte) (DocWithRefs, error) {
  var i DocWithRefs

	buf := bytes.NewReader(b)
	var dataSize uint32
	binary.Read(buf, binary.BigEndian, &dataSize)
	i.Data = make([]byte, dataSize)
  binary.Read(buf, binary.BigEndian, &i.Data)
	var refs uint32
	binary.Read(buf, binary.BigEndian, &refs)
	i.Refs = refs

	return i, nil
}
