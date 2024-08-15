package handler

import (
	"database/sql"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

type TagHandler struct {
	base *BaseHandler
}

func NewTagHandler(injector do.Injector) (*TagHandler, error) {
	return &TagHandler{do.MustInvoke[*BaseHandler](injector)}, nil
}

type tagListResp struct {
	Tags []string `json:"tags,omitempty"` //标签列表
}

// List godoc
//
//	@Tags		Tag
//	@Summary	标签列表
//	@Accept		json
//	@Produce	json
//	@Param		x-api-token	header		string	true	"登录TOKEN"
//	@Success	200			{object}	tagListResp
//	@Router		/api/file/s3PreSigned [post]
func (t TagHandler) List(c echo.Context) error {
	mySet := mapset.NewSet[string]()
	context := c.(CustomContext)
	currentUser := context.CurrentUser()
	var tags []sql.NullString
	t.base.db.Table("Memo").Select("tags").Where("userId = ?", currentUser.Id).Find(&tags)
	for _, tagLine := range tags {
		if tagLine.Valid {
			split := strings.Split(tagLine.String, ",")
			for _, tag := range split {
				if tag != "" {
					mySet.Add(tag)
				}
			}
		}
	}
	return SuccessResp(c, tagListResp{
		Tags: mySet.ToSlice(),
	})
}

func init() {
	do.Provide(nil, NewTagHandler)
}
