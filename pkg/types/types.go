// Various types for Lyskom Protocol A implementation

package types

import (
	"time"
)

type ConfNo uint16
type TextNo uint32
type AuxNo uint32
type SessionNo uint32

type AuxItem struct {
	AuxNo     AuxNo
	Tag       uint32
	Creator   ConfNo
	CreatedAt time.Time
	Flags     AuxItemFlags
	data      string
}

type AuxItemInput struct {
	Tag          uint32
	Flags        AuxItemFlags
	InheritLimit uint32
	data         string
}

type AuxItemFlags struct {
	Deleted     bool
	Inherit     bool
	Secret      bool
	HideCreator bool
	DontGarb    bool
	Reserved2   bool
	Reserved3   bool
	Reserved4   bool
}

type ConfType struct {
	RdProt    bool
	Original  bool
	Secret    bool
	LetterBox bool
}

type ExtendedConfType struct {
	RdProt         bool
	Original       bool
	Secret         bool
	LetterBox      bool
	AllowAnonymous bool
	ForbidSecret   bool
	Reserved2      bool
	Reserved3      bool
}

type AnyConfType interface {
	isConfType() bool
	BitField() string
}

func (t ConfType) isConfType() bool {
	return true
}

func (t ExtendedConfType) isConfType() bool {
	return true
}

func (t ConfType) BitField() string {
	ar := []byte("0000")
	if t.RdProt {
		ar[0] = '1'
	}
	if t.Original {
		ar[1] = '1'
	}
	if t.Secret {
		ar[2] = '1'
	}
	if t.LetterBox {
		ar[3] = '1'
	}

	return string(ar)
}

func (t ExtendedConfType) BitField() string {
	ar := []byte("00000000")
	if t.RdProt {
		ar[0] = '1'
	}
	if t.Original {
		ar[1] = '1'
	}
	if t.Secret {
		ar[2] = '1'
	}
	if t.LetterBox {
		ar[3] = '1'
	}
	if t.AllowAnonymous {
		ar[4] = '1'
	}
	if t.ForbidSecret {
		ar[5] = '1'
	}

	return string(ar)
}

type OldConferece struct {
	Name                string
	Type                ConfType
	CreationTime        time.Time
	LastWritten         time.Time
	Creator             ConfNo
	Presentation        TextNo
	Supervisor          ConfNo
	PermittedSubmitters ConfNo
	SuperConf           ConfNo
	Nice                uint32
	NoOfMembers         uint16
	FirstLocalNo        TextNo
	NoOfTexts           uint32
}

type Conference struct {
	Name                string
	Type                ExtendedConfType
	CreationTime        time.Time
	LastWritten         time.Time
	Creator             ConfNo
	Presentation        TextNo
	Supervisor          ConfNo
	PermittedSubmitters ConfNo
	MsgOfDay            TextNo
	Nice                uint32
	KeepCommented       uint32
	NoOfMembers         uint32
	Expire              uint32
	AuxItems            []AuxItem
}

type UConference struct {
	Name           string
	Type           ExtendedConfType
	HighestLocalNo TextNo
	Nice           uint32
}

type Person struct {
	Username            string
	Privileges          PrivBits
	Flags               PersonalFlags
	LastLogin           time.Time
	UserArea            TextNo
	TotalTimePresent    uint32
	Sessions            uint32
	CreatedLines        uint32
	CreatedBytes        uint32
	ReadTexts           uint32
	Testfetches         uint32
	CreatedPersons      uint16
	CreatedConferences  uint16
	FirstCreatedLocalNo uint32
	CreatedTexts        uint32
	Marks               uint16
	Conferences         uint16
}

type PersonalFlags struct {
	UnreadIsSecret bool
}

type PrivBits struct {
	Wheel             bool
	Admin             bool
	Statistic         bool
	CreatePersons     bool
	CreateConferences bool
	ChangeName        bool
}

type MembershipType struct {
	Invitation          bool
	Passive             bool
	Secret              bool
	PassiveMessageInver bool
}

type Member struct {
	Member  ConfNo
	AddedBy ConfNo
	AddedAt time.Time
	Type    MembershipType
}

type ReadRange struct {
	FirstRead TextNo
	LastRead  TextNo
}

type Membership struct {
	Position     uint32
	LastTimeRead time.Time
	Conference   ConfNo
	Priority     byte
	ReadRanges   []ReadRange
	AddedBy      ConfNo
	AddedAt      time.Time
	Type         MembershipType
}

type MembershipOld struct {
	LastTimeRead time.Time
	Conference   ConfNo
	LastTextRead TextNo
	ReadTexts    []TextNo
}

type Membership10 struct {
	Position     uint32
	LastTimeRead time.Time
	Conference   ConfNo
	Priority     byte
	LastTextRead TextNo
	ReadTests    []TextNo
	AddedBy      ConfNo
	AddedAt      time.Time
	Type         MembershipType
}

type Mark struct {
	TextNo TextNo
	Type   byte
}

type MiscInfo struct {
	Selector     uint32
	Recipient    ConfNo
	CCRecipient  ConfNo
	CommentTo    TextNo
	CommentedIn  TextNo
	FootnoteTo   TextNo
	FootnotedIn  TextNo
	LocalNo      TextNo
	ReceivedAt   time.Time
	Sender       ConfNo
	SentAt       time.Time
	BCCRecipient ConfNo
}

type InfoType uint8

const (
	Recipient = InfoType(iota)
	CCRecipient
	CommentTo
	CommentIn
	FootnoteTo
	FootnoteIn
	LocalNo
	ReceiveTime
	SentBy
	SentAt
	BCCRecipient
)

type TextStatOld struct {
	CreationTime time.Time
	Author       ConfNo
	Lines        uint32
	Characters   uint32
	Marks        uint16
	MiscInfo     []MiscInfo
}

type TextStat struct {
	CreationTime time.Time
	Author       ConfNo
	Lines        uint32
	Chars        uint32
	Marks        uint16
	MiscInfo     []MiscInfo
	AuxItems     []AuxItem
}

type WhoInfoOld struct {
	Person            ConfNo
	WorkingConference ConfNo
	WhatAmIDoing      string
}

type WhoInfo struct {
	Person            ConfNo
	WorkingConference ConfNo
	Session           uint32
	WhatAmIDoing      string
	UserName          string
}

type WhoInfoIdent struct {
	Person            ConfNo
	WorkingConference ConfNo
	Session           uint32
	WhatAmIDoing      string
	UserName          string
	HostName          string
	IdentUser         string
}

type SessionInfo struct {
	Person            ConfNo
	WorkingConference ConfNo
	Session           SessionNo
	WhatAmIDoing      string
	UserName          string
	IdleTime          uint32
	ConnectionTime    time.Time
}

type SessionInfoIdent struct {
	Person            ConfNo
	WorkingConference ConfNo
	Session           SessionNo
	WhatAmIDoing      string
	UserName          string
	HostName          string
	IdentUser         string
	IdleTime          uint32
	ConnectionTime    time.Time
}

type StaticSessionInfo struct {
	UserName       string
	HostName       string
	IdentUser      string
	ConnectionTime time.Time
}

type SessionFlags struct {
	Invisible      bool
	UserActiveUsed bool
	UserAbsent     bool
}

type DynamicSessionInfo struct {
	Session           SessionNo
	Peron             ConfNo
	WorkingConference ConfNo
	IdleTime          uint32
	Flags             SessionFlags
	WhatAmIDoing      string
}

type SchedulingInfo struct {
	Priority uint16
	Weight   uint16
}

type ConfZInfo struct {
	Name string
	Type ConfType
	No   ConfNo
}

type VersionInfo struct {
	ProtocolVersion uint32
	ServerSoftware  string
	SoftwareVersion string
}

type InfoOld struct {
	Version                          uint32
	ConferencePresentationConference ConfNo
	PersonPresentationConference     ConfNo
	MOTDConference                   ConfNo
	KomNewsConference                ConfNo
	MOTDOfLyskom                     TextNo
}
