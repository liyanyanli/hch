package service

import (
	"errors"
	//	"io/ioutil"
	"net/smtp"
	"strings"

	"bufio"

	"github.com/astaxie/beego"
	"github.com/vmware/harbor/dao"
	//	"github.com/vmware/harbor/service/scan"
	"github.com/vmware/harbor/utils/log"
)

//const logFilePath = `/var/log/image-log/`

//const numFilePath = `/var/log/image-num/`

func CheckImageSafety(to, content string, repoName string, userName string, projectName string) error {
	var otherStatus int
	hn, on := Count(content)

	if hn > 0 {
		otherStatus = 5
	} else {
		otherStatus = OtherStats(content)
	}

	check, err := dao.CheckRepoName(repoName)
	if err != nil {
		log.Errorf("Error happens check clair_result: %v", err)
		return err
	}
	if check {
		pidu, err := dao.UpdateCliarResult(repoName, userName, projectName, hn, on, content, otherStatus, 1)
		if err != nil {
			log.Errorf("Error happens insert clair_result: %v", err)
			//return err
		}
		log.Debugf("The result of update clair_result: %s", pidu)

	} else {
		//insert into mysql (repo_name,user_name,project_name,high_num,other_num, clair_result)
		pid, err := dao.AddCliarResult(repoName, userName, projectName, hn, on, content, otherStatus, 1, 0)
		if err != nil {
			log.Errorf("Error happens insert clair_result: %v", err)
			//return err
		}
		log.Debugf("The result of insert clair_result: %s", pid)
	}

	if hn > 0 {
		if strings.Contains(content, "Success") {
			log.Debugf("The image is safe")
			return nil
		} else {
			if to != "" {
				err := SendToMail(to, content)
				return err
			} else {
				err := errors.New("The user does not have a valid mailbox")
				return err
			}

		}

	}
	return nil
}
func CheckImageSize(fileName string, repoName string, userName string, projectName string, image_size float64) error {
	/*
			file_num := numFilePath + fileName + ".num"
		//get the size of image

				image_size, err := getImageSize(file_num)

			if err != nil {
				//err := errors.New("Error happens when check image")
				log.Errorf("Failed to check image: %v", err)
				return err
			}
	*/
	check, err := dao.CheckRepoName(repoName)
	if err != nil {
		log.Errorf("Error happens check clair_result: %v", err)
		return err
	}
	usedSize, err_size := dao.GetUseSizeByProject(projectName)
	if err_size != nil {
		log.Errorf("Error happens get the used quota of the project:%v ", err)
	}
	addImageSize := image_size
	log.Debugf("The used size_quota of the project : %s", usedSize)
	log.Info("upload image, update quota for repoName %s, image size %f, used size %f, exist %t", repoName, image_size, usedSize, check)
	if check {
		imageSize, err := dao.GetSizeOfImage(repoName)
		if err != nil {
			log.Errorf("Error happens get the size of the image from sql:%v ", err)
		} else {
			//exist tag, need to calculate by new size and old size
			addImageSize = image_size - imageSize
		}
		//set the result of clair and set the size of image
		pidu, err := dao.UpdateImageSize(repoName, image_size)
		if err != nil {
			log.Errorf("Error happens insert clair_result: %v", err)
			//return err
		}
		log.Debugf("The result of update clair_result: %s", pidu)

	} else {
		//insert into mysql (repo_name,user_name,project_name,high_num,other_num, clair_result)
		pid, err := dao.AddCliarResult(repoName, userName, projectName, 0, 0, "", 3, 1, image_size)
		if err != nil {
			log.Errorf("Error happens insert clair_result: %v", err)
			//return err
		}
		log.Debugf("The result of insert clair_result: %s", pid)

	}

	if err_size == nil {
		pidu, err := dao.AddProjectQuotaUsedSize(projectName, addImageSize)
		if err != nil {
			log.Errorf("Error happens when update the used quota of the project: %v", err)
			return err
		}
		log.Debugf("The result of update the quota of the project: %s", pidu)
	}

	err = checkQuota(projectName)
	if err != nil {
		log.Errorf("Error happens when send email to project admin: %v", err)
		return err
	}

	return nil
}
func SendToMail(to, body string) error {
	config, err := beego.AppConfig.GetSection("mail")
	if err != nil {
		log.Errorf("Can not load app.conf: %v", err)

	}
	user := config["from"]
	password := config["password"]
	host := config["host"] + ":" + config["port"]
	subject := "The result returned from harbor"
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err = smtp.SendMail(host, auth, user, send_to, msg)
	return err
}

func Count(content string) (int, int) {
	highNum := 0
	otheNum := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if (strings.HasPrefix(line, "CVE-") || strings.HasPrefix(line, "RHSA-")) && strings.Contains(line, "(") {
			if strings.Contains(line, "Low") {
				otheNum++
			} else if strings.Contains(line, "Negligible") {
				otheNum++
			} else if strings.Contains(line, "Medium") {
				otheNum++
			} else if strings.Contains(line, "High") || strings.Contains(line, "Critical"){
				highNum++
			} else if strings.Contains(line, "Unknown") {
				otheNum++
			}
		}
	}
	return highNum, otheNum
}

func OtherStats(content string) int {
	if strings.Contains(content, "the image isn't supported by Clair") {
		return 1
	} else if strings.Contains(content, "Success!") {
		return 2
	} else if strings.Contains(content, "CVE-") || strings.Contains(content, "RHSA-"){
		return 4
	}
	return 3
}
