// Package bitrix. Транспортный слой для получения данных из битрикса. Для них есть исходящие вебхуки.
package bitrix

import (
	"errors"
	"fmt"
	"ia-online-golang/internal/dto"
	"ia-online-golang/internal/http/responses"
	"ia-online-golang/internal/services/comment"
	"ia-online-golang/internal/services/lead"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

type BitrixController struct {
	log              *logrus.Logger
	authTokenDeal    string
	authTokenComment string
	LeadService      lead.LeadServiceI
	CommentService   comment.CommentServiceI
}

type BitrixControllerI interface {
	СhangingDeal(w http.ResponseWriter, r *http.Request)
	NewComment(w http.ResponseWriter, r *http.Request)
}

func New(log *logrus.Logger, authTokenDeal string, authTokenComment string, leadService lead.LeadServiceI, commentService comment.CommentServiceI) *BitrixController {
	return &BitrixController{
		log:              log,
		authTokenDeal:    authTokenDeal,
		authTokenComment: authTokenComment,
		LeadService:      leadService,
		CommentService:   commentService,
	}
}

// Функция для получения айди заявки при изменения статуса этой заявки. В воронке на каждый статус, есть тригер, которые при изменении статуса отправляет информацию о заявке.
func (c *BitrixController) СhangingDeal(w http.ResponseWriter, r *http.Request) {
	const op = "BitrixController.СhangingDeal"

	c.log.Debugf("%s: start", op)

	if r.Method != http.MethodPost {
		c.log.Infof("%s: method not allowed. method: %s", op, r.Method)

		w.Header().Set("Allow", http.MethodPost)
		responses.MethodNotAllowed(w)
		return
	}

	err := r.ParseForm()
	if err != nil {
		c.log.Errorf("%s: %v", op, err)

		responses.ServerError(w)
		return
	}

	var hook dto.OutgoingHookDeal

	for i := 0; ; i++ {
		key := fmt.Sprintf("document_id[%d]", i)
		if val := r.FormValue(key); val != "" {
			hook.DocumentID = append(hook.DocumentID, val)
		} else {
			break
		}
	}

	hook.Auth.Domain = r.FormValue("auth[domain]")
	hook.Auth.ClientEndpoint = r.FormValue("auth[client_endpoint]")
	hook.Auth.ServerEndpoint = r.FormValue("auth[server_endpoint]")
	hook.Auth.MemberID = r.FormValue("auth[member_id]")
	hook.Auth.ApplicationToken = r.FormValue("auth[application_token]")

	for key, values := range r.Form {
		c.log.Infof("%s: %s = %v", op, key, values)
	}

	c.log.Debugf("%s: parsing form", op)

	if hook.Auth.MemberID != c.authTokenDeal {
		c.log.Infof("%s: invalid member id", op)

		responses.Forbidden(w)
		return
	}

	c.log.Debugf("%s: correct member id", op)

	err = c.LeadService.EditDeal(r.Context(), hook.DocumentID)
	if err != nil {
		c.log.Errorf("%s: %v", op, err)

		responses.ServerError(w)
		return
	}

	c.log.Debugf("%s: deal changed", op)

	responses.Ok(w)
}

// Функция для получения айди комментария при его добавлении в заявке. Для этой ручки есть исходящий вебхук, которые срабатывает при добавлении любого коммента в crm. В сервисах проверяется айди воронки.
func (c *BitrixController) NewComment(w http.ResponseWriter, r *http.Request) {
	const op = "BitrixController.NewComment"

	c.log.Debugf("%s: start", op)

	if r.Method != http.MethodPost {
		c.log.Infof("%s: method not allowed. method: %s", op, r.Method)

		w.Header().Set("Allow", http.MethodPost)
		responses.MethodNotAllowed(w)
		return
	}

	err := r.ParseForm()
	if err != nil {
		c.log.Errorf("%s: error parsing form: %v", op, err)

		responses.ServerError(w)
		return
	}

	for key, values := range r.Form {
		c.log.Infof("%s: %s = %v", op, key, values)
	}

	var hook dto.OutgoingHookComment

	// Заполнение поля Auth
	hook.Auth.Domain = r.FormValue("auth[domain]")
	hook.Auth.ClientEndpoint = r.FormValue("auth[client_endpoint]")
	hook.Auth.ServerEndpoint = r.FormValue("auth[server_endpoint]")
	hook.Auth.MemberID = r.FormValue("auth[member_id]")
	hook.Auth.ApplicationToken = r.FormValue("auth[application_token]")

	// Заполнение ID комментария
	idStr := r.FormValue("data[FIELDS][ID]")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		// обработка ошибки: например, вернуть 400 Bad Request
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	hook.Data.Fields.ID = id

	if hook.Auth.ApplicationToken != c.authTokenComment {
		c.log.Infof("%s: invalid member id", op)

		responses.Forbidden(w)
		return
	}

	err = c.CommentService.SaveCommentFromBitrix(r.Context(), hook.Data.Fields.ID)
	if err != nil {
		if errors.Is(err, comment.ErrCommentDoesNotBelongToTheFunnel) {
			c.log.Infof("%s: %v", op, err)

			responses.Forbidden(w)
			return
		}
		c.log.Errorf("%s: %v", op, err)

		responses.ServerError(w)
		return
	}

	c.log.Debugf("%s: comment received", op)

	responses.Ok(w)
}
