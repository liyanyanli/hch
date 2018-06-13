package service

import (
	//	"bufio"
	"errors"
	//	"io/ioutil"
	//	"strconv"
	//	"strings"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/service/token"
	"github.com/vmware/harbor/utils/log"
)

/*
func getImageSize(fileName string) (float64, error) {
	var size float64
	var unit int
	//repo_tag := strings.Split(repoName, ":")
	size = 0.00
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		//fmt.Println("error")
		log.Errorf("Error happens when set the size of image: %v", err)
		return 0, err
	}
	str := string(b)
	scanner := bufio.NewScanner(strings.NewReader(str))
	log.Debugf("++++++++++++++++++++++++++++++++++")
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "IMAGE") {
			continue
		}
		str := strings.Split(line, " ")
		l := len(str)
		for i := 0; i < l; i++ {
			if len(strings.Replace(str[i], " ", "", -1)) != 0 {
				unit = i
				log.Debugf("++++++++++++++++++: %s,%d", str[unit], unit)
			}
		}
		log.Debugf("++++++++++++++++++: %s", str[unit])
		log.Debugf("++++++++++++++++++: %d", str[unit-1])
		if str[unit] == "MB" {
			//fmt.Println(str[unit-1])
			f, err := strconv.ParseFloat(str[unit-1], 64)
			if err != nil {
				//fmt.Println("err")
				log.Errorf("Error happens when set the size of image: %v", err)
				return 0, err
			}
			size = size + f
			//fmt.Println(size)
		} else if str[unit] == "kB" {
			f, err := strconv.ParseFloat(str[unit-1], 64)
			if err != nil {
				//fmt.Println("rrr")
				log.Errorf("Error happens when set the size of image: %v", err)
				return 0, err
			}
			size = size + f/1024
			//fmt.Println(size)
		} else if str[unit] == "B" {
			f, err := strconv.ParseFloat(str[unit-1], 64)
			if err != nil {
				//fmt.Println("rrr")
				log.Errorf("Error happens when set the size of image: %v", err)
				return 0, err
			}
			size = size + f/(1024*1024)
			//fmt.Println(size)
		} else if str[unit] == "GB" {
			f, err := strconv.ParseFloat(str[unit-1], 64)
			if err != nil {
				//fmt.Println("rrr")
				log.Errorf("Error happens when set the size of image: %v", err)
				return 0, err
			}
			size = size + f*1024
			//fmt.Println(size)
		}
		// lose the GB not sure "GB"or "gB"
	}
	return size, nil

}
*/
func addQuotaNum(repoName string, projectName string) error {
	check, err := dao.CheckRepoName(repoName)
	if err != nil {
		log.Errorf("Error happens when check the used quota of project: %v", err)
	}
	usedNum, err := dao.GetUseNumByProject(projectName)
	if err != nil {
		log.Errorf("Error happens get the used quota of the project:%v ", err)
		return err
	}
	log.Debugf("The used quota number of the project : %s", usedNum)

	if !check {
		//set the result of clair and set the size of image
		pidu, err := dao.AddProjectQuotaUsedNum(projectName)
		if err != nil {
			log.Errorf("Error happens when update project QuotaUsedNum : %v", err)
			return err
		}
		log.Debugf("The result of update project_quota: %s", pidu)
	}
	return nil
}
func checkQuota(projectName string) error {
	thread := token.GetThreshold()
	//flag := 0
	//var to string
	//	var projectAdmin []int
	usedNum, usedSize, quotaNum, quotaSize, err := dao.GetInforByProject(projectName)
	if err != nil {
		log.Errorf("Error happens get the  quota information of the project:%v ", err)
		return err
	}
	if (usedNum >= quotaNum) || (usedSize >= quotaSize*thread) {
		content := "Your project " + projectName + "'s quota has been used up. No one can push image again.Please clean up or update the quota of current project."
		projectID, err := dao.GetProjectID(projectName)
		if err != nil {
			log.Errorf("failed to get projectID by projectName error: %v", err)
			return err
		}
		projectAdmin, err := dao.GetProjectAdmin(projectID)
		if err != nil {
			log.Errorf("failed to get projectAdmin by projectID error: %v", err)
			return err
		}
		for _, admin := range projectAdmin {
			to, err := dao.GetEmailByUserId(admin)
			if err != nil {
				log.Errorf("Failed to select email by userID: %v : %d", err, admin)
			}
			if to != "" {
				err := SendToMail(to, content)
				if err != nil {
					log.Errorf("Failed to send email to  project admin: %v", err)
				}

			} else {
				err := errors.New("The user does not have a valid mailbox")
				log.Errorf("Failed to send email to  project admin: %v", err)
			}

		}

	}
	return nil

}
