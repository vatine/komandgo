package protocol
// Protocol implementation for the KomAndGo clinet

import (
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	
	"github.com/vatine/komandgo/pkg/hollerith"
	"github.com/vatine/komandgo/pkg/types"
)

// A callback is expected to expose two methods, one (OK) for dealing
// with successful respones and one (Error) for dealing with error
// responses. It will be called as soon as the main client loop has
// managed to map a response to a given callback and it is the
// responsibility of the callback to (quickly) consume all the
// remaining data of the response.
type Callback interface {
	OK(io.Reader)
	Error(io.Reader)
}

// The mapLock serves a dual purpose, it locks the nextRequest counter
// and it synchronises access to the asyncMap
type KomClient struct {
	mapLock sync.Mutex
	socket io.ReadWriter
	asyncMap map[uint32]Callback
	nextRequest uint32
	server *KomServer
	shutdown chan struct{}
}

func NewKomClient(name string) (*KomClient, error) {
	server, err := GetServer(name)
	if err != nil {
		return nil, err
	}
	return internalNewClient(name, server)
}

func internalNewClient(name string, server *KomServer) (*KomClient, error) {
	rv := KomClient{
		asyncMap: make(map[uint32]Callback),
		server: server,
	}
	s, err := net.Dial("tcp", name)
	if err != nil {
		return nil, err
	}
	rv.socket = s
	return &rv, nil
}

// Skips to the next linefeed character in the stream
func skipToNewline(r io.Reader) {
	c, err := readByte(r)
	for c != 10 {
		c, err = readByte(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("error skipping to newline")
		}
	}
}

// Read a single byte from an io.Reader
func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return 0, err
		}
		if n == 1 {
			return buf[0], nil
		}
	}

	return 0, nil
}

// The generic "success is empty, failure is complicated" response
type genericResponse struct {
	err error
}

type genericCallback chan genericResponse

func (c genericCallback) OK(r io.Reader) {
	c <- genericResponse{nil}
}

func (c genericCallback) Error(r io.Reader) {
	var errorCode, errorStatus, reqID uint32
	
	n, err := fmt.Fscanf(r, "%d %d", &reqID, &errorCode, &errorStatus)
	skipToNewline(r)
	if err != nil || n != 2 {
		log.WithFields(log.Fields{
			"n": n,
			"error": err,
		}).Errorf("Generic Callback, fscanf error.")
		c <- genericResponse{err}
		return
	}

	resp := genericResponse{
		err: fmt.Errorf("Generic error, code %d, status %d", errorCode, errorStatus),
	}
	c <- resp
}

// Register a callback and return the corresponding request ID
func (k *KomClient) registerCallback(c Callback) uint32 {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	reqID := k.nextRequest
	k.nextRequest++
	k.asyncMap[reqID] = c

	return reqID
}

// Return the callback associated with a specific request and delete
// the request from the mapping.
func (k *KomClient) getCallback(id uint32) (Callback, error) {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	c, ok := k.asyncMap[id]
	if !ok {
		log.WithFields(log.Fields{
			"reqID": id,
		}).Error("non-existent request id")
		return c, fmt.Errorf("Unknown request %d", id)
	}
	delete(k.asyncMap, id)
	return c, nil
	
}

// Send a protocol string to the server, handle any and all errors.
func (k *KomClient) send(s string) error {
	b := []byte(s)
	offset := 0
	remains := len(b)
	done := false

	for !done {
		sent, err := k.socket.Write(b[offset:])
		if err != nil {
			return err
		}
		if sent >= remains {
			done = true
		} else {
			offset += sent
			remains -= sent
		}
	}

	k.socket.Write([]byte{10})
	
	return nil
}

// Run a continuous read loop on the 
func (k *KomClient) receiveLoop() {
	done := false

	for !done {
		select {
		case _, ok := <- k.shutdown:
			_ = ok
			done = true
			continue
		default:
			status, _ := readByte(k.socket)
			id := readID(k.socket)
			callback, _ := k.getCallback(id)
			switch {
			case status == '=':
				callback.OK(k.socket)
			case status == '%':
				callback.Error(k.socket)
			}
		}
		
	}
}

// Read an id from the client socket, also consumer the first whitespace after
func readID(r io.Reader) uint32 {
	var done bool
	var rv uint32

	for !done {
		b, err := readByte(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"b": b,
				"rv": rv,
			}).Error("read id")
			return rv
		}
		log.WithFields(log.Fields{
			"0": '0',
			"9": '9',
			"b": b,
		}).Debug("read id")

		switch {
		case (b >= '0') && (b <= '9'):
			rv = (10 * rv) + uint32(b - '0')
		default:
			done = true
		}
	}

	return rv
}

// Various protocol messages


// Log out, but don't terminate the current session, this is protocol message #1
func (k *KomClient) asyncLogout() (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 1", reqID)
	err := k.send(req)

	return rv, err
}

// Change current conference (protocol message #2)
func (k *KomClient) asyncChangeConferece(newConf string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(newConf)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 2 %d", reqID, confNo)
	err := k.send(req)

	return rv, err
}

// Change name of a conference/person (protocol message #3)
func (k *KomClient) asyncChangeName(conference, newName string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(conference)

	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 3 %d %s", reqID, confNo, hollerith.Sprint(newName))
	return rv, k.send(req)
}

// This sends the :change-what-i-am-doing" protocol message (#4)
func (k *KomClient) asyncChangeWhatIAmDoing(msg string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 4 %s", reqID, hollerith.Sprint(msg))

	return rv, k.send(req)
}

// This sends the set-priv-bits protocol message (#7)
func (k *KomClient) asyncSetPrivBits(person types.ConfNo, privBits types.PrivBits) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 7 %d %s", reqID, person, privBits.Repr())

	return rv, k.send(req)
}

// This sends the set-passwd protocol message (#8)
func (k *KomClient) asyncChangePassword(person types.ConfNo, oldPasswd, newPasswd string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 8 %d %s %s", reqID, person, hollerith.Sprint(oldPasswd), hollerith.Sprint(newPasswd))

	return rv, k.send(req)
}

// This sends the "delete-conf"  protocol message (# 11)and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncDeleteConference(conferenceName string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confID := k.ConferenceFromName(conferenceName)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 11 %d", reqID, confID)

	err := k.send(req)
	return rv, err
}

// This sends the "sub-member" protocol message (#15) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSubMember(person, conference string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	personID := k.PersonFromName(person)
	confID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 15 %d %d", reqID, confID, personID)

	err := k.send(req)
	return rv, err
}

// This sends the set-presentation protocol message (#16) and returns
// a channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetPresentation(conference string, text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 16 %d %d", reqID, confID, text)

	err := k.send(req)
	return rv, err
}

// This sends the "login" protocol message (# 62) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncLogin(userName, password string, invisible bool) (chan genericResponse, error) {
	var visibility int
	if invisible {
		visibility = 1
	}
	rv := make(chan genericResponse)
	persNo := k.PersonFromName(userName)
	
	reqID := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 62 %d %s %d", reqID, persNo, hollerith.Sprint(password), visibility)
	err := k.send(req)

	return rv, err
}



// Various utility functions

func (k *KomClient) PersonFromName(user string) types.ConfNo {
	rv, ok := k.server.LookupUser(user)

	if ok {
		return rv
	}

	return 0
}

func (k *KomClient) ConferenceFromName(name string) types.ConfNo {
	rv, ok := k.server.LookupUser(name)

	if ok {
		return rv
	}

	rv, ok = k.server.LookupConference(name)
	if ok {
		return rv
	}

	return 0
}
