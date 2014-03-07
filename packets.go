package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
)

type ControlPacket interface {
	Pack() []byte
	Unpack([]byte)
	String() string
}

var packetNames = map[uint8]string{
	1:  "CONNECT",
	2:  "CONNACK",
	3:  "PUBLISH",
	4:  "PUBACK",
	5:  "PUBREC",
	6:  "PUBREL",
	7:  "PUBCOMP",
	8:  "SUBSCRIBE",
	9:  "SUBACK",
	10: "UNSUBSCRIBE",
	11: "UNSUBACK",
	12: "PINGREQ",
	13: "PINGRESP",
	14: "DISCONNECT",
}

const (
	CONNECT     = 1
	CONNACK     = 2
	PUBLISH     = 3
	PUBACK      = 4
	PUBREC      = 5
	PUBREL      = 6
	PUBCOMP     = 7
	SUBSCRIBE   = 8
	SUBACK      = 9
	UNSUBSCRIBE = 10
	UNSUBACK    = 11
	PINGREQ     = 12
	PINGRESP    = 13
	DISCONNECT  = 14
)

const (
	CONN_ACCEPTED          = 0x00
	CONN_REF_BAD_PROTO_VER = 0x01
	CONN_REF_ID_REJ        = 0x02
	CONN_REF_SERV_UNAVAIL  = 0x03
	CONN_REF_BAD_USER_PASS = 0x04
	CONN_REF_NOT_AUTH      = 0x05
)

func msgIdToBytes(messageId msgId) []byte {
	msgIdBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(msgIdBytes, uint16(messageId))
	return msgIdBytes
}

func bytesToMsgId(bytes []byte) msgId {
	return msgId(binary.BigEndian.Uint16(bytes))
}

func getType(typeByte []byte) byte {
	return typeByte[0] >> 4
}

func New(packetType byte) ControlPacket {
	switch packetType {
	case CONNECT:
		return &connectPacket{FixedHeader: FixedHeader{MessageType: CONNECT}, protocolName: "MQIsdp", protocolVersion: 3}
	case CONNACK:
		return &connackPacket{FixedHeader: FixedHeader{MessageType: CONNACK}}
	case DISCONNECT:
		return &disconnectPacket{FixedHeader: FixedHeader{MessageType: DISCONNECT}}
	case PUBLISH:
		return &publishPacket{FixedHeader: FixedHeader{MessageType: PUBLISH}}
	case PUBACK:
		return &pubackPacket{FixedHeader: FixedHeader{MessageType: PUBACK}}
	case PUBREC:
		return &pubrecPacket{FixedHeader: FixedHeader{MessageType: PUBREC}}
	case PUBREL:
		return &pubrelPacket{FixedHeader: FixedHeader{MessageType: PUBREL}}
	case PUBCOMP:
		return &pubcompPacket{FixedHeader: FixedHeader{MessageType: PUBCOMP}}
	case SUBSCRIBE:
		return &subscribePacket{FixedHeader: FixedHeader{MessageType: SUBSCRIBE, Qos: 1}}
	case SUBACK:
		return &subackPacket{FixedHeader: FixedHeader{MessageType: SUBACK}}
	case UNSUBSCRIBE:
		return &unsubscribePacket{FixedHeader: FixedHeader{MessageType: UNSUBSCRIBE}}
	case UNSUBACK:
		return &unsubackPacket{FixedHeader: FixedHeader{MessageType: UNSUBACK}}
	case PINGREQ:
		return &pingreqPacket{FixedHeader: FixedHeader{MessageType: PINGREQ}}
	case PINGRESP:
		return &pingrespPacket{FixedHeader: FixedHeader{MessageType: PINGRESP}}
	default:
		break
	}
	return nil
}

type FixedHeader struct {
	MessageType     byte
	Dup             byte
	Qos             byte
	Retain          byte
	remainingLength uint32
	length          int
}

func (fh FixedHeader) String() string {
	return fmt.Sprintf("%s: dup: %d qos: %d retain: %d rLength: %d", packetNames[fh.MessageType], fh.Dup, fh.Qos, fh.Retain, fh.remainingLength)
}

func (fh *FixedHeader) pack(size uint32) []byte {
	var header bytes.Buffer
	header.WriteByte(fh.MessageType<<4 | fh.Dup<<3 | fh.Qos<<1 | fh.Retain)
	header.Write(encode(size))
	return header.Bytes()
}

func (fh *FixedHeader) unpack(header byte) {
	fh.MessageType = header >> 4
	fh.Dup = (header >> 3) & 0x01
	fh.Qos = (header >> 1) & 0x03
	fh.Retain = header & 0x01
}

func encodeField(field string) []byte {
	fieldLength := make([]byte, 2)
	binary.BigEndian.PutUint16(fieldLength, uint16(len(field)))
	return append(fieldLength, []byte(field)...)
}

func decodeField(packet []byte) ([]byte, string, int) {
	if len(packet) == 0 {
		return packet, "", 0
	}
	fieldLength := binary.BigEndian.Uint16(packet[:2]) + 2
	return packet[fieldLength:], string(packet[2:fieldLength]), int(fieldLength)
}

func encode(length uint32) []byte {
	var encLength []byte
	for {
		digit := byte(length % 128)
		length /= 128
		if length > 0 {
			digit |= 0x80
		}
		encLength = append(encLength, digit)
		if length == 0 {
			break
		}
	}
	return encLength
}

func decodeLength(src *bufio.ReadWriter) uint32 {
	var rLength uint32
	var count int
	var multiplier uint32 = 1
	var digit byte
	count = 1
	for {
		digit, _ = src.ReadByte()
		rLength += uint32(digit&127) * multiplier
		if (digit & 128) == 0 {
			break
		}
		multiplier *= 128
		count++
	}
	return rLength
}

func messageType(mType byte) byte {
	return mType >> 4
}

//CONNECT packet

type connectPacket struct {
	FixedHeader
	protocolName    string
	protocolVersion byte
	cleanSession    byte
	willFlag        byte
	willQos         byte
	willRetain      byte
	usernameFlag    byte
	passwordFlag    byte
	reservedBit     byte
	keepaliveTimer  uint16

	clientIdentifier string
	willTopic        string
	willMessage      string
	username         string
	password         string
}

func (c *connectPacket) String() string {
	str := fmt.Sprintf("%s\n", c.FixedHeader)
	str += fmt.Sprintf("protocolversion: %d protocolname: %s cleansession: %d willflag: %d willqos: %d willretain: %d usernameflag: %d passwordflag: %d keepalivetimer: %d\nclientId: %s\nwilltopic: %s\nwillmessage: %s\nusername: %s\npassword: %s\n", c.protocolVersion, c.protocolName, c.cleanSession, c.willFlag, c.willQos, c.willRetain, c.usernameFlag, c.passwordFlag, c.keepaliveTimer, c.clientIdentifier, c.willTopic, c.willMessage, c.username, c.password)
	return str
}

func (c *connectPacket) Pack() []byte {
	var body []byte
	keepalive := make([]byte, 2)
	binary.BigEndian.PutUint16(keepalive, c.keepaliveTimer)
	body = append(body, encodeField(c.protocolName)...)
	body = append(body, c.protocolVersion)
	body = append(body, (c.cleanSession<<1 | c.willFlag<<2 | c.willQos<<3 | c.willRetain<<5 | c.passwordFlag<<6 | c.usernameFlag<<7))
	body = append(body, keepalive...)
	body = append(body, encodeField(c.clientIdentifier)...)
	if c.willFlag == 1 {
		body = append(body, encodeField(c.willTopic)...)
		body = append(body, encodeField(c.willMessage)...)
	}
	if c.usernameFlag == 1 {
		body = append(body, encodeField(c.username)...)
	}
	if c.passwordFlag == 1 {
		body = append(body, encodeField(c.password)...)
	}
	return append(c.FixedHeader.pack(uint32(len(body))), body...)
}

func (c *connectPacket) Unpack(packet []byte) {
	packet, c.protocolName, _ = decodeField(packet[c.FixedHeader.length:])
	c.protocolVersion = packet[0]
	options := packet[1]
	c.reservedBit = 1 & options
	c.cleanSession = 1 & (options >> 1)
	c.willFlag = 1 & (options >> 2)
	c.willQos = 3 & (options >> 3)
	c.willRetain = 1 & (options >> 5)
	c.passwordFlag = 1 & (options >> 6)
	c.usernameFlag = 1 & (options >> 7)
	c.keepaliveTimer = binary.BigEndian.Uint16(packet[2:4])
	packet, c.clientIdentifier, _ = decodeField(packet[4:])
	if c.willFlag == 1 {
		packet, c.willTopic, _ = decodeField(packet[:])
		packet, c.willMessage, _ = decodeField(packet[:])
	}
	if c.usernameFlag == 1 {
		packet, c.username, _ = decodeField(packet[:])
	}
	if c.passwordFlag == 1 {
		packet, c.password, _ = decodeField(packet[:])
	}
}

func (c *connectPacket) Validate() bool {
	if c.passwordFlag == 1 && c.usernameFlag != 1 {
		return false
	}
	if c.reservedBit != 0 {
		return false
	}
	if c.protocolName != "MQIsdp" && c.protocolName != "MQTT" {
		return false
	}
	if len(c.clientIdentifier) > 65535 || len(c.username) > 65535 || len(c.password) > 65535 {
		return false
	}
	return true
}

//CONNACK packet

type connackPacket struct {
	FixedHeader
	topicNameCompression byte
	returnCode           byte
}

func (ca *connackPacket) String() string {
	str := fmt.Sprintf("%s\n", ca.FixedHeader)
	str += fmt.Sprintf("returncode: %d", ca.returnCode)
	return str
}

func (ca *connackPacket) Pack() []byte {
	var body []byte
	body = append(body, ca.topicNameCompression)
	body = append(body, ca.returnCode)
	return append(ca.FixedHeader.pack(uint32(2)), body...)
}

func (ca *connackPacket) Unpack(packet []byte) {
	packet = packet[ca.FixedHeader.length:]
	ca.topicNameCompression = packet[0]
	ca.returnCode = packet[1]
}

//DISCONNECT packet

type disconnectPacket struct {
	FixedHeader
}

func (d *disconnectPacket) String() string {
	str := fmt.Sprintf("%s\n", d.FixedHeader)
	return str
}

func (d *disconnectPacket) Pack() []byte {
	return d.FixedHeader.pack(uint32(0))
}

func (d *disconnectPacket) Unpack(packet []byte) {
}

//PUBLISH packet

type publishPacket struct {
	FixedHeader
	topicName string
	messageId msgId
	payload   []byte
}

func (p *publishPacket) String() string {
	str := fmt.Sprintf("%s\n", p.FixedHeader)
	str += fmt.Sprintf("topicName: %s messageId: %d\n", p.topicName, p.messageId)
	str += fmt.Sprintf("payload: %s\n", string(p.payload))
	return str
}

func (p *publishPacket) Pack() []byte {
	var body []byte
	body = append(body, encodeField(p.topicName)...)
	if p.Qos > 0 {
		body = append(body, msgIdToBytes(p.messageId)...)
	}
	body = append(body, p.payload...)
	return append(p.FixedHeader.pack(uint32(len(body))), body...)
}

func (p *publishPacket) Unpack(packet []byte) {
	var skip int
	packet, p.topicName, skip = decodeField(packet[p.FixedHeader.length:])
	skip += p.FixedHeader.length
	if p.Qos > 0 {
		p.messageId = bytesToMsgId(packet[:2])
		p.payload = packet[2:]
	} else {
		p.payload = packet[:]
	}

}

//PUBACK packet

type pubackPacket struct {
	FixedHeader
	messageId msgId
}

func (pa *pubackPacket) String() string {
	str := fmt.Sprintf("%s\n", pa.FixedHeader)
	str += fmt.Sprintf("messageId: %d", pa.messageId)
	return str
}

func (pa *pubackPacket) Pack() []byte {
	return append(pa.FixedHeader.pack(uint32(2)), msgIdToBytes(pa.messageId)...)
}

func (pa *pubackPacket) Unpack(packet []byte) {
	pa.messageId = bytesToMsgId(packet[:2])
}

//PUBREC packet

type pubrecPacket struct {
	FixedHeader
	messageId msgId
}

func (pr *pubrecPacket) String() string {
	str := fmt.Sprintf("%s\n", pr.FixedHeader)
	str += fmt.Sprintf("messageId: %d", pr.messageId)
	return str
}

func (pr *pubrecPacket) Pack() []byte {
	return append(pr.FixedHeader.pack(uint32(2)), msgIdToBytes(pr.messageId)...)
}

func (pr *pubrecPacket) Unpack(packet []byte) {
	pr.messageId = bytesToMsgId(packet[:2])
}

//PUBREL packet

type pubrelPacket struct {
	FixedHeader
	messageId msgId
}

func (pr *pubrelPacket) String() string {
	str := fmt.Sprintf("%s\n", pr.FixedHeader)
	str += fmt.Sprintf("messageId: %d", pr.messageId)
	return str
}

func (pr *pubrelPacket) Pack() []byte {
	return append(pr.FixedHeader.pack(uint32(2)), msgIdToBytes(pr.messageId)...)
}

func (pr *pubrelPacket) Unpack(packet []byte) {
	pr.messageId = bytesToMsgId(packet[:2])
}

//PUBCOMP packet

type pubcompPacket struct {
	FixedHeader
	messageId msgId
}

func (pc *pubcompPacket) String() string {
	str := fmt.Sprintf("%s\n", pc.FixedHeader)
	str += fmt.Sprintf("messageId: %d", pc.messageId)
	return str
}

func (pc *pubcompPacket) Pack() []byte {
	fmt.Println("Outbound bytes", pc.FixedHeader.pack(uint32(2)))
	return append(pc.FixedHeader.pack(uint32(2)), msgIdToBytes(pc.messageId)...)
}

func (pc *pubcompPacket) Unpack(packet []byte) {
	pc.messageId = bytesToMsgId(packet[:2])
}

//SUBSCRIBE packet

type subscribePacket struct {
	FixedHeader
	messageId msgId
	payload   []byte
	topics    []string
	qoss      []uint
}

func (s *subscribePacket) String() string {
	str := fmt.Sprintf("%s\n", s.FixedHeader)
	//str += fmt.Sprintf("messageId: %d topics: %s", s.messageId, string(s.payload[:bytes.Index(s.payload, []byte{0})]))
	str += fmt.Sprintf("messageId: %d topics: %s", s.messageId, string(s.payload[:bytes.Index(s.payload, []byte{0})]))
	return str
}

func (s *subscribePacket) Pack() []byte {
	var body []byte
	body = append(body, msgIdToBytes(s.messageId)...)
	body = append(body, s.payload...)
	return append(s.FixedHeader.pack(uint32(len(body))), body...)
}

func (s *subscribePacket) Unpack(packet []byte) {
	s.messageId = bytesToMsgId(packet[0:2])
	s.payload = packet[2:]
	payload := packet[2:]
	var topic string
	for payload, topic, _ = decodeField(payload); topic != ""; payload, topic, _ = decodeField(payload) {
		s.topics = append(s.topics, topic)
		s.qoss = append(s.qoss, uint(payload[0]))
		payload = payload[1:]
	}
}

//SUBACK packet

type subackPacket struct {
	FixedHeader
	messageId   msgId
	grantedQoss []byte
}

func (sa *subackPacket) String() string {
	str := fmt.Sprintf("%s\n", sa.FixedHeader)
	str += fmt.Sprintf("messageId: %d", sa.messageId)
	return str
}

func (sa *subackPacket) Pack() []byte {
	var body []byte
	body = append(body, msgIdToBytes(sa.messageId)...)
	body = append(body, sa.grantedQoss...)
	return append(sa.FixedHeader.pack(uint32(len(body))), body...)
}

func (sa *subackPacket) Unpack(packet []byte) {
	sa.messageId = bytesToMsgId(packet[:2])
}

//UNSUBSCRIBE packet

type unsubscribePacket struct {
	FixedHeader
	messageId msgId
	payload   []byte
	topics    []string
}

func (u *unsubscribePacket) String() string {
	str := fmt.Sprintf("%s\n", u.FixedHeader)
	str += fmt.Sprintf("messageId: %d", u.messageId)
	return str
}

func (u *unsubscribePacket) Pack() []byte {
	var body []byte
	body = append(body, msgIdToBytes(u.messageId)...)
	body = append(body, u.payload...)
	return append(u.FixedHeader.pack(uint32(len(body))), body...)
}

func (u *unsubscribePacket) Unpack(packet []byte) {
	u.messageId = bytesToMsgId(packet[:2])
	u.payload = packet[2:]
	payload := packet[2:]
	var topic string
	for payload, topic, _ = decodeField(payload); topic != ""; payload, topic, _ = decodeField(payload) {
		u.topics = append(u.topics, topic)
	}
}

//UNSUBACK packet

type unsubackPacket struct {
	FixedHeader
	messageId msgId
}

func (ua *unsubackPacket) String() string {
	str := fmt.Sprintf("%s\n", ua.FixedHeader)
	str += fmt.Sprintf("messageId: %d", ua.messageId)
	return str
}

func (ua *unsubackPacket) Pack() []byte {
	return append(ua.FixedHeader.pack(uint32(2)), msgIdToBytes(ua.messageId)...)
}

func (ua *unsubackPacket) Unpack(packet []byte) {
	ua.messageId = bytesToMsgId(packet[:2])
}

//PINGREQ packet

type pingreqPacket struct {
	FixedHeader
}

func (pr *pingreqPacket) String() string {
	str := fmt.Sprintf("%s", pr.FixedHeader)
	return str
}

func (pr *pingreqPacket) Pack() []byte {
	return pr.FixedHeader.pack(uint32(0))
}

func (pr *pingreqPacket) Unpack(packet []byte) {
}

//PINGRESP packet

type pingrespPacket struct {
	FixedHeader
}

func (pr *pingrespPacket) String() string {
	str := fmt.Sprintf("%s", pr.FixedHeader)
	return str
}

func (pr *pingrespPacket) Pack() []byte {
	return pr.FixedHeader.pack(uint32(0))
}

func (pr *pingrespPacket) Unpack(packet []byte) {
}
