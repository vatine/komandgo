package protocol

// The KomServer is shared amongst multiple clients, connected to the
// same server, it keeps server-wide information so that if multiple
// clients connect to the same sever, the information can be shared.

import (
	"sync"

	"github.com/vatine/komandgo/pkg/types"
)


// The KomServer data structure
type KomServer struct {
	client *KomClient
	personLock sync.Mutex
	userNameMap map[string]types.ConfNo
	conferenceLock sync.Mutex
	conferenceMap map[string]types.ConfNo
}

var serverLock sync.Mutex
var serverMap map[string]*KomServer

func GetServer(name string) (*KomServer, error) {
	var err error
	
	serverLock.Lock()
	defer serverLock.Unlock()

	s, ok := serverMap[name]
	if !ok {
		s = new(KomServer)
		s.userNameMap = make(map[string]types.ConfNo)
		s.client, err = internalNewClient(name, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (ks *KomServer) LookupUser(user string) (types.ConfNo, bool) {
	ks.personLock.Lock()
	defer ks.personLock.Unlock()
	c, ok := ks.userNameMap[user]
	return c, ok
}

func (ks *KomServer) LookupConference(user string) (types.ConfNo, bool) {
	ks.conferenceLock.Lock()
	defer ks.conferenceLock.Unlock()
	c, ok := ks.conferenceMap[user]
	return c, ok
}
