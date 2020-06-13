package protocol

// Error codes and nicer error formatting

import (
	"fmt"
)

// Return an error that formats to a sensible error message based on
// the protocol A error code and status.
func protocolError(code, status uint32) error {
	switch {
	case code == 0:
		// no-error (0)
		//   No error has occurred. error-status is undefined. This should never happen, but it might.
		return fmt.Errorf("Error code %d, status %d, this technically means 'no error'...", code, status)
	case code == 2:
		// not-implemented (2)
		//   The call has not been implemented yet. error-status is undefined.
		return fmt.Errorf("Not implemented.")
	case code == 3:
		// obsolete-call (3)
		//   The call is obsolete and no longer implemented. error-status is undefined.
		return fmt.Errorf("Obsoleted.")
	case code == 4:
		// invalid-password (4)
		//   Attempt to set a password containing illegal characters, or to use an incorrect password.
		return fmt.Errorf("Invalid password.")
	case code == 5:
		// string-too-long (5)
		//   A string was too long (see descriptions of each call.) error-status indicates the maximum string length.
		return fmt.Errorf("String too long, max length is %d", status)
	case code == 6:
		// login-first (6)
		//   Login is required before issuing the call. error-status is undefined.
		return fmt.Errorf("Not logged in.")
	case code == 7:
		// login-disallowed (7)
		//   The system is in single-user mode. You need to be privileged to log in despite this. error-status is undefined.
		return fmt.Errorf("System in single-used mode, login not allowed.")
	case code == 8:
		// conference-zero (8)
		//   Attempt to use conference number 0. error-status is undefined.
		return fmt.Errorf("Attempting to use conference 0.")
	case code == 9:
		// undefined-conference (9)
		//   Attempt to access a non-existent or secret conference. error-status contains the conference number in question.
		return fmt.Errorf("Non-existent conference, %d", status)
	case code == 10:
		// undefined-person (10)
		//   Attempt to access a non-existent or secret person. error-status contains the person number in question.
		return fmt.Errorf("Non-existent person, %d", status)
	case code == 11:
		// access-denied (11)
		//   No read/write access to something. This might be returned in response to an attempt to create a text, when the recipient conference and its super conferences are read-only, or when attempting to add a member to a conference without enough permission to do so. error-status indicates the object to which we didn't have enough permissions to.
		return fmt.Errorf("Access denied, %d", status)
	case code == 12:
		// permission-denied (12)
		//   Not enough permissions to do something. The exact meaning of this response depends on the call. error-status indicated the object for which permission was lacking, or zero.
		return fmt.Errorf("Permission denied operating on object %d", status)
	case code == 13:
		// not-member (13)
		//   The call requires the caller to be a member of some conference that the caller is not a member of. error-status indicates the conference in question.
		return fmt.Errorf("Requires membership in conference %d", status)
	case code == 14:
		// no-such-text (14)
		//   Attempt to access a text that either does not exist or is secret in some way. error-status indicates the text number in question.
		return fmt.Errorf("Text %d is not available.", status)
	case code == 15:
		// text-zero (15)
		//   Attempt to use text number 0. error-status is undefined.
		return fmt.Errorf("Text zero.")
	case code == 16:
		// no-such-local-text (16)
		//   Attempt to access a text using a local text number that does not represent an existing text. error-status indicates the offending number.
		return fmt.Errorf("No such local text %d", status)
	case code == 17:
		// local-text-zero (17)
		//   Attempt to use local text number zero. error-status is undefined.
		return fmt.Errorf("Local text zero")
	case code == 18:
		// bad-name (18)
		//   Attempt to use a name that's too long, too short or contains invalid characters. error-status is undefined.
		return fmt.Errorf("Bad name.")
	case code == 19:
		// index-out-of-range (19)
		//   Attempt to use a number that's out of range. The range and meaning of the numbers depends on the call issued. error-status is undefined unless stated otherwise in the call documentation.
		return fmt.Errorf("Index out of range, status code is %d", status)
		// conference-exists (20)
		//   Attempt to create a conference or person with a name that's already occupied. error-status is undefined.
		// person-exists (21)
		//   Attempt to create a person with a name that's already occupied. error-status is undefined. This error code is probably not used, but you never know for sure.
		// secret-public (22)
		//   Attempt to give a conference a type with secret bit set and the rd-prot bit unset. This is an error since such a conference type is inconsistent. error-status is undefined.
		// letterbox (23)
		//   Attempt to change the letterbox flag of a conference. error-status indicates the conference number.
		// ldb-error (24)
		//   Database is corrupted. error-status is an internal code.
		// illegal-misc (25)
		//   Attempt to create an illegal misc item. error-status contains the index of the illegal item.
		// illegal-info-type (26)
		//   Attempt to use a Misc-Info type (or Info-Type value) that the server knows nothing about. error-status is the type.
		// already-recipient (27)
		//   Attempt to add a recipient that is already a recipient of the same type. error-status contains the recipient that already is.
		// already-comment (28)
		//   Attempt to add a comment to a text twice over. error-status contains the text number of the text that already is a comment.
		// already-footnote (29)
		//   Attempt to add a footnote to a text twice over. error-status contains the text number of the text that already is a footnote.
		// not-recipient (30)
		//   Attempt to remove a recipient that isn't really a recipient. error-status contains the conference number in question.
		// not-comment (31)
		//   Attempt to remove a comment link that does not exist. error-status contains the text number that isn't a comment.
		// not-footnote (32)
		//   Attempt to remove a footnote link that does not exist. error-status contains the text number that isn't a footnote.
		// recipient-limit (33)
		//   Attempt to add a recipient to a text that already has the maximum number of recipients. error-status is the text that has the maximum number of recipients.
		// comment-limit (34)
		//   Attempt to add a comment to a text that already has the maximum number of comments. error-status is the text with the maximum number of comments.
		// footnote-limit (35)
		//   Attempt to add a footnote to a text that already has the maximum number of footnote. error-status is the text with the maximum number of footnotes.
		// mark-limit (36)
		//   Attempt to add a mark to a text that already has the maximum number of marks. error-status is the text with the maximum number of marks.
		// not-author (37)
		//   Attempt to manipulate a text in a way that required the user to be the author of the text, when not in fact the author. error-status contains the text number in question.
		// no-connect (38)
		//   Currently unused.
		// out-of-memory (39)
		//   The server ran out of memory.
		// server-is-crazy (40)
		//   Currently unused.
		// client-is-crazy (41)
		//   The client used an illegal call sequence, such as calling set-client-version more than once.
		// undefined-session (42)
		//   Attempt to access a session that does not exist. error-status contains the offending session number.
		// regexp-error (43)
		//   Error using a regexp. The regexp may be invalid or the server unable to compile it for other reasons. error-status is undefined.
		// not-marked (44)
		//   Attempt to manipulate a text in a way that requires the text to be marked, when in fact it is not marked. error-status indicates the text in question.
		// temporary-failure (45)
		//   Temporary failure. Try again later. error-status is undefined.
		// long-array (46)
		//   An array sent to the server was too long. error-status is undefined.
		// anonymous-rejected (47)
		//   Attempt to send an anonymous text to a conference that does not accept anonymous texts. error-status is undefined.
		// illegal-aux-item (48)
		//   Attempt to create an invalid aux-item. Probably the tag or data are invalid. error-status contains the index in the aux-item list where the invalid item appears.
		// aux-item-permission (49)
		//   Attempt to manipulate an aux-item without enough permissions. This response is sent when attempting to delete an item set by someone else or an item that can't be deleted, and when attempting to create an item without permissions to do so. error-status contains the index at which the item appears in the aux-item list sent to the server.
		// unknown-async (50)
		//   Sent in response to a request for an asynchronous message the server does not send. The call succeeds, but this is sent as a warning to the client. error-status contains the message type the server did not understand.
		// internal-error (51)
		//   The server has encountered a possibly recoverable internal error. error-status is undefined.
		// feature-disabled (52)
		//   Attempt to use a feature that has been explicitly disabled in the server. error-status is undefined.
		// message-not-sent (53)
		//   Attempt to send an asynchronous message failed for some reason. Perhaps the recipient is not accepting messages at the moment or there are no viable members in the recipient of the message. error-status is undefined.
		// invalid-membership-type (54)
		//   A requested membership type was not compatible with restrictions set on the server or on a specific conference. error-status is undefined unless specifically mentioned in the documentation for a specific call.
		// invalid-range (55)
		//   The lower limit of a supplied range is greater than the upper limit. error-status is undefined.
		// invalid-range-list (56)
		//   The lower limit of a supplied range is not greater than the upper limit of the previous range in the list. error-status is undefined.
		// undefined-measurement (57)
		//   A request for a measurement that the server doesn't make has been made. error-status is undefined.
		// priority-denied (58)
		//   You don't have enough privileges to lower your priority. error-status indicates the lowest priority that you have access to.
		// weight-denied (59)
		//   You don't have enough privileges to set the specified weight.
		// weight-zero (60)
		//   The scheduling weight must be non-zero. error-status is undefined.
		// bad-bool (61)
		//   An argument of type BOOL was given a value that is neither 0 nor 1. error-status is undefined.
	}

	return fmt.Errorf("Generic error, code is %d, status is %d", code, status)
}
