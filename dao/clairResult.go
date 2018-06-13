package dao

import (
	"github.com/vmware/harbor/models"
)

func AddCliarResult(repo_name string, user_name string, project_name string, high_num int, other_num int, clair_result string, other_status int, CVE_status int, image_size float64) (int64, error) {
	o := GetOrmer()
	sql := `insert into clair_result(repo_name,user_name,project_name,high_num,other_num,clair_result,other_status,CVE_status,image_size) values(?,?,?,?,?,?,?,?,?)`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(repo_name, user_name, project_name, high_num, other_num, clair_result, other_status, CVE_status, image_size)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	return id, err
}

func GetUnsaftImgNumByUser(userName string) (int, error) {
	sql := `select count(clair_id) from clair_result where user_name = ? and high_num <> 0`
	var num int
	err := GetOrmer().Raw(sql, userName).QueryRow(&num)
	return num, err
}

func GetSumImgNumByUser(userName string) (int, error) {
	sql := `select count(clair_id) from clair_result where user_name = ?`
	var num int
	err := GetOrmer().Raw(sql, userName).QueryRow(&num)
	return num, err
}

func GetMildNumByUser(userName string) (int, error) {
	sql := `select count(clair_id) from clair_result where user_name = ? and high_num = 0 and other_num > 0 ; `
	var num int
	err := GetOrmer().Raw(sql, userName).QueryRow(&num)
	return num, err
}

//yanli
func GetOtherStatysByUser(userName string) ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select cr.clair_id,cr.repo_name,cr.user_name,cr.project_name,cr.high_num,cr.other_num,cr.clair_result,cr.other_status from clair_result cr where user_name = ?`

	var result []models.ClairResult

	if _, err := o.Raw(sql, userName).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetUnsaftImgNumByproject(projectName string) (int, error) {
	sql := `select count(clair_id) from clair_result where project_name = ? and high_num <> 0`
	var num int
	err := GetOrmer().Raw(sql, projectName).QueryRow(&num)
	return num, err
}

func GetSumImgNumByProject(projectName string) (int, error) {
	sql := `select count(clair_id) from clair_result where project_name = ?`
	var num int
	err := GetOrmer().Raw(sql, projectName).QueryRow(&num)
	return num, err
}

func GetMildNumByProject(userName string) (int, error) {
	sql := `select count(clair_id) from clair_result where project_name = ? and high_num = 0 and other_num > 0 ; `
	var num int
	err := GetOrmer().Raw(sql, userName).QueryRow(&num)
	return num, err
}

//yanli
func GetOtherStatysByProject(projectName string) ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select cr.clair_id,cr.repo_name,cr.user_name,cr.project_name,cr.high_num,cr.other_num,cr.clair_result,cr.other_status from clair_result cr where project_name = ?`

	var result []models.ClairResult

	if _, err := o.Raw(sql, projectName).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetHighNumByRepoName(repoName string) (int, error) {
	sql := `select high_num from clair_result where repo_name = ?;`
	var highNum int
	err := GetOrmer().Raw(sql, repoName).QueryRow(&highNum)
	return highNum, err
}

func GetOtherNumByRepoName(repoName string) (int, error) {
	sql := `select other_num from clair_result where repo_name = ?`
	var otherNum int
	err := GetOrmer().Raw(sql, repoName).QueryRow(&otherNum)
	return otherNum, err
}

func GetClairResultByRepoName(repoName string) (string, error) {
	sql := `select clair_result from clair_result where repo_name = ?`
	var result string
	err := GetOrmer().Raw(sql, repoName).QueryRow(&result)
	return result, err
}

func UpdateClairResult(imagename string, result string, highNum int, otherNum int, CVE_status int, other_status int) error {
	o := GetOrmer()

	sql := "update clair_result set clair_result= ?, high_num = ?, other_num = ?, CVE_status = ?,other_status = ?   where repo_name = ?;"

	_, err := o.Raw(sql, result, highNum, otherNum, CVE_status, other_status, imagename).Exec()

	return err
}

func UpdateClairResultSize(imagename string, result string, highNum int, otherNum int, CVE_status int, other_status int,image_size float64) error {
	o := GetOrmer()

	sql := "update clair_result set clair_result= ?, high_num = ?, other_num = ?, CVE_status = ?,other_status = ?,image_size=?   where repo_name = ?;"

	_, err := o.Raw(sql, result, highNum, otherNum, CVE_status, other_status,image_size, imagename).Exec()

	return err
}

func GetCVEStatusByRepoName(repoName string) (int, error) {
	sql := `select CVE_status from clair_result where repo_name = ?`
	var cveStatus int
	err := GetOrmer().Raw(sql, repoName).QueryRow(&cveStatus)
	return cveStatus, err
}

func GetClairRepoNameByUserName(userName string) ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select cr.clair_id,cr.repo_name,cr.user_name,cr.project_name,cr.high_num,cr.other_num,cr.clair_result,cr.CVE_status from clair_result cr where cr.user_name = ?`

	var result []models.ClairResult

	if _, err := o.Raw(sql, userName).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetClairRepoNameByProjectName(projectName string) ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select cr.clair_id,cr.repo_name,cr.user_name,cr.project_name,cr.high_num,cr.other_num,cr.clair_result,cr.CVE_status from clair_result cr where cr.project_name = ?`

	var result []models.ClairResult

	if _, err := o.Raw(sql, projectName).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetClairAllRepoName() ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select cr.clair_id,cr.repo_name,cr.user_name,cr.project_name,cr.high_num,cr.other_num,cr.clair_result,cr.CVE_status from clair_result cr`
	var result []models.ClairResult

	if _, err := o.Raw(sql).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetOtherStatusByRepoName(repoNameList string) ([]models.ClairResult, error) {
	o := GetOrmer()
	sql := `select other_status from clair_result where repo_name in ` + repoNameList
	var result []models.ClairResult

	if _, err := o.Raw(sql).QueryRows(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func InitCVE() error {
	o := GetOrmer()

	sql := "update clair_result set CVE_status= 0"

	_, err := o.Raw(sql).Exec()

	return err
}

func GetOtherStatueByRepoName(repoName string) (int, error) {
	sql := `select other_status from clair_result where repo_name = ?`
	var otherNum int
	err := GetOrmer().Raw(sql, repoName).QueryRow(&otherNum)
	return otherNum, err
}

func CheckRepoName(repoName string) (bool, error) {
	sql := `select count(clair_id) from clair_result where repo_name = ?`
	var num int
	err := GetOrmer().Raw(sql, repoName).QueryRow(&num)

	if num == 0 {
		return false, err
	} else {
		return true, err
	}
}

func UpdateCliarResult(repo_name string, user_name string, project_name string, high_num int, other_num int, clair_result string, other_status int, CVE_status int) (int64, error) {
	o := GetOrmer()
	sql := `update clair_result set user_name = ?, project_name = ?,high_num = ?,other_num = ?, clair_result = ?,other_status = ?,CVE_status = ? where repo_name = ?`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(user_name, project_name, high_num, other_num, clair_result, other_status, CVE_status, repo_name)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	return id, err
}

func UpdateImageSize(repo_name string, image_size float64) (int64, error) {
	o := GetOrmer()
	sql := `update clair_result set image_size = ? where repo_name = ?`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(image_size, repo_name)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	return id, err
}
