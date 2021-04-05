package sessions

import uuid "github.com/satori/go.uuid"

// SessionDO model
type SessionDO struct {
	SessionID string    `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (s *SessionDO) FromProto(session *Session) (*SessionDO, error) {
	userUUID, err := uuid.FromString(session.GetUserID())
	if err != nil {
		return nil, err
	}
	s.UserID = userUUID
	s.SessionID = session.GetSessionID()
	return s, nil
}
