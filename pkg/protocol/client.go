package protocol

// Protocol implementation for the KomAndGo client

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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

// Read error code and status from a reader, return them in that order.
// If an error occurs durung reating, terurn zeroes and the error.
func readError(r io.Reader) (uint32, uint32, error) {
	var errorCode, errorStatus, reqID uint32

	n, err := fmt.Fscanf(r, "%d %d", &reqID, &errorCode, &errorStatus)
	skipToNewline(r)
	if err != nil || n != 2 {
		log.WithFields(log.Fields{
			"n":     n,
			"error": err,
		}).Errorf("Generic Callback, fscanf error.")
		return 0, 0, err
	}

	return errorCode, errorStatus, nil
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
	code, status, err := readError(r)

	resp := getTextResponse{
		err: protocolError(code, status),
	}

	if err != nil {
		resp.err = err
	}
	go func() { g <- resp; close(g) }()
}

type zConfArrayResponse struct {
	confs []types.ConfZInfo
	err   error
}
type zConfArrayResponseCallback chan zConfArrayResponse

func (zca zConfArrayResponseCallback) OK(r io.Reader) {
	items := readUInt32(r)

	var rv []types.ConfZInfo
	ar, err := readDelimitedList('{', '}', r)

	// We should now have a string representation of the entire
	// array contents in ar, so we can just start working from
	// there.
	offset := 0
	for item := uint32(0); item < items; item++ {
		log.WithFields(log.Fields{
			"looking-at": ar[offset : offset+10],
			"offset":     offset,
		}).Debug("loop start")
		name, next := hollerith.FromString(ar, offset)
		log.WithFields(log.Fields{
			"looking-at": ar[next : next+6],
			"offset":     next,
		}).Debug("parsing conf type")
		confType := utils.ParseConfType(ar, next+1)
		next += 5
		log.WithFields(log.Fields{
			"looking-at": ar[next : next+2],
			"offset":     next,
		}).Debug("parsing conf no")
		tmp, next := utils.ReadUInt32FromString(ar, next)
		confNo := types.ConfNo(tmp)
		conf := types.ConfZInfo{Name: name, Type: confType, No: confNo}
		//log.WithFields(log.Fields{
		//	"next":     next,
		//	"confNo":   confNo,
		//	"confType": confType,
		//	"name":     name,
		//	"item":     item,
		//}).Debug("Reading zConfArray")
		offset = next
		rv = append(rv, conf)
	}

	go func() { zca <- zConfArrayResponse{confs: rv, err: err} }()
}

func (zca zConfArrayResponseCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	resp := zConfArrayResponse{
		err: protocolError(code, status),
	}

	if err != nil {
		resp.err = err
	}
	go func() { zca <- resp; close(zca) }()
}

// The version-info response
type versionInfoResponse struct {
	info types.VersionInfo
	err  error
}

type versionInfoResponseCallback chan versionInfoResponse

func (vi versionInfoResponseCallback) OK(r io.Reader) {
	var rv versionInfoResponse
	var err error

	rv.info.ProtocolVersion = readUInt32(r)
	rv.info.ServerSoftware, err = hollerith.Scan(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("reading server software")
		rv.err = err
	}
	rv.info.SoftwareVersion, err = hollerith.Scan(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("reading software version")
		rv.err = err
	}

	go func() { vi <- rv; close(vi) }()
}

func (vi versionInfoResponseCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	resp := versionInfoResponse{
		err: protocolError(code, status),
	}

	if err != nil {
		resp.err = err
	}
	go func() { vi <- resp; close(vi) }()
}

// The get-time response
type timeResponseCallback chan time.Time

func (t timeResponseCallback) Error(r io.Reader) {
	// this should never fail
}

func readTime(r io.Reader) time.Time {
	sec := int(readUInt32(r))
	min := int(readUInt32(r))
	hour := int(readUInt32(r))
	mday := int(readUInt32(r))
	mon := int(readUInt32(r))
	year := int(readUInt32(r))
	_ = int(readUInt32(r))
	_ = int(readUInt32(r))
	_ = int(readUInt32(r))

	return time.Date(1900+year, time.Month(mon), mday, hour, min, sec, 0, time.UTC)
}

func (t timeResponseCallback) OK(r io.Reader) {
	tstamp := readTime(r)
	go func() { t <- tstamp; close(t) }()
}

type personStat struct {
	person types.Person
	err    error
}

type personStatCallback chan personStat

func (ps personStatCallback) OK(r io.Reader) {
	var person types.Person
	var err error

	_, err = hollerith.Scan(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warning("personStatCallback.OK() - scanning last login failed.")
	}
	person.Privileges = types.ReadPrivBits(r)
	person.Flags = types.ReadPersonalFlags(r)
	person.LastLogin = readTime(r)
	person.UserArea = types.TextNo(readUInt32(r))
	person.TotalTimePresent = readUInt32(r)
	person.Sessions = readUInt32(r)
	person.CreatedLines = readUInt32(r)
	person.CreatedBytes = readUInt32(r)
	person.ReadTexts = readUInt32(r)
	person.Testfetches = readUInt32(r)
	person.CreatedPersons = readUInt16(r)
	person.CreatedConferences = readUInt16(r)
	person.FirstCreatedLocalNo = readUInt32(r)
	person.CreatedTexts = readUInt32(r)
	person.Marks = readUInt16(r)
	person.Conferences = readUInt16(r)

	go func() { ps <- personStat{person: person}; close(ps) }()
}

func (ps personStatCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { ps <- personStat{err: err}; close(ps) }()
}

type unreadConfs struct {
	unread []types.ConfNo
	err    error
}

type unreadConfsCallback chan unreadConfs

func (uc unreadConfsCallback) OK(r io.Reader) {
	confs := readUInt32(r)
	confArr := make([]types.ConfNo, confs)

	ar, err := readDelimitedList('{', '}', r)
	skipToNewline(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"confs": confs,
		}).Error("ureadConfsCallback.OK() - failed reading array")
		go func() { uc <- unreadConfs{err: err}; close(uc) }()
		return
	}

	tmp := strings.Split(ar, " ")
	log.WithFields(log.Fields{
		"confs": confs,
		"array": ar,
		"tmp":   tmp,
	}).Debug("unreadConfsCallback.OK() - array read")

	rv := unreadConfs{}

	strPos := 0
	for tmp[strPos] == "{" {
		strPos++
	}

	for ix := 0; ix < int(confs); ix++ {
		n, err := strconv.Atoi(tmp[strPos])
		if err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"ix":          ix,
				"strPos":      strPos,
				"tmp[strPos]": tmp[strPos],
			}).Error("unreadConfsCallback.OK() - parsing conf no")
			rv.unread = confArr[0:ix]
			rv.err = err
			go func() { uc <- rv; close(uc) }()
		}
		strPos++
		confArr[ix] = types.ConfNo(n)
	}

	rv.unread = confArr
	go func() { uc <- rv; close(uc) }()
}

func (uc unreadConfsCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { uc <- unreadConfs{err: err}; close(uc) }()

}

// Struct and callback suitable for a who-am-i call
type whoAmIResponse struct {
	session types.SessionNo
	err     error
}
type whoAmICallback chan whoAmIResponse

func (w whoAmICallback) OK(r io.Reader) {
	s := readUInt32(r)
	go func() { w <- whoAmIResponse{session: types.SessionNo(s)}; close(w) }()
}

func (w whoAmICallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { w <- whoAmIResponse{err: err}; close(w) }()
}

type textResponse struct {
	text types.TextNo
	err  error
}
type textResponseCallback chan textResponse

func (lt textResponseCallback) OK(r io.Reader) {
	textNo := types.TextNo(readUInt32(r))
	go func() { lt <- textResponse{text: textNo}; close(lt) }()
}

func (lt textResponseCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { lt <- textResponse{err: err}; close(lt) }()
}

type stringResponse struct {
	str string
	err error
}
type stringResponseCallback chan stringResponse

func (s stringResponseCallback) OK(r io.Reader) {
	str, err := hollerith.Scan(r)
	go func() { s <- stringResponse{str: str, err: err}; close(s) }()
}

func (s stringResponseCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { s <- stringResponse{err: err}; close(s) }()
}

type uConfResponse struct {
	uConf types.UConference
	err   error
}
type uConfResponseCallback chan uConfResponse

func (uc uConfResponseCallback) OK(r io.Reader) {
	var ucon types.UConference
	var resp uConfResponse

	ucon.Name, resp.err = hollerith.Scan(r)
	ucon.Type = types.ReadExtendedConfType(r)
	utils.ReadByte(r)
	ucon.HighestLocalNo = types.TextNo(readUInt32(r))
	ucon.Nice = readUInt32(r)
	resp.uConf = ucon

	go func() { uc <- resp; close(uc) }()
}

func (uc uConfResponseCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { uc <- uConfResponse{err: err}; close(uc) }()
}

type queryAsyncResponse struct {
	messages []uint32
	err      error
}

type queryAsyncCallback chan queryAsyncResponse

func (qac queryAsyncCallback) OK(r io.Reader) {
	acceptedCalls, err := types.ReadUInt32Array(r)
	if err != nil {
		go func() {
			qac <- queryAsyncResponse{err: err}
			close(qac)
		}()
	}

	go func() {
		qac <- queryAsyncResponse{messages: acceptedCalls}
		close(qac)
	}()

}

func (qac queryAsyncCallback) Error(r io.Reader) {
	code, status, err := readError(r)

	if err == nil {
		err = protocolError(code, status)
	}

	go func() { qac <- queryAsyncResponse{err: err}; close(qac) }()
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

// Read an uint32 from the client socket, also consume the first
// whitespace after the number.
func readUInt16(r io.Reader) uint16 {
	var done bool
	var rv uint16

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
			rv = (10 * rv) + uint16(b-'0')
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
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(conference)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 30 %d %d %d", reqID, textNo, confNo, recipientType)

	err := k.send(req)
	return rv, err
}

// This sends the sub-recipient message (#31) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncSubRecipient(textNo types.TextNo, conference string) (chan genericResponse, error) {
	rv := make(chan genericResponse)

	confNo := k.ConferenceFromName(conference)

	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 31 %d %d", reqID, textNo, confNo)

	err := k.send(req)
	return rv, err
}

// This sends the add-comment message (#32) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncAddComment(text, commentTo types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 32 %d %d", reqID, text, commentTo)
	err := k.send(req)
	return rv, err
}

// This sends the sub-comment message (#33) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncSubComment(text, commentTo types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 33 %d %d", reqID, text, commentTo)
	err := k.send(req)
	return rv, err
}

// This sends the get-time message (#35) and returns a channel
// suitable for getting the time.
func (k *KomClient) asyncGetTime() (chan time.Time, error) {
	rv := make(chan time.Time)
	reqID := k.registerCallback(timeResponseCallback(rv))
	req := fmt.Sprintf("%d 35", reqID)
	err := k.send(req)
	return rv, err
}

// This sends the add-footnote message (#37) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncAddFootnote(text, footnoteTo types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 37 %d %d", reqID, text, footnoteTo)
	err := k.send(req)
	return rv, err
}

// This sends the sub-footnote message (#38) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncSubFootnote(text, footnoteTo types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 38 %d %d", reqID, text, footnoteTo)
	err := k.send(req)
	return rv, err
}

// This sends the set-unread message (#40) and returns a channel
// suitable for getting a success or an error.
func (k *KomClient) asyncSetUnread(conference string, unread uint32) (chan genericResponse, error) {
	confNo := k.ConferenceFromName(conference)

	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 40 %d %d", reqID, confNo, unread)

	err := k.send(req)
	return rv, err
}

// This sends the set-motd-of-lyskom message (#41) and returns a
// channel suitable for getting a success or an error.
func (k *KomClient) asyncSetMOTDOfLysKom(text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 41 %d", reqID, text)
	err := k.send(req)
	return rv, err
}

// This sends the enable message (#42) and returns a channel suitable
// for getting a success or an error.
func (k *KomClient) asyncEnable(level uint8) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 42 %d", reqID, level)
	err := k.send(req)
	return rv, err
}

// This sends the SyncKom message (#43) and returns a channel suitable
// for getting a success or an error.
func (k *KomClient) asyncSyncKom() (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 43", reqID)
	err := k.send(req)
	return rv, err
}

// This sends the ShutdownKom message (#44) and returns a channel suitable
// for getting a success or an error.
func (k *KomClient) asyncShutdownKom(eVal uint8) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 44 %d", reqID, eVal)
	err := k.send(req)
	return rv, err
}

// This sends the get-person-stat message (#49) and returns a channel
// suitabe for passing a successful response or an error through.
func (k *KomClient) asyncGetPersonStat(person types.ConfNo) (chan personStat, error) {
	rv := make(chan personStat)
	reqID := k.registerCallback(personStatCallback(rv))
	req := fmt.Sprintf("%d 49 %d", reqID, person)
	err := k.send(req)
	return rv, err
}

// This sends the get-unread-confs (#52) protocol message and returns
// a channel suitable for passing a successful response or an error
// through.
func (k *KomClient) asyncGetUnreadConfs(person types.ConfNo) (chan unreadConfs, error) {
	rv := make(chan unreadConfs)
	reqID := k.registerCallback(unreadConfsCallback(rv))
	req := fmt.Sprintf("%d 52 %d", reqID, person)
	err := k.send(req)
	return rv, err
}

// This sends the send-message (#53) protocol message and returns
// a channel suitable for passing a successful response or an error
// through.
func (k *KomClient) asyncSendMessage(recipient types.ConfNo, message string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 53 %d %s", reqID, recipient, hollerith.Sprint(message))
	err := k.send(req)
	return rv, err
}

// This sends the disconnect (#55) protocol message and returns
// a channel suitable for passing a successful response or an error
// through.
func (k *KomClient) asyncDisconnect(session types.SessionNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 55 %d", reqID, session)
	err := k.send(req)
	return rv, err
}

// This sends the who-am-i (#56) protocol message and returns
// a channel suitable for passing a successful response or an error
// through.
func (k *KomClient) asyncWhoAmI() (chan whoAmIResponse, error) {
	rv := make(chan whoAmIResponse)
	reqID := k.registerCallback(whoAmICallback(rv))
	req := fmt.Sprintf("%d 56", reqID)
	err := k.send(req)
	return rv, err
}

// This send the set-user-area (#57) protocol message and returns a
// channel suitable for getting a success or error code through.
func (k *KomClient) asyncSetUserArea(who types.ConfNo, text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 57 %d %d", reqID, who, text)
	err := k.send(req)
	return rv, err
}

// This sends the get-last-text (#58) protocol message and returns a
// channel suitable for reading the response or an error through.
func (k *KomClient) asyncGetLastText(when time.Time) (chan textResponse, error) {
	rv := make(chan textResponse)
	reqID := k.registerCallback(textResponseCallback(rv))
	req := fmt.Sprintf("%d 58 %s", reqID, types.StringTime(when))
	err := k.send(req)
	return rv, err
}

// This sends the find-next-text-no (#60) protocol message, returning
// a channel suitable to read the reponse or an error from.
func (k *KomClient) asyncFindNextTextNo(text types.TextNo) (chan textResponse, error) {
	rv := make(chan textResponse)
	reqID := k.registerCallback(textResponseCallback(rv))
	req := fmt.Sprintf("%d 60 %d", reqID, text)
	err := k.send(req)
	return rv, err
}

// This sends the find-previous-text-no (#61) protocol message, returning
// a channel suitable to read the reponse or an error from.
func (k *KomClient) asyncFindPreviousTextNo(text types.TextNo) (chan textResponse, error) {
	rv := make(chan textResponse)
	reqID := k.registerCallback(textResponseCallback(rv))
	req := fmt.Sprintf("%d 61 %d", reqID, text)
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

// This sends the "set-client-version" protocol message (#69) and
// returns a channel suitable to see if there was an error or not.
func (k *KomClient) asyncSetClientVersion(name, version string) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 69 %s %s", reqID, hollerith.Sprint(name), hollerith.Sprint(version))
	err := k.send(req)

	return rv, err
}

// This sends the "get-client-name" protocol message (#70) and returns
// a channel suitable to get the response or an error from.
func (k *KomClient) asyncGetClientName(session uint32) (chan stringResponse, error) {
	rv := make(chan stringResponse)
	reqID := k.registerCallback(stringResponseCallback(rv))
	req := fmt.Sprintf("%d 70 %d", reqID, session)
	err := k.send(req)

	return rv, err
}

// This sends the "get-client-version" protocol message (#71) and returns
// a channel suitable to get the response or an error from.
func (k *KomClient) asyncGetClientVersion(session uint32) (chan stringResponse, error) {
	rv := make(chan stringResponse)
	reqID := k.registerCallback(stringResponseCallback(rv))
	req := fmt.Sprintf("%d 71 %d", reqID, session)
	err := k.send(req)

	return rv, err
}

// This sends the "mark-text" protocol message (#72) and returns a
// channel suitable for reading success or error from.
func (k *KomClient) asyncMarkText(text types.TextNo, mark uint8) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 72 %d %d", reqID, text, mark)
	err := k.send(req)

	return rv, err
}

// This sends the "unmark-text" protocol message (#73) and returns a
// channel suitable for reading success or error from.
func (k *KomClient) asyncUnmarkText(text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 73 %d", reqID, text)
	err := k.send(req)

	return rv, err
}

// This sends the "re-z-lookup" protocol message (#74) and returns a
// channel suitable for reading the response or an error from.
func (k *KomClient) asyncReZLookup(re string, wantPersons bool, wantConferences bool) (chan zConfArrayResponse, error) {
	rv := make(chan zConfArrayResponse)
	reqID := k.registerCallback(zConfArrayResponseCallback(rv))
	persons := 0
	confs := 0
	if wantPersons {
		persons = 1
	}
	if wantConferences {
		confs = 1
	}
	req := fmt.Sprintf("%d 74 %s %d %d", reqID, hollerith.Sprint(re), persons, confs)
	err := k.send(req)

	return rv, err
}

// This sends the "get-version-info" protocol message (#75) and
// returns a channel to read a response or an error from
func (k *KomClient) asyncGetVersionInfo() (chan versionInfoResponse, error) {
	rv := make(chan versionInfoResponse)
	reqID := k.registerCallback(versionInfoResponseCallback(rv))
	req := fmt.Sprintf("%d 75", reqID)
	err := k.send(req)
	return rv, err
}

// This sends the "lookup-z-name" protocol message (#76) and returns a
// channel suitabe to read the reponse or an error from.
func (k *KomClient) asyncLookupZName(name string, wantConferences, wantPersons bool) (chan zConfArrayResponse, error) {
	rv := make(chan zConfArrayResponse)
	reqID := k.registerCallback(zConfArrayResponseCallback(rv))
	persons := 0
	confs := 0
	if wantPersons {
		persons = 1
	}
	if wantConferences {
		confs = 1
	}
	req := fmt.Sprintf("%d 76 %s %d %d", reqID, hollerith.Sprint(name), persons, confs)
	err := k.send(req)

	return rv, err
}

// This sends the "set-last-read" protocol message (#77) and returns a
// generic channel suitable for reading success or failure from.
func (k *KomClient) asyncSetLastRead(conf types.ConfNo, text types.TextNo) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 77 %d %d", reqID, conf, text)
	err := k.send(req)

	return rv, err
}

// This sends the "get-uconf-stat" protocol message (#78) and returns
// a channel suitable for reading the result or an error from.
func (k *KomClient) asyncGetUConfStat(conf types.ConfNo) (chan uConfResponse, error) {
	rv := make(chan uConfResponse)
	reqID := k.registerCallback(uConfResponseCallback(rv))
	req := fmt.Sprintf("%d 78 %d", reqID, conf)
	err := k.send(req)

	return rv, err
}

// This sends the "set-info" protocol message (#79) and returns a
// channel suitable for reading an OK or an error from.
func (k *KomClient) asyncSetInfo(info types.InfoOld) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 79 %d %d %d %d %d %d", reqID, info.Version, info.ConferencePresentationConference, info.PersonPresentationConference, info.MOTDConference, info.KomNewsConference, info.MOTDOfLyskom)
	err := k.send(req)

	return rv, err
}

// This sends the "accept-async" protocol message (#80) and returns a
// channel suitable for reading an OK or an error from.
func (k *KomClient) asyncAcceptAsync(msgs []uint32) (chan genericResponse, error) {
	rv := make(chan genericResponse)
	reqID := k.registerCallback(genericCallback(rv))
	req := fmt.Sprintf("%d 80 %s", reqID, types.UInt32Array(msgs))
	err := k.send(req)

	return rv, err
}

// This sends the "query-async" protocol message (#81) and returns a
// channel suitable for reading the answer or an error from.
func (k *KomClient) asyncQueryAsync() (chan queryAsyncResponse, error) {
	rv := make(chan queryAsyncResponse)
	reqID := k.registerCallback(queryAsyncCallback(rv))
	req := fmt.Sprintf("%d 81", reqID)
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
