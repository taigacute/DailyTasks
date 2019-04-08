// Package model provides ...
package model

import (
	"database/sql"
	"html/template"
	"log"
	"strconv"
	"strings"
	"time"

	md "github.com/shurcooL/github_flavored_markdown"
	"github.com/taigacute/DailyTasks/database"
)

//Task is the Struct used to indentify tasks
type Task struct {
	Id           int           `json:"id"`
	Title        string        `json:"title"`
	Content      string        `json:"content"`
	ContentHtml  template.HTML `json:"content_html"`
	Created      string        `json:"created"`
	Priority     string        `json:"priority"`
	Category     string        `json:"category"`
	Referer      string        `json:"referer,omitempty"`
	Comments     []Comment     `json:"comments,omitempty"`
	IsOverdue    bool          `json:"isoverdue,omitempty"`
	IsHidden     int           `json:"ishidden,omitempty"`
	CompletedMsg string        `json:"completedmsg,omitempty"`
}

//Tasks a slice Task
type Tasks []Task

var (
	tasks Tasks
	task  Task
	//TaskCreated time
	TaskCreated time.Time
	context     Context
	getTaskSQL  string
	rows        *sql.Rows
	taskStatus  map[string]int
)

//AddTask will add a new task
func (tk *Task) AddTask(title, content, category string, taskPriority int, username string, hidden int) error {
	log.Println("AddTask : started function")
	var err error
	userID, err := GetUserID(username)
	if err != nil && (title != "" || content != "") {
		return err
	}
	if category == "" {
		err = database.TaskExec("insert into task(title, content, priority, task_status_id, created_date, last_modified_at, user_id,hide) values(?,?,?,?,datetime(), datetime(),?,?)", title, content, taskPriority, taskStatus["PENDING"], userID, hidden)
	} else {
		categoryID := GetCategoryByName(username, category)
		err = database.TaskExec("insert into task(title, content, priority, created_date, last_modified_at, cat_id, task_status_id, user_id,hide) values(?,?,?,datetime(), datetime(), ?,?,?,?)", title, content, taskPriority, categoryID, taskStatus["PENDING"], userID, hidden)
	}
	return err
}

//GetAllTasks will return all tasks context
func (tk *Task) GetAllTasks(uname, status, category string) (Context, error) {
	log.Println("get task for ", status)
	comments, err := GetComments(uname)
	if err != nil {
		return context, err
	}
	basicSQL := "select t.id, title, content, created_date, priority, case when c.name is null then 'NA' else c.name end from task t, status s, user u left outer join  category c on c.id=t.cat_id where u.username=? and s.id=t.task_status_id and u.id=t.user_id "
	if category == "" {
		switch status {
		case "pending":
			getTaskSQL = basicSQL + "and s.status = 'PENDING' and t.hide !=1"
		case "deleted":
			getTaskSQL = basicSQL + "and s.status = 'DELETED' and t.hide !=1"
		case "completed":
			getTaskSQL = basicSQL + "and s.status = 'COMPLETE'and t.hide !=1"
		}
		getTaskSQL += " order by t.created_date asc"
		rows = database.TaskQueryRows(getTaskSQL, uname, uname)
	} else {
		status = category
		if category == "UNCATEGORIZED" {
			getTaskSQL = "select t.id, title, content, created_date, priority, 'UNCATEGORIZED' from task t, status s, user u where u.username=? and s.id=t.task_status_id and u.id=t.user_id and t.cat_id=0  and  s.status='PENDING'  order by priority desc, created_date asc, finish_date asc"
			rows = database.TaskQueryRows(getTaskSQL, uname)
		} else {
			getTaskSQL = basicSQL + " and name = ?  and  s.status='PENDING'  order by priority desc, created_date asc, finish_date asc"
			rows = database.TaskQueryRows(getTaskSQL, uname, category)
		}
	}
	defer rows.Close()
	for rows.Next() {
		task = Task{}
		err = rows.Scan(&task.Id, &task.Title, &task.Content, &TaskCreated, &task.Priority, &task.Category)
		taskCompleted := 0
		totalTasks := 0
		if strings.HasPrefix(task.Content, "- [") {
			for _, value := range strings.Split(task.Content, "\n") {
				if strings.HasPrefix(value, "- [x]") {
					taskCompleted += 1
				}
				totalTasks += 1
			}
			task.CompletedMsg = strconv.Itoa(taskCompleted) + "complete out of" + strconv.Itoa(totalTasks)
		}
		task.ContentHtml = template.HTML(md.Markdown([]byte(task.Content)))
		if err != nil {
			log.Println(err)
		}
		if comments[task.Id] != nil {
			task.Comments = comments[task.Id]
		}
		TaskCreated = TaskCreated.Local()
		task.Created = TaskCreated.Format("Jan 2 2006")
		tasks = append(tasks, task)
	}
	context = Context{
		Tasks:      tasks,
		Navigation: status,
	}
	return context, nil
}

//AddFile will return a error
func AddFile(fileName, token, username string) error {
	userID, err := GetUserID(username)
	if err != nil {
		return err
	}
	err = database.TaskExec("insert into files values(?,?,?,datetime())", fileName, token, userID)
	return err
}