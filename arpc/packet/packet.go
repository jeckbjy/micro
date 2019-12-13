package packet

import (
	"errors"
	"time"

	"github.com/jeckbjy/gsk/arpc"
	"github.com/jeckbjy/gsk/codec"
	"github.com/jeckbjy/gsk/util/buffer"
)

func New() arpc.Packet {
	return &Packet{}
}

type Packet struct {
	contentType uint
	command     uint
	ack         bool
	status      uint
	seqID       string
	msgID       uint16
	name        string
	method      string
	service     string
	extras      []string
	heads       map[string]string
	body        interface{}
	buffer      *buffer.Buffer
	codec       codec.Codec
	internal    interface{} // 以下字段不需要序列化
	ttl         time.Duration
	retry       int
	priority    int
	callInfo    *arpc.CallInfo
}

func (p *Packet) Reset() {
	*p = Packet{}
}

func (p *Packet) IsAck() bool {
	return p.ack
}

func (p *Packet) SetAck(ack bool) {
	p.ack = ack
}

func (p *Packet) Status() uint {
	return p.status
}

func (p *Packet) SetStatus(status uint) {
	p.status = status
}

func (p *Packet) ContentType() arpc.ContentType {
	return arpc.ContentType(p.contentType)
}

func (p *Packet) SetContentType(ct arpc.ContentType) {
	p.contentType = uint(ct)
}

func (p *Packet) Command() arpc.CommandType {
	return arpc.CommandType(p.command)
}

func (p *Packet) SetCommand(ct arpc.CommandType) {
	p.command = uint(ct)
}

func (p *Packet) SeqID() string {
	return p.seqID
}

func (p *Packet) SetSeqID(id string) {
	p.seqID = id
}

func (p *Packet) MsgID() uint16 {
	return p.msgID
}

func (p *Packet) SetMsgID(id uint16) {
	p.msgID = id
}

func (p *Packet) Name() string {
	return p.name
}

func (p *Packet) SetName(name string) {
	p.name = name
}

func (p *Packet) Method() string {
	return p.method
}

func (p *Packet) SetMethod(m string) {
	p.method = m
}

func (p *Packet) Service() string {
	return p.service
}

func (p *Packet) SetService(service string) {
	p.service = service
}

func (p *Packet) Extra(key uint) string {
	if key < uint(len(p.extras)) {
		return p.extras[key]
	}

	return ""
}

func (p *Packet) SetExtra(key uint, value string) error {
	if key >= arpc.HFExtraMax {
		return errors.New("bad extra key, must be in [0-6]")
	}

	if key >= uint(len(p.extras)) {
		extras := make([]string, key+1)
		copy(extras, p.extras)
	}
	p.extras[key] = value

	return nil
}

func (p *Packet) Head(key string) string {
	if p.heads != nil {
		return p.heads[key]
	}

	return ""
}

func (p *Packet) SetHead(key string, value string) {
	if p.heads == nil {
		p.heads = make(map[string]string)
	}

	p.heads[key] = value
}

func (p *Packet) Body() interface{} {
	return p.body
}

func (p *Packet) SetBody(body interface{}) {
	p.body = body
}

func (p *Packet) Codec() codec.Codec {
	return p.codec
}

func (p *Packet) SetCodec(c codec.Codec) {
	p.codec = c
}

func (p *Packet) Buffer() *buffer.Buffer {
	return p.buffer
}

func (p *Packet) SetBuffer(b *buffer.Buffer) {
	p.buffer = b
}

func (p *Packet) Internal() interface{} {
	return p.internal
}

func (p *Packet) SetInternal(value interface{}) {
	p.internal = value
}

func (p *Packet) TTL() time.Duration {
	return p.ttl
}

func (p *Packet) SetTTL(ttl time.Duration) {
	p.ttl = ttl
}

func (p *Packet) Retry() int {
	return p.retry
}

func (p *Packet) SetRetry(retry int) {
	p.retry = retry
}

func (p *Packet) Priority() int {
	return p.priority
}

func (p *Packet) SetPriority(value int) {
	p.priority = value
}

func (p *Packet) CallInfo() *arpc.CallInfo {
	return p.callInfo
}

func (p *Packet) SetCallInfo(info *arpc.CallInfo) {
	p.callInfo = info
}

func (p *Packet) Encode() error {
	if p.buffer == nil {
		p.buffer = buffer.New()
	}

	// encode head
	w := Writer{}
	w.Init()
	w.WriteBool(p.ack, 1<<arpc.HFAck)
	w.WriteUint(p.status, 1<<arpc.HFStatus)
	w.WriteUint(p.contentType, 1<<arpc.HFContentType)
	w.WriteUint(p.command, 1<<arpc.HFCommand)
	w.WriteString(p.seqID, 1<<arpc.HFSeqID)
	w.WriteUint(uint(p.msgID), 1<<arpc.HFMsgID)
	w.WriteString(p.name, 1<<arpc.HFName)
	w.WriteString(p.method, 1<<arpc.HFMethod)
	w.WriteString(p.service, 1<<arpc.HFService)
	w.WriteMap(p.heads, 1<<arpc.HFHeadMap)
	// extras
	if len(p.extras) > 0 {
		for i, v := range p.extras {
			w.WriteString(v, 1<<(uint(i)+arpc.HFExtra))
		}
	}

	p.buffer.Append(w.Flush())

	// encode body
	if p.body != nil {
		if body, ok := p.body.(*buffer.Buffer); ok {
			// 已经序列化好
			p.buffer.AppendBuffer(body)
		} else if p.codec != nil {
			buf := buffer.New()
			if err := p.codec.Encode(buf, p.body); err != nil {
				return err
			}
			p.buffer.AppendBuffer(buf)
		} else {
			return errors.New("packet no codec")
		}
	}

	return nil
}

func (p *Packet) Decode() (err error) {
	r := Reader{}
	if err := r.Init(p.buffer); err != nil {
		return err
	}

	// decode head
	r.ReadBool(&p.ack, 1<<arpc.HFAck)
	if err := r.ReadUint(&p.status, 1<<arpc.HFStatus); err != nil {
		return err
	}
	if err := r.ReadUint(&p.contentType, 1<<arpc.HFContentType); err != nil {
		return err
	}
	if err := r.ReadUint(&p.command, 1<<arpc.HFCommand); err != nil {
		return err
	}
	if err := r.ReadString(&p.seqID, 1<<arpc.HFSeqID); err != nil {
		return err
	}
	if err := r.ReadUint16(&p.msgID, 1<<arpc.HFMsgID); err != nil {
		return err
	}
	if err := r.ReadString(&p.name, 1<<arpc.HFName); err != nil {
		return err
	}
	if err := r.ReadString(&p.method, 1<<arpc.HFMethod); err != nil {
		return err
	}
	if err := r.ReadString(&p.service, 1<<arpc.HFService); err != nil {
		return err
	}
	if err := r.ReadMap(&p.heads, 1<<arpc.HFHeadMap); err != nil {
		return err
	}

	// extra
	if r.HasFlag(uint64(arpc.HFExtraMask)) {
		for i := arpc.HFExtra; i < arpc.HFMax; i++ {
			s, err := r.ReadStringDirect()
			if err != nil {
				return err
			}
			if s == "" {
				continue
			}
			key := i - arpc.HFExtra
			_ = p.SetExtra(uint(key), s)
		}
	}

	// lazy decode body
	if p.body != nil && p.codec != nil {
		if err := p.codec.Decode(p.buffer, p.body); err != nil {
			return err
		}
	}

	return nil
}

func (p *Packet) DecodeBody(msg interface{}) error {
	if p.codec == nil {
		return errors.New("no codec")
	}

	if err := p.codec.Decode(p.buffer, msg); err != nil {
		return err
	}

	p.body = msg
	return nil
}
