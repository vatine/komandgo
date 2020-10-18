package protocol

// Protocol implementation for the KomAndGo client

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vatine/komandgo/pkg/hollerith"
	"github.com/vatine/komandgo/pkg/types"
	"github.com/vatine/komandgo/pkg/utils"
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
	mapLock     sync.Mutex
	socket      io.ReadWriter
	asyncMap    map[uint32]Callback
	nextRequest uint32
	server      *KomServer
	shutdown    chan struct{}
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
		server:   server,
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
	c, err := utils.ReadByte(r)
	for c != 10 {
		c, err = utils.ReadByte(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("error skipping to newline")
			return
		}
	}
}

// Read from a stream until we've read a full delimited list, then
// return a string that is composed of the read bytes. If the start is
// sent in as 0, we skip matching the start. On error, simply return
// what's been read so far and the error seen
func readDelimitedList(start, end byte, r io.Reader) (string, error) {
	var rv []byte

	b, err := utils.ReadByte(r)
	if err != nil || (start != 0 && b != start) {
		log.WithFields(log.Fields{
			"b":     b,
			"start": start,
			"end":   end,
		}).Error("Unexpected start of list")
		return "", fmt.Errorf("Unexpected start '%c'", b)
	}
	rv = append(rv, b)

	for {
		b, err := utils.ReadByte(r)
		if err != nil {
			return string(rv), err
		}
		rv = append(rv, b)
		if b == end {
			return string(rv), nil
		}
	}

	return "", nil
}

// The generic "success is empty, failure is complicated" response
type genericResponse struct {
	code   uint32
	status uint32
	err    error
}

type genericCallback chan genericResponse

func (c genericCallback) OK(r io.Reader) {
	go func() { c <- genericResponse{0, 0, nil}; close(c) }()
}

func (c genericCallback) Error(r io.Reader) {
	var errorCode, errorStatus, reqID uint32

	n, err := fmt.Fscanf(r, "%d %d", &reqID, &errorCode, &errorStatus)
	skipToNewline(r)
	if err != nil || n != 2 {
		log.WithFields(log.Fields{
			"n":     n,
			"error": err,
		}).Errorf("Generic Callback, fscanf error.")
		go func() { c <- genericResponse{0, 0, err}; close(c) }()
		return
	}

	resp := genericResponse{
		code:   errorCode,
		status: errorStatus,
		err:    protocolError(errorCode, errorStatus),
	}
	go func() { c <- resp; close(c) }()
}

// The get-marks response structure
type getMarksResponse struct {
	marks []types.Mark
	err   error
}

type getMarksCallback chan getMarksResponse

func (g getMarksCallback) OK(r io.Reader) {
	marks := readUInt32(r)
	marksArr := make([]types.Mark, marks)

	ar, err := readDelimitedList('{', '}', r)
	skipToNewline(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"marks": marks,
		}).Error("Failed to read delimited list")
		go func() { g <- getMarksResponse{err: err}; close(g) }()
		return
	}

	tmp := strings.Split(ar, " ")
	log.WithFields(log.Fields{
		"marks": marks,
		"array": ar,
		"tmp":   tmp,
	}).Debug("get marks")

	rv := getMarksResponse{}

	strPos := 0
	if tmp[strPos] == "{" {
		strPos++
	}

	for ix := 0; ix < int(marks); ix++ {
		n, err := strconv.Atoi(tmp[strPos])
		if err != nil {
			log.WithFields(log.Fields{
				"ix":          ix,
				"strPos":      strPos,
				"tmp[strPos]": tmp[strPos],
			}).Error("parsing textNo")
			rv.marks = marksArr[0:ix]
			rv.err = err
			go func() { g <- rv; close(g) }()
			return
		}
		strPos++
		marksArr[ix].TextNo = types.TextNo(n)

		n, err = strconv.Atoi(tmp[strPos])
		if err != nil {
			log.WithFields(log.Fields{
				"ix":          ix,
				"strPos":      strPos,
				"tmp[strPos]": tmp[strPos],
			}).Error("parsing textNo")
			rv.marks = marksArr[0:ix]
			rv.err = err
			go func() { g <- rv; close(g) }()
			return
		}
		strPos++
		marksArr[ix].Type = byte(n)
	}
	rv.marks = marksArr
	go func() { g <- rv; close(g) }()
}

func (g getMarksCallback) Error(r io.Reader) {
	var errorCode, errorStatus, reqID uint32

	n, err := fmt.Fscanf(r, "%d %d", &reqID, &errorCode, &errorStatus)
	skipToNewline(r)
	if err != nil || n != 2 {
		log.WithFields(log.Fields{
			"n":     n,
			"error": err,
		}).Errorf("Generic Callback, fscanf error.")
		go func() { g <- getMarksResponse{err: err}; close(g) }()
		return
	}

	resp := getMarksResponse{
		err: fmt.Errorf("Generic error, code %d, status %d", errorCode, errorStatus),
	}
	go func() { g <- resp; close(g) }()
}

// The get-text reponse structure
type getTextResponse struct {
	text string
	err  error
}
type getTextCallback chan getTextResponse

func (g getTextCallback) OK(r io.Reader) {
	s, err := hollerith.Scan(r)

	go func() {
		g <- getTextResponse{s, err}
		close(g)
	}()
}

func (g getTextCallback) Error(r io.Reader) {
	var errorCode, errorStatus, reqID uint32

	n, err := fmt.Fscanf(r, "%d %d", &reqID, &errorCode, &errorStatus)
	skipToNewline(r)
	if err != nil || n != 2 {
		log.WithFields(log.Fields{
			"n":     n,
			"error": err,
		}).Errorf("GetText Callback, fscanf error.")
		go func() { g <- getTextResponse{err: err}; close(g) }()
		return
	}

	resp := getTextResponse{
		err: fmt.Errorf("Generic error, code %d, status %d", errorCode, errorStatus),
	}
	go func() { g <- resp; close(g) }()
}

// Read an uint32 from the client socket, also consume the first
// whitespace after the number.
func readUInt32(r io.Reader) uint32 {
	var done bool
	var rv uint32

	for !done {
		b, err := utils.ReadByte(r)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"b":     b,
				"rv":    rv,
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
			rv = (10 * rv) + uint32(b-'0')
		default:
			done = true
		}
	}

	return rv
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
		case _, ok := <-k.shutdown:
			_ = ok
			done = true
			continue
		default:
			status, _ := utils.ReadByte(k.socket)
			id := readUInt32(k.socket)
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

// This sends the set-etc-motd protocol message (#17) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetEtcMotd(conference string, text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 17 %d %d", reqID, confID, text)

	err := k.send(req)
	return rv, err
}

// This sends the set-supervisor protocol message (#18) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetSupervisor(conference, admin string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)
	adminID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 18 %d %d", reqID, confID, adminID)

	err := k.send(req)
	return rv, err
}

// This sends the set-permitter-submitters protocol message (#19) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetPermitterSubmitters(conference, permitted string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)
	permSubID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 19 %d %d", reqID, confID, permSubID)

	err := k.send(req)
	return rv, err
}

// This sends the set-super-conf protocol message (#20) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetSuperConf(conference, permitted string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)
	superID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 20 %d %d", reqID, confID, superID)

	err := k.send(req)
	return rv, err
}

// This sends the set-conf-type protocol message (#21) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetConfTypef(conference string, confType types.AnyConfType) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 21 %d %s", reqID, confID, confType.BitField())

	err := k.send(req)
	return rv, err
}

// This sends the set-garb-nice protocol message (#22) and returns a
// channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetGarbNice(conference string, nice uint32) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	confID := k.ConferenceFromName(conference)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 22 %d %d", reqID, confID, nice)

	err := k.send(req)
	return rv, err
}

// This sends the get-marks protocol message (#23) and returns a
// channel suitable to return an array of marks or an error.
func (k *KomClient) asyncGetMarks() (chan getMarksResponse, error) {
	rv := make(chan getMarksResponse)

	reqID := k.registerCallback(getMarksCallback(rv))
	req := fmt.Sprintf("%d 23", reqID)

	err := k.send(req)
	return rv, err
}

// This sends the get-text protocol message (#25) and returns a
// channel suitable to return the text and/or an error.
func (k *KomClient) asyncGetText(textNo types.TextNo, start, end uint32) (chan getTextResponse, error) {
	rv := make(chan getTextResponse)

	reqID := k.registerCallback(getTextCallback(rv))
	req := fmt.Sprintf("%d 25 %d %d %d", reqID, textNo, start, end)

	err := k.send(req)
	return rv, err
}

// This sends the mark-as-read message (#27) and returns a channel
// suitable to get success or error.
func (k *KomClient) asyncMarkAsRead(conference string, texts []types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confID := k.ConferenceFromName(conference)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 27 %d %s", reqID, confID, types.TextNoArray(texts))

	err := k.send(req)
	return rv, err
}

// This sends the delete-text message (#29) and returns a channel
// suitable for getting success or an error.
func (k *KomClient) asyncDeleteText(text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 29 %d", reqID, text)

	err := k.send(req)
	return rv, err
}

// This sends the add-recipient message (#30) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncAddRecipient(textNo types.TextNo, conference string, recipientType types.InfoType) (chan genericResponse, error) {
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

// This sends the "set-client-version" protocol message (#69) and
// returns a channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetClientVersion(name, version string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 69 %s %s", reqID, hollerith.Sprint(name), hollerith.Sprint(version))
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
