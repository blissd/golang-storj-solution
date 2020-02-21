package session

import (
	"fmt"
	"io"
	"net"
)

type state byte

const (
	initial state = 1
	started state = 2
	sending state = 3
)

const (

	// Encoded sender type
	MsgSend byte = 1

	// Encoded receiver type
	MsgRecv byte = 2

	// Encoded secret code
	msgSecretCode byte = 3

	// Encoded file name.
	msgFileName byte = 4

	// Encoded file length
	msgFileLength byte = 5
)

type Session struct {
	conn net.Conn
}

func (s *Session) Close() error {
	return s.conn.Close()
}

func New(addr string) (*Session, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	return &Session{conn: conn}, nil
}

func Attach(conn net.Conn) *Session {
	return &Session{conn: conn}
}

// get the first message sent to a new connection
func (s *Session) FirstByte() (byte, error) {
	bs, err := s.nextFrame()
	if err != nil {
		return 0, fmt.Errorf("first byte: %w", err)
	}
	if bs[0] != 1 {
		return 0, fmt.Errorf("must have length of 1, but is %v", bs[0])
	}

	return bs[1], nil
}

func (s *Session) SendFileName(name string) error {
	return s.sendString(msgFileName, name)
}

func (s *Session) RecvFileName() (string, error) {
	v, err := s.recvString(msgFileName)
	return v, err
}

func (s *Session) SendSecret(secret string) error {
	return s.sendString(msgSecretCode, secret)
}

func (s *Session) RecvSecret() (string, error) {
	v, err := s.recvString(msgSecretCode)
	return v, err
}

func (s *Session) SendFileLength(length uint32) error {
	bs, err := encodeUint32(msgFileLength, length)
	if err != nil {
		return fmt.Errorf("send file length: %w", err)
	}

	_, err = s.conn.Write(bs)
	if err != nil {
		return fmt.Errorf("send file length: %w", err)
	}
	return nil
}

func (s *Session) RecvFileLength() (uint32, error) {
	f, err := s.nextFrame()
	if err != nil {
		return 0, fmt.Errorf("recv file length: %w", err)
	}

	ft, v, err := decodeUint32(f)
	if err != nil {
		return 0, fmt.Errorf("recv file length: %w", err)
	} else if ft != msgFileLength {
		return 0, fmt.Errorf("expected %v, got %v", msgFileLength, ft)
	}
	return v, err
}

// Informs server that client is a receiver.
// Informs sender that receiver is connected and ready.
func (s *Session) SendSendReady() error {
	bs, err := EncodeByte(MsgSend)
	if err != nil {
		return fmt.Errorf("send ready: %w", err)
	}
	_, err = s.conn.Write(bs)
	return err
}

// Informs server that client is a receiver.
// Informs sender that receiver is connected and ready.
func (s *Session) SendRecvReady() error {
	bs, err := EncodeByte(MsgRecv)
	if err != nil {
		return fmt.Errorf("recv ready: %w", err)
	}
	_, err = s.conn.Write(bs)
	return err
}

// Informs server that client is a receiver.
// Informs sender that receiver is connected and ready.
func (s *Session) WaitForRecv() error {
	bs, err := s.nextFrame()
	if err != nil {
		return fmt.Errorf("wait for recv: %w", err)
	}
	b, err := DecodeByte(bs)
	if err != nil {
		return err
	}
	if b != MsgRecv {
		return fmt.Errorf("expected %v, got %v", MsgRecv, b)
	}
	return nil
}

func (s *Session) Send(r io.Reader) error {
	_, err := io.Copy(s.conn, r)
	return err
}

func (s *Session) Recv(w io.Writer, length int32) error {
	_, err := io.CopyN(w, s.conn, int64(length))
	return err
}

// reads the next from from the connection
func (s *Session) nextFrame() ([]byte, error) {
	length := make([]byte, 1)
	_, err := s.conn.Read(length)
	if err != nil {
		return nil, fmt.Errorf("next frame: %w", err)
	}

	frame := make([]byte, length[0]+1)
	frame[0] = length[0]
	_, err = s.conn.Read(frame[1:])
	return frame, err
}

func (s *Session) sendString(fieldType byte, v string) error {
	bs, err := encodeString(fieldType, v)
	if err != nil {
		return fmt.Errorf("send string: %w", err)
	}

	_, err = s.conn.Write(bs)
	if err != nil {
		return fmt.Errorf("send string: %w", err)
	}

	return nil
}

func (s *Session) recvString(fieldType byte) (string, error) {
	f, err := s.nextFrame()
	if err != nil {
		return "", fmt.Errorf("recv string: %w", err)
	}

	ft, v, err := decodeString(f)

	if err != nil {
		return "", fmt.Errorf("recv string: %w", err)
	} else if ft != fieldType {
		return "", fmt.Errorf("expected %v, got %v", fieldType, ft)
	}

	return v, nil
}