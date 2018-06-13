package service

import (
	"net/smtp"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/utils/log"
)

type EmailAccessLog struct {
	RepoName string `json:"ImageName"`
	RepoTag  string `json:"ImageTag"`
	Overdue  string `json:"Overdue"`
}

func Detection() {
	for {

		userNum, err := dao.GetAllUserID()
		if err != nil {
			log.Errorf("error happens when get user from sql: %v", err)
		}

		adminDeTime := "admin_detectTime"
		t, err := dao.GetAdminTime(adminDeTime)
		if err != nil {
			log.Errorf("error happens when get Imagelist from sql: %v", err)
		}
		if t == 0 {
			log.Errorf("error happens when get no time: %v", err)
		}

		for _, u := range userNum {
			userName, err := dao.GetUserName(u)
			mail, err := dao.GetEmailByUsername(userName)
			if err != nil {
				log.Errorf("Failed to select email: %v", err)
			}

			image_result, err := dao.GetEmailImageList(u, t)
			if err != nil {
				log.Errorf("error happens when get Imagelist from sql: %s", err)
			}

			var body string
			email_log := []EmailAccessLog{}
			var eml EmailAccessLog
			for _, im := range image_result {
				eml.RepoName = im.RepoName
				eml.RepoTag = im.RepoTag
				eml.Overdue = strconv.FormatFloat(im.Overdue, 'f', 0, 64)

				vt := reflect.TypeOf(eml)
				vv := reflect.ValueOf(eml)
				for i := 0; i < vt.NumField(); i++ {
					f := vt.Field(i)
					chKey := f.Tag.Get("json")
					k, v := chKey, vv.FieldByName(f.Name).String()
					body = body + k + ": " + v + "\n"
				}
				body = body + "\n"

				email_log = append(email_log, eml)
			}

			if body != "" {
				err = SendMail(mail, body)
				if err != nil {
					log.Errorf("error happens when send email : %s", err)
				}
			}
		}
		adminCyTime := "admin_cycleTime"
		ct, err := dao.GetAdminTime(adminCyTime)
		time.Sleep(time.Duration(ct) * 24 * time.Hour)
	}
}

func SendMail(to, b string) error {
	config, err := beego.AppConfig.GetSection("mail")
	if err != nil {
		log.Errorf("Can not load app.conf: %v", err)

	}
	user := config["from"]
	password := config["password"]
	host := config["host"] + ":" + config["port"]
	subject := "The Result From Extended Image Regular Check !"
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	body := "The following images are superimposed, so please manage them :" + "\n" + b
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err = smtp.SendMail(host, auth, user, send_to, msg)
	return err
}
