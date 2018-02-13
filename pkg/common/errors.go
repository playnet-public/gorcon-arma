package common

import "errors"

var (
	//ErrDisconnect .
	ErrDisconnect = errors.New("Connection lost")
	//ErrConnectionNil .
	ErrConnectionNil = errors.New("Connection is nil")
	//ErrTimeout .
	ErrTimeout = errors.New("Connection timeout")
	//ErrInvalidLogin .
	ErrInvalidLogin = errors.New("Login Invalid")
	//ErrLoginFailed .
	ErrLoginFailed = errors.New("Login failed")
	//ErrUnknownPacketType .
	ErrUnknownPacketType = errors.New("Received Unknown Packet Type")
	//ErrInvalidLoginPacket .
	ErrInvalidLoginPacket = errors.New("Received invalid Login Packet")
	//ErrInvalidChecksum .
	ErrInvalidChecksum = errors.New("Received invalid Packet Checksum")
	//ErrInvalidSizeNoHeader .
	ErrInvalidSizeNoHeader = errors.New("Invalid Packet Size, no Header found")
	//ErrInvalidSizeNoSequence .
	ErrInvalidSizeNoSequence = errors.New("Invalid Packet Size, no Sequence found")
	//ErrInvalidHeaderSize .
	ErrInvalidHeaderSize = errors.New("Invalid Packet Header Size")
	//ErrInvalidHeaderSyntax .
	ErrInvalidHeaderSyntax = errors.New("Invalid Packet Header Syntax")
	//ErrInvalidHeaderEnd .
	ErrInvalidHeaderEnd = errors.New("Invalid Packet Header end")
	//ErrInvalidSize .
	ErrInvalidSize = errors.New("Packet size too")
	//ErrConnected is not really an error but a information state
	ErrConnected = errors.New("Client Connected")
	//ErrKeepAliveAsync .
	ErrKeepAliveAsync = errors.New("keepAlive packets are out of sync")
	//ErrUnknownEventType .
	ErrUnknownEventType = errors.New("unknown event type")
	//ErrEscalationAttempt .
	ErrEscalationAttempt = errors.New("escalation attempt")
	//ErrUnableToParse .
	ErrUnableToParse = errors.New("unable to parse")
)
