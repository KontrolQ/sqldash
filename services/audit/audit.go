package audit

import (
	"net/http"
	"time"

	"sqldash/models"
	repository "sqldash/repositories/event"
	"sqldash/utils/collections"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Record(actor string, databaseName string, action string, detail string) {
	held := &models.Event{
		OccurredAt: time.Now(),
		Database:   databaseName,
		Action:     action,
		Detail:     detail,
		Actor:      actor,
	}

	if createError := repository.Create(held); createError != nil {
		logger.Errorf(LogPrefix, RecordFailedLog, action, createError)
	}
}

func Page(databaseName string, action string, page int) (*LogContext, *fiber.Error) {
	if page < 1 {
		page = 1
	}

	records, total, findError := repository.Page(databaseName, action, (page-1)*PageSize, PageSize)
	if findError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, LogUnavailable)
	}

	actions, actionsError := repository.Actions(databaseName)
	if actionsError != nil {
		actions = nil
	}

	shown := &LogContext{
		Title:    Title,
		Heading:  Title,
		Database: databaseName,
		Action:   action,
		Page:     page,
		Total:    total,
		Pages:    int((total + PageSize - 1) / PageSize),
		Choices:  choicesFor(actions),
		Carry:    carryOf(databaseName, action),
	}

	if shown.Pages > 0 && shown.Page > shown.Pages {
		shown.Page = shown.Pages
	}

	shown.PreviousPage = shown.Page - 1
	shown.NextPage = shown.Page + 1

	for _, record := range records {
		shown.Entries = append(shown.Entries, EntryView{
			When:     record.OccurredAt.Local().Format(MomentLayout),
			Ago:      readableAgo(record.OccurredAt),
			Database: record.Database,
			Action:   record.Action,
			Label:    LabelFor(record.Action),
			Tone:     toneOf(record.Action),
			Detail:   record.Detail,
			Actor:    record.Actor,
		})
	}

	if total > 0 {
		shown.FirstRow = (shown.Page-1)*PageSize + 1
		shown.LastRow = shown.FirstRow + len(shown.Entries) - 1
	}

	return shown, nil
}

func choicesFor(actions []string) []collections.Option {
	choices := []collections.Option{{Value: "", Label: EveryAction}}

	for _, action := range actions {
		choices = append(choices, collections.Option{Value: action, Label: LabelFor(action)})
	}

	return choices
}

func carryOf(databaseName string, action string) string {
	carry := ""

	if databaseName != "" {
		carry += DatabaseParameter + "=" + databaseName + "&"
	}

	if action != "" {
		carry += ActionParameter + "=" + action + "&"
	}

	return carry
}

func readableAgo(when time.Time) string {
	gap := time.Since(when)

	if gap < time.Minute {
		return JustNow
	}

	if gap < time.Hour {
		return plural(int(gap.Minutes()), MinuteWord)
	}

	if gap < DayHours*time.Hour {
		return plural(int(gap.Hours()), HourWord)
	}

	return plural(int(gap.Hours()/DayHours), DayWord)
}
