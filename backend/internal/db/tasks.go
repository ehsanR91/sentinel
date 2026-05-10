package db

import (
	"fmt"
	"strings"
)

type Task struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Command      string `json:"command"`
	ScheduleKind string `json:"schedule_kind"` // manual | interval
	ScheduleExpr string `json:"schedule_expr"` // seconds for interval
	Enabled      bool   `json:"enabled"`
	CreatedBy    string `json:"created_by"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type TaskRun struct {
	ID          int64  `json:"id"`
	TaskID      int64  `json:"task_id"`
	StartedAt   int64  `json:"started_at"`
	EndedAt     int64  `json:"ended_at"`
	Status      string `json:"status"`
	TriggeredBy string `json:"triggered_by"`
	Output      string `json:"output"`
	ExitCode    int    `json:"exit_code"`
}

func ListTasks() ([]Task, error) {
	rows, err := db.Query(`SELECT id,name,description,command,schedule_kind,schedule_expr,enabled,created_by,created_at,updated_at FROM tasks ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var t Task
		var enabled int
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Command, &t.ScheduleKind, &t.ScheduleExpr, &enabled, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err == nil {
			t.Enabled = enabled == 1
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

func GetTask(id int64) (*Task, error) {
	var t Task
	var enabled int
	err := db.QueryRow(`SELECT id,name,description,command,schedule_kind,schedule_expr,enabled,created_by,created_at,updated_at FROM tasks WHERE id=?`, id).
		Scan(&t.ID, &t.Name, &t.Description, &t.Command, &t.ScheduleKind, &t.ScheduleExpr, &enabled, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	return &t, nil
}

func CreateTask(t *Task) (int64, error) {
	now := unixNow()
	res, err := db.Exec(`INSERT INTO tasks(name,description,command,schedule_kind,schedule_expr,enabled,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		t.Name, t.Description, t.Command, t.ScheduleKind, t.ScheduleExpr, boolToInt(t.Enabled), t.CreatedBy, now, now)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func UpdateTask(t *Task) error {
	_, err := db.Exec(`UPDATE tasks SET name=?, description=?, command=?, schedule_kind=?, schedule_expr=?, enabled=?, updated_at=? WHERE id=?`,
		t.Name, t.Description, t.Command, t.ScheduleKind, t.ScheduleExpr, boolToInt(t.Enabled), unixNow(), t.ID)
	return err
}

func DeleteTask(id int64) error {
	_, err := db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	return err
}

func CreateTaskRun(taskID int64, triggeredBy string) (int64, error) {
	res, err := db.Exec(`INSERT INTO task_runs(task_id, started_at, status, triggered_by) VALUES(?,?,?,?)`, taskID, unixNow(), "running", triggeredBy)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func FinishTaskRun(runID int64, status string, output string, exitCode int) error {
	if len(output) > 32768 {
		output = output[len(output)-32768:]
	}
	_, err := db.Exec(`UPDATE task_runs SET ended_at=?, status=?, output=?, exit_code=? WHERE id=?`, unixNow(), status, output, exitCode, runID)
	return err
}

func ListTaskRuns(limit int) ([]TaskRun, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := db.Query(`SELECT id,task_id,started_at,ended_at,status,triggered_by,output,exit_code FROM task_runs ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TaskRun, 0)
	for rows.Next() {
		var tr TaskRun
		if err := rows.Scan(&tr.ID, &tr.TaskID, &tr.StartedAt, &tr.EndedAt, &tr.Status, &tr.TriggeredBy, &tr.Output, &tr.ExitCode); err == nil {
			out = append(out, tr)
		}
	}
	return out, rows.Err()
}

func TaskStats() (map[string]int, error) {
	stats := map[string]int{"total": 0, "enabled": 0, "runs": 0, "failed_runs": 0}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&total); err != nil {
		return nil, err
	}
	stats["total"] = total

	var enabled int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE enabled=1`).Scan(&enabled); err != nil {
		return nil, err
	}
	stats["enabled"] = enabled

	var runs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_runs`).Scan(&runs); err != nil {
		return nil, err
	}
	stats["runs"] = runs

	var failedRuns int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_runs WHERE status='failed'`).Scan(&failedRuns); err != nil {
		return nil, err
	}
	stats["failed_runs"] = failedRuns
	return stats, nil
}

func LastRunByTask(taskID int64) (*TaskRun, error) {
	var tr TaskRun
	err := db.QueryRow(`SELECT id,task_id,started_at,ended_at,status,triggered_by,output,exit_code FROM task_runs WHERE task_id=? ORDER BY started_at DESC LIMIT 1`, taskID).
		Scan(&tr.ID, &tr.TaskID, &tr.StartedAt, &tr.EndedAt, &tr.Status, &tr.TriggeredBy, &tr.Output, &tr.ExitCode)
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

func LatestRunsByTaskIDs(taskIDs []int64) (map[int64]*TaskRun, error) {
	result := make(map[int64]*TaskRun, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for index, taskID := range taskIDs {
		placeholders[index] = "?"
		args[index] = taskID
	}

	query := fmt.Sprintf(`
		SELECT tr.id,tr.task_id,tr.started_at,tr.ended_at,tr.status,tr.triggered_by,tr.output,tr.exit_code
		FROM task_runs tr
		JOIN (
			SELECT task_id, MAX(started_at) AS max_started_at
			FROM task_runs
			WHERE task_id IN (%s)
			GROUP BY task_id
		) latest ON latest.task_id = tr.task_id AND latest.max_started_at = tr.started_at
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tr TaskRun
		if err := rows.Scan(&tr.ID, &tr.TaskID, &tr.StartedAt, &tr.EndedAt, &tr.Status, &tr.TriggeredBy, &tr.Output, &tr.ExitCode); err != nil {
			return nil, err
		}
		copyRun := tr
		result[tr.TaskID] = &copyRun
	}

	return result, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func ValidateTask(t *Task) error {
	if t.Name == "" {
		return fmt.Errorf("name is required")
	}
	if t.Command == "" {
		return fmt.Errorf("command is required")
	}
	if t.ScheduleKind == "" {
		t.ScheduleKind = "manual"
	}
	if t.ScheduleKind != "manual" && t.ScheduleKind != "interval" {
		return fmt.Errorf("unsupported schedule_kind")
	}
	return nil
}
