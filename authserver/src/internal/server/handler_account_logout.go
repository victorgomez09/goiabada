package server

import (
	"net/http"

	"github.com/leodip/goiabada/internal/common"
	"github.com/leodip/goiabada/internal/constants"
	"github.com/leodip/goiabada/internal/lib"
)

func (s *Server) handleAccountLogoutGet() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		sess, err := s.sessionStore.Get(r, common.SessionName)
		if err != nil {
			s.internalServerError(w, r, err)
			return
		}

		sessionIdentifier := ""
		if r.Context().Value(common.ContextKeySessionIdentifier) != nil {
			sessionIdentifier = r.Context().Value(common.ContextKeySessionIdentifier).(string)
		}

		userId := uint(0)

		if len(sessionIdentifier) > 0 {
			userSession, err := s.database.GetUserSessionBySessionIdentifier(sessionIdentifier)
			if err != nil {
				s.internalServerError(w, r, err)
				return
			}
			if userSession != nil {
				err := s.database.DeleteUserSession(userSession.Id)
				if err != nil {
					s.internalServerError(w, r, err)
					return
				}

				lib.LogAudit(constants.AuditDeletedUserSession, map[string]interface{}{
					"userId":        userSession.UserId,
					"userSessionId": userSession.Id,
					"loggedInUser":  s.getLoggedInSubject(r),
				})
				userId = userSession.UserId
			}
		}

		// clear the session state
		sess.Values = make(map[interface{}]interface{})
		err = sess.Save(r, w)
		if err != nil {
			s.internalServerError(w, r, err)
			return
		}

		if userId > 0 {
			lib.LogAudit(constants.AuditLogout, map[string]interface{}{
				"userId":       userId,
				"loggedInUser": s.getLoggedInSubject(r),
			})
		}

		http.Redirect(w, r, lib.GetBaseUrl()+"/account/profile", http.StatusFound)
	}
}
