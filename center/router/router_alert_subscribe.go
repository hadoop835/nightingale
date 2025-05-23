package router

import (
	"net/http"
	"time"

	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/strx"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

// Return all, front-end search and paging
func (rt *Router) alertSubscribeGets(c *gin.Context) {
	bgid := ginx.UrlParamInt64(c, "id")
	lst, err := models.AlertSubscribeGets(rt.Ctx, bgid)
	ginx.Dangerous(err)

	ugcache := make(map[int64]*models.UserGroup)
	rulecache := make(map[int64]string)

	for i := 0; i < len(lst); i++ {
		ginx.Dangerous(lst[i].FillUserGroups(rt.Ctx, ugcache))
		ginx.Dangerous(lst[i].FillRuleNames(rt.Ctx, rulecache))
		ginx.Dangerous(lst[i].FillDatasourceIds(rt.Ctx))
		ginx.Dangerous(lst[i].DB2FE())
	}

	ginx.NewRender(c).Data(lst, err)
}

func (rt *Router) alertSubscribeGetsByGids(c *gin.Context) {
	gids := strx.IdsInt64ForAPI(ginx.QueryStr(c, "gids", ""), ",")
	if len(gids) > 0 {
		for _, gid := range gids {
			rt.bgroCheck(c, gid)
		}
	} else {
		me := c.MustGet("user").(*models.User)
		if !me.IsAdmin() {
			var err error
			gids, err = models.MyBusiGroupIds(rt.Ctx, me.Id)
			ginx.Dangerous(err)

			if len(gids) == 0 {
				ginx.NewRender(c).Data([]int{}, nil)
				return
			}
		}
	}

	lst, err := models.AlertSubscribeGetsByBGIds(rt.Ctx, gids)
	ginx.Dangerous(err)

	ugcache := make(map[int64]*models.UserGroup)
	rulecache := make(map[int64]string)

	for i := 0; i < len(lst); i++ {
		ginx.Dangerous(lst[i].FillUserGroups(rt.Ctx, ugcache))
		ginx.Dangerous(lst[i].FillRuleNames(rt.Ctx, rulecache))
		ginx.Dangerous(lst[i].FillDatasourceIds(rt.Ctx))
		ginx.Dangerous(lst[i].DB2FE())
	}

	ginx.NewRender(c).Data(lst, err)
}

func (rt *Router) alertSubscribeGet(c *gin.Context) {
	subid := ginx.UrlParamInt64(c, "sid")

	sub, err := models.AlertSubscribeGet(rt.Ctx, "id=?", subid)
	ginx.Dangerous(err)

	if sub == nil {
		ginx.NewRender(c, 404).Message("No such alert subscribe")
		return
	}

	ugcache := make(map[int64]*models.UserGroup)
	ginx.Dangerous(sub.FillUserGroups(rt.Ctx, ugcache))

	rulecache := make(map[int64]string)
	ginx.Dangerous(sub.FillRuleNames(rt.Ctx, rulecache))
	ginx.Dangerous(sub.FillDatasourceIds(rt.Ctx))
	ginx.Dangerous(sub.DB2FE())

	ginx.NewRender(c).Data(sub, nil)
}

func (rt *Router) alertSubscribeAdd(c *gin.Context) {
	var f models.AlertSubscribe
	ginx.BindJSON(c, &f)

	username := c.MustGet("username").(string)
	f.CreateBy = username
	f.UpdateBy = username
	f.GroupId = ginx.UrlParamInt64(c, "id")

	if f.GroupId <= 0 {
		ginx.Bomb(http.StatusBadRequest, "group_id invalid")
	}

	ginx.NewRender(c).Message(f.Add(rt.Ctx))
}

func (rt *Router) alertSubscribePut(c *gin.Context) {
	var fs []models.AlertSubscribe
	ginx.BindJSON(c, &fs)

	timestamp := time.Now().Unix()
	username := c.MustGet("username").(string)
	for i := 0; i < len(fs); i++ {
		fs[i].UpdateBy = username
		fs[i].UpdateAt = timestamp
		//After adding the function of batch subscription alert rules, rule_ids is used instead of rule_id.
		//When the subscription rules are updated, set rule_id=0 to prevent the wrong subscription caused by the old rule_id.
		fs[i].RuleId = 0
		ginx.Dangerous(fs[i].Update(
			rt.Ctx,
			"name",
			"disabled",
			"prod",
			"cate",
			"datasource_ids",
			"cluster",
			"rule_id",
			"rule_ids",
			"tags",
			"redefine_severity",
			"new_severity",
			"redefine_channels",
			"new_channels",
			"user_group_ids",
			"update_at",
			"update_by",
			"webhooks",
			"for_duration",
			"redefine_webhooks",
			"severities",
			"extra_config",
			"busi_groups",
			"note",
			"notify_rule_ids",
		))
	}

	ginx.NewRender(c).Message(nil)
}

func (rt *Router) alertSubscribeDel(c *gin.Context) {
	var f idsForm
	ginx.BindJSON(c, &f)
	f.Verify()

	ginx.NewRender(c).Message(models.AlertSubscribeDel(rt.Ctx, f.Ids))
}

func (rt *Router) alertSubscribeGetsByService(c *gin.Context) {
	lst, err := models.AlertSubscribeGetsByService(rt.Ctx)
	ginx.NewRender(c).Data(lst, err)
}
