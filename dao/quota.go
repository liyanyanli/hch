package dao

//	"github.com/vmware/harbor/models"

//add objectquota to sql
func AddProjectQuota(project_id int64, project_name string, quota_num int, use_num int, quota_size float64, use_size float64) (int64, error) {
	o := GetOrmer()
	sql := `insert into project_quota(project_id,project_name , quota_num, use_num , quota_size,  use_size ) values(?,?,?,?,?,?)`
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(project_id, project_name, quota_num, use_num, quota_size, use_size)
	if err != nil {
		return 0, err
	}
	quotaid, err := r.LastInsertId()
	return quotaid, err
}

//set the quota of the project
func UpdateProjectQuota(project_name string, quota_num int, quota_size float64) (int64, error) {
	o := GetOrmer()

	sql := "update project_quota set quota_num= ?, quota_size = ?  where project_name = ?;"
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(quota_num, quota_size, project_name)
	id, err := r.LastInsertId()
	return id, err
}

//set the used quota of the project
func UpdateProjectQuotaUsedSize(project_name string, used_size float64) (int64, error) {
	o := GetOrmer()

	sql := "update project_quota set  use_size = ?  where project_name = ?;"
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(used_size, project_name)
	id, err := r.LastInsertId()
	return id, err
}

//add project quota used size for upload image
func AddProjectQuotaUsedSize(project_name string, imageSize float64) (int64, error) {
	o := GetOrmer()

	sql := "update project_quota set  use_size = use_size + ?  where project_name = ?;"
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(imageSize, project_name)
	id, err := r.LastInsertId()
	return id, err
}

func UpdateProjectQuotaUsedNum(project_name string, used_num int) (int64, error) {
	o := GetOrmer()

	sql := "update project_quota set use_num= ?  where project_name = ?;"
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(used_num, project_name)
	id, err := r.LastInsertId()
	return id, err
}

func AddProjectQuotaUsedNum(project_name string) (int64, error) {
	o := GetOrmer()
	sql := "update project_quota set use_num= use_num + 1 where project_name = ?;"
	p, err := o.Raw(sql).Prepare()
	if err != nil {
		return 0, err
	}
	r, err := p.Exec(project_name)
	id, err := r.LastInsertId()
	return id, err
}

//update the use of the quota
func UpdateUseQuota(project_name string, use_num int, use_size float64) error {
	o := GetOrmer()
	sql := "update project_quota set use_num= ?, use_size = ?  where project_name = ?;"
	_, err := o.Raw(sql, use_num, use_size, project_name).Exec()
	return err
}

//update the use of the quota while delete tag
func UpdateUseQuotaForDelete(project_name string, repoName string) error {
	o := GetOrmer()
	sql := `update project_quota set use_num = use_num - 1, use_size = use_size - (select image_size from clair_result where repo_name = ?)
           where exists (select image_size from clair_result where repo_name = ?) and project_name= ?;`
	_, err := o.Raw(sql, repoName, repoName, project_name).Exec()
	return err
}

//get the use of the project quota
func GetUseSizeByProject(project_name string) (float64, error) {
	sql := `select use_size from project_quota where project_name = ?`
	var size float64
	err := GetOrmer().Raw(sql, project_name).QueryRow(&size)
	return size, err
}

//get the use of the project quota
func GetUseNumByProject(project_name string) (int, error) {
	sql := `select use_num from project_quota where project_name = ?`
	var num int
	err := GetOrmer().Raw(sql, project_name).QueryRow(&num)
	return num, err
}

//get the use of the project quota
func GetSizeOfImage(image_name string) (float64, error) {
	sql := `select image_size from clair_result where repo_name = ?`
	var image_size float64
	err := GetOrmer().Raw(sql, image_name).QueryRow(&image_size)
	return image_size, err
}

//get the quota information of the project by projectname
func GetInforByProject(project_name string) (int, float64, int, float64, error) {
	sql := `select use_num ,use_size,quota_num,quota_size from project_quota where project_name = ?`
	var use_num, quota_num int
	var use_size, quota_size float64
	err := GetOrmer().Raw(sql, project_name).QueryRow(&use_num, &use_size, &quota_num, &quota_size)
	return use_num, use_size, quota_num, quota_size, err
}

//@lili update the quota  of the project.
func UpdateQuota(projectId int64, quota_num int, quota_size float64) error {
	o := GetOrmer()
	sql := "update project_quota set quota_num = ? ,quota_size=? where project_id = ?"
	_, err := o.Raw(sql, quota_num, quota_size, projectId).Exec()
	return err
}

// DeleteProjectQuota ...
func DeleteQuota(id int64) error {
	sql := `update project_quota 
		set deleted = 1,project_name = concat(project_name,"#",project_id) 
		where project_id = ?`
	_, err := GetOrmer().Raw(sql, id).Exec()
	return err
}

//delete image from clair_result
/*
func DelectImage(repo_name string) error {
	sql := `update clair_result
		set deleted = 1,repo_name = concat(repo_name,"#",clair_id)
		where repo_name = ?`
	_, err := GetOrmer().Raw(sql, repo_name).Exec()
	return err
}
*/
//delete image from clair_result
func DelectImage(repo_name string) error {
	sql := `delete from clair_result where repo_name=?`
	_, err := GetOrmer().Raw(sql, repo_name).Exec()
	return err
}

//delete image size from quota,cut the size and num from use
