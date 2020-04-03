// Various types for Lyskom Protocol A implementation

package types

import (
	"time"
)

type ConfNo uint16
type TextNo uint32
type AuxNo uint32

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
	Resevred2      bool
	Reserved3      bool
}

type AnyConfType interface {
	isConfType() bool
}

func (t ConfType) isConfType() bool {
	return true
}

func (t ExtendedConfType) isConfType() bool {
	return true
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
	CreatiomTime        time.Time
	LastWritten         time.Time
	Creator             ConfNo
	Presentation        TextNo
	Supervisor          ConfNo
	PermittedSubmitters ConfNo
	MsgOfDay            TextNo
	Nice                uint32
	KeepCommented       uint32
	NoOfMembers         uint32
	expire              uint32
	AuxItems            []AuxItem
}

type Person struct {
	Username            string
	Privileges          PrivBits
	Flags               PersonalFlags
	LastLogin           time.Time
	UserArea            TextNo
	TitalTimePresent    unit32
	Sessions            uint32
	CreatedLines        uint32
	CreatedBytes        uint32
	ReadTexts           uint32
	Testfetches         uint32
	CreatedPersons      uint16
	CreatedConferences  uint16
	FirstCreatedLocalNo uint32
	CreatedTextx        uint32
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
	LastTextRead textNo
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
	WorkingConference ConfNO
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
	UserName          srting
	IdleTime          uint32
	ConnectionTime    time.Time
}

type SessionInfoIdent struct {
	Person            ConfNo
	WorkingConference ConfNo
	Session           SessionNo
	WhatAmIDoing      string
	UserName          srting
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
