package rolecommands

import (
	"database/sql"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/common/cplogs"
	"github.com/ThatBathroom/yagpdb/common/pubsub"
	schEvtsModels "github.com/ThatBathroom/yagpdb/common/scheduledevents2/models"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/ThatBathroom/yagpdb/rolecommands/models"
	"github.com/ThatBathroom/yagpdb/web"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"goji.io"
	"goji.io/pat"
)

//go:embed assets/rolecommands.html
var PageHTML string

var (
	panelLogKeyNewCommand        = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_new_command", FormatString: "Created a new role command: %s"})
	panelLogKeyUpdatedCommand    = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_updated_command", FormatString: "Updated role command: %s"})
	panelLogKeyRemovedCommand    = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_removed_command", FormatString: "Removed role command: %d"})
	panelLogKeyRemoveAllCommands = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_removed_all_command", FormatString: "Removed all role command in group: %s"})

	panelLogKeyNewGroup     = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_new_group", FormatString: "Created a new role group: %s"})
	panelLogKeyUpdatedGroup = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_updated_group", FormatString: "Updated role group: %s"})
	panelLogKeyRemovedGroup = cplogs.RegisterActionFormat(&cplogs.ActionFormat{Key: "rolecommands_removed_group", FormatString: "Removed role group: %d"})
)

type FormCommand struct {
	ID           int64
	Name         string `valid:",1,100,trimspace"`
	Role         int64  `valid:"role,false"`
	Group        int64
	RequireRoles []int64 `valid:"role,true"`
	IgnoreRoles  []int64 `valid:"role,true"`
}

type FormGroup struct {
	ID           int64
	Name         string  `valid:",1,100,trimspace"`
	RequireRoles []int64 `valid:"role,true"`
	IgnoreRoles  []int64 `valid:"role,true"`

	Mode int

	MultipleMax int `valid:"0,250"`
	MultipleMin int `valid:"0,250"`

	SingleAutoToggleOff   bool
	SingleRequireOne      bool
	TemporaryRoleDuration int `valid:"0,1440"`
}

func (p *Plugin) InitWeb() {
	web.AddHTMLTemplate("rolecommands/assets/rolecommands.html", PageHTML)

	web.AddSidebarItem(web.SidebarCategoryTools, &web.SidebarItem{
		Name: "Role Commands",
		URL:  "rolecommands/",
		Icon: "fas fa-tags",
	})

	// Setup SubMuxer
	subMux := goji.SubMux()
	web.CPMux.Handle(pat.New("/rolecommands/*"), subMux)
	web.CPMux.Handle(pat.New("/rolecommands"), subMux)

	subMux.Use(web.RequireBotMemberMW)
	subMux.Use(web.RequirePermMW(discordgo.PermissionManageRoles))

	// Setup routes
	getIndexHandler := web.ControllerHandler(HandleGetIndex, "cp_rolecommands")
	getGroupHandler := web.ControllerHandler(func(w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
		groupIDRaw := pat.Param(r, "groupID")
		parsed, _ := strconv.ParseInt(groupIDRaw, 10, 64)
		return HandleGetGroup(parsed, w, r)
	}, "cp_rolecommands")

	subMux.Handle(pat.Get("/"), getIndexHandler)
	subMux.Handle(pat.Get("/group/:groupID"), getGroupHandler)

	// either serve the group page or the index page, kinda convoluted but eh
	getIndexpPostHandler := web.ControllerHandler(func(w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
		if r.FormValue("GroupID") != "" {
			parsed, _ := strconv.ParseInt(r.FormValue("GroupID"), 10, 64)
			return HandleGetGroup(parsed, w, r)
		}

		if r.FormValue("Group") != "" {
			parsed, _ := strconv.ParseInt(r.FormValue("Group"), 10, 64)
			return HandleGetGroup(parsed, w, r)
		}

		_, tmpl = web.GetBaseCPContextData(r.Context())
		if idInterface, ok := tmpl["GroupID"]; ok {
			return HandleGetGroup(idInterface.(int64), w, r)
		}

		return HandleGetIndex(w, r)
	}, "cp_rolecommands")

	subMux.Handle(pat.Post("/new_cmd"), web.ControllerPostHandler(HandleNewCommand, getIndexpPostHandler, FormCommand{}))
	subMux.Handle(pat.Post("/update_cmd"), web.ControllerPostHandler(HandleUpdateCommand, getIndexpPostHandler, FormCommand{}))
	subMux.Handle(pat.Post("/remove_cmd"), web.ControllerPostHandler(HandleRemoveCommand, getIndexpPostHandler, nil))
	subMux.Handle(pat.Post("/move_cmd"), web.ControllerPostHandler(HandleMoveCommand, getIndexpPostHandler, nil))
	subMux.Handle(pat.Post("/delete_rolecmds"), web.ControllerPostHandler(HandleDeleteRoleCommands, getIndexpPostHandler, nil))

	subMux.Handle(pat.Post("/new_group"), web.ControllerPostHandler(HandleNewGroup, getIndexpPostHandler, FormGroup{}))
	subMux.Handle(pat.Post("/update_group"), web.ControllerPostHandler(HandleUpdateGroup, getIndexpPostHandler, FormGroup{}))
	subMux.Handle(pat.Post("/remove_group"), web.ControllerPostHandler(HandleRemoveGroup, getIndexpPostHandler, nil))
}

func HandleGetIndex(w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	ungroupedCommands, err := models.RoleCommands(qm.Where("guild_id = ?", g.ID), qm.Where("role_group_id is null")).AllG(r.Context())
	if err != nil {
		return tmpl, err
	}
	sort.Slice(ungroupedCommands, RoleCommandsLessFunc(ungroupedCommands))

	tmpl["LoneCommands"] = ungroupedCommands

	groups, err := models.RoleGroups(qm.Where(models.RoleGroupColumns.GuildID+" = ?", g.ID), qm.OrderBy("id asc")).AllG(r.Context())
	if err != nil {
		return tmpl, err
	}

	tmpl["Groups"] = groups

	return tmpl, nil
}

func HandleGetGroup(groupID int64, w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())
	groups, err := models.RoleGroups(qm.Where(models.RoleGroupColumns.GuildID+" = ?", g.ID), qm.OrderBy("id asc")).AllG(r.Context())
	if err != nil {
		return tmpl, err
	}

	tmpl["Groups"] = groups

	var currentGroup *models.RoleGroup
	for _, v := range groups {
		if v.ID == groupID {
			tmpl["CurrentGroup"] = v
			currentGroup = v
			break
		}
	}

	if currentGroup != nil {
		commands, err := currentGroup.RoleCommands().AllG(r.Context())
		if err != nil {
			return tmpl, err
		}
		sort.Slice(commands, RoleCommandsLessFunc(commands))

		tmpl["Commands"] = commands
	} else {
		// Fallback
		return HandleGetIndex(w, r)
	}

	return tmpl, nil
}

func HandleNewCommand(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	form := r.Context().Value(common.ContextKeyParsedForm).(*FormCommand)
	form.Name = strings.TrimSpace(form.Name)

	if c, _ := models.RoleCommands(qm.Where(models.RoleCommandColumns.GuildID+"=?", g.ID)).CountG(r.Context()); c >= 1000 {
		tmpl.AddAlerts(web.ErrorAlert("Max 1000 role commands allowed"))
		return tmpl, nil
	}

	if existing, err := models.RoleCommands(qm.Where(models.RoleCommandColumns.GuildID+"=?", g.ID), qm.Where(models.RoleCommandColumns.Name+" ILIKE ?", form.Name),
		qm.Load(models.RoleCommandRels.RoleGroup)).OneG(r.Context()); err == nil {
		if existing.R.RoleGroup == nil {
			tmpl.AddAlerts(web.ErrorAlert("Already a role command with that name in the ungrouped section; delete it or use a different name"))
		} else {
			tmpl.AddAlerts(web.ErrorAlert(`Already a role command with that name in the "` + existing.R.RoleGroup.Name + `" group; delete it or use a different name`))
		}
		return tmpl, nil
	}

	model := &models.RoleCommand{
		Name:    form.Name,
		GuildID: g.ID,

		Role:         form.Role,
		RequireRoles: form.RequireRoles,
		IgnoreRoles:  form.IgnoreRoles,
	}

	if form.Group != -1 {
		group, err := models.RoleGroups(qm.Where(models.RoleGroupColumns.GuildID+"=?", g.ID), qm.Where(models.RoleGroupColumns.ID+"=?", form.Group)).OneG(r.Context())
		if err != nil {
			return tmpl, err
		}

		model.RoleGroupID = null.Int64From(group.ID)
	}

	const q = `
		SELECT max(position)
		FROM role_commands
		WHERE $1::bigint IS NULL AND role_group_id IS NULL
			OR role_group_ID = $1::bigint
	`
	var maxExistingPos sql.NullInt64
	if err := common.PQ.QueryRow(q, model.RoleGroupID).Scan(&maxExistingPos); err != nil {
		return tmpl, err
	} else if maxExistingPos.Valid {
		model.Position = maxExistingPos.Int64 + 1 // place new role command last
	}

	err := model.InsertG(r.Context(), boil.Infer())
	if err == nil {
		go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyNewCommand, &cplogs.Param{Type: cplogs.ParamTypeString, Value: form.Name}))
		sendEvictMenuCachePubSub(g.ID)
	}

	return tmpl, err
}

func HandleUpdateCommand(w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	formCmd := r.Context().Value(common.ContextKeyParsedForm).(*FormCommand)

	cmd, err := models.FindRoleCommandG(r.Context(), formCmd.ID)
	if err != nil {
		return
	}

	if cmd.GuildID != g.ID {
		return tmpl.AddAlerts(web.ErrorAlert("That's not your command")), nil
	}

	cmd.Name = formCmd.Name
	cmd.Role = formCmd.Role
	cmd.IgnoreRoles = formCmd.IgnoreRoles
	cmd.RequireRoles = formCmd.RequireRoles

	groupChanged := cmd.RoleGroupID.Int64 != formCmd.Group
	if !cmd.RoleGroupID.Valid && formCmd.Group <= 0 {
		groupChanged = false // special case
	}

	if groupChanged {

		// validate group change
		if formCmd.Group != -1 {
			group, err := models.FindRoleGroupG(r.Context(), formCmd.Group)
			if err != nil {
				return tmpl, err
			}
			if group.GuildID != g.ID {
				return tmpl.AddAlerts(web.ErrorAlert("That's not your group")), nil
			}

			cmd.RoleGroupID = null.Int64From(group.ID)
		} else {
			cmd.RoleGroupID.Valid = false
		}

		// delete all related options since this is now pointing to a different group
		_, err := models.RoleMenuOptions(qm.Where("role_command_id = ?", cmd.ID)).DeleteAll(r.Context(), common.PQ)
		if err != nil {
			web.CtxLogger(r.Context()).WithError(err).Error("[rolecommands] failed clearing menu options on group change")
		}
	}

	_, err = cmd.UpdateG(r.Context(),
		boil.Whitelist(models.RoleCommandColumns.Name, models.RoleCommandColumns.Role, models.RoleCommandColumns.IgnoreRoles,
			models.RoleCommandColumns.RequireRoles, models.RoleCommandColumns.RoleGroupID))
	if err == nil {
		go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyUpdatedCommand, &cplogs.Param{Type: cplogs.ParamTypeString, Value: cmd.Name}))
		sendEvictMenuCachePubSub(g.ID)
	}
	return
}

func HandleMoveCommand(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())
	commands, err := models.RoleCommands(qm.Where("guild_id=?", g.ID)).AllG(r.Context())
	if err != nil {
		return tmpl, err
	}

	tID, err := strconv.ParseInt(r.FormValue("ID"), 10, 32)
	if err != nil {
		return tmpl, err
	}

	var targetCmd *models.RoleCommand
	for _, v := range commands {
		if v.ID == tID {
			targetCmd = v
			break
		}
	}

	if targetCmd == nil {
		return tmpl, errors.New("RoleCommand not found")
	}

	commandsInGroup := make([]*models.RoleCommand, 0, len(commands))

	// Sort all relevant commands
	for _, v := range commands {
		if (!targetCmd.RoleGroupID.Valid && !v.RoleGroupID.Valid) || (targetCmd.RoleGroupID.Valid && v.RoleGroupID.Valid && targetCmd.RoleGroupID.Int64 == v.RoleGroupID.Int64) {
			commandsInGroup = append(commandsInGroup, v)
		}
	}

	sort.Slice(commandsInGroup, RoleCommandsLessFunc(commandsInGroup))

	isUp := r.FormValue("dir") == "1"

	// Move the position
	for i := 0; i < len(commandsInGroup); i++ {
		v := commandsInGroup[i]

		v.Position = int64(i)
		if v.ID == tID {
			if isUp {
				if i == 0 {
					// Can't move further up
					continue
				}

				v.Position--
				commandsInGroup[i-1].Position = int64(i)
			} else {
				if i == len(commandsInGroup)-1 {
					// Can't move further down
					continue
				}
				v.Position++
				commandsInGroup[i+1].Position = int64(i)
				i++
			}
		}
	}

	for _, v := range commandsInGroup {
		_, lErr := v.UpdateG(r.Context(), boil.Whitelist(models.RoleCommandColumns.Position))
		if lErr != nil {
			err = lErr
		}
	}
	sendEvictMenuCachePubSub(g.ID)

	return tmpl, err
}

func HandleDeleteRoleCommands(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	idParsed, _ := strconv.ParseInt(r.FormValue("group"), 10, 64)

	var qmRoleGroupID qm.QueryMod
	if idParsed == -1 {
		qmRoleGroupID = qm.Where("role_group_id IS NULL")
	} else {
		qmRoleGroupID = qm.Where("role_group_id=?", idParsed)
	}
	result, err := models.RoleCommands(qm.Where("guild_id=?", g.ID), qmRoleGroupID).DeleteAll(r.Context(), common.PQ)
	if err != nil {
		return nil, err
	}

	if result > 0 {
		id := r.FormValue("group")
		if id == "" {
			id = "Ungrouped"
		}
		go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyRemoveAllCommands, &cplogs.Param{Type: cplogs.ParamTypeString, Value: id}))
		sendEvictMenuCachePubSub(g.ID)
	}

	return tmpl, nil
}

func HandleRemoveCommand(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	idParsed, _ := strconv.ParseInt(r.FormValue("ID"), 10, 64)
	_, err := models.RoleCommands(qm.Where("guild_id=?", g.ID), qm.Where("id=?", idParsed)).DeleteAll(r.Context(), common.PQ)
	if err != nil {
		return nil, err
	}

	go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyRemovedCommand, &cplogs.Param{Type: cplogs.ParamTypeInt, Value: idParsed}))
	sendEvictMenuCachePubSub(g.ID)

	return tmpl, nil
}

func HandleNewGroup(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	form := r.Context().Value(common.ContextKeyParsedForm).(*FormGroup)
	form.Name = strings.TrimSpace(form.Name)

	if c, _ := models.RoleGroups(qm.Where("guild_id=?", g.ID)).CountG(r.Context()); c >= 1000 {
		tmpl.AddAlerts(web.ErrorAlert("Max 1000 role groups allowed"))
		return tmpl, nil
	}

	if c, _ := models.RoleGroups(qm.Where("guild_id=?", g.ID), qm.Where("name ILIKE ?", form.Name)).CountG(r.Context()); c > 0 {
		tmpl.AddAlerts(web.ErrorAlert("Already a role group with that name"))
		return tmpl, nil
	}

	model := &models.RoleGroup{
		Name:    form.Name,
		GuildID: g.ID,

		RequireRoles: form.RequireRoles,
		IgnoreRoles:  form.IgnoreRoles,
		Mode:         int64(form.Mode),

		MultipleMax:         int64(form.MultipleMax),
		MultipleMin:         int64(form.MultipleMin),
		SingleRequireOne:    form.SingleRequireOne,
		SingleAutoToggleOff: form.SingleAutoToggleOff,
	}

	err := model.InsertG(r.Context(), boil.Infer())
	if err != nil {
		return tmpl, err
	}

	go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyNewGroup, &cplogs.Param{Type: cplogs.ParamTypeString, Value: model.Name}))
	sendEvictMenuCachePubSub(g.ID)

	tmpl["GroupID"] = model.ID

	return tmpl, nil
}

func HandleUpdateGroup(w http.ResponseWriter, r *http.Request) (tmpl web.TemplateData, err error) {
	g, tmpl := web.GetBaseCPContextData(r.Context())

	formGroup := r.Context().Value(common.ContextKeyParsedForm).(*FormGroup)

	group, err := models.RoleGroups(qm.Where("guild_id=?", g.ID), qm.Where("id=?", formGroup.ID)).OneG(r.Context())
	if err != nil {
		return
	}

	group.Name = formGroup.Name
	group.IgnoreRoles = formGroup.IgnoreRoles
	group.RequireRoles = formGroup.RequireRoles
	group.SingleRequireOne = formGroup.SingleRequireOne
	group.SingleAutoToggleOff = formGroup.SingleAutoToggleOff
	group.MultipleMax = int64(formGroup.MultipleMax)
	group.MultipleMin = int64(formGroup.MultipleMin)
	group.Mode = int64(formGroup.Mode)
	group.TemporaryRoleDuration = formGroup.TemporaryRoleDuration

	tmpl["GroupID"] = group.ID

	_, err = group.UpdateG(r.Context(), boil.Infer())
	if err != nil {
		return
	}

	go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyUpdatedGroup, &cplogs.Param{Type: cplogs.ParamTypeString, Value: group.Name}))
	sendEvictMenuCachePubSub(g.ID)

	if group.TemporaryRoleDuration < 1 {
		_, err = schEvtsModels.ScheduledEvents(qm.Where("event_name='remove_member_role' AND guild_id = ? AND (data->>'group_id')::bigint = ?", g.ID, group.ID)).DeleteAll(r.Context(), common.PQ)
	}

	return
}

func HandleRemoveGroup(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, _ := web.GetBaseCPContextData(r.Context())

	idParsed, _ := strconv.ParseInt(r.FormValue("ID"), 10, 64)
	_, err := models.RoleGroups(qm.Where("guild_id=?", g.ID), qm.Where("id=?", idParsed)).DeleteAll(r.Context(), common.PQ)
	if err == nil {
		go cplogs.RetryAddEntry(web.NewLogEntryFromContext(r.Context(), panelLogKeyRemovedGroup, &cplogs.Param{Type: cplogs.ParamTypeInt, Value: idParsed}))
		sendEvictMenuCachePubSub(g.ID)
	}
	return nil, err
}

var _ web.PluginWithServerHomeWidget = (*Plugin)(nil)

func (p *Plugin) LoadServerHomeWidget(w http.ResponseWriter, r *http.Request) (web.TemplateData, error) {
	g, templateData := web.GetBaseCPContextData(r.Context())
	templateData["WidgetTitle"] = "Role commands"
	templateData["SettingsPath"] = "/rolecommands/"

	numCommands, err := models.RoleCommands(qm.Where("guild_id = ?", g.ID)).CountG(r.Context())

	if err != nil {
		return templateData, err
	}

	numGroups, err := models.RoleGroups(qm.Where("guild_id = ?", g.ID)).CountG(r.Context())

	if numCommands > 0 {
		templateData["WidgetEnabled"] = true
	} else {
		templateData["WidgetDisabled"] = true
	}

	templateData["WidgetBody"] = template.HTML(fmt.Sprintf(`<ul>
		<li>Active role commands: <code>%d</code></li>
		<li>Active role groups: <code>%d</code></li>
		</ul>`, numCommands, numGroups))

	return templateData, err
}

func sendEvictMenuCachePubSub(guildID int64) {
	err := pubsub.Publish("role_commands_evict_menus", guildID, nil)
	if err != nil {
		logger.WithError(err).WithField("guild", guildID).Error("failed evicting rolemenu cache")
	}
}
