package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ccfos/nightingale/v6/pkg/ctx"
	"github.com/toolkits/pkg/logger"
)

type AlertHisEvent struct {
	Id                 int64             `json:"id" gorm:"primaryKey"`
	Cate               string            `json:"cate"`
	IsRecovered        int               `json:"is_recovered"`
	DatasourceId       int64             `json:"datasource_id"`
	Cluster            string            `json:"cluster"`
	GroupId            int64             `json:"group_id"`
	GroupName          string            `json:"group_name"` // busi group name
	Hash               string            `json:"hash"`
	RuleId             int64             `json:"rule_id"`
	RuleName           string            `json:"rule_name"`
	RuleNote           string            `json:"rule_note"`
	RuleProd           string            `json:"rule_prod"`
	RuleAlgo           string            `json:"rule_algo"`
	Severity           int               `json:"severity"`
	PromForDuration    int               `json:"prom_for_duration"`
	PromQl             string            `json:"prom_ql"`
	RuleConfig         string            `json:"-" gorm:"rule_config"` // rule config
	RuleConfigJson     interface{}       `json:"rule_config" gorm:"-"` // rule config for fe
	PromEvalInterval   int               `json:"prom_eval_interval"`
	Callbacks          string            `json:"-"`
	CallbacksJSON      []string          `json:"callbacks" gorm:"-"`
	RunbookUrl         string            `json:"runbook_url"`
	NotifyRecovered    int               `json:"notify_recovered"`
	NotifyChannels     string            `json:"-"`
	NotifyChannelsJSON []string          `json:"notify_channels" gorm:"-"`
	NotifyGroups       string            `json:"-"`
	NotifyGroupsJSON   []string          `json:"notify_groups" gorm:"-"`
	NotifyGroupsObj    []UserGroup       `json:"notify_groups_obj" gorm:"-"`
	TargetIdent        string            `json:"target_ident"`
	TargetNote         string            `json:"target_note"`
	TriggerTime        int64             `json:"trigger_time"`
	TriggerValue       string            `json:"trigger_value"`
	RecoverTime        int64             `json:"recover_time"`
	LastEvalTime       int64             `json:"last_eval_time"`
	Tags               string            `json:"-"`
	TagsJSON           []string          `json:"tags" gorm:"-"`
	OriginalTags       string            `json:"-"`                       // for db
	OriginalTagsJSON   []string          `json:"original_tags"  gorm:"-"` // for fe
	Annotations        string            `json:"-"`
	AnnotationsJSON    map[string]string `json:"annotations" gorm:"-"` // for fe
	NotifyCurNumber    int               `json:"notify_cur_number"`    // notify: current number
	FirstTriggerTime   int64             `json:"first_trigger_time"`   // 连续告警的首次告警时间
	ExtraConfig        interface{}       `json:"extra_config" gorm:"-"`
	NotifyRuleIds      []int64           `json:"notify_rule_ids" gorm:"serializer:json"`

	NotifyVersion int                `json:"notify_version" gorm:"-"`
	NotifyRules   []*EventNotifyRule `json:"notify_rules" gorm:"-"`
}

func (e *AlertHisEvent) TableName() string {
	return "alert_his_event"
}

func (e *AlertHisEvent) Add(ctx *ctx.Context) error {
	return Insert(ctx, e)
}

func (e *AlertHisEvent) DB2FE() {
	e.NotifyChannelsJSON = strings.Fields(e.NotifyChannels)
	e.NotifyGroupsJSON = strings.Fields(e.NotifyGroups)
	e.CallbacksJSON = strings.Fields(e.Callbacks)
	e.TagsJSON = strings.Split(e.Tags, ",,")
	e.OriginalTagsJSON = strings.Split(e.OriginalTags, ",,")

	if len(e.Annotations) > 0 {
		err := json.Unmarshal([]byte(e.Annotations), &e.AnnotationsJSON)
		if err != nil {
			e.AnnotationsJSON = make(map[string]string)
			e.AnnotationsJSON["error"] = e.Annotations
		}
	}

	json.Unmarshal([]byte(e.RuleConfig), &e.RuleConfigJson)
}

func (e *AlertHisEvent) FillNotifyGroups(ctx *ctx.Context, cache map[int64]*UserGroup) error {
	// some user-group already deleted ?
	count := len(e.NotifyGroupsJSON)
	if count == 0 {
		e.NotifyGroupsObj = []UserGroup{}
		return nil
	}

	for i := range e.NotifyGroupsJSON {
		id, err := strconv.ParseInt(e.NotifyGroupsJSON[i], 10, 64)
		if err != nil {
			continue
		}

		ug, has := cache[id]
		if has {
			e.NotifyGroupsObj = append(e.NotifyGroupsObj, *ug)
			continue
		}

		ug, err = UserGroupGetById(ctx, id)
		if err != nil {
			return err
		}

		if ug != nil {
			e.NotifyGroupsObj = append(e.NotifyGroupsObj, *ug)
			cache[id] = ug
		}
	}

	return nil
}

// func (e *AlertHisEvent) FillTaskTplName(ctx *ctx.Context, cache map[int64]*UserGroup) error {

// }

func AlertHisEventTotal(
	ctx *ctx.Context, prods []string, bgids []int64, stime, etime int64, severity int,
	recovered int, dsIds []int64, cates []string, ruleId int64, query string) (int64, error) {
	session := DB(ctx).Model(&AlertHisEvent{}).Where("last_eval_time between ? and ?", stime, etime)

	if len(prods) > 0 {
		session = session.Where("rule_prod in ?", prods)
	}

	if len(bgids) > 0 {
		session = session.Where("group_id in ?", bgids)
	}

	if severity >= 0 {
		session = session.Where("severity = ?", severity)
	}

	if recovered >= 0 {
		session = session.Where("is_recovered = ?", recovered)
	}

	if len(dsIds) > 0 {
		session = session.Where("datasource_id in ?", dsIds)
	}

	if len(cates) > 0 {
		session = session.Where("cate in ?", cates)
	}

	if ruleId > 0 {
		session = session.Where("rule_id = ?", ruleId)
	}

	if query != "" {
		arr := strings.Fields(query)
		for i := 0; i < len(arr); i++ {
			qarg := "%" + arr[i] + "%"
			session = session.Where("rule_name like ? or tags like ?", qarg, qarg)
		}
	}

	return Count(session)
}

func AlertHisEventGets(ctx *ctx.Context, prods []string, bgids []int64, stime, etime int64,
	severity int, recovered int, dsIds []int64, cates []string, ruleId int64, query string,
	limit, offset int) ([]AlertHisEvent, error) {
	session := DB(ctx).Where("last_eval_time between ? and ?", stime, etime)

	if len(prods) != 0 {
		session = session.Where("rule_prod in ?", prods)
	}

	if len(bgids) > 0 {
		session = session.Where("group_id in ?", bgids)
	}

	if severity >= 0 {
		session = session.Where("severity = ?", severity)
	}

	if recovered >= 0 {
		session = session.Where("is_recovered = ?", recovered)
	}

	if len(dsIds) > 0 {
		session = session.Where("datasource_id in ?", dsIds)
	}

	if len(cates) > 0 {
		session = session.Where("cate in ?", cates)
	}

	if ruleId > 0 {
		session = session.Where("rule_id = ?", ruleId)
	}

	if query != "" {
		arr := strings.Fields(query)
		for i := 0; i < len(arr); i++ {
			qarg := "%" + arr[i] + "%"
			session = session.Where("rule_name like ? or tags like ?", qarg, qarg)
		}
	}

	var lst []AlertHisEvent
	err := session.Order("trigger_time desc, id desc").Limit(limit).Offset(offset).Find(&lst).Error

	if err == nil {
		for i := 0; i < len(lst); i++ {
			lst[i].DB2FE()
		}
	}

	return lst, err
}

func AlertHisEventGet(ctx *ctx.Context, where string, args ...interface{}) (*AlertHisEvent, error) {
	var lst []*AlertHisEvent
	err := DB(ctx).Where(where, args...).Find(&lst).Error
	if err != nil {
		return nil, err
	}

	if len(lst) == 0 {
		return nil, nil
	}

	lst[0].DB2FE()
	lst[0].FillNotifyGroups(ctx, make(map[int64]*UserGroup))

	return lst[0], nil
}

func AlertHisEventGetById(ctx *ctx.Context, id int64) (*AlertHisEvent, error) {
	return AlertHisEventGet(ctx, "id=?", id)
}

func AlertHisEventBatchDelete(ctx *ctx.Context, timestamp int64, severities []int, limit int) (int64, error) {
	db := DB(ctx).Where("last_eval_time < ?", timestamp)
	if len(severities) > 0 {
		db = db.Where("severity IN (?)", severities)
	}
	res := db.Limit(limit).Delete(&AlertHisEvent{})
	return res.RowsAffected, res.Error
}

func (m *AlertHisEvent) UpdateFieldsMap(ctx *ctx.Context, fields map[string]interface{}) error {
	return DB(ctx).Model(m).Updates(fields).Error
}

func AlertHisEventUpgradeToV6(ctx *ctx.Context, dsm map[string]Datasource) error {
	var lst []*AlertHisEvent
	err := DB(ctx).Where("trigger_time > ?", time.Now().Unix()-3600*24*30).Limit(10000).Order("id desc").Find(&lst).Error
	if err != nil {
		return err
	}

	for i := 0; i < len(lst); i++ {
		ds, exists := dsm[lst[i].Cluster]
		if !exists {
			continue
		}
		lst[i].DatasourceId = ds.Id

		ruleConfig := PromRuleConfig{
			Queries: []PromQuery{
				{
					PromQl:   lst[i].PromQl,
					Severity: lst[i].Severity,
				},
			},
		}
		b, _ := json.Marshal(ruleConfig)
		lst[i].RuleConfig = string(b)

		if lst[i].RuleProd == "" {
			lst[i].RuleProd = METRIC
		}

		if lst[i].Cate == "" {
			lst[i].Cate = PROMETHEUS
		}

		err = lst[i].UpdateFieldsMap(ctx, map[string]interface{}{
			"datasource_id": lst[i].DatasourceId,
			"rule_config":   lst[i].RuleConfig,
			"rule_prod":     lst[i].RuleProd,
			"cate":          lst[i].Cate,
		})
		if err != nil {
			logger.Errorf("update alert rule:%d datasource ids failed, %v", lst[i].Id, err)
		}
	}
	return nil
}

func EventPersist(ctx *ctx.Context, event *AlertCurEvent) error {
	has, err := AlertCurEventExists(ctx, "hash=?", event.Hash)
	if err != nil {
		return fmt.Errorf("event_persist_check_exists_fail: %v rule_id=%d hash=%s", err, event.RuleId, event.Hash)
	}

	his := event.ToHis(ctx)

	// 不管是告警还是恢复，全量告警里都要记录
	if err := his.Add(ctx); err != nil {
		return fmt.Errorf("add his event error:%v", err)
	}

	if has {
		// 活跃告警表中有记录，删之
		err = AlertCurEventDelByHash(ctx, event.Hash)
		if err != nil {
			return fmt.Errorf("event_del_cur_fail: %v hash=%s", err, event.Hash)
		}

		if !event.IsRecovered {
			// 恢复事件，从活跃告警列表彻底删掉，告警事件，要重新加进来新的event
			// use his id as cur id
			event.Id = his.Id
			if event.Id > 0 {
				if err := event.Add(ctx); err != nil {
					return fmt.Errorf("add cur event err:%v", err)
				}
			}
		}

		// use his id as cur id
		event.Id = his.Id
		return nil
	}

	// use his id as cur id
	event.Id = his.Id

	if event.IsRecovered {
		// alert_cur_event表里没有数据，表示之前没告警，结果现在报了恢复，神奇....理论上不应该出现的
		return nil
	}

	if event.Id > 0 {
		if err := event.Add(ctx); err != nil {
			return fmt.Errorf("add cur event error:%v", err)
		}
	}

	return nil
}

func AlertHisEventGetByIds(ctx *ctx.Context, ids []int64) ([]*AlertHisEvent, error) {
	var lst []*AlertHisEvent

	if len(ids) == 0 {
		return lst, nil
	}

	err := DB(ctx).Where("id in ?", ids).Order("trigger_time desc").Find(&lst).Error
	if err == nil {
		for i := 0; i < len(lst); i++ {
			lst[i].DB2FE()
		}
	}

	return lst, err
}

func (e *AlertHisEvent) ToCur() *AlertCurEvent {
	cur := AlertCurEvent{
		Id:                 e.Id,
		Cate:               e.Cate,
		Cluster:            e.Cluster,
		DatasourceId:       e.DatasourceId,
		GroupId:            e.GroupId,
		GroupName:          e.GroupName,
		Hash:               e.Hash,
		RuleId:             e.RuleId,
		RuleName:           e.RuleName,
		RuleProd:           e.RuleProd,
		RuleAlgo:           e.RuleAlgo,
		RuleNote:           e.RuleNote,
		Severity:           e.Severity,
		PromForDuration:    e.PromForDuration,
		PromQl:             e.PromQl,
		PromEvalInterval:   e.PromEvalInterval,
		RuleConfig:         e.RuleConfig,
		RuleConfigJson:     e.RuleConfigJson,
		Callbacks:          e.Callbacks,
		RunbookUrl:         e.RunbookUrl,
		NotifyRecovered:    e.NotifyRecovered,
		NotifyChannels:     e.NotifyChannels,
		NotifyGroups:       e.NotifyGroups,
		Annotations:        e.Annotations,
		AnnotationsJSON:    e.AnnotationsJSON,
		TargetIdent:        e.TargetIdent,
		TargetNote:         e.TargetNote,
		TriggerTime:        e.TriggerTime,
		TriggerValue:       e.TriggerValue,
		Tags:               e.Tags,
		TagsJSON:           strings.Split(e.Tags, ",,"),
		OriginalTags:       e.OriginalTags,
		LastEvalTime:       e.LastEvalTime,
		NotifyCurNumber:    e.NotifyCurNumber,
		FirstTriggerTime:   e.FirstTriggerTime,
		IsRecovered:        e.IsRecovered == 1,
		TriggerValues:      e.TriggerValue,
		CallbacksJSON:      e.CallbacksJSON,
		NotifyChannelsJSON: e.NotifyChannelsJSON,
		NotifyGroupsJSON:   e.NotifyGroupsJSON,
		OriginalTagsJSON:   e.OriginalTagsJSON,
	}

	cur.SetTagsMap()
	return &cur
}
