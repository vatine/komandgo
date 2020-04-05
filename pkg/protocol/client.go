package protocol
// Protocol implementation for the KomAndGo clinet

import (
	"fmt"
	"io"
	"net"
	"sync"

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
	asyncMap map[int32]Callback
	nextRequest int32
	server *KomServer
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
		asyncMap: make(map[int32]Callback),
		server: server,
	}
	s, err := net.Dial("tcp", name)
	if err != nil {
		return nil, err
	}
	rv.socket = s
	return &rv, nil
}

// The generic "success is empty, failure is complicated" response
type genericResponse struct {
	err error
}

type genericCallback chan genericResponse

func (c genericCallback) OK(io.Reader) {
	c <- genericResponse{nil}
}

func (c genericCallback) Error(io.Reader) {
}

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

// Various protocol messages


// Log out, but don't terminate the current session, this is protocol request #1
func (k *KomClient) asyncLogout() (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 1", reqId)
	err := k.send(req)

	return rv, err
}

// Change current conference (protocol message #2
func (k *KomClient) asyncChangeConferece(newConf string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(newConf)

	reqId := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 2 %d", reqId, confNo)
	err := k.send(req)

	return rv, err
}

// Change name of a conference/person (protocol message #3
func (k *KomClient) asyncChangeName(conference, newName string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(conference)

	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 3 %d %s", reqId, confNo, hollerith.Sprint(newName))
	return rv, k.send(req)
}

// This sends the :change-what-i-am-doing" protocol request (#4)
func (k *KomClient) asyncCHangeWhatIAmDoing(msg string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 4 %s", reqId, hollerith.Sprint(msg))

	return rv, k.send(req)
}

// This sends the set-priv-bits protocol message (#7)
func (k *KomClient) asyncSetPrivBits(person types.ConfNo, privBits types.PrivBits) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 7 %d %s", reqId, person, privBits.Repr())

	return rv, k.send(req)
}

// This sends the set-passwd protocol message (#8)
func (k *KomClient) asyncChangePassword(person types.ConfNo, oldPasswd, newPasswd string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 8 %d %s %s", reqId, person, hollerith.Sprint(oldPasswd), hollerith.Sprint(newPasswd))

	return rv, k.send(req)
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
	
	reqId := k.registerCallback(genericCallback(rv))

	req := fmt.Sprintf("%d 62 %d %s %d", reqId, persNo, hollerith.Sprint(password), visibility)
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


func (k *KomClient) registerCallback(c Callback) int32 {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	reqId := k.nextRequest
	k.nextRequest++
	k.asyncMap[reqId] = c

	return reqId
}
