package comment

import (
	"encoding/json"
	"errors"
	"fmt"
	"ia-online-golang/internal/dto"
	"ia-online-golang/internal/http/responses"
	CommentService "ia-online-golang/internal/services/comment"
	"ia-online-golang/internal/utils"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type CommentController struct {
	log            *logrus.Logger
	validator      *validator.Validate
	CommentService CommentService.CommentServiceI
}

type CommentControllerI interface {
	SaveComment(w http.ResponseWriter, r *http.Request)
}

func New(log *logrus.Logger, validator *validator.Validate, commentService CommentService.CommentServiceI) *CommentController {
	return &CommentController{
		log:            log,
		validator:      validator,
		CommentService: commentService,
	}
}

func (c *CommentController) SaveComment(w http.ResponseWriter, r *http.Request) {
	const op = "CommentController.SaveComment"

	c.log.Debugf("%s: start", op)

	if r.Method != http.MethodPost {
		c.log.Infof("%s: method not allowed. method: %s", op, r.Method)

		w.Header().Set("Allow", http.MethodPost)
		responses.MethodNotAllowed(w)
		return
	}

	c.log.Debugf("%s: method id correct", op)

	var comment dto.AddCommentDTO
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		c.log.Infof("%s: decode error", op)

		responses.InvalidRequest(w)
		return
	}

	c.log.Debugf("%s: decode completed", op)

	// Валидируем данные
	if err := c.validator.Struct(comment); err != nil {
		c.log.Infof("%s: validation error", op)

		responses.ValidationError(w, utils.FormatValidationErrors(err))
		return
	}

	c.log.Debugf("%s: validation completed", op)

	result, err := c.CommentService.SaveComment(r.Context(), comment.IdLead, comment.Comment)
	if err != nil {
		if errors.Is(err, CommentService.ErrLeadNotFound) {
			c.log.Infof("%s: lead not found", op)

			responses.LeadNotFound(w)
			return
		} else if errors.Is(err, CommentService.ErrLeadDoesNotBelongToUser) {
			c.log.Infof("%s: lead does not belong to user", op)

			responses.Forbidden(w)
			return
		} else {
			c.log.Errorf("%s: %v", op, err)

			responses.ServerError(w)
			return
		}
	}

	c.log.Debugf("%s: add comment", op)

	w.Header().Set("Content-Type", "application/json")

	fmt.Println(result)

	json.NewEncoder(w).Encode(result)
}
